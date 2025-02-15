package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/mujehoxe/blogo/docs"
)

// @title Blog API
// @version 1.0
// @description API for managing blog posts.
// @BasePath /

// BlogPost represents a blog post with metadata
// @swagger:model
type BlogPost struct {
	ID              int64    `json:"id"`
	Title           string   `json:"title"`
	MetaDescription string   `json:"meta_description"`
	FocusKeyword    string   `json:"focus_keyword"`
	UrlKeyword      string   `json:"url_keyword"`
	Image           string   `json:"image"`
	Tags            []string `json:"tags"`
	Topic           string   `json:"topic"`
	Service         string   `json:"service"`
	Industry        string   `json:"industry"`
	Priority        string   `json:"priority" enums:"maximum,high,normal"`
	Description     string   `json:"description"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

// SEOData represents SEO metadata for a blog post
// @swagger:model
type SEOData struct {
	// The JSON-LD context
	Context string `json:"@context"`

	// The type of schema (e.g., BlogPosting)
	Type string `json:"@type"`

	// The headline of the blog post
	Headline string `json:"headline"`

	// The keywords associated with the blog post
	Keywords string `json:"keywords"`

	// The image URL for the blog post
	Image string `json:"image"`

	// The canonical URL of the blog post
	URL string `json:"url"`
}

// URL represents an entry in the sitemap
// @swagger:model
type URL struct {
	// The URL of the blog post
	Loc string `xml:"loc" json:"loc"`

	// The change frequency of the URL
	Change string `xml:"changefreq" json:"changefreq"`

	// The priority of the URL in the sitemap
	Priority string `xml:"priority" json:"priority"`
}

// Sitemap represents the structure of the sitemap.xml
// @swagger:model
type Sitemap struct {
	XMLName xml.Name `xml:"urlset" json:"-"`

	// List of URLs in the sitemap
	Urls []URL `xml:"url" json:"urls"`
}

// PaginatedResponse represents a paginated list of blog posts
// @swagger:model
type PaginatedResponse struct {
	// List of blog posts
	Posts []BlogPost `json:"posts"`

	// Total number of blog posts
	TotalPosts int `json:"totalPosts"`

	// Current page number
	Page int `json:"page"`

	// Number of items per page
	PageSize int `json:"pageSize"`

	// Total number of pages
	TotalPages int `json:"totalPages"`
}

var db *sql.DB

// Add transaction wrapper
func withTransaction(fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func writeErrorResponse(w http.ResponseWriter, status int, message string) error {
	return writeJSONResponse(w, status, map[string]string{"error": message})
}

const (
	maxFileSize       = 10 << 20 // 10MB
	allowedImageTypes = "image/jpeg,image/png,image/gif"
)

func validateAndSaveFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// Check file size
	if header.Size > maxFileSize {
		return "", fmt.Errorf("file size exceeds maximum allowed size")
	}

	// Check file type
	contentType := header.Header.Get("Content-Type")
	if !strings.Contains(allowedImageTypes, contentType) {
		return "", fmt.Errorf("unsupported file type: %s", contentType)
	}

	// Create safe filename
	filename := filepath.Clean(header.Filename)
	filepath := filepath.Join("uploads", filename)

	// Save file with proper permissions
	dst, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(filepath) // Cleanup on failure
		return "", err
	}

	return filepath, nil
}

var PriorityWeight = map[string]int{
	"maximum": 3,
	"high":    2,
	"normal":  1,
}

// Create a handler wrapper for non-HandlerFunc handlers
func wrapHandler(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}
}

// CORS middleware to handle cross-origin requests
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func fileServerHandler(dir string) http.HandlerFunc {
	// Create a file server handler for the uploads directory
	fs := http.FileServer(http.Dir(dir))

	return func(w http.ResponseWriter, r *http.Request) {
		// Remove "/uploads/" prefix from the URL path
		urlPath := strings.TrimPrefix(r.URL.Path, "/uploads/")

		// Security check: prevent directory traversal
		if strings.Contains(urlPath, "..") {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// Clean the path
		cleanPath := filepath.Clean(urlPath)

		// Update the request URL path
		r.URL.Path = cleanPath

		// Set headers for image caching (optional)
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		w.Header().Set("Expires", "31536000")

		// Serve the file
		fs.ServeHTTP(w, r)
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Warning: No .env file found. Using default values if available.")
	}

	// Initialize SQLite database
	var err error
	db, err = sql.Open("sqlite3", "../blog.db")
	if err != nil {
		log.Fatal("❌ Failed to open database:", err)
	}
	defer db.Close()

	// Create tables if they don't exist
	createTables()

	// Apply CORS middleware to all routes
	http.HandleFunc("/blog", corsMiddleware(createBlogHandler))
	http.HandleFunc("/blog/", corsMiddleware(blogHandler))
	http.HandleFunc("/blogs", corsMiddleware(listBlogsHandler))
	http.HandleFunc("/sitemap.xml", corsMiddleware(sitemapHandler))

	// For the swagger handler, we need to wrap it since it's an http.Handler
	http.HandleFunc("/swagger/", corsMiddleware(wrapHandler(httpSwagger.WrapHandler)))

	http.HandleFunc("/uploads/", corsMiddleware(fileServerHandler("./uploads")))

	log.Println("🚀 Server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func createTables() {
	query := `
	CREATE TABLE IF NOT EXISTS blog_posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		meta_description TEXT,
		focus_keyword TEXT,
		url_keyword TEXT NOT NULL,
		image TEXT,
		tags TEXT,
		topic TEXT,
		service TEXT,
		industry TEXT,
		priority TEXT,
		description TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_url_keyword ON blog_posts(url_keyword);
	CREATE INDEX IF NOT EXISTS idx_priority ON blog_posts(priority);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("❌ Failed to create tables:", err)
	}
}

// listBlogsHandler handles listing blogs with pagination
// @Summary List blog posts
// @Description Get a paginated list of blog posts
// @Tags blogs
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Number of items per page"
// @Success 200 {object} PaginatedResponse
// @Failure 500 {object} map[string]string
// @Router /blogs [get]
func listBlogsHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	sortByPriority := r.URL.Query().Get("sort") == "priority"

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// Count total posts
	var totalPosts int
	err := db.QueryRow("SELECT COUNT(*) FROM blog_posts").Scan(&totalPosts)
	if err != nil {
		http.Error(w, "Could not count blog posts", http.StatusInternalServerError)
		return
	}

	// Prepare query
	query := "SELECT * FROM blog_posts"
	if sortByPriority {
		query += " ORDER BY CASE priority WHEN 'maximum' THEN 1 WHEN 'high' THEN 2 WHEN 'normal' THEN 3 ELSE 4 END"
	}
	query += " LIMIT ? OFFSET ?"

	offset := (page - 1) * pageSize
	rows, err := db.Query(query, pageSize, offset)
	if err != nil {
		http.Error(w, "Could not fetch blog posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []BlogPost
	for rows.Next() {
		var tagsJSON string
		var post BlogPost
		err := rows.Scan(
			&post.ID, &post.Title, &post.MetaDescription, &post.FocusKeyword,
			&post.UrlKeyword, &post.Image, &tagsJSON, &post.Topic,
			&post.Service, &post.Industry, &post.Priority, &post.Description,
			&post.CreatedAt, &post.UpdatedAt,
		)
		// Unmarshal the tags JSON if it's not empty
		if tagsJSON != "" {
			if err := json.Unmarshal([]byte(tagsJSON), &post.Tags); err != nil {
				fmt.Println("Error unmarshaling tags:", err)
				http.Error(w, "Error unmarshaling tags", http.StatusInternalServerError)
				post.Tags = []string{}
			}
		}

		if err != nil {
			continue
		}
		posts = append(posts, post)
	}

	totalPages := (totalPosts + pageSize - 1) / pageSize

	response := PaginatedResponse{
		Posts:      posts,
		TotalPosts: totalPosts,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// blogHandler retrieves a blog post by its URL keyword
// @Summary Get a blog post
// @Description Retrieve a blog post by its URL keyword
// @Tags blogs
// @Accept json
// @Produce json
// @Param urlKeyword path string true "URL Keyword of the blog post"
// @Success 200 {object} BlogPost
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /blog/{urlKeyword} [get]
func blogHandler(w http.ResponseWriter, r *http.Request) {
	urlKeyword := r.URL.Path[len("/blog/"):]

	var tagsJSON string

	var blog BlogPost
	err := db.QueryRow(`
		SELECT id, title, meta_description, focus_keyword, url_keyword,
			image, tags, topic, service, industry, priority, description,
		  created_at, updated_at
		FROM blog_posts WHERE url_keyword = ?`, urlKeyword).Scan(
		&blog.ID, &blog.Title, &blog.MetaDescription, &blog.FocusKeyword,
		&blog.UrlKeyword, &blog.Image, &tagsJSON, &blog.Topic,
		&blog.Service, &blog.Industry, &blog.Priority, &blog.Description,
		&blog.CreatedAt, &blog.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Blog post not found", http.StatusNotFound)
		return
	} else if err != nil {
		fmt.Println(err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Unmarshal the tags JSON if it's not empty
	if tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &blog.Tags); err != nil {
			fmt.Println("Error unmarshaling tags:", err)
			http.Error(w, "Error unmarshaling tags", http.StatusInternalServerError)
			blog.Tags = []string{}
		}
	}

	blogURL := "/blog/" + blog.UrlKeyword

	seoData := SEOData{
		Context:  "https://schema.org",
		Type:     "BlogPosting",
		Headline: blog.Title,
		Keywords: blog.FocusKeyword,
		Image:    blog.Image,
		URL:      blogURL,
	}

	response := map[string]interface{}{
		"blog":      blog,
		"seoData":   seoData,
		"canonical": blogURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sitemapHandler generates a sitemap
// @Summary Generate sitemap.xml
// @Description Generate an XML sitemap of blog posts
// @Tags sitemap
// @Produce xml
// @Success 200 {object} Sitemap
// @Failure 500 {object} map[string]string
// @Router /sitemap.xml [get]
func sitemapHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT url_keyword, priority FROM blog_posts")

	if err != nil {
		http.Error(w, "Could not generate sitemap", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var urls []URL

	for rows.Next() {
		var urlKeyword, priority string
		if err := rows.Scan(&urlKeyword, &priority); err != nil {
			continue
		}

		urls = append(urls, URL{
			Loc:      "/blog/" + urlKeyword,
			Change:   "weekly",
			Priority: priority,
		})
	}

	sitemap := Sitemap{Urls: urls}
	w.Header().Set("Content-Type", "application/xml")
	xml.NewEncoder(w).Encode(sitemap)
}

// createBlogHandler creates a new blog post with image upload
// @Summary Create a new blog post
// @Description Create a new blog post with metadata and an optional image upload
// @Tags blogs
// @Accept multipart/form-data
// @Produce json
// @Param title formData string true "Title"
// @Param meta_description formData string false "Meta Description"
// @Param focus_keyword formData string false "Focus Keyword"
// @Param url_keyword formData string true "URL Keyword"
// @Param tags formData array false "Tags (comma-separated values or multiple fields)"
// @Param topic formData string false "Topic"
// @Param service formData string false "Service"
// @Param industry formData string false "Industry"
// @Param priority formData string false "Priority"
// @Param description formData string true "Description"
// @Param image formData file false "Image file (optional)"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /blog [post]
func createBlogHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	blog, err := validateBlogPost(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Handle file upload
	if file, header, err := r.FormFile("image"); err == nil {
		if filepath, err := validateAndSaveFile(file, header); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid file: %v", err))
			return
		} else {
			blog.Image = filepath
		}
	}

	tagsJSON, err := json.Marshal(blog.Tags)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to process tags")
		return
	}

	// Use transaction for database operation
	err = withTransaction(func(tx *sql.Tx) error {
		result, err := tx.Exec(`
        INSERT INTO blog_posts (
            title, meta_description, focus_keyword, url_keyword,
            image, tags, topic, service, industry, priority, description
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			blog.Title, blog.MetaDescription, blog.FocusKeyword, blog.UrlKeyword,
			blog.Image, string(tagsJSON), blog.Topic, blog.Service, blog.Industry,
			blog.Priority, blog.Description,
		)
		if err != nil {
			return err
		}
		blog.ID, err = result.LastInsertId()
		return err
	})

	if err != nil {
		if blog.Image != "" {
			os.Remove(blog.Image) // Cleanup uploaded file on DB failure
		}
		fmt.Println(err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create blog post")
		return
	}

	if err := writeJSONResponse(w, http.StatusCreated, map[string]interface{}{
		"message": "Blog post created successfully",
		"url":     "/blog/" + blog.UrlKeyword,
		"id":      blog.ID,
		"image":   blog.Image,
		"tags":    blog.Tags,
	}); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func validateBlogPost(r *http.Request) (BlogPost, error) {
	var blog BlogPost

	// Required field validation
	blog.Title = strings.TrimSpace(r.FormValue("title"))
	if blog.Title == "" {
		return blog, fmt.Errorf("title is required")
	}

	blog.Description = strings.TrimSpace(r.FormValue("description"))
	if blog.Description == "" {
		return blog, fmt.Errorf("description is required")
	}

	blog.UrlKeyword = strings.TrimSpace(r.FormValue("url_keyword"))
	if blog.UrlKeyword == "" {
		return blog, fmt.Errorf("url_keyword is required")
	}

	// Validate URL keyword format (alphanumeric with hyphens)
	if !regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString(blog.UrlKeyword) {
		return blog, fmt.Errorf("url_keyword must contain only letters, numbers, and hyphens")
	}

	// Validate priority
	blog.Priority = strings.TrimSpace(r.FormValue("priority"))
	if blog.Priority != "" {
		validPriorities := map[string]bool{
			"maximum": true,
			"high":    true,
			"normal":  true,
		}
		if !validPriorities[blog.Priority] {
			return blog, fmt.Errorf("invalid priority value: must be maximum, high, or normal")
		}
	} else {
		blog.Priority = "normal" // Default priority
	}

	// Process and validate tags
	var tags []string
	if rawTags := r.Form["tags"]; len(rawTags) > 0 {
		for _, tagField := range rawTags {
			splitTags := strings.Split(tagField, ",")
			for _, tag := range splitTags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					// Optional: Add more specific tag validation here
					// e.g., length limits, character restrictions
					if len(tag) > 50 {
						return blog, fmt.Errorf("tag length cannot exceed 50 characters: %s", tag)
					}
					tags = append(tags, tag)
				}
			}
		}
	}
	blog.Tags = tags

	// Optional fields
	blog.MetaDescription = strings.TrimSpace(r.FormValue("meta_description"))
	blog.FocusKeyword = strings.TrimSpace(r.FormValue("focus_keyword"))
	blog.Topic = strings.TrimSpace(r.FormValue("topic"))
	blog.Service = strings.TrimSpace(r.FormValue("service"))
	blog.Industry = strings.TrimSpace(r.FormValue("industry"))

	// Optional field validations
	if len(blog.MetaDescription) > 160 {
		return blog, fmt.Errorf("meta description cannot exceed 160 characters")
	}

	if len(blog.Title) > 100 {
		return blog, fmt.Errorf("title cannot exceed 100 characters")
	}

	// Check for duplicate URL keyword
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM blog_posts WHERE url_keyword = ?)", blog.UrlKeyword).Scan(&exists)
	if err != nil {
		return blog, fmt.Errorf("failed to check URL keyword uniqueness: %v", err)
	}
	if exists {
		return blog, fmt.Errorf("url_keyword already exists")
	}

	return blog, nil
}

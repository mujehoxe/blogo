package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/mujehoxe/blogo/docs"
)

// @title Blog API
// @version 1.0
// @description API for managing blog posts.
// @host localhost:8080
// @BasePath /

// BlogPost represents a blog post with metadata
// @swagger:model
type BlogPost struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	MetaDescription string `json:"meta_description"`
	FocusKeyword    string `json:"focus_keyword"`
	UrlKeyword      string `json:"url_keyword"`
	Image           string `json:"image"`
	Tags            string `json:"tags"`
	Topic           string `json:"topic"`
	Service         string `json:"service"`
	Industry        string `json:"industry"`
	Priority        string `json:"priority" enums:"maximum,high,normal"`
	Description     string `json:"description"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
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
var domain string

var PriorityWeight = map[string]int{
	"maximum": 3,
	"high":    2,
	"normal":  1,
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Warning: No .env file found. Using default values if available.")
	}

	domain = os.Getenv("BASE_URL")
	if domain == "" {
		log.Fatal("❌ BASE_URL is not set in the environment variables or .env file")
	}

	// Initialize SQLite database
	var err error
	db, err = sql.Open("sqlite3", "./blog.db")
	if err != nil {
		log.Fatal("❌ Failed to open database:", err)
	}
	defer db.Close()

	// Create tables if they don't exist
	createTables()

	http.HandleFunc("/blog", createBlogHandler)
	http.HandleFunc("/blog/", blogHandler)
	http.HandleFunc("/blogs", listBlogsHandler)
	http.HandleFunc("/sitemap.xml", sitemapHandler)
	http.HandleFunc("/swagger/", httpSwagger.WrapHandler)

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
		var post BlogPost
		err := rows.Scan(
			&post.ID, &post.Title, &post.MetaDescription, &post.FocusKeyword,
			&post.UrlKeyword, &post.Image, &post.Tags, &post.Topic,
			&post.Service, &post.Industry, &post.Priority, &post.Description,
			&post.CreatedAt, &post.UpdatedAt,
		)
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

	var blog BlogPost
	err := db.QueryRow(`
		SELECT id, title, meta_description, focus_keyword, url_keyword,
			image, tags, topic, service, industry, priority, description,
		  created_at, updated_at
		FROM blog_posts WHERE url_keyword = ?`, urlKeyword).Scan(
		&blog.ID, &blog.Title, &blog.MetaDescription, &blog.FocusKeyword,
		&blog.UrlKeyword, &blog.Image, &blog.Tags, &blog.Topic,
		&blog.Service, &blog.Industry, &blog.Priority, &blog.Description,
		&blog.CreatedAt, &blog.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Blog post not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	blogURL := domain + "/blog/" + blog.UrlKeyword

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
			Loc:      domain + "/blog/" + urlKeyword,
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
// @Param tags formData string false "Tags"
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
	err := r.ParseMultipartForm(10 << 20) // Limit file size to 10MB
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Extract form values
	blog := BlogPost{
		Title:           r.FormValue("title"),
		MetaDescription: r.FormValue("meta_description"),
		FocusKeyword:    r.FormValue("focus_keyword"),
		UrlKeyword:      r.FormValue("url_keyword"),
		Tags:            r.FormValue("tags"),
		Topic:           r.FormValue("topic"),
		Service:         r.FormValue("service"),
		Industry:        r.FormValue("industry"),
		Priority:        r.FormValue("priority"),
		Description:     r.FormValue("description"),
	}

	if blog.UrlKeyword == "" || blog.Title == "" || blog.Description == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Handle image upload
	file, header, err := r.FormFile("image")
	if err == nil { // File exists
		defer file.Close()

		// Ensure uploads directory exists
		uploadDir := "./uploads"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.Mkdir(uploadDir, os.ModePerm)
		}

		// Save file
		filePath := uploadDir + "/" + header.Filename
		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Failed to save image", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		_, err = dst.ReadFrom(file)
		if err != nil {
			http.Error(w, "Failed to write image file", http.StatusInternalServerError)
			return
		}

		blog.Image = filePath // Save image path
	}

	// Insert into database
	result, err := db.Exec(`
		INSERT INTO blog_posts (
			title, meta_description, focus_keyword, url_keyword,
			image, tags, topic, service, industry, priority, description
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		blog.Title, blog.MetaDescription, blog.FocusKeyword, blog.UrlKeyword,
		blog.Image, blog.Tags, blog.Topic, blog.Service, blog.Industry,
		blog.Priority, blog.Description,
	)

	if err != nil {
		fmt.Print(err)
		http.Error(w, "Failed to create blog post", http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	blog.ID = id

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Blog post created successfully",
		"url":     domain + "/blog/" + blog.UrlKeyword,
		"id":      id,
		"image":   blog.Image,
	})
}

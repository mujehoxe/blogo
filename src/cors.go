package main

import "net/http"

// CORSMiddleware wraps an http.HandlerFunc and adds CORS headers
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from localhost on common development ports
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000") // React default port
		// Add other common development ports if needed
		// w.Header().Add("Access-Control-Allow-Origin", "http://localhost:8000")
		// w.Header().Add("Access-Control-Allow-Origin", "http://localhost:5173") // Vite default port

		// Allow common HTTP methods
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// Allow common headers
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Allow credentials (if needed)
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

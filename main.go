package main

import (
	"bufio"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"s3go/internal/api"
	"s3go/internal/db"
)

//go:embed static/*
var staticFS embed.FS

// loadEnv loads environment variables from a local .env file if present
func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return // Ignore if .env file is missing, fallback to actual system env vars
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove surrounding quotes if they exist
			value = strings.Trim(value, `"'`)

			// Only set env var if it doesn't already exist in the environment
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, value)
			}
		}
	}
}

func main() {
	// Load environment variables from .env file
	loadEnv()

	// Pre-flight check: Verify S3GO_ENCRYPTION_KEY is set
	if os.Getenv("S3GO_ENCRYPTION_KEY") == "" {
		log.Fatalf("FATAL: S3GO_ENCRYPTION_KEY environment variable is not set! Please set S3GO_ENCRYPTION_KEY in your environment or .env file.")
	}

	// Initialize SQLite Database
	dbPath := "s3go.db"
	if err := db.InitDB(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Local JSON database initialized successfully")

	// API Routes (using Go 1.22 path patterns)
	http.HandleFunc("POST /api/connections", api.CreateConnectionHandler)
	http.HandleFunc("GET /api/connections", api.ListConnectionsHandler)
	http.HandleFunc("DELETE /api/connections/{id}", api.DeleteConnectionHandler)
	http.HandleFunc("PUT /api/connections/{id}", api.UpdateConnectionHandler)

	http.HandleFunc("GET /api/connections/{id}/files", api.ListFilesHandler)
	http.HandleFunc("POST /api/connections/{id}/presigned-url", api.GeneratePresignedURLHandler)
	http.HandleFunc("DELETE /api/connections/{id}/files", api.DeleteFilesHandler)
	http.HandleFunc("GET /api/connections/{id}/preview", api.GetFilePreviewHandler)

	// Serve Static Files from Embedded FS
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("Failed to load embedded static folder: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(subFS)))

	// Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("S3Go Server starting on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

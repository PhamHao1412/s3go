package main

import (
	"embed"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"

	"s3go/cmd/serverd/router"
	"s3go/internal/app"
	v1 "s3go/internal/controller/rest/v1"
	s3Infra "s3go/internal/infra/s3"
	"s3go/internal/persistence"
	"s3go/internal/persistence/connection"
	connectionService "s3go/internal/service/connection"
	s3Service "s3go/internal/service/s3"

	"database/sql"
	_ "github.com/lib/pq"
)

//go:embed static/*
var staticFS embed.FS

func initDatabase(cfg app.Config) (*sql.DB, string, error) {
	if cfg.DatabaseURL != "" {
		db, err := sql.Open("postgres", cfg.DatabaseURL)
		if err != nil {
			return nil, "", err
		}

		if err := db.Ping(); err != nil {
			db.Close()
			return nil, "", err
		}

		// Auto schema migration
		query := `
		CREATE TABLE IF NOT EXISTS connections (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			access_key VARCHAR(255) NOT NULL,
			secret_key_encrypted TEXT NOT NULL,
			region VARCHAR(255) NOT NULL,
			bucket VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL
		);`
		if _, err := db.Exec(query); err != nil {
			db.Close()
			return nil, "", err
		}

		log.Println("DB: Successfully connected and initialized PostgreSQL database")
		return db, "", nil
	}

	// Local JSON Fallback mode
	jsonFilePath := cfg.DBPath
	if jsonFilePath == "" {
		jsonFilePath = "connections.json"
	}

	if err := connection.EnsureDBFileExists(jsonFilePath); err != nil {
		return nil, "", err
	}

	log.Printf("DB: Successfully initialized local JSON database at %s", jsonFilePath)
	return nil, jsonFilePath, nil
}

func main() {
	// 1. Load Configurations
	cfg := app.LoadConfig()

	// 2. Pre-flight check: Verify S3GO_ENCRYPTION_KEY is set
	if cfg.EncryptionKey == "" {
		log.Fatalf("FATAL: S3GO_ENCRYPTION_KEY environment variable is not set! Please set S3GO_ENCRYPTION_KEY in your environment or .env file.")
	}

	// 3. Initialize Database
	pgDB, jsonFilePath, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 4. Instantiate Unit of Work and Gateways
	readOnlyUOW := persistence.NewReadOnlyUnitOfWork(pgDB, jsonFilePath)
	writableUOW := persistence.NewWritableUnitOfWork(pgDB, jsonFilePath)
	s3GW := s3Infra.NewClient()

	// 5. Instantiate Services
	connSvc := connectionService.New(readOnlyUOW, writableUOW, s3GW)
	s3Svc := s3Service.New(readOnlyUOW, s3GW)

	// 6. Instantiate Controllers
	ctrlV1 := v1.New(connSvc, s3Svc)

	// 7. Initialize Router and Register Routes
	rtr := router.New(ctrlV1)
	mux := http.DefaultServeMux
	rtr.RegisterRoutes(mux)

	// 8. Serve Embedded Static Files
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("Failed to load embedded static folder: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(subFS)))

	// 9. Start Server with IP whitelisting middleware
	mainHandler := IPMiddleware(mux, cfg.AllowedIPs)
	log.Printf("S3Go Server starting on http://localhost:%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mainHandler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// getClientIP extracts the real client IP address, handling proxy headers from Render/Cloudflare.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	ip := r.RemoteAddr
	if strings.Contains(ip, ":") {
		host, _, err := net.SplitHostPort(ip)
		if err == nil {
			return host
		}
	}
	return ip
}

// IPMiddleware blocks access to the application if the client IP is not whitelisted.
func IPMiddleware(next http.Handler, allowedIPsStr string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		log.Printf("INFO: Incoming request from IP: %s (%s %s)", clientIP, r.Method, r.URL.Path)

		if allowedIPsStr == "" {
			// Whitelist is not configured, bypass checks
			next.ServeHTTP(w, r)
			return
		}

		// Parse comma-separated allowed IPs
		allowedIPs := make(map[string]bool)
		for _, ip := range strings.Split(allowedIPsStr, ",") {
			trimmed := strings.TrimSpace(ip)
			if trimmed != "" {
				allowedIPs[trimmed] = true
			}
		}

		// Always allow local loopback for easy local development/testing
		if clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost" {
			next.ServeHTTP(w, r)
			return
		}

		if !allowedIPs[clientIP] {
			log.Printf("SECURITY: Access blocked for unauthorized IP: %s (Request: %s %s)", clientIP, r.Method, r.URL.Path)
			http.Error(w, "Forbidden: Access denied from your IP address.", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

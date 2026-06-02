package app

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	EncryptionKey string
	DatabaseURL   string
	DBPath        string
	AllowedIPs    string
	Port          string
	BypassBucket  string
	BypassFolder  string
}

// LoadConfig loads environment variables and returns a Config struct.
func LoadConfig() Config {
	loadEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		EncryptionKey: os.Getenv("S3GO_ENCRYPTION_KEY"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		DBPath:        os.Getenv("DB_PATH"),
		AllowedIPs:    os.Getenv("ALLOWED_IPS"),
		Port:          port,
		BypassBucket:  os.Getenv("BYPASS_BUCKET"),
		BypassFolder:  os.Getenv("BYPASS_FOLDER"),
	}
}

// loadEnv loads environment variables from a local .env file if present.
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

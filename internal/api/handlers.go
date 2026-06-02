package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"s3go/internal/crypto"
	"s3go/internal/db"
	"s3go/internal/s3"

	s3Client "github.com/aws/aws-sdk-go-v2/service/s3"
)

// GenerateUUID generates a random UUID v4 for connection IDs.
func GenerateUUID() string {
	uuid := make([]byte, 16)
	_, _ = rand.Read(uuid)
	// Set version 4 (random) and variant (RFC 4122)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// ErrorResponse represents a standardized error JSON structure.
type ErrorResponse struct {
	Error string `json:"error"`
}

// SendJSON sends a JSON response with status code.
func SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// SendError sends a standardized error response.
func SendError(w http.ResponseWriter, status int, errMsg string) {
	SendJSON(w, status, ErrorResponse{Error: errMsg})
}

// CreateConnectionRequest holds connection payload.
type CreateConnectionRequest struct {
	Name      string `json:"name"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
	Bucket    string `json:"bucket"` // Now optional
}

// CreateConnectionHandler verifies S3 credentials, encrypts Secret Key, and saves the connection.
func CreateConnectionHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Validate required fields (bucket is now optional!)
	if req.Name == "" || req.AccessKey == "" || req.SecretKey == "" || req.Region == "" {
		SendError(w, http.StatusBadRequest, "Fields (name, access_key, secret_key, region) are required")
		return
	}

	// 1. Verify S3 credentials and bucket existence (if specified) before saving
	err := s3.VerifyCredentials(req.AccessKey, req.SecretKey, req.Region, req.Bucket)
	if err != nil {
		SendError(w, http.StatusUnauthorized, fmt.Sprintf("S3 Verification Failed: %v", err))
		return
	}

	// 2. Encrypt Secret Key securely
	encryptedSecret, err := crypto.Encrypt(req.SecretKey)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Failed to encrypt credentials")
		return
	}

	// 3. Save connection configuration to SQLite DB
	conn := &db.Connection{
		ID:                 GenerateUUID(),
		Name:               req.Name,
		AccessKey:          req.AccessKey,
		SecretKeyEncrypted: encryptedSecret,
		Region:             req.Region,
		Bucket:             req.Bucket,
	}

	if err := db.SaveConnection(conn); err != nil {
		SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save connection: %v", err))
		return
	}

	SendJSON(w, http.StatusCreated, map[string]string{
		"status": "success",
		"id":     conn.ID,
	})
}

// UpdateConnectionRequest holds update connection payload.
type UpdateConnectionRequest struct {
	Name      string `json:"name"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"` // Optional
	Region    string `json:"region"`
	Bucket    string `json:"bucket"` // Optional
}

// UpdateConnectionHandler verifies and updates an S3 connection.
func UpdateConnectionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	var req UpdateConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Validate required fields
	if req.Name == "" || req.AccessKey == "" || req.Region == "" {
		SendError(w, http.StatusBadRequest, "Fields (name, access_key, region) are required")
		return
	}

	// Retrieve existing connection configuration
	conn, err := db.GetConnection(id)
	if err != nil {
		SendError(w, http.StatusNotFound, fmt.Sprintf("Connection not found: %v", err))
		return
	}

	var decryptedSecret string
	var encryptedSecret string

	if req.SecretKey == "" {
		// Keep existing secret key. Decrypt it to verify credentials.
		decryptedSecret, err = crypto.Decrypt(conn.SecretKeyEncrypted)
		if err != nil {
			SendError(w, http.StatusInternalServerError, "Failed to decrypt existing credentials")
			return
		}
		encryptedSecret = conn.SecretKeyEncrypted
	} else {
		// Use new secret key
		decryptedSecret = req.SecretKey
		// Encrypt the new secret key securely
		encryptedSecret, err = crypto.Encrypt(req.SecretKey)
		if err != nil {
			SendError(w, http.StatusInternalServerError, "Failed to encrypt new credentials")
			return
		}
	}

	// Verify the updated S3 credentials and bucket existence (if specified) before saving
	err = s3.VerifyCredentials(req.AccessKey, decryptedSecret, req.Region, req.Bucket)
	if err != nil {
		SendError(w, http.StatusUnauthorized, fmt.Sprintf("S3 Verification Failed: %v", err))
		return
	}

	// Update the connection object
	conn.Name = req.Name
	conn.AccessKey = req.AccessKey
	conn.SecretKeyEncrypted = encryptedSecret
	conn.Region = req.Region
	conn.Bucket = req.Bucket

	if err := db.SaveConnection(conn); err != nil {
		SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update connection: %v", err))
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{
		"status": "success",
		"id":     conn.ID,
	})
}

// ListConnectionsHandler returns all saved connections without their encrypted secret keys.
func ListConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	conns, err := db.ListConnections()
	if err != nil {
		SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list connections: %v", err))
		return
	}
	SendJSON(w, http.StatusOK, conns)
}

// DeleteConnectionHandler deletes a connection config.
func DeleteConnectionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	if err := db.DeleteConnection(id); err != nil {
		SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete connection: %v", err))
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// getS3ClientAndBucketForKey resolves connection from DB, decrypts secret key, and dynamically parses
// the S3 bucket name and actual S3 object key from the path if the connection has no specific bucket bound.
func getS3ClientAndBucketForKey(id string, keyOrPrefix string) (*s3Client.Client, string, string, error) {
	conn, err := db.GetConnection(id)
	if err != nil {
		return nil, "", "", fmt.Errorf("connection not found in database: %w", err)
	}

	secretKey, err := crypto.Decrypt(conn.SecretKeyEncrypted)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	client, err := s3.NewS3Client(conn.AccessKey, secretKey, conn.Region)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to initialize S3 client: %w", err)
	}

	// Scenario A: Connection bound to a single bucket
	if conn.Bucket != "" {
		return client, conn.Bucket, keyOrPrefix, nil
	}

	// Scenario B: All-buckets explorer mode
	if keyOrPrefix == "" {
		return client, "", "", nil
	}

	// First part of the key/prefix is the bucket name, remainder is the actual S3 key
	parts := strings.Split(keyOrPrefix, "/")
	bucketName := parts[0]
	actualKey := strings.Join(parts[1:], "/")

	return client, bucketName, actualKey, nil
}

// ListFilesHandler browses and searches S3. If connection has no bucket, the root lists all buckets.
func ListFilesHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	prefix := r.URL.Query().Get("prefix")
	search := r.URL.Query().Get("search")

	// Get S3 Client and verified Bucket name/actual path
	client, bucket, actualPrefix, err := getS3ClientAndBucketForKey(id, prefix)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	// If bucket is empty, list all available buckets in the S3 account!
	if bucket == "" {
		buckets, err := s3.ListBuckets(client)
		if err != nil {
			SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list S3 buckets: %v", err))
			return
		}
		SendJSON(w, http.StatusOK, buckets)
		return
	}

	// Otherwise, list objects inside the resolved bucket
	items, err := s3.ListObjects(client, bucket, actualPrefix, search)
	if err != nil {
		SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list S3 objects: %v", err))
		return
	}

	// If the connection was not bound to a bucket, the prefix contains the bucket name.
	// We need to prepend the bucket name to the returned keys so the frontend navigates correctly!
	conn, _ := db.GetConnection(id)
	if conn != nil && conn.Bucket == "" {
		for i := range items {
			items[i].Key = bucket + "/" + items[i].Key
		}
	}

	SendJSON(w, http.StatusOK, items)
}

// PresignedURLRequest payload for link generation.
type PresignedURLRequest struct {
	Key     string `json:"key"`
	Action  string `json:"action"`  // "download" or "upload"
	Expires int    `json:"expires"` // expiration in minutes
}

// PresignedURLResponse holds generated URL.
type PresignedURLResponse struct {
	URL string `json:"url"`
}

// GeneratePresignedURLHandler generates a download (GET) or upload (PUT) S3 URL.
func GeneratePresignedURLHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	var req PresignedURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Key == "" || req.Action == "" {
		SendError(w, http.StatusBadRequest, "key and action parameters are required")
		return
	}

	if req.Expires <= 0 {
		req.Expires = 15 // Default to 15 minutes
	}

	client, bucket, actualKey, err := getS3ClientAndBucketForKey(id, req.Key)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	expiration := time.Duration(req.Expires) * time.Minute
	var presignedURL string

	if req.Action == "download" {
		presignedURL, err = s3.GeneratePresignedGET(client, bucket, actualKey, expiration)
	} else if req.Action == "upload" {
		presignedURL, err = s3.GeneratePresignedPUT(client, bucket, actualKey, expiration)
	} else {
		SendError(w, http.StatusBadRequest, "action must be either 'download' or 'upload'")
		return
	}

	if err != nil {
		SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate URL: %v", err))
		return
	}

	SendJSON(w, http.StatusOK, PresignedURLResponse{URL: presignedURL})
}

// DeleteFilesRequest holds a list of keys to delete.
type DeleteFilesRequest struct {
	Keys []string `json:"keys"`
}

// DeleteFilesHandler deletes multiple objects from the S3 bucket.
func DeleteFilesHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	var req DeleteFilesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if len(req.Keys) == 0 {
		SendError(w, http.StatusBadRequest, "keys array cannot be empty")
		return
	}

	// Resolve bucket name and client from the first key
	client, bucket, _, err := getS3ClientAndBucketForKey(id, req.Keys[0])
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	// Convert all keys to actual S3 keys (stripping bucket prefix if list-all mode)
	var actualKeys []string
	for _, k := range req.Keys {
		_, _, actKey, err := getS3ClientAndBucketForKey(id, k)
		if err == nil {
			actualKeys = append(actualKeys, actKey)
		}
	}

	err = s3.DeleteObjects(client, bucket, actualKeys)
	if err != nil {
		SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete objects: %v", err))
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// FilePreviewResponse holds file content.
type FilePreviewResponse struct {
	Content string `json:"content"`
}

// GetFilePreviewHandler reads and returns the preview string of text-based files.
func GetFilePreviewHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		SendError(w, http.StatusBadRequest, "key parameter is required")
		return
	}

	client, bucket, actualKey, err := getS3ClientAndBucketForKey(id, key)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	content, err := s3.GetFilePreview(client, bucket, actualKey)
	if err != nil {
		SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read file preview: %v", err))
		return
	}

	SendJSON(w, http.StatusOK, FilePreviewResponse{Content: content})
}

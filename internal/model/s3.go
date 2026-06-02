package model

import "time"

type S3Item struct {
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	IsDir        bool      `json:"is_dir"`
}

type PresignedURLInput struct {
	Key     string `json:"key"`
	Action  string `json:"action"`  // "download" or "upload"
	Expires int    `json:"expires"` // expiration in minutes
}

type PresignedURLResponse struct {
	URL string `json:"url"`
}

type DeleteFilesInput struct {
	Keys []string `json:"keys"`
}

type FilePreviewResponse struct {
	Content string `json:"content"`
}

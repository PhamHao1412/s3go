package model

import "time"

type ConnectionDetail struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	AccessKey string    `json:"access_key"`
	Region    string    `json:"region"`
	Bucket    string    `json:"bucket"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateConnectionInput struct {
	Name      string `json:"name"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
	Bucket    string `json:"bucket"`
}

type CreateConnectionResponse struct {
	Status string `json:"status"`
	ID     string `json:"id"`
}

type UpdateConnectionInput struct {
	Name      string `json:"name"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
	Bucket    string `json:"bucket"`
}

type UpdateConnectionResponse struct {
	Status string `json:"status"`
	ID     string `json:"id"`
}

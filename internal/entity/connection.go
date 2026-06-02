package entity

import "time"

type Connection struct {
	ID                 string
	Name               string
	AccessKey          string
	SecretKeyEncrypted string
	Region             string
	Bucket             string
	CreatedAt          time.Time
}

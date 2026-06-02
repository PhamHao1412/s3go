package s3

import (
	"context"
	"io"
	"time"

	"s3go/internal/model"
)

type Client interface {
	VerifyCredentials(ctx context.Context, accessKey, secretKey, region, bucket string) error
	ListBuckets(ctx context.Context, accessKey, secretKey, region string) ([]model.S3Item, error)
	ListObjects(ctx context.Context, accessKey, secretKey, region, bucket, prefix, search string) ([]model.S3Item, error)
	GeneratePresignedGET(ctx context.Context, accessKey, secretKey, region, bucket, key string, expiration time.Duration) (string, error)
	GeneratePresignedPUT(ctx context.Context, accessKey, secretKey, region, bucket, key string, expiration time.Duration) (string, error)
	DeleteObjects(ctx context.Context, accessKey, secretKey, region, bucket string, keys []string) error
	GetFilePreview(ctx context.Context, accessKey, secretKey, region, bucket, key string) (string, error)
	UploadObject(ctx context.Context, accessKey, secretKey, region, bucket, key string, body io.Reader, size int64, contentType string) error
}

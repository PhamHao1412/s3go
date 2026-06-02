package s3

import (
	"context"
	"fmt"
	"strings"

	s3Infra "s3go/internal/infra/s3"
	"s3go/internal/model"
	"s3go/internal/persistence"
	"s3go/internal/pkg/crypto"
)

type Service interface {
	ListFiles(ctx context.Context, id string, prefix, search string) ([]model.S3Item, error)
	GetFilePreview(ctx context.Context, id string, key string) (string, error)
	GeneratePresignedURL(ctx context.Context, id string, input model.PresignedURLInput) (model.PresignedURLResponse, error)
	DeleteFiles(ctx context.Context, id string, input model.DeleteFilesInput) error
}

type service struct {
	readOnlyUOW persistence.ReadOnlyUnitOfWork
	s3Gateway   s3Infra.Client
}

func New(
	readOnlyUOW persistence.ReadOnlyUnitOfWork,
	s3Gateway s3Infra.Client,
) Service {
	return &service{
		readOnlyUOW: readOnlyUOW,
		s3Gateway:   s3Gateway,
	}
}

// getCredentialsAndBucketForKey resolves credentials and S3 details for a connection and key/prefix.
func (s *service) getCredentialsAndBucketForKey(ctx context.Context, id string, keyOrPrefix string) (accessKey, secretKey, region, bucket, actualKey string, err error) {
	conn, err := s.readOnlyUOW.Connection().GetByID(ctx, id)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("connection not found in database: %w", err)
	}

	secretKey, err = crypto.Decrypt(conn.SecretKeyEncrypted)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// Scenario A: Connection bound to a single bucket
	if conn.Bucket != "" {
		return conn.AccessKey, secretKey, conn.Region, conn.Bucket, keyOrPrefix, nil
	}

	// Scenario B: All-buckets explorer mode
	if keyOrPrefix == "" {
		return conn.AccessKey, secretKey, conn.Region, "", "", nil
	}

	// First part of the key/prefix is the bucket name, remainder is the actual S3 key
	parts := strings.Split(keyOrPrefix, "/")
	bucketName := parts[0]
	actKey := strings.Join(parts[1:], "/")

	return conn.AccessKey, secretKey, conn.Region, bucketName, actKey, nil
}

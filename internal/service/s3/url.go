package s3

import (
	"context"
	"fmt"
	"time"

	"s3go/internal/model"
)

func (s *service) GeneratePresignedURL(ctx context.Context, id string, input model.PresignedURLInput) (model.PresignedURLResponse, error) {
	accessKey, secretKey, region, bucket, actualKey, err := s.getCredentialsAndBucketForKey(ctx, id, input.Key)
	if err != nil {
		return model.PresignedURLResponse{}, err
	}

	expires := input.Expires
	if expires <= 0 {
		expires = 15 // Default to 15 minutes
	}
	expiration := time.Duration(expires) * time.Minute

	var presignedURL string
	if input.Action == "download" {
		presignedURL, err = s.s3Gateway.GeneratePresignedGET(ctx, accessKey, secretKey, region, bucket, actualKey, expiration)
	} else if input.Action == "upload" {
		presignedURL, err = s.s3Gateway.GeneratePresignedPUT(ctx, accessKey, secretKey, region, bucket, actualKey, expiration)
	} else {
		return model.PresignedURLResponse{}, fmt.Errorf("action must be either 'download' or 'upload'")
	}

	if err != nil {
		return model.PresignedURLResponse{}, err
	}

	return model.PresignedURLResponse{URL: presignedURL}, nil
}

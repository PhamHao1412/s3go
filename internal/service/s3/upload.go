package s3

import (
	"context"
	"io"
)

func (s *service) UploadFile(ctx context.Context, id string, key string, body io.Reader, size int64, contentType string) error {
	accessKey, secretKey, region, bucket, actualKey, err := s.getCredentialsAndBucketForKey(ctx, id, key)
	if err != nil {
		return err
	}

	return s.s3Gateway.UploadObject(ctx, accessKey, secretKey, region, bucket, actualKey, body, size, contentType)
}

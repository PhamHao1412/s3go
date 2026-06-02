package s3

import (
	"context"
	"s3go/internal/model"
)

func (s *service) DeleteFiles(ctx context.Context, id string, input model.DeleteFilesInput) error {
	if len(input.Keys) == 0 {
		return nil
	}

	// Resolve bucket name and credentials from the first key
	accessKey, secretKey, region, bucket, _, err := s.getCredentialsAndBucketForKey(ctx, id, input.Keys[0])
	if err != nil {
		return err
	}

	// Convert all keys to actual S3 keys (stripping bucket prefix if list-all mode)
	var actualKeys []string
	for _, k := range input.Keys {
		_, _, _, _, actKey, err := s.getCredentialsAndBucketForKey(ctx, id, k)
		if err == nil {
			actualKeys = append(actualKeys, actKey)
		}
	}

	return s.s3Gateway.DeleteObjects(ctx, accessKey, secretKey, region, bucket, actualKeys)
}

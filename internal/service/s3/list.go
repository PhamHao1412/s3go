package s3

import (
	"context"

	"s3go/internal/model"
)

func (s *service) ListFiles(ctx context.Context, id string, prefix, search string) ([]model.S3Item, error) {
	accessKey, secretKey, region, bucket, actualPrefix, err := s.getCredentialsAndBucketForKey(ctx, id, prefix)
	if err != nil {
		return nil, err
	}

	// If bucket is empty, list all available buckets in the S3 account!
	if bucket == "" {
		buckets, err := s.s3Gateway.ListBuckets(ctx, accessKey, secretKey, region)
		if err != nil {
			return nil, err
		}
		return buckets, nil
	}

	// Otherwise, list objects inside the resolved bucket
	items, err := s.s3Gateway.ListObjects(ctx, accessKey, secretKey, region, bucket, actualPrefix, search)
	if err != nil {
		return nil, err
	}

	// If the connection was not bound to a bucket, the prefix contains the bucket name.
	// We need to prepend the bucket name to the returned keys so the frontend navigates correctly!
	conn, _ := s.readOnlyUOW.Connection().GetByID(ctx, id)
	if conn != nil && conn.Bucket == "" {
		for i := range items {
			items[i].Key = bucket + "/" + items[i].Key
		}
	}

	return items, nil
}

package s3

import "context"

func (s *service) GetFilePreview(ctx context.Context, id string, key string) (string, error) {
	accessKey, secretKey, region, bucket, actualKey, err := s.getCredentialsAndBucketForKey(ctx, id, key)
	if err != nil {
		return "", err
	}

	return s.s3Gateway.GetFilePreview(ctx, accessKey, secretKey, region, bucket, actualKey)
}

package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"s3go/internal/model"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type client struct{}

func NewClient() Client {
	return &client{}
}

// newS3Client creates a new AWS S3 Client using static credentials.
func (c *client) newS3Client(ctx context.Context, accessKey, secretKey, region string) (*awsS3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		)),
	)
	if err != nil {
		return nil, err
	}
	return awsS3.NewFromConfig(cfg), nil
}

// VerifyCredentials checks if the provided credentials are valid.
func (c *client) VerifyCredentials(ctx context.Context, accessKey, secretKey, region, bucket string) error {
	s3Client, err := c.newS3Client(ctx, accessKey, secretKey, region)
	if err != nil {
		return err
	}

	verifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// If bucket is empty, verify credentials by listing all buckets in the account
	if bucket == "" {
		_, err = s3Client.ListBuckets(verifyCtx, &awsS3.ListBucketsInput{})
		return err
	}

	// 1. Try HeadBucket first
	_, err = s3Client.HeadBucket(verifyCtx, &awsS3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		return nil
	}

	// 2. Check if the error is an authentication error (bad Key or Secret)
	errStr := err.Error()
	if strings.Contains(errStr, "SignatureDoesNotMatch") ||
		strings.Contains(errStr, "InvalidAccessKeyId") ||
		strings.Contains(errStr, "InvalidToken") {
		return fmt.Errorf("invalid AWS credentials (Access Key ID or Secret Key is incorrect)")
	}

	// 3. Fallback: Try a minimal ListObjectsV2 to see if list permissions work
	_, listErr := s3Client.ListObjectsV2(verifyCtx, &awsS3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int32(1),
	})
	if listErr == nil {
		return nil
	}

	listErrStr := listErr.Error()
	if strings.Contains(listErrStr, "SignatureDoesNotMatch") ||
		strings.Contains(listErrStr, "InvalidAccessKeyId") ||
		strings.Contains(listErrStr, "InvalidToken") {
		return fmt.Errorf("invalid AWS credentials (Access Key ID or Secret Key is incorrect)")
	}

	// 4. If both failed with Forbidden/AccessDenied (403), the keys are actually VALID,
	// but the user's IAM policy restricts bucket-level actions.
	return nil
}

// ListBuckets retrieves all buckets available under the credentials.
func (c *client) ListBuckets(ctx context.Context, accessKey, secretKey, region string) ([]model.S3Item, error) {
	s3Client, err := c.newS3Client(ctx, accessKey, secretKey, region)
	if err != nil {
		return nil, err
	}

	listCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	output, err := s3Client.ListBuckets(listCtx, &awsS3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	var items []model.S3Item
	for _, b := range output.Buckets {
		items = append(items, model.S3Item{
			Key:          aws.ToString(b.Name) + "/",
			Name:         aws.ToString(b.Name),
			IsDir:        true,
			LastModified: aws.ToTime(b.CreationDate),
		})
	}
	return items, nil
}

// isGarbage checks if a folder or file name looks like trash/logs (contains special characters or is a UUID)
func isGarbage(name string) bool {
	// 1. Check for special characters
	if strings.Contains(name, "*") || strings.Contains(name, "#") {
		return true
	}

	// 2. Check if it looks like a UUID (length 36, contains 4 hyphens)
	if len(name) == 36 && strings.Count(name, "-") == 4 {
		return true
	}

	return false
}

// ListObjects retrieves objects inside a bucket under a prefix, supporting searching and structured directories.
func (c *client) ListObjects(ctx context.Context, accessKey, secretKey, region, bucket, prefix, search string) ([]model.S3Item, error) {
	s3Client, err := c.newS3Client(ctx, accessKey, secretKey, region)
	if err != nil {
		return nil, err
	}

	bypassBucket := os.Getenv("BYPASS_BUCKET")
	bypassFolder := os.Getenv("BYPASS_FOLDER")

	if bypassBucket != "" && bypassFolder != "" {
		bypassFolderSlash := bypassFolder
		if !strings.HasSuffix(bypassFolderSlash, "/") {
			bypassFolderSlash += "/"
		}

		if bucket == bypassBucket && prefix == "" && search == "" {
			return []model.S3Item{
				{
					Key:   bypassFolderSlash,
					Name:  strings.TrimSuffix(bypassFolder, "/"),
					IsDir: true,
				},
			}, nil
		}
	}

	// Increase timeout to 180 seconds (3 minutes) as requested
	listCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	var items []model.S3Item
	maxItems := 2000 // Return at most 2000 valid items to prevent browser lag & infinite backend loops

	if search != "" {
		searchPrefix := prefix
		if bypassBucket != "" && bypassFolder != "" && bucket == bypassBucket && prefix == "" {
			bypassFolderSlash := bypassFolder
			if !strings.HasSuffix(bypassFolderSlash, "/") {
				bypassFolderSlash += "/"
			}
			searchPrefix = bypassFolderSlash // Focus search inside bypass folder to avoid massive timeout
		}

		paginator := awsS3.NewListObjectsV2Paginator(s3Client, &awsS3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
			Prefix: aws.String(searchPrefix),
		})

		searchLower := strings.ToLower(search)
		scannedCount := 0
		maxScan := 15000 // Scan up to 15,000 S3 objects

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(listCtx)
			if err != nil {
				return nil, err
			}

			scannedCount += len(page.Contents)

			for _, obj := range page.Contents {
				key := aws.ToString(obj.Key)

				parts := strings.Split(key, "/")
				var name string
				if len(parts) > 0 {
					name = parts[len(parts)-1]
					if name == "" && len(parts) > 1 {
						name = parts[len(parts)-2]
					}
				}

				if isGarbage(name) {
					continue
				}

				if strings.Contains(strings.ToLower(key), searchLower) {
					items = append(items, model.S3Item{
						Key:          key,
						Name:         name,
						Size:         aws.ToInt64(obj.Size),
						LastModified: aws.ToTime(obj.LastModified),
						IsDir:        strings.HasSuffix(key, "/"),
					})

					if len(items) >= maxItems {
						break
					}
				}
			}

			if scannedCount >= maxScan || len(items) >= maxItems {
				break
			}
		}
		return items, nil
	}

	// Normal list (structured browsing with delimiter)
	paginator := awsS3.NewListObjectsV2Paginator(s3Client, &awsS3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(listCtx)
		if err != nil {
			return nil, err
		}

		// 1. Process directories (CommonPrefixes)
		for _, commonPrefix := range page.CommonPrefixes {
			dirKey := aws.ToString(commonPrefix.Prefix)

			trimmed := strings.TrimSuffix(dirKey, "/")
			parts := strings.Split(trimmed, "/")
			name := parts[len(parts)-1]

			if isGarbage(name) {
				continue
			}

			items = append(items, model.S3Item{
				Key:   dirKey,
				Name:  name,
				IsDir: true,
			})

			if len(items) >= maxItems {
				break
			}
		}

		// 2. Process files (Contents)
		if len(items) < maxItems {
			for _, obj := range page.Contents {
				key := aws.ToString(obj.Key)

				if key == prefix {
					continue
				}

				parts := strings.Split(key, "/")
				name := parts[len(parts)-1]

				if isGarbage(name) {
					continue
				}

				items = append(items, model.S3Item{
					Key:          key,
					Name:         name,
					Size:         aws.ToInt64(obj.Size),
					LastModified: aws.ToTime(obj.LastModified),
					IsDir:        false,
				})

				if len(items) >= maxItems {
					break
				}
			}
		}

		if len(items) >= maxItems {
			break
		}
	}

	return items, nil
}

// GeneratePresignedGET creates a temporary link to download/view an object.
func (c *client) GeneratePresignedGET(ctx context.Context, accessKey, secretKey, region, bucket, key string, expiration time.Duration) (string, error) {
	s3Client, err := c.newS3Client(ctx, accessKey, secretKey, region)
	if err != nil {
		return "", err
	}

	presigner := awsS3.NewPresignClient(s3Client)
	req, err := presigner.PresignGetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, awsS3.WithPresignExpires(expiration))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

// GeneratePresignedPUT creates a temporary link to upload an object directly from the browser.
func (c *client) GeneratePresignedPUT(ctx context.Context, accessKey, secretKey, region, bucket, key string, expiration time.Duration) (string, error) {
	s3Client, err := c.newS3Client(ctx, accessKey, secretKey, region)
	if err != nil {
		return "", err
	}

	presigner := awsS3.NewPresignClient(s3Client)
	req, err := presigner.PresignPutObject(ctx, &awsS3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, awsS3.WithPresignExpires(expiration))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

// DeleteObjects deletes multiple files at once from the bucket.
func (c *client) DeleteObjects(ctx context.Context, accessKey, secretKey, region, bucket string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	s3Client, err := c.newS3Client(ctx, accessKey, secretKey, region)
	if err != nil {
		return err
	}

	var objects []types.ObjectIdentifier
	for _, key := range keys {
		objects = append(objects, types.ObjectIdentifier{
			Key: aws.String(key),
		})
	}

	_, err = s3Client.DeleteObjects(ctx, &awsS3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	})
	return err
}

// GetFilePreview downloads the first 2MB of a text file from S3 to preview it.
func (c *client) GetFilePreview(ctx context.Context, accessKey, secretKey, region, bucket, key string) (string, error) {
	s3Client, err := c.newS3Client(ctx, accessKey, secretKey, region)
	if err != nil {
		return "", err
	}

	previewCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	output, err := s3Client.GetObject(previewCtx, &awsS3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}
	defer output.Body.Close()

	limitReader := io.LimitReader(output.Body, 2*1024*1024)
	content, err := io.ReadAll(limitReader)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// UploadObject uploads raw binary data to S3.
func (c *client) UploadObject(ctx context.Context, accessKey, secretKey, region, bucket, key string, body io.Reader, size int64, contentType string) error {
	s3Client, err := c.newS3Client(ctx, accessKey, secretKey, region)
	if err != nil {
		return err
	}

	uploadCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	input := &awsS3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	}

	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}

	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	_, err = s3Client.PutObject(uploadCtx, input)
	return err
}

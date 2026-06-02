package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Item represents a file or directory in S3.
type S3Item struct {
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	IsDir        bool      `json:"is_dir"`
}

// NewS3Client creates a new AWS S3 Client using static credentials.
func NewS3Client(accessKey, secretKey, region string) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		)),
	)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), nil
}

// VerifyCredentials checks if the provided credentials are valid.
func VerifyCredentials(accessKey, secretKey, region, bucket string) error {
	client, err := NewS3Client(accessKey, secretKey, region)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// If bucket is empty, verify credentials by listing all buckets in the account
	if bucket == "" {
		_, err = client.ListBuckets(ctx, &s3.ListBucketsInput{})
		return err
	}

	// 1. Try HeadBucket first
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
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
	_, listErr := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
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

	// 4. If both failed with Forbidden/AccessDenied (403), the keys are actually VALID (otherwise AWS S3
	// would have thrown an Auth error), but the user's IAM policy restricts bucket-level actions.
	// We will still allow saving the connection, because the user may have direct object-level access
	// (like s3:PutObject or s3:GetObject on bucket/*) which works perfectly for presigned uploads/downloads.
	return nil
}

// ListBuckets retrieves all buckets available under the credentials.
func ListBuckets(client *s3.Client) ([]S3Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	output, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	var items []S3Item
	for _, b := range output.Buckets {
		items = append(items, S3Item{
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
	// 1. Check for special characters used in their system's garbage/log folders
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
func ListObjects(client *s3.Client, bucket, prefix, search string) ([]S3Item, error) {
	bypassBucket := os.Getenv("BYPASS_BUCKET")
	bypassFolder := os.Getenv("BYPASS_FOLDER")

	if bypassBucket != "" && bypassFolder != "" {
		bypassFolderSlash := bypassFolder
		if !strings.HasSuffix(bypassFolderSlash, "/") {
			bypassFolderSlash += "/"
		}

		// SPECIAL CASE: If we are at the root level of the massive configured bucket,
		// only return the bypass folder to completely bypass the million garbage folders and prevent timeouts!
		if bucket == bypassBucket && prefix == "" && search == "" {
			return []S3Item{
				{
					Key:   bypassFolderSlash,
					Name:  strings.TrimSuffix(bypassFolder, "/"),
					IsDir: true,
				},
			}, nil
		}
	}

	// Increase timeout to 180 seconds (3 minutes) as requested
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	var items []S3Item
	maxItems := 2000 // Return at most 2000 valid items to prevent browser lag & infinite backend loops

	// If there is a search query, list recursively and filter locally
	if search != "" {
		searchPrefix := prefix
		if bypassBucket != "" && bypassFolder != "" && bucket == bypassBucket && prefix == "" {
			bypassFolderSlash := bypassFolder
			if !strings.HasSuffix(bypassFolderSlash, "/") {
				bypassFolderSlash += "/"
			}
			searchPrefix = bypassFolderSlash // Focus search inside bypass folder to avoid massive timeout
		}

		paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
			Prefix: aws.String(searchPrefix),
		})

		searchLower := strings.ToLower(search)
		scannedCount := 0
		maxScan := 15000 // Scan up to 15,000 S3 objects

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			scannedCount += len(page.Contents)

			for _, obj := range page.Contents {
				key := aws.ToString(obj.Key)

				// Extract file or directory name
				parts := strings.Split(key, "/")
				var name string
				if len(parts) > 0 {
					name = parts[len(parts)-1]
					if name == "" && len(parts) > 1 {
						name = parts[len(parts)-2]
					}
				}

				// Skip garbage items
				if isGarbage(name) {
					continue
				}

				if strings.Contains(strings.ToLower(key), searchLower) {
					items = append(items, S3Item{
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
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		// 1. Process directories (CommonPrefixes)
		for _, commonPrefix := range page.CommonPrefixes {
			dirKey := aws.ToString(commonPrefix.Prefix)

			// Get directory display name (e.g. "images/" -> "images")
			trimmed := strings.TrimSuffix(dirKey, "/")
			parts := strings.Split(trimmed, "/")
			name := parts[len(parts)-1]

			// Skip garbage folders!
			if isGarbage(name) {
				continue
			}

			items = append(items, S3Item{
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

				// Skip the directory placeholder object itself
				if key == prefix {
					continue
				}

				parts := strings.Split(key, "/")
				name := parts[len(parts)-1]

				// Skip garbage files!
				if isGarbage(name) {
					continue
				}

				items = append(items, S3Item{
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
			break // Stop loading pages once we have 2000 clean items
		}
	}

	return items, nil
}

// GeneratePresignedGET creates a temporary link to download/view an object.
func GeneratePresignedGET(client *s3.Client, bucket, key string, expiration time.Duration) (string, error) {
	presigner := s3.NewPresignClient(client)

	req, err := presigner.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

// GeneratePresignedPUT creates a temporary link to upload an object directly from the browser.
func GeneratePresignedPUT(client *s3.Client, bucket, key string, expiration time.Duration) (string, error) {
	presigner := s3.NewPresignClient(client)

	req, err := presigner.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

// DeleteObjects deletes multiple files at once from the bucket.
func DeleteObjects(client *s3.Client, bucket string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	var objects []types.ObjectIdentifier
	for _, key := range keys {
		objects = append(objects, types.ObjectIdentifier{
			Key: aws.String(key),
		})
	}

	_, err := client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	})
	return err
}

// GetFilePreview downloads the first 2MB of a text file from S3 to preview it.
func GetFilePreview(client *s3.Client, bucket, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}
	defer output.Body.Close()

	// Read up to 2MB (Safety limit to prevent RAM bloat on large files)
	limitReader := io.LimitReader(output.Body, 2*1024*1024)
	content, err := io.ReadAll(limitReader)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

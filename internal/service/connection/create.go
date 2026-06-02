package connection

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"s3go/internal/entity"
	"s3go/internal/model"
	"s3go/internal/persistence"
	"s3go/internal/pkg/crypto"
)

// generateUUID generates a random UUID v4 for connection IDs.
func generateUUID() string {
	uuid := make([]byte, 16)
	_, _ = rand.Read(uuid)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func (s *service) CreateConnection(ctx context.Context, input model.CreateConnectionInput) (model.CreateConnectionResponse, error) {
	// 1. Verify credentials and bucket
	err := s.s3Client.VerifyCredentials(ctx, input.AccessKey, input.SecretKey, input.Region, input.Bucket)
	if err != nil {
		return model.CreateConnectionResponse{}, fmt.Errorf("S3 Verification Failed: %w", err)
	}

	// 2. Encrypt Secret Key
	encryptedSecret, err := crypto.Encrypt(input.SecretKey)
	if err != nil {
		return model.CreateConnectionResponse{}, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// 3. Save to database
	conn := &entity.Connection{
		ID:                 generateUUID(),
		Name:               input.Name,
		AccessKey:          input.AccessKey,
		SecretKeyEncrypted: encryptedSecret,
		Region:             input.Region,
		Bucket:             input.Bucket,
		CreatedAt:          time.Now(),
	}

	err = s.writableUOW.DoInTx(ctx, func(tx persistence.WritableUnitOfWork) error {
		return tx.Connection().Save(ctx, conn)
	})
	if err != nil {
		return model.CreateConnectionResponse{}, err
	}

	return model.CreateConnectionResponse{
		Status: "success",
		ID:     conn.ID,
	}, nil
}

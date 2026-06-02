package connection

import (
	"context"
	"fmt"

	"s3go/internal/model"
	"s3go/internal/persistence"
	"s3go/internal/pkg/crypto"
)

func (s *service) UpdateConnection(ctx context.Context, id string, input model.UpdateConnectionInput) (model.UpdateConnectionResponse, error) {
	// 1. Retrieve existing connection
	conn, err := s.readOnlyUOW.Connection().GetByID(ctx, id)
	if err != nil {
		return model.UpdateConnectionResponse{}, fmt.Errorf("connection not found: %w", err)
	}

	var decryptedSecret string
	var encryptedSecret string

	if input.SecretKey == "" {
		decryptedSecret, err = crypto.Decrypt(conn.SecretKeyEncrypted)
		if err != nil {
			return model.UpdateConnectionResponse{}, fmt.Errorf("failed to decrypt existing credentials: %w", err)
		}
		encryptedSecret = conn.SecretKeyEncrypted
	} else {
		decryptedSecret = input.SecretKey
		encryptedSecret, err = crypto.Encrypt(input.SecretKey)
		if err != nil {
			return model.UpdateConnectionResponse{}, fmt.Errorf("failed to encrypt new credentials: %w", err)
		}
	}

	// 2. Verify updated S3 credentials
	err = s.s3Client.VerifyCredentials(ctx, input.AccessKey, decryptedSecret, input.Region, input.Bucket)
	if err != nil {
		return model.UpdateConnectionResponse{}, fmt.Errorf("S3 Verification Failed: %w", err)
	}

	// 3. Save connection
	conn.Name = input.Name
	conn.AccessKey = input.AccessKey
	conn.SecretKeyEncrypted = encryptedSecret
	conn.Region = input.Region
	conn.Bucket = input.Bucket

	err = s.writableUOW.DoInTx(ctx, func(tx persistence.WritableUnitOfWork) error {
		return tx.Connection().Save(ctx, conn)
	})
	if err != nil {
		return model.UpdateConnectionResponse{}, err
	}

	return model.UpdateConnectionResponse{
		Status: "success",
		ID:     conn.ID,
	}, nil
}

package connection

import (
	"context"
	"s3go/internal/persistence"
)

func (s *service) DeleteConnection(ctx context.Context, id string) error {
	return s.writableUOW.DoInTx(ctx, func(tx persistence.WritableUnitOfWork) error {
		return tx.Connection().Delete(ctx, id)
	})
}

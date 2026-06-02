package connection

import (
	"context"
	"s3go/internal/entity"
)

func (s *service) GetConnection(ctx context.Context, id string) (*entity.Connection, error) {
	return s.readOnlyUOW.Connection().GetByID(ctx, id)
}

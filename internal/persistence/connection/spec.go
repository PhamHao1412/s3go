package connection

import (
	"context"
	"s3go/internal/entity"
)

type ReadOnlyRepository interface {
	GetByID(ctx context.Context, id string) (*entity.Connection, error)
	List(ctx context.Context) ([]*entity.Connection, error)
}

type WritableRepository interface {
	ReadOnlyRepository
	Save(ctx context.Context, conn *entity.Connection) error
	Delete(ctx context.Context, id string) error
}

package connection

import (
	"context"

	"s3go/internal/entity"
	s3Infra "s3go/internal/infra/s3"
	"s3go/internal/model"
	"s3go/internal/persistence"
)

type Service interface {
	CreateConnection(ctx context.Context, input model.CreateConnectionInput) (model.CreateConnectionResponse, error)
	UpdateConnection(ctx context.Context, id string, input model.UpdateConnectionInput) (model.UpdateConnectionResponse, error)
	DeleteConnection(ctx context.Context, id string) error
	GetConnection(ctx context.Context, id string) (*entity.Connection, error)
	ListConnections(ctx context.Context) ([]*model.ConnectionDetail, error)
}

type service struct {
	readOnlyUOW persistence.ReadOnlyUnitOfWork
	writableUOW persistence.WritableUnitOfWork
	s3Client    s3Infra.Client
}

func New(
	readOnlyUOW persistence.ReadOnlyUnitOfWork,
	writableUOW persistence.WritableUnitOfWork,
	s3Client s3Infra.Client,
) Service {
	return &service{
		readOnlyUOW: readOnlyUOW,
		writableUOW: writableUOW,
		s3Client:    s3Client,
	}
}

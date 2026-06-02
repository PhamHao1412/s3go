package connection

import (
	"context"

	"s3go/internal/model"
)

func (s *service) ListConnections(ctx context.Context) ([]*model.ConnectionDetail, error) {
	conns, err := s.readOnlyUOW.Connection().List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*model.ConnectionDetail, 0, len(conns))
	for _, conn := range conns {
		result = append(result, &model.ConnectionDetail{
			ID:        conn.ID,
			Name:      conn.Name,
			AccessKey: conn.AccessKey,
			Region:    conn.Region,
			Bucket:    conn.Bucket,
			CreatedAt: conn.CreatedAt,
		})
	}
	return result, nil
}

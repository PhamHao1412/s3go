package v1

import (
	"s3go/internal/service/connection"
	"s3go/internal/service/s3"
)

func New(connectionSvc connection.Service, s3Svc s3.Service) *Controller {
	return &Controller{
		connectionSvc: connectionSvc,
		s3Svc:         s3Svc,
	}
}

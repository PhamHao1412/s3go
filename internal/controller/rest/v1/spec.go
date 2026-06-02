package v1

import (
	"s3go/internal/service/connection"
	"s3go/internal/service/s3"
)

type Controller struct {
	connectionSvc connection.Service
	s3Svc         s3.Service
}

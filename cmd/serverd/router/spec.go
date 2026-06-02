package router

import (
	v1 "s3go/internal/controller/rest/v1"
)

type Router struct {
	ctrlV1 *v1.Controller
}

func New(ctrlV1 *v1.Controller) *Router {
	return &Router{
		ctrlV1: ctrlV1,
	}
}

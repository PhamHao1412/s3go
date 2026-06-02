package router

import (
	"net/http"
)

func (rtr *Router) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/connections", rtr.ctrlV1.CreateConnection)
	mux.HandleFunc("GET /api/connections", rtr.ctrlV1.ListConnections)
	mux.HandleFunc("DELETE /api/connections/{id}", rtr.ctrlV1.DeleteConnection)
	mux.HandleFunc("PUT /api/connections/{id}", rtr.ctrlV1.UpdateConnection)

	mux.HandleFunc("GET /api/connections/{id}/files", rtr.ctrlV1.ListFiles)
	mux.HandleFunc("POST /api/connections/{id}/presigned-url", rtr.ctrlV1.GeneratePresignedURL)
	mux.HandleFunc("DELETE /api/connections/{id}/files", rtr.ctrlV1.DeleteFiles)
	mux.HandleFunc("GET /api/connections/{id}/preview", rtr.ctrlV1.GetFilePreview)
	mux.HandleFunc("POST /api/connections/{id}/upload", rtr.ctrlV1.UploadFile)
}

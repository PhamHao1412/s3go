package v1

import (
	"fmt"
	"net/http"

	"s3go/internal/controller/rest/v1/dto"
	"s3go/internal/model"
)

func (c *Controller) ListFiles(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		dto.SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	prefix := r.URL.Query().Get("prefix")
	search := r.URL.Query().Get("search")

	items, err := c.s3Svc.ListFiles(r.Context(), id, prefix, search)
	if err != nil {
		dto.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SendJSON(w, http.StatusOK, items)
}

func (c *Controller) GeneratePresignedURL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		dto.SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	var input model.PresignedURLInput
	if err := dto.BindJSON(r, &input); err != nil {
		dto.SendError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if input.Key == "" || input.Action == "" {
		dto.SendError(w, http.StatusBadRequest, "key and action parameters are required")
		return
	}

	resp, err := c.s3Svc.GeneratePresignedURL(r.Context(), id, input)
	if err != nil {
		dto.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SendJSON(w, http.StatusOK, resp)
}

func (c *Controller) DeleteFiles(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		dto.SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	var input model.DeleteFilesInput
	if err := dto.BindJSON(r, &input); err != nil {
		dto.SendError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if len(input.Keys) == 0 {
		dto.SendError(w, http.StatusBadRequest, "keys array cannot be empty")
		return
	}

	err := c.s3Svc.DeleteFiles(r.Context(), id, input)
	if err != nil {
		dto.SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete objects: %v", err))
		return
	}

	dto.SendJSON(w, http.StatusOK, model.StatusResponse{Status: "success"})
}

func (c *Controller) GetFilePreview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		dto.SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		dto.SendError(w, http.StatusBadRequest, "key parameter is required")
		return
	}

	content, err := c.s3Svc.GetFilePreview(r.Context(), id, key)
	if err != nil {
		dto.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SendJSON(w, http.StatusOK, model.FilePreviewResponse{Content: content})
}

func (c *Controller) UploadFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		dto.SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		dto.SendError(w, http.StatusBadRequest, "key query parameter is required")
		return
	}

	contentType := r.Header.Get("Content-Type")

	err := c.s3Svc.UploadFile(r.Context(), id, key, r.Body, r.ContentLength, contentType)
	if err != nil {
		dto.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SendJSON(w, http.StatusOK, model.StatusResponse{Status: "success"})
}

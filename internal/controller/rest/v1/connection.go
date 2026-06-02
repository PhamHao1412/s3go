package v1

import (
	"fmt"
	"net/http"

	"s3go/internal/controller/rest/v1/dto"
	"s3go/internal/model"
)

func (c *Controller) CreateConnection(w http.ResponseWriter, r *http.Request) {
	var input model.CreateConnectionInput
	if err := dto.BindJSON(r, &input); err != nil {
		dto.SendError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if input.Name == "" || input.AccessKey == "" || input.SecretKey == "" || input.Region == "" {
		dto.SendError(w, http.StatusBadRequest, "Fields (name, access_key, secret_key, region) are required")
		return
	}

	resp, err := c.connectionSvc.CreateConnection(r.Context(), input)
	if err != nil {
		dto.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SendJSON(w, http.StatusCreated, resp)
}

func (c *Controller) UpdateConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		dto.SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	var input model.UpdateConnectionInput
	if err := dto.BindJSON(r, &input); err != nil {
		dto.SendError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if input.Name == "" || input.AccessKey == "" || input.Region == "" {
		dto.SendError(w, http.StatusBadRequest, "Fields (name, access_key, region) are required")
		return
	}

	resp, err := c.connectionSvc.UpdateConnection(r.Context(), id, input)
	if err != nil {
		dto.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dto.SendJSON(w, http.StatusOK, resp)
}

func (c *Controller) ListConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := c.connectionSvc.ListConnections(r.Context())
	if err != nil {
		dto.SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list connections: %v", err))
		return
	}

	dto.SendJSON(w, http.StatusOK, conns)
}

func (c *Controller) DeleteConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		dto.SendError(w, http.StatusBadRequest, "Connection ID is required")
		return
	}

	err := c.connectionSvc.DeleteConnection(r.Context(), id)
	if err != nil {
		dto.SendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete connection: %v", err))
		return
	}

	dto.SendJSON(w, http.StatusOK, model.StatusResponse{Status: "success"})
}

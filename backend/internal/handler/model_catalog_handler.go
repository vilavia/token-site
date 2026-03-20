package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ModelCatalogHandler handles model catalog requests.
type ModelCatalogHandler struct {
	modelCatalogService *service.ModelCatalogService
}

// NewModelCatalogHandler creates a new ModelCatalogHandler.
func NewModelCatalogHandler(modelCatalogService *service.ModelCatalogService) *ModelCatalogHandler {
	return &ModelCatalogHandler{
		modelCatalogService: modelCatalogService,
	}
}

// ListModels returns all available models.
// GET /api/v1/models
func (h *ModelCatalogHandler) ListModels(c *gin.Context) {
	models, err := h.modelCatalogService.ListModels(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"models": models})
}

// GetModel returns a single model by ID.
// GET /api/v1/models/:id
func (h *ModelCatalogHandler) GetModel(c *gin.Context) {
	modelID := c.Param("id")

	model, err := h.modelCatalogService.GetModel(c.Request.Context(), modelID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if model == nil {
		response.NotFound(c, "Model not found: "+modelID)
		return
	}

	response.Success(c, model)
}

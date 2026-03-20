package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterModelRoutes 注册模型目录相关路由
func RegisterModelRoutes(
	v1 *gin.RouterGroup,
	h *handler.ModelCatalogHandler,
	jwtAuth middleware.JWTAuthMiddleware,
) {
	models := v1.Group("/models")
	models.Use(gin.HandlerFunc(jwtAuth))
	{
		models.GET("", h.ListModels)
		models.GET("/:id", h.GetModel)
	}
}

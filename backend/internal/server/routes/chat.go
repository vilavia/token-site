package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterChatRoutes registers web chat routes under /api/v1/chat.
func RegisterChatRoutes(
	v1 *gin.RouterGroup,
	h *handler.ChatHandler,
	historyHandler *handler.ChatHistoryHandler,
	jwtAuth middleware.JWTAuthMiddleware,
) {
	chat := v1.Group("/chat")
	chat.Use(gin.HandlerFunc(jwtAuth))
	{
		chat.POST("/completions", h.ChatCompletion)
		chat.GET("/models", h.ListChatModels)
	}

	// Chat history (JWT auth)
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	history := authenticated.Group("/conversations")
	history.GET("", historyHandler.ListConversations)
	history.POST("", historyHandler.CreateConversation)
	history.PUT("/:id", historyHandler.UpdateConversation)
	history.DELETE("/:id", historyHandler.DeleteConversation)
	history.GET("/:id/messages", historyHandler.GetMessages)
	history.POST("/:id/messages", historyHandler.AddMessage)
}

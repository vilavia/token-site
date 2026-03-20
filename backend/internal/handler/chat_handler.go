package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ChatHandler handles web chat requests for JWT-authenticated users.
// It resolves an API key for the requested model, sets up the gateway context,
// and delegates to the existing gateway handlers for request processing.
type ChatHandler struct {
	chatService    *service.ChatService
	gatewayHandler *GatewayHandler
	openaiHandler  *OpenAIGatewayHandler
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(
	chatService *service.ChatService,
	gatewayHandler *GatewayHandler,
	openaiHandler *OpenAIGatewayHandler,
) *ChatHandler {
	return &ChatHandler{
		chatService:    chatService,
		gatewayHandler: gatewayHandler,
		openaiHandler:  openaiHandler,
	}
}

// chatCompletionRequest is used only to peek at the model field from the request body.
type chatCompletionRequest struct {
	Model string `json:"model"`
}

// ChatCompletion handles POST /api/v1/chat/completions.
// It uses JWT auth, resolves an API key for the requested model,
// injects gateway context, and forwards to the OpenAI gateway handler.
func (h *ChatHandler) ChatCompletion(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Read body once so we can peek at the model field, then restore it for the gateway handler.
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "Failed to read request body"}})
		return
	}
	// Restore body for downstream handler.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var req chatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil || req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "Missing or invalid 'model' field"}})
		return
	}

	apiKey, err := h.chatService.FindOrCreateAPIKeyForModel(c.Request.Context(), subject.UserID, req.Model)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "No available provider for model: " + req.Model}})
		return
	}

	// Inject gateway context to mimic what APIKeyAuthMiddleware sets.
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{
		UserID:      subject.UserID,
		Concurrency: subject.Concurrency,
	})
	if apiKey.User != nil {
		c.Set(string(middleware2.ContextKeyUserRole), apiKey.User.Role)
	}

	// Delegate to the OpenAI gateway handler which handles chat completions.
	h.openaiHandler.ChatCompletions(c)
}

// ListChatModels handles GET /api/v1/chat/models.
// Returns model IDs available to the authenticated user via their API keys.
func (h *ChatHandler) ListChatModels(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	models, err := h.chatService.GetAvailableChatModels(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"models": models})
}

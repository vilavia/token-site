package handler

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ChatHistoryHandler handles chat history CRUD endpoints.
type ChatHistoryHandler struct {
	chatHistoryService *service.ChatHistoryService
}

// NewChatHistoryHandler creates a new ChatHistoryHandler.
func NewChatHistoryHandler(svc *service.ChatHistoryService) *ChatHistoryHandler {
	return &ChatHistoryHandler{chatHistoryService: svc}
}

// ListConversations handles GET /api/v1/chat/conversations.
func (h *ChatHistoryHandler) ListConversations(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	conversations, err := h.chatHistoryService.ListConversations(c.Request.Context(), subject.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list conversations"})
		return
	}

	response.Success(c, conversations)
}

type createConversationRequest struct {
	Title string `json:"title"`
	Model string `json:"model"`
}

// CreateConversation handles POST /api/v1/chat/conversations.
func (h *ChatHistoryHandler) CreateConversation(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	title := req.Title
	if title == "" {
		title = "New Chat"
	}

	conv, err := h.chatHistoryService.CreateConversation(c.Request.Context(), subject.UserID, title, req.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	response.Success(c, conv)
}

type updateConversationRequest struct {
	Title string `json:"title"`
	Model string `json:"model"`
}

// UpdateConversation handles PUT /api/v1/chat/conversations/:id.
func (h *ChatHistoryHandler) UpdateConversation(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	convID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req updateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.chatHistoryService.UpdateConversation(c.Request.Context(), subject.UserID, convID, req.Title, req.Model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update conversation"})
		return
	}

	response.Success(c, nil)
}

// DeleteConversation handles DELETE /api/v1/chat/conversations/:id.
func (h *ChatHistoryHandler) DeleteConversation(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	convID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	if err := h.chatHistoryService.DeleteConversation(c.Request.Context(), subject.UserID, convID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete conversation"})
		return
	}

	response.Success(c, nil)
}

// GetMessages handles GET /api/v1/chat/conversations/:id/messages.
func (h *ChatHistoryHandler) GetMessages(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	convID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	messages, err := h.chatHistoryService.GetMessages(c.Request.Context(), subject.UserID, convID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}

	response.Success(c, messages)
}

type addMessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AddMessage handles POST /api/v1/chat/conversations/:id/messages.
func (h *ChatHistoryHandler) AddMessage(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	convID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req addMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Role == "" || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role and content are required"})
		return
	}

	msg, err := h.chatHistoryService.AddMessage(c.Request.Context(), subject.UserID, convID, req.Role, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add message"})
		return
	}

	response.Success(c, msg)
}

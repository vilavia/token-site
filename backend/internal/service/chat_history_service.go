package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const (
	maxConversationsPerUser = 100     // Max conversations per user
	maxMessagesPerConversation = 500  // Max messages per conversation
	maxMessageContentBytes = 100_000  // Max single message content size (100KB)
)

// ChatConversation represents a chat conversation record.
type ChatConversation struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatMessageRecord represents a single message within a conversation.
type ChatMessageRecord struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

// ChatHistoryService provides CRUD operations for chat conversations and messages.
type ChatHistoryService struct {
	db *sql.DB
}

// NewChatHistoryService creates a new ChatHistoryService.
func NewChatHistoryService(db *sql.DB) *ChatHistoryService {
	return &ChatHistoryService{db: db}
}

// ListConversations returns the most recent conversations for a user.
func (s *ChatHistoryService) ListConversations(ctx context.Context, userID int64) ([]ChatConversation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, title, model, created_at, updated_at
		FROM chat_conversations
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var conversations []ChatConversation
	for rows.Next() {
		var c ChatConversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.Model, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		conversations = append(conversations, c)
	}
	return conversations, rows.Err()
}

// CreateConversation creates a new conversation and returns it.
func (s *ChatHistoryService) CreateConversation(ctx context.Context, userID int64, title, model string) (*ChatConversation, error) {
	// Check conversation limit
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_conversations WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return nil, err
	}
	if count >= maxConversationsPerUser {
		// Auto-delete oldest conversation to make room
		_, _ = s.db.ExecContext(ctx, `
			DELETE FROM chat_conversations WHERE id = (
				SELECT id FROM chat_conversations WHERE user_id = $1 ORDER BY updated_at ASC LIMIT 1
			)`, userID)
	}

	c := &ChatConversation{}
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO chat_conversations (user_id, title, model)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, title, model, created_at, updated_at
	`, userID, title, model).Scan(&c.ID, &c.UserID, &c.Title, &c.Model, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// UpdateConversation updates the title and model of a conversation owned by the user.
func (s *ChatHistoryService) UpdateConversation(ctx context.Context, userID, convID int64, title, model string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE chat_conversations
		SET title = $3, model = $4, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, convID, userID, title, model)
	return err
}

// DeleteConversation deletes a conversation owned by the user.
func (s *ChatHistoryService) DeleteConversation(ctx context.Context, userID, convID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM chat_conversations
		WHERE id = $1 AND user_id = $2
	`, convID, userID)
	return err
}

// GetMessages returns all messages for a conversation owned by the user.
func (s *ChatHistoryService) GetMessages(ctx context.Context, userID, convID int64) ([]ChatMessageRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.conversation_id, m.role, m.content, m.created_at
		FROM chat_messages m
		JOIN chat_conversations c ON m.conversation_id = c.id
		WHERE c.id = $1 AND c.user_id = $2
		ORDER BY m.created_at ASC
	`, convID, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var messages []ChatMessageRecord
	for rows.Next() {
		var m ChatMessageRecord
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// AddMessage adds a message to a conversation and updates the conversation's updated_at.
// If the conversation title is "New Chat" and the role is "user", it auto-titles with the first 40 chars of content.
func (s *ChatHistoryService) AddMessage(ctx context.Context, userID, convID int64, role, content string) (*ChatMessageRecord, error) {
	// Validate content size
	if len(content) > maxMessageContentBytes {
		return nil, fmt.Errorf("message too long: %d bytes exceeds limit of %d", len(content), maxMessageContentBytes)
	}

	// Check message count limit
	var msgCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_messages WHERE conversation_id = $1`, convID).Scan(&msgCount); err != nil {
		return nil, err
	}
	if msgCount >= maxMessagesPerConversation {
		return nil, fmt.Errorf("conversation has reached the maximum of %d messages", maxMessagesPerConversation)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// Verify ownership and get current title
	var currentTitle string
	err = tx.QueryRowContext(ctx, `
		SELECT title FROM chat_conversations WHERE id = $1 AND user_id = $2
	`, convID, userID).Scan(&currentTitle)
	if err != nil {
		return nil, err
	}

	// Insert message
	msg := &ChatMessageRecord{}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO chat_messages (conversation_id, role, content)
		VALUES ($1, $2, $3)
		RETURNING id, conversation_id, role, content, created_at
	`, convID, role, content).Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Auto-title if title is "New Chat" and role is "user"
	if currentTitle == "New Chat" && role == "user" {
		newTitle := content
		if len(newTitle) > 40 {
			newTitle = newTitle[:40]
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE chat_conversations SET title = $3, updated_at = NOW() WHERE id = $1 AND user_id = $2
		`, convID, userID, newTitle)
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE chat_conversations SET updated_at = NOW() WHERE id = $1 AND user_id = $2
		`, convID, userID)
	}
	if err != nil {
		return nil, err
	}

	return msg, tx.Commit()
}

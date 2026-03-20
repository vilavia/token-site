package service

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// ChatService provides web chat functionality for JWT-authenticated users.
// It resolves the appropriate API key for a given model and user, creating
// an internal "__web_chat__" key if none exists.
type ChatService struct {
	gatewayService *GatewayService
	apiKeyService  *APIKeyService
}

// NewChatService creates a new ChatService.
func NewChatService(gatewayService *GatewayService, apiKeyService *APIKeyService) *ChatService {
	return &ChatService{
		gatewayService: gatewayService,
		apiKeyService:  apiKeyService,
	}
}

// GetAvailableChatModels returns model IDs available to the user via their existing API keys.
// It collects all group IDs from the user's active keys and aggregates models from each group.
func (s *ChatService) GetAvailableChatModels(ctx context.Context, userID int64) ([]string, error) {
	keys, _, err := s.apiKeyService.List(ctx, userID, pagination.DefaultPagination(), APIKeyListFilters{})
	if err != nil {
		return nil, fmt.Errorf("list user api keys: %w", err)
	}

	// Collect unique group IDs from the user's keys.
	seen := make(map[int64]bool)
	var groupIDs []*int64
	for i := range keys {
		k := &keys[i]
		if k.GroupID != nil && !seen[*k.GroupID] {
			seen[*k.GroupID] = true
			gid := *k.GroupID
			groupIDs = append(groupIDs, &gid)
		}
	}

	// Always include models available with no group restriction.
	modelSet := make(map[string]bool)
	for _, gid := range groupIDs {
		models := s.gatewayService.GetAvailableModels(ctx, gid, "")
		for _, m := range models {
			modelSet[m] = true
		}
	}

	result := make([]string, 0, len(modelSet))
	for m := range modelSet {
		result = append(result, m)
	}
	return result, nil
}

// FindOrCreateAPIKeyForModel finds an existing active API key for the user whose group
// serves the given modelID. If none is found, it creates a new internal key named
// "__web_chat__" in an appropriate group and returns it.
func (s *ChatService) FindOrCreateAPIKeyForModel(ctx context.Context, userID int64, modelID string) (*APIKey, error) {
	keys, _, err := s.apiKeyService.List(ctx, userID, pagination.DefaultPagination(), APIKeyListFilters{})
	if err != nil {
		return nil, fmt.Errorf("list user api keys: %w", err)
	}

	// Look for an existing key whose group serves the requested model.
	for i := range keys {
		k := &keys[i]
		if !k.IsActive() {
			continue
		}
		models := s.gatewayService.GetAvailableModels(ctx, k.GroupID, "")
		for _, m := range models {
			if m == modelID {
				// Found a suitable key — load full key (with User/Group populated).
				full, err := s.apiKeyService.GetByID(ctx, k.ID)
				if err != nil {
					return nil, fmt.Errorf("get api key by id: %w", err)
				}
				return full, nil
			}
		}
	}

	// No suitable key found — find a group that serves the model and create one.
	groupID, err := s.findGroupForModel(ctx, userID, modelID)
	if err != nil {
		return nil, err
	}

	created, err := s.apiKeyService.Create(ctx, userID, CreateAPIKeyRequest{
		Name:    "__web_chat__",
		GroupID: groupID,
	})
	if err != nil {
		return nil, fmt.Errorf("create web chat api key: %w", err)
	}

	// Load full object so User/Group are populated.
	full, err := s.apiKeyService.GetByID(ctx, created.ID)
	if err != nil {
		return nil, fmt.Errorf("get created api key: %w", err)
	}
	return full, nil
}

// findGroupForModel returns a group ID that serves the given model and is accessible
// to the user, or nil if the model is available without a group restriction.
func (s *ChatService) findGroupForModel(ctx context.Context, userID int64, modelID string) (*int64, error) {
	availableGroups, err := s.apiKeyService.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get available groups: %w", err)
	}

	for _, g := range availableGroups {
		gid := g.ID
		models := s.gatewayService.GetAvailableModels(ctx, &gid, "")
		for _, m := range models {
			if m == modelID {
				result := gid
				return &result, nil
			}
		}
	}

	// Check if the model is available without any group.
	models := s.gatewayService.GetAvailableModels(ctx, nil, "")
	for _, m := range models {
		if m == modelID {
			return nil, nil
		}
	}

	return nil, fmt.Errorf("no group found that serves model %q for user %d", modelID, userID)
}

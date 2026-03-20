package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// ChatService provides web chat functionality for JWT-authenticated users.
type ChatService struct {
	gatewayService *GatewayService
	apiKeyService  *APIKeyService
	billing        *BillingService
}

// NewChatService creates a new ChatService.
func NewChatService(gatewayService *GatewayService, apiKeyService *APIKeyService, billing *BillingService) *ChatService {
	return &ChatService{
		gatewayService: gatewayService,
		apiKeyService:  apiKeyService,
		billing:        billing,
	}
}

// GetAvailableChatModels returns model IDs available to the user for web chat.
// Shows all models that are dynamically available and have valid pricing.
func (s *ChatService) GetAvailableChatModels(ctx context.Context, userID int64) ([]string, error) {
	modelSet := make(map[string]bool)

	// 1. Collect models from user's existing API keys
	keys, _, err := s.apiKeyService.List(ctx, userID, pagination.DefaultPagination(), APIKeyListFilters{})
	if err != nil {
		return nil, fmt.Errorf("list user api keys: %w", err)
	}
	seen := make(map[int64]bool)
	for i := range keys {
		k := &keys[i]
		if k.GroupID != nil && !seen[*k.GroupID] {
			seen[*k.GroupID] = true
			models := s.gatewayService.GetAvailableModels(ctx, k.GroupID, "")
			for _, m := range models {
				modelSet[m] = true
			}
		}
	}

	// 2. Also check all groups the user can access (even without existing keys)
	availableGroups, err := s.apiKeyService.GetAvailableGroups(ctx, userID)
	if err == nil {
		for _, g := range availableGroups {
			gid := g.ID
			if !seen[gid] {
				seen[gid] = true
				models := s.gatewayService.GetAvailableModels(ctx, &gid, "")
				for _, m := range models {
					modelSet[m] = true
				}
			}
		}
	}

	// 3. Also include globally available models (no group restriction)
	globalModels := s.gatewayService.GetAvailableModels(ctx, nil, "")
	for _, m := range globalModels {
		modelSet[m] = true
	}

	// 4. Filter: only mainstream models with valid pricing (same rules as catalog)
	result := make([]string, 0)
	for m := range modelSet {
		if !isDisplayModel(m) {
			continue
		}
		if pricing, err := s.billing.GetModelPricing(m); err == nil && pricing != nil {
			if pricing.InputPricePerToken > 0 || pricing.OutputPricePerToken > 0 {
				result = append(result, m)
			}
		}
	}
	sort.Strings(result)
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

	full, err := s.apiKeyService.GetByID(ctx, created.ID)
	if err != nil {
		return nil, fmt.Errorf("get created api key: %w", err)
	}
	return full, nil
}

// findGroupForModel returns a group ID that serves the given model.
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

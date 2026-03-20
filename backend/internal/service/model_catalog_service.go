package service

import (
	"context"
	"strings"
)

// ModelInfo holds public-facing metadata for a model.
type ModelInfo struct {
	ID                string   `json:"id"`
	DisplayName       string   `json:"display_name"`
	Provider          string   `json:"provider"`
	ContextWindow     int      `json:"context_window"`
	MaxOutputTokens   int      `json:"max_output_tokens"`
	InputPriceMTok    float64  `json:"input_price_per_mtok"`
	OutputPriceMTok   float64  `json:"output_price_per_mtok"`
	InputFormats      []string `json:"input_formats"`
	OutputFormats     []string `json:"output_formats"`
	Description       string   `json:"description"`
}

// modelMetadata holds static specs for a known model.
type modelMetadata struct {
	DisplayName     string
	Provider        string
	ContextWindow   int
	MaxOutputTokens int
	InputFormats    []string
	OutputFormats   []string
	Description     string
}

// ModelCatalogService provides model listing and detail functionality.
type ModelCatalogService struct {
	gateway *GatewayService
	billing *BillingService
}

// NewModelCatalogService creates a new ModelCatalogService.
func NewModelCatalogService(gateway *GatewayService, billing *BillingService) *ModelCatalogService {
	return &ModelCatalogService{
		gateway: gateway,
		billing: billing,
	}
}

// ListModels returns all available models enriched with pricing and metadata.
func (s *ModelCatalogService) ListModels(ctx context.Context) ([]*ModelInfo, error) {
	modelIDs := s.gateway.GetAvailableModels(ctx, nil, "")
	models := make([]*ModelInfo, 0, len(modelIDs))
	for _, id := range modelIDs {
		info := s.buildModelInfo(id)
		models = append(models, info)
	}
	return models, nil
}

// GetModel returns info for a single model by ID, or nil if not found.
func (s *ModelCatalogService) GetModel(ctx context.Context, modelID string) (*ModelInfo, error) {
	modelIDs := s.gateway.GetAvailableModels(ctx, nil, "")
	for _, id := range modelIDs {
		if id == modelID {
			return s.buildModelInfo(id), nil
		}
	}
	return nil, nil
}

// buildModelInfo constructs a ModelInfo for a given model ID.
func (s *ModelCatalogService) buildModelInfo(modelID string) *ModelInfo {
	meta := s.getModelMetadata(modelID)
	info := &ModelInfo{
		ID:              modelID,
		DisplayName:     meta.DisplayName,
		Provider:        meta.Provider,
		ContextWindow:   meta.ContextWindow,
		MaxOutputTokens: meta.MaxOutputTokens,
		InputFormats:    meta.InputFormats,
		OutputFormats:   meta.OutputFormats,
		Description:     meta.Description,
	}

	// Enrich with pricing from BillingService.
	if pricing, err := s.billing.GetModelPricing(modelID); err == nil && pricing != nil {
		info.InputPriceMTok = pricing.InputPricePerToken * 1_000_000
		info.OutputPriceMTok = pricing.OutputPricePerToken * 1_000_000
	}

	return info
}

// getModelMetadata returns static metadata for known models.
// Falls back to inferred values for unknown models.
func (s *ModelCatalogService) getModelMetadata(modelID string) modelMetadata {
	textImage := []string{"text", "image"}
	textOnly := []string{"text"}

	registry := map[string]modelMetadata{
		"claude-sonnet-4-20250514": {
			DisplayName:     "Claude Sonnet 4",
			Provider:        "Anthropic",
			ContextWindow:   200000,
			MaxOutputTokens: 16000,
			InputFormats:    textImage,
			OutputFormats:   textOnly,
			Description:     "Anthropic's balanced model offering strong performance and efficiency.",
		},
		"claude-opus-4-20250514": {
			DisplayName:     "Claude Opus 4",
			Provider:        "Anthropic",
			ContextWindow:   200000,
			MaxOutputTokens: 32000,
			InputFormats:    textImage,
			OutputFormats:   textOnly,
			Description:     "Anthropic's most capable model for complex, nuanced tasks.",
		},
		"claude-haiku-3-5-20241022": {
			DisplayName:     "Claude Haiku 3.5",
			Provider:        "Anthropic",
			ContextWindow:   200000,
			MaxOutputTokens: 8000,
			InputFormats:    textImage,
			OutputFormats:   textOnly,
			Description:     "Anthropic's fastest and most compact model for lightweight tasks.",
		},
		"gpt-4o": {
			DisplayName:     "GPT-4o",
			Provider:        "OpenAI",
			ContextWindow:   128000,
			MaxOutputTokens: 16000,
			InputFormats:    textImage,
			OutputFormats:   textOnly,
			Description:     "OpenAI's flagship multimodal model combining speed and intelligence.",
		},
		"gpt-4o-mini": {
			DisplayName:     "GPT-4o Mini",
			Provider:        "OpenAI",
			ContextWindow:   128000,
			MaxOutputTokens: 16000,
			InputFormats:    textImage,
			OutputFormats:   textOnly,
			Description:     "OpenAI's efficient small model for fast, cost-effective tasks.",
		},
		"gemini-2.0-flash": {
			DisplayName:     "Gemini 2.0 Flash",
			Provider:        "Google",
			ContextWindow:   1000000,
			MaxOutputTokens: 8000,
			InputFormats:    textImage,
			OutputFormats:   textOnly,
			Description:     "Google's fast, versatile model with a 1M token context window.",
		},
		"gemini-2.5-pro": {
			DisplayName:     "Gemini 2.5 Pro",
			Provider:        "Google",
			ContextWindow:   1000000,
			MaxOutputTokens: 64000,
			InputFormats:    textImage,
			OutputFormats:   textOnly,
			Description:     "Google's most capable reasoning model with a 1M token context window.",
		},
	}

	if meta, ok := registry[modelID]; ok {
		return meta
	}

	// Fallback: infer basic metadata from the model name.
	return modelMetadata{
		DisplayName:     modelID,
		Provider:        s.guessProvider(modelID),
		ContextWindow:   0,
		MaxOutputTokens: 0,
		InputFormats:    textOnly,
		OutputFormats:   textOnly,
		Description:     "",
	}
}

// guessProvider infers a provider name from the model ID string.
func (s *ModelCatalogService) guessProvider(modelID string) string {
	lower := strings.ToLower(modelID)
	switch {
	case strings.Contains(lower, "claude"):
		return "Anthropic"
	case strings.Contains(lower, "gpt") || strings.Contains(lower, "o1") || strings.Contains(lower, "o3") || strings.Contains(lower, "codex"):
		return "OpenAI"
	case strings.Contains(lower, "gemini"):
		return "Google"
	case strings.Contains(lower, "mistral") || strings.Contains(lower, "mixtral"):
		return "Mistral"
	case strings.Contains(lower, "llama"):
		return "Meta"
	default:
		return "Unknown"
	}
}

package service

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ModelInfo holds public-facing metadata for a model.
type ModelInfo struct {
	ID                    string   `json:"id"`
	DisplayName           string   `json:"display_name"`
	Provider              string   `json:"provider"`
	ContextWindow         int      `json:"context_window"`
	MaxOutputTokens       int      `json:"max_output_tokens"`
	InputPriceMTok        float64  `json:"input_price_per_mtok"`
	OutputPriceMTok       float64  `json:"output_price_per_mtok"`
	OfficialInputPriceMTok  float64 `json:"official_input_price_per_mtok"`
	OfficialOutputPriceMTok float64 `json:"official_output_price_per_mtok"`
	InputFormats          []string `json:"input_formats"`
	OutputFormats         []string `json:"output_formats"`
	Description           string   `json:"description"`
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
	gateway   *GatewayService
	billing   *BillingService
	groupRepo GroupRepository
}

// NewModelCatalogService creates a new ModelCatalogService.
func NewModelCatalogService(gateway *GatewayService, billing *BillingService, groupRepo GroupRepository) *ModelCatalogService {
	return &ModelCatalogService{
		gateway:   gateway,
		billing:   billing,
		groupRepo: groupRepo,
	}
}

// getMinRateMultiplier returns the lowest rate multiplier across all active groups.
func (s *ModelCatalogService) getMinRateMultiplier(ctx context.Context) float64 {
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil || len(groups) == 0 {
		return 1.0
	}
	min := groups[0].RateMultiplier
	for _, g := range groups[1:] {
		if g.RateMultiplier < min {
			min = g.RateMultiplier
		}
	}
	if min <= 0 {
		return 1.0
	}
	return min
}

// dateSuffixRe matches model IDs ending with a date like -2025-08-07 or -20250514
var dateSuffixRe = regexp.MustCompile(`-\d{4}-\d{2}-\d{2}$|^.+-\d{8}$`)

// isDisplayModel returns true if a model should be shown in the catalog/chat.
// Filters out: date-suffixed duplicates, API-only variants (codex/realtime/audio),
// legacy models (gpt-3.5, gpt-4 non-o/non-dot), and preview versions.
// New models from any provider will pass through automatically as long as they
// don't match these patterns.
func isDisplayModel(id string) bool {
	lower := strings.ToLower(id)

	// Skip date-suffixed variants (gpt-5.4-2026-03-05 is same as gpt-5.4)
	if dateSuffixRe.MatchString(lower) {
		return false
	}

	// Skip API-only / developer variants
	for _, kw := range []string{"codex", "realtime", "audio", "search", "instruct"} {
		if strings.Contains(lower, kw) {
			return false
		}
	}

	// Skip -chat / -chat-latest aliases
	if strings.HasSuffix(lower, "-chat") || strings.HasSuffix(lower, "-chat-latest") {
		return false
	}

	// Skip chatgpt- prefix (internal)
	if strings.HasPrefix(lower, "chatgpt-") {
		return false
	}

	// Skip preview/beta versions (gpt-4.5-preview, o1-preview, etc.)
	if strings.Contains(lower, "preview") {
		return false
	}

	// Skip legacy models: gpt-3.5-*, gpt-4 (exact), gpt-4-turbo, gpt-4-*
	// But keep gpt-4o, gpt-4.1, gpt-4.5 etc (they contain a dot or 'o' after '4')
	if strings.HasPrefix(lower, "gpt-3.5") || strings.HasPrefix(lower, "gpt-3") {
		return false
	}
	if lower == "gpt-4" {
		return false
	}
	if strings.HasPrefix(lower, "gpt-4-") {
		return false
	}

	return true
}

// extractModelVersion extracts a numeric version from a model ID for sorting.
// gpt-5.4 -> 5.4, gpt-5.4-pro -> 5.4, gpt-4o -> 4.5 (o=0.5 bonus), gpt-4.1-mini -> 4.1
// o3 -> 3.0, o3-mini -> 3.0, o4-mini -> 4.0
// claude-opus-4 -> 4.0, gemini-2.5-pro -> 2.5
// Returns 0 for unrecognizable models.
func extractModelVersion(id string) float64 {
	lower := strings.ToLower(id)

	// o-series: o1, o3, o4 -> extract the number
	if strings.HasPrefix(lower, "o") && len(lower) > 1 && lower[1] >= '0' && lower[1] <= '9' {
		numStr := ""
		for _, c := range lower[1:] {
			if c >= '0' && c <= '9' || c == '.' {
				numStr += string(c)
			} else {
				break
			}
		}
		if v, err := strconv.ParseFloat(numStr, 64); err == nil {
			return v
		}
	}

	// GPT series: gpt-5.4, gpt-4o, gpt-4.1
	if strings.HasPrefix(lower, "gpt-") {
		rest := lower[4:] // "5.4-pro", "4o-mini", "4.1"
		numStr := ""
		for _, c := range rest {
			if c >= '0' && c <= '9' || c == '.' {
				numStr += string(c)
			} else {
				break
			}
		}
		if v, err := strconv.ParseFloat(numStr, 64); err == nil {
			// gpt-4o gets a 0.5 bonus over gpt-4 (it's newer)
			if strings.HasPrefix(rest, numStr) && len(rest) > len(numStr) && rest[len(numStr)] == 'o' {
				v += 0.5
			}
			return v
		}
	}

	// Claude: extract version from name like claude-opus-4, claude-sonnet-4
	if strings.Contains(lower, "claude") {
		parts := strings.Split(lower, "-")
		for _, p := range parts {
			if v, err := strconv.ParseFloat(p, 64); err == nil {
				return v
			}
		}
	}

	// Gemini: gemini-2.5-pro, gemini-2.0-flash
	if strings.Contains(lower, "gemini") {
		re := regexp.MustCompile(`(\d+\.?\d*)`)
		matches := re.FindStringSubmatch(lower)
		if len(matches) > 1 {
			if v, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return v
			}
		}
	}

	return 0
}

// ListModels returns available models enriched with pricing and metadata.
// Only shows mainstream models with valid pricing — no date variants, legacy, or codex models.
// Sorted by provider, then by official output price descending (strongest/most expensive first).
// This order is fully dynamic — new models automatically appear in the right position
// based on their pricing from the upstream litellm data.
func (s *ModelCatalogService) ListModels(ctx context.Context) ([]*ModelInfo, error) {
	rm := s.getMinRateMultiplier(ctx)
	modelIDs := s.gateway.GetAvailableModels(ctx, nil, "")
	models := make([]*ModelInfo, 0)
	for _, id := range modelIDs {
		if !isDisplayModel(id) {
			continue
		}
		info := s.buildModelInfo(id, rm)
		if info.OfficialInputPriceMTok <= 0 && info.OfficialOutputPriceMTok <= 0 {
			continue
		}
		models = append(models, info)
	}
	// Sort by provider, then by version number descending (newest first).
	// Version is extracted from model ID: gpt-5.4 -> 5.4, o3 -> 3, etc.
	// This is fully automatic — new models sort correctly without code changes.
	sort.Slice(models, func(i, j int) bool {
		if models[i].Provider != models[j].Provider {
			return models[i].Provider < models[j].Provider
		}
		vi, vj := extractModelVersion(models[i].ID), extractModelVersion(models[j].ID)
		if vi != vj {
			return vi > vj // higher version first
		}
		// Same version: pro > base > mini > nano (by price)
		pi := models[i].OfficialOutputPriceMTok
		pj := models[j].OfficialOutputPriceMTok
		if pi != pj {
			return pi > pj
		}
		return models[i].DisplayName > models[j].DisplayName
	})
	return models, nil
}

// GetModel returns info for a single model by ID, or nil if not available.
func (s *ModelCatalogService) GetModel(ctx context.Context, modelID string) (*ModelInfo, error) {
	rm := s.getMinRateMultiplier(ctx)
	modelIDs := s.gateway.GetAvailableModels(ctx, nil, "")
	for _, id := range modelIDs {
		if id == modelID {
			return s.buildModelInfo(id, rm), nil
		}
	}
	return nil, nil
}

// buildModelInfo constructs a ModelInfo for a given model ID.
func (s *ModelCatalogService) buildModelInfo(modelID string, rateMultiplier float64) *ModelInfo {
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

	// Enrich with pricing from BillingService (upstream pricing data).
	if pricing, err := s.billing.GetModelPricing(modelID); err == nil && pricing != nil {
		officialIn := pricing.InputPricePerToken * 1_000_000
		officialOut := pricing.OutputPricePerToken * 1_000_000
		info.OfficialInputPriceMTok = officialIn
		info.OfficialOutputPriceMTok = officialOut
		// Our price = official * lowest group rate multiplier
		info.InputPriceMTok = officialIn * rateMultiplier
		info.OutputPriceMTok = officialOut * rateMultiplier
	}

	return info
}

// getModelMetadata returns metadata for known models, with inferred fallback.
// This is only for enriching display — models shown are determined by account config.
func (s *ModelCatalogService) getModelMetadata(modelID string) modelMetadata {
	textImage := []string{"text", "image"}
	textOnly := []string{"text"}

	// Static metadata for common models (enrichment only, not a filter)
	registry := map[string]modelMetadata{
		// Anthropic
		"claude-sonnet-4-20250514":  {DisplayName: "Claude Sonnet 4", Provider: "Anthropic", ContextWindow: 200000, MaxOutputTokens: 16000, InputFormats: textImage, OutputFormats: textOnly, Description: "Balanced model with strong performance."},
		"claude-opus-4-20250514":    {DisplayName: "Claude Opus 4", Provider: "Anthropic", ContextWindow: 200000, MaxOutputTokens: 32000, InputFormats: textImage, OutputFormats: textOnly, Description: "Most capable model for complex tasks."},
		"claude-haiku-3-5-20241022": {DisplayName: "Claude Haiku 3.5", Provider: "Anthropic", ContextWindow: 200000, MaxOutputTokens: 8000, InputFormats: textImage, OutputFormats: textOnly, Description: "Fastest and most compact model."},
		// OpenAI
		"gpt-4o":        {DisplayName: "GPT-4o", Provider: "OpenAI", ContextWindow: 128000, MaxOutputTokens: 16384, InputFormats: textImage, OutputFormats: textOnly, Description: "Flagship multimodal model."},
		"gpt-4o-mini":   {DisplayName: "GPT-4o Mini", Provider: "OpenAI", ContextWindow: 128000, MaxOutputTokens: 16384, InputFormats: textImage, OutputFormats: textOnly, Description: "Efficient small model for fast tasks."},
		"gpt-4.1":       {DisplayName: "GPT-4.1", Provider: "OpenAI", ContextWindow: 1047576, MaxOutputTokens: 32768, InputFormats: textImage, OutputFormats: textOnly, Description: "Flagship model with 1M context."},
		"gpt-4.1-mini":  {DisplayName: "GPT-4.1 Mini", Provider: "OpenAI", ContextWindow: 1047576, MaxOutputTokens: 32768, InputFormats: textImage, OutputFormats: textOnly, Description: "Fast and affordable with 1M context."},
		"gpt-4.1-nano":  {DisplayName: "GPT-4.1 Nano", Provider: "OpenAI", ContextWindow: 1047576, MaxOutputTokens: 32768, InputFormats: textImage, OutputFormats: textOnly, Description: "Smallest and fastest model."},
		"o1":            {DisplayName: "o1", Provider: "OpenAI", ContextWindow: 200000, MaxOutputTokens: 100000, InputFormats: textImage, OutputFormats: textOnly, Description: "Reasoning model for complex problems."},
		"o1-mini":       {DisplayName: "o1 Mini", Provider: "OpenAI", ContextWindow: 128000, MaxOutputTokens: 65536, InputFormats: textOnly, OutputFormats: textOnly, Description: "Compact reasoning model for STEM."},
		"o3":            {DisplayName: "o3", Provider: "OpenAI", ContextWindow: 200000, MaxOutputTokens: 100000, InputFormats: textImage, OutputFormats: textOnly, Description: "Advanced reasoning model."},
		"o3-mini":       {DisplayName: "o3 Mini", Provider: "OpenAI", ContextWindow: 200000, MaxOutputTokens: 100000, InputFormats: textOnly, OutputFormats: textOnly, Description: "Efficient reasoning at lower cost."},
		"o4-mini":       {DisplayName: "o4 Mini", Provider: "OpenAI", ContextWindow: 200000, MaxOutputTokens: 100000, InputFormats: textImage, OutputFormats: textOnly, Description: "Latest compact reasoning model."},
		"gpt-5":         {DisplayName: "GPT-5", Provider: "OpenAI", ContextWindow: 1047576, MaxOutputTokens: 32768, InputFormats: textImage, OutputFormats: textOnly, Description: "OpenAI's most advanced model."},
		"gpt-5-mini":    {DisplayName: "GPT-5 Mini", Provider: "OpenAI", ContextWindow: 1047576, MaxOutputTokens: 32768, InputFormats: textImage, OutputFormats: textOnly, Description: "Efficient model with GPT-5 capabilities."},
		"gpt-5-nano":    {DisplayName: "GPT-5 Nano", Provider: "OpenAI", ContextWindow: 1047576, MaxOutputTokens: 32768, InputFormats: textImage, OutputFormats: textOnly, Description: "Fastest GPT-5 variant for lightweight tasks."},
		"gpt-5-pro":     {DisplayName: "GPT-5 Pro", Provider: "OpenAI", ContextWindow: 1047576, MaxOutputTokens: 32768, InputFormats: textImage, OutputFormats: textOnly, Description: "Premium GPT-5 variant for complex workloads."},
		"gpt-5.1":       {DisplayName: "GPT-5.1", Provider: "OpenAI", ContextWindow: 1047576, MaxOutputTokens: 32768, InputFormats: textImage, OutputFormats: textOnly, Description: "Improved GPT-5 with enhanced capabilities."},
		"gpt-5.2":       {DisplayName: "GPT-5.2", Provider: "OpenAI", ContextWindow: 1047576, MaxOutputTokens: 32768, InputFormats: textImage, OutputFormats: textOnly, Description: "Further refined GPT-5 generation."},
		"gpt-5.2-pro":   {DisplayName: "GPT-5.2 Pro", Provider: "OpenAI", ContextWindow: 1047576, MaxOutputTokens: 32768, InputFormats: textImage, OutputFormats: textOnly, Description: "Premium variant of GPT-5.2."},
		"gpt-5.4":       {DisplayName: "GPT-5.4", Provider: "OpenAI", ContextWindow: 1047576, MaxOutputTokens: 32768, InputFormats: textImage, OutputFormats: textOnly, Description: "Latest GPT-5 generation model."},
		"o3-pro":        {DisplayName: "o3 Pro", Provider: "OpenAI", ContextWindow: 200000, MaxOutputTokens: 100000, InputFormats: textImage, OutputFormats: textOnly, Description: "Premium reasoning model for the most demanding tasks."},
		// Google
		"gemini-2.0-flash": {DisplayName: "Gemini 2.0 Flash", Provider: "Google", ContextWindow: 1048576, MaxOutputTokens: 8192, InputFormats: textImage, OutputFormats: textOnly, Description: "Fast model with 1M context."},
		"gemini-2.5-pro":   {DisplayName: "Gemini 2.5 Pro", Provider: "Google", ContextWindow: 1048576, MaxOutputTokens: 65536, InputFormats: textImage, OutputFormats: textOnly, Description: "Most capable reasoning model."},
		"gemini-2.5-flash": {DisplayName: "Gemini 2.5 Flash", Provider: "Google", ContextWindow: 1048576, MaxOutputTokens: 65536, InputFormats: textImage, OutputFormats: textOnly, Description: "Fast reasoning model."},
	}

	if meta, ok := registry[modelID]; ok {
		return meta
	}

	// Fallback: infer basic metadata from the model name
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
	case strings.Contains(lower, "gpt") || strings.Contains(lower, "o1") || strings.Contains(lower, "o3") || strings.Contains(lower, "o4") || strings.Contains(lower, "codex") || strings.Contains(lower, "chatgpt"):
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

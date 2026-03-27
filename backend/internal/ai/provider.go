package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TokenUsage tracks token consumption for cost accounting.
type TokenUsage struct {
	PromptTokens     int     `json:"promptTokens"`
	CompletionTokens int     `json:"completionTokens"`
	EstimatedCost    float64 `json:"estimatedCost"`
}

// AIProvider defines the interface for LLM provider adapters.
type AIProvider interface {
	GeneratePlan(ctx context.Context, prompt string) (string, TokenUsage, error)
	Name() string
}

// ProviderRuntimeConfig contains runtime provider configuration.
type ProviderRuntimeConfig struct {
	Provider   string
	Model      string
	APIKey     string
	BaseURL    string
	UseMock    bool
	HTTPClient *http.Client
}

// ProviderTimeoutError indicates the provider exceeded its deadline.
type ProviderTimeoutError struct{ Provider string }

func (e ProviderTimeoutError) Error() string {
	return fmt.Sprintf("%s provider timed out", e.Provider)
}

// ProviderInvalidOutputError indicates the LLM returned non-parseable JSON.
type ProviderInvalidOutputError struct {
	Provider string
	RawBody  string
}

func (e ProviderInvalidOutputError) Error() string {
	return fmt.Sprintf("%s returned invalid JSON output", e.Provider)
}

// ProviderAPIError indicates an upstream provider HTTP/API error.
type ProviderAPIError struct {
	Provider   string
	StatusCode int
	Message    string
}

func (e ProviderAPIError) Error() string {
	return fmt.Sprintf("%s provider api error: status=%d message=%s", e.Provider, e.StatusCode, e.Message)
}

// ProviderCircuitOpenError indicates circuit breaker rejection.
type ProviderCircuitOpenError struct {
	Provider   string
	RetryAfter time.Duration
}

func (e ProviderCircuitOpenError) Error() string {
	return fmt.Sprintf("%s provider circuit is open (retry_after=%s)", e.Provider, e.RetryAfter)
}

const (
	providerFailureThreshold = 3
	providerOpenDuration     = 60 * time.Second
	defaultHTTPTimeout       = 20 * time.Second
)

type providerCircuitState struct {
	ConsecutiveFailures int
	OpenUntil           time.Time
}

var (
	providerCircuitMu sync.Mutex
	providerCircuit   = map[string]providerCircuitState{}
)

func BuildProvider(cfg ProviderRuntimeConfig) AIProvider {
	name := normalizeProviderName(cfg.Provider)
	if cfg.UseMock {
		return GetProvider(name)
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}

	switch name {
	case "openai":
		return &openAIProvider{
			model:      defaultIfEmpty(strings.TrimSpace(cfg.Model), "gpt-4.1-mini"),
			apiKey:     strings.TrimSpace(cfg.APIKey),
			baseURL:    strings.TrimSpace(cfg.BaseURL),
			httpClient: client,
		}
	case "anthropic":
		return &anthropicProvider{
			model:      defaultIfEmpty(strings.TrimSpace(cfg.Model), "claude-sonnet-4-20250514"),
			apiKey:     strings.TrimSpace(cfg.APIKey),
			baseURL:    strings.TrimSpace(cfg.BaseURL),
			httpClient: client,
		}
	case "google":
		return &geminiProvider{
			model:      defaultIfEmpty(strings.TrimSpace(cfg.Model), "gemini-2.0-flash"),
			apiKey:     strings.TrimSpace(cfg.APIKey),
			baseURL:    strings.TrimSpace(cfg.BaseURL),
			httpClient: client,
		}
	default:
		return GetProvider("openai")
	}
}

func normalizeProviderName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case "gemini":
		return "google"
	default:
		return normalized
	}
}

func beforeProviderCall(provider string) error {
	name := normalizeProviderName(provider)
	now := time.Now().UTC()

	providerCircuitMu.Lock()
	defer providerCircuitMu.Unlock()

	state := providerCircuit[name]
	if state.OpenUntil.After(now) {
		return ProviderCircuitOpenError{
			Provider:   name,
			RetryAfter: state.OpenUntil.Sub(now),
		}
	}

	if !state.OpenUntil.IsZero() && !state.OpenUntil.After(now) {
		state.OpenUntil = time.Time{}
		state.ConsecutiveFailures = 0
		providerCircuit[name] = state
	}
	return nil
}

func markProviderSuccess(provider string) {
	name := normalizeProviderName(provider)
	providerCircuitMu.Lock()
	delete(providerCircuit, name)
	providerCircuitMu.Unlock()
}

func markProviderFailure(provider string) {
	name := normalizeProviderName(provider)
	now := time.Now().UTC()

	providerCircuitMu.Lock()
	defer providerCircuitMu.Unlock()

	state := providerCircuit[name]
	state.ConsecutiveFailures++
	if state.ConsecutiveFailures >= providerFailureThreshold {
		state.OpenUntil = now.Add(providerOpenDuration)
		state.ConsecutiveFailures = 0
	}
	providerCircuit[name] = state
}

// --- Mock Provider Implementations ---

// MockOpenAIProvider simulates an OpenAI provider.
type MockOpenAIProvider struct{}

func (p *MockOpenAIProvider) Name() string { return "openai" }

func (p *MockOpenAIProvider) GeneratePlan(ctx context.Context, prompt string) (string, TokenUsage, error) {
	select {
	case <-ctx.Done():
		return "", TokenUsage{}, ProviderTimeoutError{Provider: "openai"}
	default:
	}

	result := mockDraftPayload("OpenAI Generated Plan", prompt)
	usage := TokenUsage{PromptTokens: len(prompt) / 4, CompletionTokens: len(result) / 4, EstimatedCost: 0.015}
	return result, usage, nil
}

// MockAnthropicProvider simulates an Anthropic provider.
type MockAnthropicProvider struct{}

func (p *MockAnthropicProvider) Name() string { return "anthropic" }

func (p *MockAnthropicProvider) GeneratePlan(ctx context.Context, prompt string) (string, TokenUsage, error) {
	select {
	case <-ctx.Done():
		return "", TokenUsage{}, ProviderTimeoutError{Provider: "anthropic"}
	default:
	}

	result := mockDraftPayload("Anthropic Generated Plan", prompt)
	usage := TokenUsage{PromptTokens: len(prompt) / 4, CompletionTokens: len(result) / 4, EstimatedCost: 0.012}
	return result, usage, nil
}

// MockGoogleProvider simulates a Google provider.
type MockGoogleProvider struct{}

func (p *MockGoogleProvider) Name() string { return "google" }

func (p *MockGoogleProvider) GeneratePlan(ctx context.Context, prompt string) (string, TokenUsage, error) {
	select {
	case <-ctx.Done():
		return "", TokenUsage{}, ProviderTimeoutError{Provider: "google"}
	default:
	}

	result := mockDraftPayload("Google Generated Plan", prompt)
	usage := TokenUsage{PromptTokens: len(prompt) / 4, CompletionTokens: len(result) / 4, EstimatedCost: 0.008}
	return result, usage, nil
}

// GetProvider returns a mock provider by name.
func GetProvider(name string) AIProvider {
	switch normalizeProviderName(name) {
	case "openai":
		return &MockOpenAIProvider{}
	case "anthropic":
		return &MockAnthropicProvider{}
	case "google":
		return &MockGoogleProvider{}
	default:
		return &MockOpenAIProvider{}
	}
}

func mockDraftPayload(title, prompt string) string {
	payload := map[string]any{
		"title":   title,
		"summary": "AI-generated itinerary based on user preferences",
		"days": []map[string]any{
			{
				"date":  time.Now().Format("2006-01-02"),
				"theme": "Cultural exploration",
				"items": []map[string]any{
					{"title": "Morning temple visit", "itemType": "place_visit", "confidence": "high"},
					{"title": "Lunch at local restaurant", "itemType": "meal", "confidence": "medium"},
					{"title": "Afternoon sightseeing", "itemType": "place_visit", "confidence": "high"},
				},
			},
		},
	}
	data, _ := json.Marshal(payload)
	_ = prompt
	return string(data)
}

// ParseStructuredOutput attempts to parse the LLM output as JSON.
func ParseStructuredOutput(provider, rawOutput string) (map[string]any, error) {
	normalized := normalizeJSONOutput(rawOutput)
	var result map[string]any
	if err := json.Unmarshal([]byte(normalized), &result); err != nil {
		return nil, ProviderInvalidOutputError{Provider: provider, RawBody: rawOutput}
	}
	return result, nil
}

func normalizeJSONOutput(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```JSON")
		trimmed = strings.TrimPrefix(trimmed, "```")
		trimmed = strings.TrimSuffix(strings.TrimSpace(trimmed), "```")
		trimmed = strings.TrimSpace(trimmed)
	}

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		return strings.TrimSpace(trimmed[start : end+1])
	}
	return trimmed
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	type timeout interface {
		Timeout() bool
	}
	var te timeout
	return errors.As(err, &te) && te.Timeout()
}

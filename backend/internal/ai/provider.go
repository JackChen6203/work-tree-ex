package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	switch strings.ToLower(name) {
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
	return string(data)
}

// ParseStructuredOutput attempts to parse the LLM output as JSON.
func ParseStructuredOutput(provider, rawOutput string) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal([]byte(rawOutput), &result); err != nil {
		return nil, ProviderInvalidOutputError{Provider: provider, RawBody: rawOutput}
	}
	return result, nil
}

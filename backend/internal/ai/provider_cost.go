package ai

import (
	"math"
	"strings"
)

type tokenPricing struct {
	InputPerMillion  float64
	OutputPerMillion float64
}

func estimateOpenAICost(model string, promptTokens, completionTokens int) float64 {
	pricing := tokenPricing{InputPerMillion: 2.00, OutputPerMillion: 8.00}
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "gpt-4.1-mini":
		pricing = tokenPricing{InputPerMillion: 0.40, OutputPerMillion: 1.60}
	case "gpt-4.1":
		pricing = tokenPricing{InputPerMillion: 2.00, OutputPerMillion: 8.00}
	}
	return estimateCost(pricing, promptTokens, completionTokens)
}

func estimateAnthropicCost(model string, promptTokens, completionTokens int) float64 {
	pricing := tokenPricing{InputPerMillion: 3.00, OutputPerMillion: 15.00}
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "claude-sonnet-4-20250514":
		pricing = tokenPricing{InputPerMillion: 3.00, OutputPerMillion: 15.00}
	}
	return estimateCost(pricing, promptTokens, completionTokens)
}

func estimateGeminiCost(model string, promptTokens, completionTokens int) float64 {
	pricing := tokenPricing{InputPerMillion: 0.35, OutputPerMillion: 1.05}
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "gemini-2.0-flash":
		pricing = tokenPricing{InputPerMillion: 0.35, OutputPerMillion: 1.05}
	}
	return estimateCost(pricing, promptTokens, completionTokens)
}

func estimateCost(pricing tokenPricing, promptTokens, completionTokens int) float64 {
	promptCost := (float64(promptTokens) / 1_000_000) * pricing.InputPerMillion
	completionCost := (float64(completionTokens) / 1_000_000) * pricing.OutputPerMillion
	return math.Round((promptCost+completionCost)*1_000_000) / 1_000_000
}

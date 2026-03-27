package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type openAIProvider struct {
	model      string
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func (p *openAIProvider) Name() string { return "openai" }

func (p *openAIProvider) GeneratePlan(ctx context.Context, prompt string) (string, TokenUsage, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return "", TokenUsage{}, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusUnauthorized, Message: "missing api key"}
	}
	if err := beforeProviderCall(p.Name()); err != nil {
		return "", TokenUsage{}, err
	}

	requestBody := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a strict JSON itinerary generator. Return JSON only."},
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.2,
	}
	rawBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", TokenUsage{}, err
	}

	endpoint := strings.TrimRight(defaultIfEmpty(p.baseURL, "https://api.openai.com/v1"), "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawBody))
	if err != nil {
		return "", TokenUsage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	respBody, statusCode, err := doProviderRequest(p.httpClient, req)
	if err != nil {
		markProviderFailure(p.Name())
		if isTimeoutError(err) {
			return "", TokenUsage{}, ProviderTimeoutError{Provider: p.Name()}
		}
		return "", TokenUsage{}, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusBadGateway, Message: err.Error()}
	}
	if statusCode >= http.StatusBadRequest {
		markProviderFailure(p.Name())
		return "", TokenUsage{}, ProviderAPIError{Provider: p.Name(), StatusCode: statusCode, Message: truncateBody(respBody)}
	}

	var payload struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		markProviderFailure(p.Name())
		return "", TokenUsage{}, ProviderInvalidOutputError{Provider: p.Name(), RawBody: string(respBody)}
	}
	if len(payload.Choices) == 0 {
		markProviderFailure(p.Name())
		return "", TokenUsage{}, ProviderInvalidOutputError{Provider: p.Name(), RawBody: string(respBody)}
	}

	content, err := extractOpenAIContent(payload.Choices[0].Message.Content)
	if err != nil {
		markProviderFailure(p.Name())
		return "", TokenUsage{}, ProviderInvalidOutputError{Provider: p.Name(), RawBody: string(respBody)}
	}

	usage := TokenUsage{
		PromptTokens:     payload.Usage.PromptTokens,
		CompletionTokens: payload.Usage.CompletionTokens,
		EstimatedCost:    estimateOpenAICost(p.model, payload.Usage.PromptTokens, payload.Usage.CompletionTokens),
	}
	markProviderSuccess(p.Name())
	return content, usage, nil
}

type anthropicProvider struct {
	model      string
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func (p *anthropicProvider) Name() string { return "anthropic" }

func (p *anthropicProvider) GeneratePlan(ctx context.Context, prompt string) (string, TokenUsage, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return "", TokenUsage{}, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusUnauthorized, Message: "missing api key"}
	}
	if err := beforeProviderCall(p.Name()); err != nil {
		return "", TokenUsage{}, err
	}

	requestBody := map[string]any{
		"model":       p.model,
		"max_tokens":  1800,
		"temperature": 0.2,
		"system":      "You are a strict JSON itinerary generator. Return JSON only.",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	rawBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", TokenUsage{}, err
	}

	endpoint := strings.TrimRight(defaultIfEmpty(p.baseURL, "https://api.anthropic.com"), "/") + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawBody))
	if err != nil {
		return "", TokenUsage{}, err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	respBody, statusCode, err := doProviderRequest(p.httpClient, req)
	if err != nil {
		markProviderFailure(p.Name())
		if isTimeoutError(err) {
			return "", TokenUsage{}, ProviderTimeoutError{Provider: p.Name()}
		}
		return "", TokenUsage{}, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusBadGateway, Message: err.Error()}
	}
	if statusCode >= http.StatusBadRequest {
		markProviderFailure(p.Name())
		return "", TokenUsage{}, ProviderAPIError{Provider: p.Name(), StatusCode: statusCode, Message: truncateBody(respBody)}
	}

	var payload struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		markProviderFailure(p.Name())
		return "", TokenUsage{}, ProviderInvalidOutputError{Provider: p.Name(), RawBody: string(respBody)}
	}

	content := ""
	for _, part := range payload.Content {
		if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
			content = part.Text
			break
		}
	}
	if strings.TrimSpace(content) == "" {
		markProviderFailure(p.Name())
		return "", TokenUsage{}, ProviderInvalidOutputError{Provider: p.Name(), RawBody: string(respBody)}
	}

	usage := TokenUsage{
		PromptTokens:     payload.Usage.InputTokens,
		CompletionTokens: payload.Usage.OutputTokens,
		EstimatedCost:    estimateAnthropicCost(p.model, payload.Usage.InputTokens, payload.Usage.OutputTokens),
	}
	markProviderSuccess(p.Name())
	return content, usage, nil
}

type geminiProvider struct {
	model      string
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func (p *geminiProvider) Name() string { return "google" }

func (p *geminiProvider) GeneratePlan(ctx context.Context, prompt string) (string, TokenUsage, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return "", TokenUsage{}, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusUnauthorized, Message: "missing api key"}
	}
	if err := beforeProviderCall(p.Name()); err != nil {
		return "", TokenUsage{}, err
	}

	requestBody := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature":      0.2,
			"responseMimeType": "application/json",
		},
	}
	rawBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", TokenUsage{}, err
	}

	base := strings.TrimRight(defaultIfEmpty(p.baseURL, "https://generativelanguage.googleapis.com/v1beta"), "/")
	endpoint, err := buildGeminiEndpoint(base, p.model, p.apiKey)
	if err != nil {
		return "", TokenUsage{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawBody))
	if err != nil {
		return "", TokenUsage{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	respBody, statusCode, err := doProviderRequest(p.httpClient, req)
	if err != nil {
		markProviderFailure(p.Name())
		if isTimeoutError(err) {
			return "", TokenUsage{}, ProviderTimeoutError{Provider: p.Name()}
		}
		return "", TokenUsage{}, ProviderAPIError{Provider: p.Name(), StatusCode: http.StatusBadGateway, Message: err.Error()}
	}
	if statusCode >= http.StatusBadRequest {
		markProviderFailure(p.Name())
		return "", TokenUsage{}, ProviderAPIError{Provider: p.Name(), StatusCode: statusCode, Message: truncateBody(respBody)}
	}

	var payload struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		markProviderFailure(p.Name())
		return "", TokenUsage{}, ProviderInvalidOutputError{Provider: p.Name(), RawBody: string(respBody)}
	}
	if len(payload.Candidates) == 0 || len(payload.Candidates[0].Content.Parts) == 0 {
		markProviderFailure(p.Name())
		return "", TokenUsage{}, ProviderInvalidOutputError{Provider: p.Name(), RawBody: string(respBody)}
	}

	content := strings.TrimSpace(payload.Candidates[0].Content.Parts[0].Text)
	if content == "" {
		markProviderFailure(p.Name())
		return "", TokenUsage{}, ProviderInvalidOutputError{Provider: p.Name(), RawBody: string(respBody)}
	}

	usage := TokenUsage{
		PromptTokens:     payload.UsageMetadata.PromptTokenCount,
		CompletionTokens: payload.UsageMetadata.CandidatesTokenCount,
		EstimatedCost:    estimateGeminiCost(p.model, payload.UsageMetadata.PromptTokenCount, payload.UsageMetadata.CandidatesTokenCount),
	}
	markProviderSuccess(p.Name())
	return content, usage, nil
}

func doProviderRequest(client *http.Client, req *http.Request) ([]byte, int, error) {
	httpClient := client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func extractOpenAIContent(content any) (string, error) {
	switch v := content.(type) {
	case string:
		return strings.TrimSpace(v), nil
	case []any:
		builder := strings.Builder{}
		for _, entry := range v {
			part, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			text, _ := part["text"].(string)
			if strings.TrimSpace(text) != "" {
				if builder.Len() > 0 {
					builder.WriteString("\n")
				}
				builder.WriteString(text)
			}
		}
		out := strings.TrimSpace(builder.String())
		if out == "" {
			return "", errors.New("empty openai content array")
		}
		return out, nil
	default:
		return "", errors.New("unknown openai content type")
	}
}

func buildGeminiEndpoint(base, model, apiKey string) (string, error) {
	modelName := strings.TrimSpace(model)
	if modelName == "" {
		modelName = "gemini-2.0-flash"
	}
	u, err := url.Parse(base + "/models/" + modelName + ":generateContent")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("key", apiKey)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func truncateBody(body []byte) string {
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		return "upstream provider error"
	}
	if len(raw) > 240 {
		return raw[:240]
	}
	return raw
}

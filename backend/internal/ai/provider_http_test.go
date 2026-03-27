package ai

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProviderGeneratePlanSuccess(t *testing.T) {
	resetProviderCircuit()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "unexpected endpoint", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-openai-key" {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"{\"title\":\"OpenAI\",\"summary\":\"ok\",\"days\":[]}"}}],
			"usage":{"prompt_tokens":120,"completion_tokens":45}
		}`))
	}))
	defer server.Close()

	provider := BuildProvider(ProviderRuntimeConfig{
		Provider: "openai",
		Model:    "gpt-4.1-mini",
		APIKey:   "test-openai-key",
		BaseURL:  server.URL + "/v1",
	})

	raw, usage, err := provider.GeneratePlan(context.Background(), "hello")
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if usage.PromptTokens != 120 || usage.CompletionTokens != 45 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	if usage.EstimatedCost <= 0 {
		t.Fatalf("expected positive cost, got %f", usage.EstimatedCost)
	}

	if _, err := ParseStructuredOutput(provider.Name(), raw); err != nil {
		t.Fatalf("ParseStructuredOutput error: %v", err)
	}
}

func TestAnthropicProviderGeneratePlanSuccess(t *testing.T) {
	resetProviderCircuit()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
			http.Error(w, "unexpected endpoint", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("x-api-key"); got != "test-anthropic-key" {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"content":[{"type":"text","text":"{\"title\":\"Claude\",\"summary\":\"ok\",\"days\":[]}"}],
			"usage":{"input_tokens":80,"output_tokens":32}
		}`))
	}))
	defer server.Close()

	provider := BuildProvider(ProviderRuntimeConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-20250514",
		APIKey:   "test-anthropic-key",
		BaseURL:  server.URL,
	})

	raw, usage, err := provider.GeneratePlan(context.Background(), "plan")
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if usage.PromptTokens != 80 || usage.CompletionTokens != 32 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	if _, err := ParseStructuredOutput(provider.Name(), raw); err != nil {
		t.Fatalf("ParseStructuredOutput error: %v", err)
	}
}

func TestGeminiProviderGeneratePlanSuccess(t *testing.T) {
	resetProviderCircuit()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1beta/models/gemini-2.0-flash:generateContent" {
			http.Error(w, "unexpected endpoint", http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("key") != "test-gemini-key" {
			http.Error(w, "missing key query", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates":[{"content":{"parts":[{"text":"{\"title\":\"Gemini\",\"summary\":\"ok\",\"days\":[]}"}]}}],
			"usageMetadata":{"promptTokenCount":90,"candidatesTokenCount":30}
		}`))
	}))
	defer server.Close()

	provider := BuildProvider(ProviderRuntimeConfig{
		Provider: "google",
		Model:    "gemini-2.0-flash",
		APIKey:   "test-gemini-key",
		BaseURL:  server.URL + "/v1beta",
	})

	raw, usage, err := provider.GeneratePlan(context.Background(), "plan")
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if usage.PromptTokens != 90 || usage.CompletionTokens != 30 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	if _, err := ParseStructuredOutput(provider.Name(), raw); err != nil {
		t.Fatalf("ParseStructuredOutput error: %v", err)
	}
}

func TestProviderCircuitBreakerOpenAfterConsecutiveFailures(t *testing.T) {
	resetProviderCircuit()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "provider failure", http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := BuildProvider(ProviderRuntimeConfig{
		Provider: "openai",
		Model:    "gpt-4.1-mini",
		APIKey:   "test-openai-key",
		BaseURL:  server.URL + "/v1",
	})

	for i := 0; i < providerFailureThreshold; i++ {
		_, _, err := provider.GeneratePlan(context.Background(), "plan")
		var apiErr ProviderAPIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected ProviderAPIError before circuit opens, got %T %v", err, err)
		}
	}

	_, _, err := provider.GeneratePlan(context.Background(), "plan")
	var circuitErr ProviderCircuitOpenError
	if !errors.As(err, &circuitErr) {
		t.Fatalf("expected ProviderCircuitOpenError, got %T %v", err, err)
	}
}

func TestDecryptProviderAPIKeyEncV1(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	t.Setenv("LLM_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))

	envelope := mustEncryptEnvelope(t, key, "real-secret-key")
	out, err := decryptProviderAPIKey(envelope)
	if err != nil {
		t.Fatalf("decryptProviderAPIKey error: %v", err)
	}
	if out != "real-secret-key" {
		t.Fatalf("unexpected decrypt result: %s", out)
	}

	legacy, err := decryptProviderAPIKey("enc_legacy_secret")
	if err != nil {
		t.Fatalf("legacy fallback error: %v", err)
	}
	if legacy != "legacy_secret" {
		t.Fatalf("unexpected legacy result: %s", legacy)
	}
}

func mustEncryptEnvelope(t *testing.T, key []byte, plaintext string) string {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("new gcm: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("nonce read: %v", err)
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return "encv1:" + base64.RawStdEncoding.EncodeToString(nonce) + ":" + base64.RawStdEncoding.EncodeToString(ciphertext)
}

func resetProviderCircuit() {
	providerCircuitMu.Lock()
	defer providerCircuitMu.Unlock()
	providerCircuit = map[string]providerCircuitState{}
}

package ai

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrProviderConfigNotFound = errors.New("provider config not found")
	ErrProviderKeyDecrypt     = errors.New("provider key decrypt failed")
	ErrProviderUnsupported    = errors.New("provider is not supported")
)

func resolveProviderRuntimeConfig(ctx context.Context, providerConfigID string) (ProviderRuntimeConfig, error) {
	if getPool() == nil {
		return resolveProviderRuntimeConfigMemory(providerConfigID), nil
	}
	return resolveProviderRuntimeConfigPostgres(ctx, providerConfigID)
}

func resolveProviderRuntimeConfigMemory(providerConfigID string) ProviderRuntimeConfig {
	name := strings.ToLower(strings.TrimSpace(providerConfigID))
	switch {
	case strings.Contains(name, "anthropic"), strings.Contains(name, "claude"):
		return ProviderRuntimeConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			UseMock:  true,
		}
	case strings.Contains(name, "google"), strings.Contains(name, "gemini"):
		return ProviderRuntimeConfig{
			Provider: "google",
			Model:    "gemini-2.0-flash",
			UseMock:  true,
		}
	default:
		return ProviderRuntimeConfig{
			Provider: "openai",
			Model:    "gpt-4.1-mini",
			UseMock:  true,
		}
	}
}

func resolveProviderRuntimeConfigPostgres(ctx context.Context, providerConfigID string) (ProviderRuntimeConfig, error) {
	p := getPool()
	if p == nil {
		return ProviderRuntimeConfig{}, errors.New("postgres ai store not configured")
	}
	if _, err := uuid.Parse(strings.TrimSpace(providerConfigID)); err != nil {
		return ProviderRuntimeConfig{}, ErrProviderConfigNotFound
	}

	var (
		provider     string
		model        string
		encryptedKey string
		baseURL      string
	)
	err := p.QueryRow(ctx, `
		SELECT provider, model, encrypted_key, COALESCE(base_url, '')
		FROM llm_provider_configs
		WHERE id = $1::uuid
		  AND is_active = true
	`, providerConfigID).Scan(&provider, &model, &encryptedKey, &baseURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderRuntimeConfig{}, ErrProviderConfigNotFound
	}
	if err != nil {
		return ProviderRuntimeConfig{}, err
	}

	apiKey, err := decryptProviderAPIKey(strings.TrimSpace(encryptedKey))
	if err != nil {
		return ProviderRuntimeConfig{}, fmt.Errorf("%w: %v", ErrProviderKeyDecrypt, err)
	}

	normalizedProvider := normalizeProviderName(provider)
	if normalizedProvider != "openai" && normalizedProvider != "anthropic" && normalizedProvider != "google" {
		return ProviderRuntimeConfig{}, fmt.Errorf("%w: %s", ErrProviderUnsupported, provider)
	}

	return ProviderRuntimeConfig{
		Provider: normalizedProvider,
		Model:    strings.TrimSpace(model),
		APIKey:   strings.TrimSpace(apiKey),
		BaseURL:  strings.TrimSpace(baseURL),
		UseMock:  false,
	}, nil
}

func decryptProviderAPIKey(envelope string) (string, error) {
	value := strings.TrimSpace(envelope)
	if value == "" {
		return "", errors.New("empty encrypted key")
	}

	if strings.HasPrefix(value, "encv1:") {
		parts := strings.SplitN(value, ":", 3)
		if len(parts) != 3 {
			return "", errors.New("invalid envelope format")
		}

		key, err := loadProviderEncryptionKey()
		if err != nil {
			return "", err
		}

		nonce, err := decodeBase64(parts[1])
		if err != nil {
			return "", fmt.Errorf("decode nonce: %w", err)
		}
		ciphertext, err := decodeBase64(parts[2])
		if err != nil {
			return "", fmt.Errorf("decode ciphertext: %w", err)
		}

		block, err := aes.NewCipher(key)
		if err != nil {
			return "", fmt.Errorf("create cipher: %w", err)
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return "", fmt.Errorf("create gcm: %w", err)
		}
		if len(nonce) != gcm.NonceSize() {
			return "", fmt.Errorf("invalid nonce size: got=%d want=%d", len(nonce), gcm.NonceSize())
		}

		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return "", fmt.Errorf("decrypt payload: %w", err)
		}
		return strings.TrimSpace(string(plaintext)), nil
	}

	// Backward-compatible fallback:
	// Existing dev data may be stored as "enc_<raw-key>" before AES envelope rollout.
	if strings.HasPrefix(value, "enc_") {
		legacy := strings.TrimSpace(strings.TrimPrefix(value, "enc_"))
		if legacy == "" {
			return "", errors.New("legacy encrypted key payload is empty")
		}
		return legacy, nil
	}

	// Treat plain text as development fallback.
	return value, nil
}

func loadProviderEncryptionKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("LLM_ENCRYPTION_KEY"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("LLM_PROVIDER_ENCRYPTION_KEY"))
	}
	if raw == "" {
		return nil, errors.New("LLM_ENCRYPTION_KEY is required for AES-256-GCM key envelopes")
	}

	if len(raw) == 32 {
		return []byte(raw), nil
	}

	decoded, err := decodeBase64(raw)
	if err == nil {
		if len(decoded) != 32 {
			return nil, fmt.Errorf("invalid decoded key length: %d (expected 32)", len(decoded))
		}
		return decoded, nil
	}

	return nil, errors.New("encryption key must be 32-byte raw string or base64-encoded 32 bytes")
}

func decodeBase64(input string) ([]byte, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return nil, errors.New("empty base64 payload")
	}

	decoders := []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.URLEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
	}
	for _, decode := range decoders {
		decoded, err := decode(value)
		if err == nil {
			return decoded, nil
		}
	}
	return nil, errors.New("invalid base64 payload")
}

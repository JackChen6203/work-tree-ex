package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type pushMessage struct {
	Title string
	Body  string
	Data  map[string]string
}

type pushGatewayResult struct {
	SuccessCount  int
	FailureCount  int
	Retryable     bool
	InvalidTokens []string
}

type pushGateway interface {
	Send(ctx context.Context, tokens []string, msg pushMessage) (pushGatewayResult, error)
}

type noopPushGateway struct{}

func (g *noopPushGateway) Send(_ context.Context, tokens []string, _ pushMessage) (pushGatewayResult, error) {
	return pushGatewayResult{
		SuccessCount: len(tokens),
		FailureCount: 0,
		Retryable:    false,
	}, nil
}

type httpFCMPushGateway struct {
	client    *http.Client
	endpoint  string
	serverKey string
}

func (g *httpFCMPushGateway) Send(ctx context.Context, tokens []string, msg pushMessage) (pushGatewayResult, error) {
	if len(tokens) == 0 {
		return pushGatewayResult{}, nil
	}

	payload := map[string]any{
		"registration_ids": tokens,
		"notification": map[string]string{
			"title": msg.Title,
			"body":  msg.Body,
		},
		"data":     msg.Data,
		"priority": "high",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return pushGatewayResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint, bytes.NewReader(body))
	if err != nil {
		return pushGatewayResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+g.serverKey)

	resp, err := g.client.Do(req)
	if err != nil {
		return pushGatewayResult{
			SuccessCount: 0,
			FailureCount: len(tokens),
			Retryable:    isRetryableFCMError(err),
		}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return pushGatewayResult{}, err
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		return pushGatewayResult{
			SuccessCount: 0,
			FailureCount: len(tokens),
			Retryable:    true,
		}, errors.New("fcm temporary failure")
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return pushGatewayResult{
			SuccessCount: 0,
			FailureCount: len(tokens),
			Retryable:    false,
		}, errors.New("fcm request rejected")
	}

	var fcmResp struct {
		Success int `json:"success"`
		Failure int `json:"failure"`
		Results []struct {
			Error string `json:"error"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &fcmResp); err != nil {
		return pushGatewayResult{}, err
	}

	out := pushGatewayResult{
		SuccessCount: fcmResp.Success,
		FailureCount: fcmResp.Failure,
	}
	for idx, result := range fcmResp.Results {
		errCode := strings.TrimSpace(result.Error)
		if errCode == "" {
			continue
		}
		if isInvalidTokenErrorCode(errCode) && idx < len(tokens) {
			out.InvalidTokens = append(out.InvalidTokens, tokens[idx])
			continue
		}
		if isRetryableErrorCode(errCode) {
			out.Retryable = true
		}
	}
	return out, nil
}

var (
	pushGatewayOnce sync.Once
	globalGateway   pushGateway
	pushGatewayMu   sync.RWMutex
	overrideGateway pushGateway
)

func getPushGateway(_ context.Context) pushGateway {
	pushGatewayMu.RLock()
	custom := overrideGateway
	pushGatewayMu.RUnlock()
	if custom != nil {
		return custom
	}

	pushGatewayOnce.Do(func() {
		globalGateway = buildPushGateway()
	})
	if globalGateway == nil {
		return &noopPushGateway{}
	}
	return globalGateway
}

func buildPushGateway() pushGateway {
	serverKey := strings.TrimSpace(os.Getenv("FCM_SERVER_KEY"))
	if serverKey == "" {
		return &noopPushGateway{}
	}

	endpoint := strings.TrimSpace(os.Getenv("FCM_SEND_ENDPOINT"))
	if endpoint == "" {
		endpoint = "https://fcm.googleapis.com/fcm/send"
	}

	log.Printf("notifications: FCM HTTP gateway enabled")
	return &httpFCMPushGateway{
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
		endpoint:  endpoint,
		serverKey: serverKey,
	}
}

func setPushGatewayForTest(gateway pushGateway) {
	pushGatewayMu.Lock()
	defer pushGatewayMu.Unlock()
	overrideGateway = gateway
}

func resetPushGatewayForTest() {
	pushGatewayMu.Lock()
	overrideGateway = nil
	pushGatewayMu.Unlock()

	pushGatewayOnce = sync.Once{}
	globalGateway = nil
}

func isRetryableFCMError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unavailable") ||
		strings.Contains(message, "internal") ||
		strings.Contains(message, "deadline exceeded") ||
		strings.Contains(message, "timeout")
}

func isInvalidTokenErrorCode(code string) bool {
	value := strings.ToLower(strings.TrimSpace(code))
	return value == "invalidregistration" || value == "notregistered"
}

func isRetryableErrorCode(code string) bool {
	value := strings.ToLower(strings.TrimSpace(code))
	return value == "unavailable" ||
		value == "internalservererror" ||
		value == "devicemessagerateexceeded" ||
		value == "topicsmessagerateexceeded"
}

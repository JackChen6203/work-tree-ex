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

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
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

type firebaseAdminPushGateway struct {
	client *messaging.Client
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

func (g *firebaseAdminPushGateway) Send(ctx context.Context, tokens []string, msg pushMessage) (pushGatewayResult, error) {
	if len(tokens) == 0 {
		return pushGatewayResult{}, nil
	}

	multicast := &messaging.MulticastMessage{
		Tokens: tokens,
		Data:   msg.Data,
		Notification: &messaging.Notification{
			Title: msg.Title,
			Body:  msg.Body,
		},
		Webpush: &messaging.WebpushConfig{
			Notification: &messaging.WebpushNotification{
				Title: msg.Title,
				Body:  msg.Body,
			},
		},
	}

	resp, err := g.client.SendEachForMulticast(ctx, multicast)
	if err != nil {
		return pushGatewayResult{
			SuccessCount: 0,
			FailureCount: len(tokens),
			Retryable:    isRetryableFCMError(err),
		}, err
	}

	out := pushGatewayResult{
		SuccessCount: resp.SuccessCount,
		FailureCount: resp.FailureCount,
	}
	for idx, item := range resp.Responses {
		if item == nil || item.Success || item.Error == nil {
			continue
		}
		if isInvalidTokenError(item.Error) && idx < len(tokens) {
			out.InvalidTokens = append(out.InvalidTokens, tokens[idx])
			continue
		}
		if isRetryableFCMError(item.Error) {
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
	if gateway, ok := buildFirebaseAdminGateway(); ok {
		return gateway
	}

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

func buildFirebaseAdminGateway() (pushGateway, bool) {
	credentialsJSON := strings.TrimSpace(os.Getenv("FCM_SERVICE_ACCOUNT_JSON"))
	credentialsFile := strings.TrimSpace(os.Getenv("FCM_SERVICE_ACCOUNT_FILE"))
	projectID := strings.TrimSpace(os.Getenv("FCM_PROJECT_ID"))
	if credentialsJSON == "" && credentialsFile == "" {
		return nil, false
	}

	var cfg *firebase.Config
	if projectID != "" {
		cfg = &firebase.Config{ProjectID: projectID}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	credentialsPath := credentialsFile
	if credentialsPath == "" {
		tmpFile, err := os.CreateTemp("", "fcm-service-account-*.json")
		if err != nil {
			log.Printf("notifications: failed to create temp credentials file: %v", err)
			return nil, false
		}
		if _, err := tmpFile.WriteString(credentialsJSON); err != nil {
			_ = tmpFile.Close()
			log.Printf("notifications: failed to write temp credentials file: %v", err)
			return nil, false
		}
		if err := tmpFile.Chmod(0o600); err != nil {
			_ = tmpFile.Close()
			log.Printf("notifications: failed to chmod temp credentials file: %v", err)
			return nil, false
		}
		if err := tmpFile.Close(); err != nil {
			log.Printf("notifications: failed to close temp credentials file: %v", err)
			return nil, false
		}
		credentialsPath = tmpFile.Name()
	}
	if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentialsPath); err != nil {
		log.Printf("notifications: failed to set GOOGLE_APPLICATION_CREDENTIALS: %v", err)
		return nil, false
	}

	app, err := firebase.NewApp(ctx, cfg)
	if err != nil {
		log.Printf("notifications: failed to init Firebase Admin SDK: %v", err)
		return nil, false
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		log.Printf("notifications: failed to init Firebase messaging client: %v", err)
		return nil, false
	}

	log.Printf("notifications: Firebase Admin SDK gateway enabled")
	return &firebaseAdminPushGateway{client: client}, true
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
		strings.Contains(message, "resource-exhausted") ||
		strings.Contains(message, "rate exceeded") ||
		strings.Contains(message, "deadline exceeded") ||
		strings.Contains(message, "timeout")
}

func isInvalidTokenError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "not registered") ||
		strings.Contains(message, "registration-token-not-registered") ||
		strings.Contains(message, "invalid registration token") ||
		strings.Contains(message, "invalid-argument")
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

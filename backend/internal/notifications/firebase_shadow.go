package notifications

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	firebase "firebase.google.com/go/v4"
	firebaseDB "firebase.google.com/go/v4/db"
)

type shadowSyncInput struct {
	EventType      string
	ResourceID     string
	UserID         string
	NotificationID string
	Payload        map[string]any
	Timestamp      time.Time
}

type shadowSyncer interface {
	Sync(ctx context.Context, in shadowSyncInput) error
}

type noopShadowSyncer struct{}

func (s *noopShadowSyncer) Sync(_ context.Context, _ shadowSyncInput) error {
	return nil
}

type disabledShadowSyncer struct {
	reason string
}

func (s *disabledShadowSyncer) Sync(_ context.Context, _ shadowSyncInput) error {
	reason := strings.TrimSpace(s.reason)
	if reason == "" {
		reason = "firebase shadow sync is enabled but unavailable"
	}
	return errors.New(reason)
}

type realtimeDBShadowSyncer struct {
	client     *firebaseDB.Client
	pathPrefix string
}

func (s *realtimeDBShadowSyncer) Sync(ctx context.Context, in shadowSyncInput) error {
	if s == nil || s.client == nil {
		return errors.New("firebase shadow client not initialized")
	}

	userSegment := pathSafeSegment(in.UserID, "default")
	key := sanitizeShadowKey(in.NotificationID)
	if key == "" {
		key = fmt.Sprintf("evt-%d", in.Timestamp.UnixNano())
	}

	payload := map[string]any{
		"eventType":      strings.TrimSpace(in.EventType),
		"resourceId":     strings.TrimSpace(in.ResourceID),
		"userId":         strings.TrimSpace(in.UserID),
		"notificationId": strings.TrimSpace(in.NotificationID),
		"createdAt":      in.Timestamp.UTC().Format(time.RFC3339Nano),
		"payload":        in.Payload,
	}
	refPath := strings.Trim(strings.Join([]string{s.pathPrefix, userSegment, "outbox_events", key}, "/"), "/")
	return s.client.NewRef(refPath).Set(ctx, payload)
}

var (
	shadowSyncOnce     sync.Once
	shadowSyncMu       sync.RWMutex
	globalShadowSyncer shadowSyncer
	overrideShadowSync shadowSyncer
)

func syncFirebaseShadow(ctx context.Context, eventType, resourceID string, payload map[string]any, userID, notificationID string) error {
	input := shadowSyncInput{
		EventType:      strings.TrimSpace(eventType),
		ResourceID:     strings.TrimSpace(resourceID),
		UserID:         strings.TrimSpace(userID),
		NotificationID: strings.TrimSpace(notificationID),
		Payload:        payload,
		Timestamp:      time.Now().UTC(),
	}
	return getShadowSyncer().Sync(ctx, input)
}

func getShadowSyncer() shadowSyncer {
	shadowSyncMu.RLock()
	custom := overrideShadowSync
	shadowSyncMu.RUnlock()
	if custom != nil {
		return custom
	}

	shadowSyncOnce.Do(func() {
		globalShadowSyncer = buildShadowSyncer()
	})
	if globalShadowSyncer == nil {
		return &noopShadowSyncer{}
	}
	return globalShadowSyncer
}

func buildShadowSyncer() shadowSyncer {
	if !envBool("FIREBASE_SHADOW_ENABLED", false) {
		return &noopShadowSyncer{}
	}

	databaseURL := strings.TrimSpace(os.Getenv("FIREBASE_DATABASE_URL"))
	if databaseURL == "" {
		reason := "firebase shadow sync enabled but FIREBASE_DATABASE_URL is empty"
		log.Printf("notifications: %s", reason)
		return &disabledShadowSyncer{reason: reason}
	}

	credentialsJSON := strings.TrimSpace(os.Getenv("FCM_SERVICE_ACCOUNT_JSON"))
	credentialsFile := strings.TrimSpace(os.Getenv("FCM_SERVICE_ACCOUNT_FILE"))
	if credentialsJSON == "" && credentialsFile == "" {
		reason := "firebase shadow sync enabled but service account credentials are missing"
		log.Printf("notifications: %s", reason)
		return &disabledShadowSyncer{reason: reason}
	}

	projectID := strings.TrimSpace(os.Getenv("FCM_PROJECT_ID"))
	cfg := &firebase.Config{
		DatabaseURL: databaseURL,
	}
	if projectID != "" {
		cfg.ProjectID = projectID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	credentialsPath := credentialsFile
	if credentialsPath == "" {
		tmpFile, err := os.CreateTemp("", "firebase-shadow-service-account-*.json")
		if err != nil {
			log.Printf("notifications: failed to create temp shadow credentials file: %v", err)
			return &disabledShadowSyncer{reason: "failed to create shadow credentials file"}
		}
		if _, err := tmpFile.WriteString(credentialsJSON); err != nil {
			_ = tmpFile.Close()
			log.Printf("notifications: failed to write temp shadow credentials file: %v", err)
			return &disabledShadowSyncer{reason: "failed to write shadow credentials file"}
		}
		if err := tmpFile.Chmod(0o600); err != nil {
			_ = tmpFile.Close()
			log.Printf("notifications: failed to chmod temp shadow credentials file: %v", err)
			return &disabledShadowSyncer{reason: "failed to chmod shadow credentials file"}
		}
		if err := tmpFile.Close(); err != nil {
			log.Printf("notifications: failed to close temp shadow credentials file: %v", err)
			return &disabledShadowSyncer{reason: "failed to close shadow credentials file"}
		}
		credentialsPath = tmpFile.Name()
	}

	if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentialsPath); err != nil {
		log.Printf("notifications: failed to set GOOGLE_APPLICATION_CREDENTIALS for shadow sync: %v", err)
		return &disabledShadowSyncer{reason: "failed to set GOOGLE_APPLICATION_CREDENTIALS"}
	}

	app, err := firebase.NewApp(ctx, cfg)
	if err != nil {
		log.Printf("notifications: failed to init Firebase app for shadow sync: %v", err)
		return &disabledShadowSyncer{reason: "failed to init firebase app for shadow sync"}
	}

	client, err := app.Database(ctx)
	if err != nil {
		log.Printf("notifications: failed to init Firebase Database client: %v", err)
		return &disabledShadowSyncer{reason: "failed to init firebase database client"}
	}

	prefix := strings.Trim(strings.TrimSpace(os.Getenv("FIREBASE_SHADOW_PATH_PREFIX")), "/")
	if prefix == "" {
		prefix = "shadow"
	}
	log.Printf("notifications: firebase shadow sync enabled (prefix=%s)", prefix)
	return &realtimeDBShadowSyncer{
		client:     client,
		pathPrefix: prefix,
	}
}

func pathSafeSegment(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, ":", "_")
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.Trim(value, "._")
	if value == "" {
		return fallback
	}
	return value
}

func sanitizeShadowKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, ":", "_")
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.Trim(value, "._")
	return value
}

func envBool(key string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func setShadowSyncerForTest(syncer shadowSyncer) {
	shadowSyncMu.Lock()
	defer shadowSyncMu.Unlock()
	overrideShadowSync = syncer
}

func resetShadowSyncerForTest() {
	shadowSyncMu.Lock()
	overrideShadowSync = nil
	shadowSyncMu.Unlock()

	shadowSyncOnce = sync.Once{}
	globalShadowSyncer = nil
}

package notifications

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"
)

func ConsumeOutboxEvent(ctx context.Context, eventType, resourceID string, payload map[string]any) error {
	now := time.Now().UTC()
	in := triggerInput{
		EventType:  strings.TrimSpace(eventType),
		ResourceID: strings.TrimSpace(resourceID),
		Title:      inferOutboxTitle(eventType, payload),
		Body:       inferOutboxBody(payload),
		Link:       inferOutboxLink(resourceID, payload),
		UserID:     defaultNotificationUserID,
	}

	notifID := ""
	if defaultDeliveryPrefs.InApp {
		if getPool() != nil {
			id, err := createNotificationPostgres(ctx, in, now)
			if err != nil {
				return err
			}
			notifID = id
		} else {
			notificationsMu.Lock()
			notifID = "n-" + strconv.Itoa(len(items)+1)
			items = append(items, notification{
				ID:        notifID,
				Type:      in.EventType,
				Title:     in.Title,
				Body:      in.Body,
				Link:      in.Link,
				CreatedAt: now,
			})
			notificationsMu.Unlock()
		}
	}
	if notifID == "" {
		notifID = "evt-" + strconv.FormatInt(now.UnixNano(), 10)
	}

	if defaultDeliveryPrefs.Push {
		tokens, err := listActivePushTokens(ctx, in.UserID)
		pd := &pushDelivery{
			NotificationID: notifID,
			Status:         "sent",
			RetryCount:     0,
			LastAttemptAt:  now,
		}
		if err != nil {
			pd.Status = "failed"
		} else if len(tokens) > 0 {
			status, retryCount, invalidTokens, _ := sendPushWithRetry(ctx, tokens, pushMessage{
				Title: in.Title,
				Body:  in.Body,
				Data: map[string]string{
					"notificationId": notifID,
					"eventType":      in.EventType,
					"resourceId":     in.ResourceID,
					"link":           in.Link,
				},
			})
			pd.Status = status
			pd.RetryCount = retryCount
			if pd.Status == "dlq" {
				log.Printf("notifications: worker push delivery moved to dlq (notification_id=%s event_type=%s resource_id=%s)", notifID, in.EventType, in.ResourceID)
			}
			if len(invalidTokens) > 0 {
				_ = deactivateFCMTokens(ctx, invalidTokens)
			}
		}

		notificationsMu.Lock()
		pushDeliveries[notifID] = pd
		notificationsMu.Unlock()
	}

	return nil
}

func inferOutboxTitle(eventType string, payload map[string]any) string {
	if value, ok := payload["title"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	title := strings.TrimSpace(eventType)
	if title == "" {
		title = "sync.event"
	}
	return "Sync: " + title
}

func inferOutboxBody(payload map[string]any) string {
	if value, ok := payload["body"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return "A new update was synchronized."
}

func inferOutboxLink(resourceID string, payload map[string]any) string {
	if value, ok := payload["link"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	tripID := strings.TrimSpace(resourceID)
	if tripID == "" {
		return "/dashboard"
	}
	return "/trips/" + tripID
}

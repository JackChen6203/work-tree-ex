package notifications

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/mailer"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/users"
)

func StartDigestWorkers(ctx context.Context, dailyInterval, weeklyInterval time.Duration) {
	if dailyInterval <= 0 {
		dailyInterval = 24 * time.Hour
	}
	if weeklyInterval <= 0 {
		weeklyInterval = 7 * 24 * time.Hour
	}

	go runDigestLoop(ctx, dailyInterval, "daily", 24*time.Hour)
	go runDigestLoop(ctx, weeklyInterval, "weekly", 7*24*time.Hour)
}

func runDigestLoop(ctx context.Context, interval time.Duration, frequency string, window time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runDigestCycle(ctx, frequency, window)
		}
	}
}

func runDigestCycle(ctx context.Context, frequency string, window time.Duration) {
	userIDs, err := listDigestUserIDs(ctx, window)
	if err != nil {
		log.Printf("notifications: digest cycle failed to list users (%s): %v", frequency, err)
		return
	}
	if len(userIDs) == 0 {
		return
	}

	now := time.Now().UTC()
	since := now.Add(-window)
	for _, userID := range userIDs {
		prefs := users.ResolveDeliveryPreferences(userID, "trip_update_digest")
		if !prefs.Email {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(prefs.DigestFrequency), strings.TrimSpace(frequency)) {
			continue
		}

		recipient := strings.TrimSpace(prefs.EmailRecipient)
		if getPool() != nil {
			if value, err := lookupUserEmailPostgres(ctx, userID); err == nil && strings.TrimSpace(value) != "" {
				recipient = strings.TrimSpace(value)
			}
		}
		if recipient == "" {
			continue
		}

		digestEntries, err := listDigestEntries(ctx, userID, since, 20)
		if err != nil {
			log.Printf("notifications: digest cycle failed to list entries (%s user=%s): %v", frequency, userID, err)
			continue
		}
		if len(digestEntries) == 0 {
			continue
		}

		message := mailer.BuildTripDigestMessage(recipient, prefs.Locale, frequency, now, digestEntries)
		if err := mailer.Send(ctx, message); err != nil {
			log.Printf("notifications: digest email send failed (%s user=%s): %v", frequency, userID, err)
			continue
		}
		log.Printf("notifications: digest email sent (%s user=%s entries=%d)", frequency, userID, len(digestEntries))
	}
}

func listDigestUserIDs(ctx context.Context, window time.Duration) ([]string, error) {
	if getPool() != nil {
		p := getPool()
		rows, err := p.Query(ctx, `
			SELECT DISTINCT user_id::text
			FROM notifications
			WHERE created_at >= $1
		`, time.Now().UTC().Add(-window))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		userIDs := make([]string, 0)
		for rows.Next() {
			var userID string
			if err := rows.Scan(&userID); err != nil {
				return nil, err
			}
			userID = strings.TrimSpace(userID)
			if userID != "" {
				userIDs = append(userIDs, userID)
			}
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return userIDs, nil
	}

	return []string{defaultNotificationUserID}, nil
}

func listDigestEntries(ctx context.Context, userID string, since time.Time, limit int) ([]mailer.DigestEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	if getPool() != nil {
		p := getPool()
		rows, err := p.Query(ctx, `
			SELECT title, body, COALESCE(link, ''), created_at
			FROM notifications
			WHERE user_id = $1::uuid
			  AND created_at >= $2
			ORDER BY created_at DESC
			LIMIT $3
		`, userID, since, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		entries := make([]mailer.DigestEntry, 0, limit)
		for rows.Next() {
			var (
				title     string
				body      string
				link      string
				createdAt time.Time
			)
			if err := rows.Scan(&title, &body, &link, &createdAt); err != nil {
				return nil, err
			}
			entries = append(entries, mailer.DigestEntry{
				Title:     strings.TrimSpace(title),
				Body:      strings.TrimSpace(body),
				Link:      strings.TrimSpace(link),
				CreatedAt: createdAt.UTC().Format("2006-01-02 15:04"),
			})
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return entries, nil
	}

	notificationsMu.RLock()
	defer notificationsMu.RUnlock()

	entries := make([]mailer.DigestEntry, 0, limit)
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item.CreatedAt.Before(since) {
			continue
		}
		entries = append(entries, mailer.DigestEntry{
			Title:     strings.TrimSpace(item.Title),
			Body:      strings.TrimSpace(item.Body),
			Link:      strings.TrimSpace(item.Link),
			CreatedAt: item.CreatedAt.UTC().Format("2006-01-02 15:04"),
		})
		if len(entries) >= limit {
			break
		}
	}
	return entries, nil
}

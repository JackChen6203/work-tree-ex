package notifications

import (
	"context"
	"fmt"
	"strings"

	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/mailer"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/users"
)

func sendNotificationEmail(ctx context.Context, userID string, in triggerInput, notificationID string) error {
	snapshot := users.ResolveDeliveryPreferences(userID, in.EventType)
	recipient := strings.TrimSpace(snapshot.EmailRecipient)
	if getPool() != nil {
		if dbRecipient, err := lookupUserEmailPostgres(ctx, userID); err == nil && strings.TrimSpace(dbRecipient) != "" {
			recipient = strings.TrimSpace(dbRecipient)
		}
	}
	if recipient == "" {
		trimmedUserID := strings.TrimSpace(userID)
		if strings.Contains(trimmedUserID, "@") {
			recipient = trimmedUserID
		} else {
			recipient = "user-" + trimmedUserID + "@time-tree.local"
		}
	}

	link := strings.TrimSpace(in.Link)
	if link == "" {
		link = "/dashboard"
	}

	message := mailer.Message{
		To:      []string{recipient},
		Subject: in.Title,
		Text:    fmt.Sprintf("%s\n\nOpen: %s\nNotification ID: %s", in.Body, link, strings.TrimSpace(notificationID)),
		HTML: fmt.Sprintf(
			"<p>%s</p><p><a href=\"%s\">Open notification</a></p><p>Notification ID: <code>%s</code></p>",
			htmlEscape(in.Body),
			htmlEscape(link),
			htmlEscape(strings.TrimSpace(notificationID)),
		),
	}
	return mailer.Send(ctx, message)
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(strings.TrimSpace(value))
}

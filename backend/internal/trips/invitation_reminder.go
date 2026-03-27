package trips

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/mailer"
)

var (
	invitationReminderMu   sync.Mutex
	invitationReminderSent = map[string]time.Time{}
)

func StartInvitationReminderWorker(ctx context.Context, interval, lookahead time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}
	if lookahead <= 0 {
		lookahead = 24 * time.Hour
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sendInvitationReminders(ctx, lookahead)
			}
		}
	}()
}

func sendInvitationReminders(ctx context.Context, lookahead time.Duration) {
	items, err := listExpiringInvitations(ctx, lookahead)
	if err != nil {
		log.Printf("trips: failed to list expiring invitations for reminder: %v", err)
		return
	}
	for _, inv := range items {
		if isInvitationReminderSent(inv.ID) {
			continue
		}

		tripName := resolveTripNameForReminder(ctx, inv.TripID)
		message := mailer.BuildInviteReminderMessage(
			inv.Email,
			defaultReminderLocale(),
			inviterName(inv.InvitedBy),
			tripName,
			buildInvitationReminderURL(inv),
			inv.ExpiresAt,
		)
		if err := mailer.Send(ctx, message); err != nil {
			log.Printf("trips: invitation reminder email failed (invitation_id=%s email=%s): %v", inv.ID, inv.Email, err)
			continue
		}

		markInvitationReminderSent(inv.ID)
		log.Printf("trips: invitation reminder email sent (invitation_id=%s email=%s)", inv.ID, inv.Email)
	}
	cleanupInvitationReminderSent()
}

func listExpiringInvitations(ctx context.Context, lookahead time.Duration) ([]invitation, error) {
	now := time.Now().UTC()
	cutoff := now.Add(lookahead)
	if getCollaborationPool() != nil {
		return listExpiringInvitationsPostgres(ctx, now, cutoff)
	}

	invitationMu.RLock()
	defer invitationMu.RUnlock()

	items := make([]invitation, 0)
	for _, inv := range invitationByID {
		if inv == nil {
			continue
		}
		if strings.TrimSpace(inv.Status) != "pending" {
			continue
		}
		if inv.ExpiresAt.Before(now) || inv.ExpiresAt.After(cutoff) {
			continue
		}
		items = append(items, *inv)
	}
	return items, nil
}

func resolveTripNameForReminder(ctx context.Context, tripID string) string {
	item, err := activeRepository.Get(ctx, tripID)
	if err != nil || strings.TrimSpace(item.Name) == "" {
		return "Your trip"
	}
	return strings.TrimSpace(item.Name)
}

func buildInvitationReminderURL(inv invitation) string {
	baseURL := strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return fmt.Sprintf(
		"%s/invitations/accept?tripId=%s&invitationId=%s",
		baseURL,
		url.QueryEscape(strings.TrimSpace(inv.TripID)),
		url.QueryEscape(strings.TrimSpace(inv.ID)),
	)
}

func defaultReminderLocale() string {
	value := strings.TrimSpace(os.Getenv("DEFAULT_LOCALE"))
	if value == "" {
		return "zh-TW"
	}
	return value
}

func isInvitationReminderSent(invitationID string) bool {
	invitationReminderMu.Lock()
	defer invitationReminderMu.Unlock()
	_, ok := invitationReminderSent[strings.TrimSpace(invitationID)]
	return ok
}

func markInvitationReminderSent(invitationID string) {
	invitationReminderMu.Lock()
	defer invitationReminderMu.Unlock()
	invitationReminderSent[strings.TrimSpace(invitationID)] = time.Now().UTC()
}

func cleanupInvitationReminderSent() {
	invitationReminderMu.Lock()
	defer invitationReminderMu.Unlock()
	cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour)
	for invitationID, sentAt := range invitationReminderSent {
		if sentAt.Before(cutoff) {
			delete(invitationReminderSent, invitationID)
		}
	}
}

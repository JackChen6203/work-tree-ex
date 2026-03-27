package mailer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type testSender struct {
	err error
}

func (s testSender) Send(context.Context, Message) error {
	return s.err
}

func TestFallbackSenderUsesFallbackOnPrimaryError(t *testing.T) {
	sender := &fallbackSender{
		primary:  testSender{err: errors.New("primary failed")},
		fallback: testSender{},
	}

	if err := sender.Send(context.Background(), Message{To: []string{"demo@example.com"}}); err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}
}

func TestBuildMagicLinkMessageLocales(t *testing.T) {
	zh := BuildMagicLinkMessage("demo@example.com", "zh-TW", "123456", "https://example.com", time.Now().Add(10*time.Minute))
	if !strings.Contains(zh.Subject, "驗證碼") {
		t.Fatalf("expected zh subject, got %s", zh.Subject)
	}
	if !strings.Contains(zh.Text, "123456") {
		t.Fatalf("expected code in zh text")
	}

	en := BuildMagicLinkMessage("demo@example.com", "en-US", "654321", "https://example.com", time.Now().Add(10*time.Minute))
	if !strings.Contains(strings.ToLower(en.Subject), "magic") {
		t.Fatalf("expected en subject, got %s", en.Subject)
	}
	if !strings.Contains(en.HTML, "654321") {
		t.Fatalf("expected code in en html")
	}
}

func TestBuildInviteReminderMessageLocales(t *testing.T) {
	zh := BuildInviteReminderMessage(
		"demo@example.com",
		"zh-TW",
		"Ariel",
		"Tokyo Trip",
		"https://example.com/invitations/accept",
		time.Now().Add(24*time.Hour),
	)
	if !strings.Contains(zh.Subject, "提醒") {
		t.Fatalf("expected zh reminder subject, got %s", zh.Subject)
	}
	if !strings.Contains(zh.HTML, "接受邀請") {
		t.Fatalf("expected zh reminder html to contain accept action, got %s", zh.HTML)
	}

	en := BuildInviteReminderMessage(
		"demo@example.com",
		"en-US",
		"Ariel",
		"Tokyo Trip",
		"https://example.com/invitations/accept",
		time.Now().Add(24*time.Hour),
	)
	if !strings.Contains(strings.ToLower(en.Subject), "reminder") {
		t.Fatalf("expected en reminder subject, got %s", en.Subject)
	}
	if !strings.Contains(en.Text, "Tokyo Trip") {
		t.Fatalf("expected en reminder text to contain trip name, got %s", en.Text)
	}
}

func TestBuildTripDigestMessage(t *testing.T) {
	generatedAt := time.Now().UTC()
	entries := []DigestEntry{
		{
			Title:     "Trip updated",
			Body:      "Flight time changed",
			Link:      "/trips/t-1",
			CreatedAt: generatedAt.Format("2006-01-02 15:04"),
		},
	}

	daily := BuildTripDigestMessage("demo@example.com", "en-US", "daily", generatedAt, entries)
	if !strings.Contains(strings.ToLower(daily.Subject), "daily") {
		t.Fatalf("expected daily digest subject, got %s", daily.Subject)
	}
	if !strings.Contains(daily.Text, "Flight time changed") {
		t.Fatalf("expected digest text to contain entry body, got %s", daily.Text)
	}

	weeklyZh := BuildTripDigestMessage("demo@example.com", "zh-TW", "weekly", generatedAt, nil)
	if !strings.Contains(weeklyZh.Subject, "每週") {
		t.Fatalf("expected zh weekly digest subject, got %s", weeklyZh.Subject)
	}
	if !strings.Contains(weeklyZh.HTML, "本期沒有新的更新") {
		t.Fatalf("expected zh empty digest html message, got %s", weeklyZh.HTML)
	}
}

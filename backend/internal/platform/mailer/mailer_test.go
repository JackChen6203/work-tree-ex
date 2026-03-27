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

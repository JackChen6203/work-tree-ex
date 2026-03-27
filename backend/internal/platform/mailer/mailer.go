package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Message struct {
	To      []string
	Subject string
	HTML    string
	Text    string
}

type Sender interface {
	Send(ctx context.Context, message Message) error
}

type noopSender struct{}

func (s *noopSender) Send(_ context.Context, _ Message) error {
	return nil
}

type resendSender struct {
	client   *http.Client
	endpoint string
	apiKey   string
	from     string
}

type sendgridSender struct {
	client   *http.Client
	endpoint string
	apiKey   string
	from     string
}

type fallbackSender struct {
	primary  Sender
	fallback Sender
}

func (s *fallbackSender) Send(ctx context.Context, message Message) error {
	if err := s.primary.Send(ctx, message); err == nil {
		return nil
	} else if s.fallback != nil {
		if fallbackErr := s.fallback.Send(ctx, message); fallbackErr == nil {
			return nil
		} else {
			return fmt.Errorf("primary and fallback email provider failed: %w", errors.Join(err, fallbackErr))
		}
	} else {
		return err
	}
}

func (s *resendSender) Send(ctx context.Context, message Message) error {
	if len(message.To) == 0 {
		return errors.New("resend message requires recipients")
	}

	body, err := json.Marshal(map[string]any{
		"from":    s.from,
		"to":      message.To,
		"subject": message.Subject,
		"html":    message.HTML,
		"text":    message.Text,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	raw, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("resend API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
}

func (s *sendgridSender) Send(ctx context.Context, message Message) error {
	if len(message.To) == 0 {
		return errors.New("sendgrid message requires recipients")
	}

	toList := make([]map[string]string, 0, len(message.To))
	for _, recipient := range message.To {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			continue
		}
		toList = append(toList, map[string]string{"email": recipient})
	}
	if len(toList) == 0 {
		return errors.New("sendgrid message requires non-empty recipients")
	}

	body, err := json.Marshal(map[string]any{
		"personalizations": []map[string]any{
			{"to": toList},
		},
		"from": map[string]string{
			"email": s.from,
		},
		"subject": message.Subject,
		"content": []map[string]string{
			{"type": "text/plain", "value": message.Text},
			{"type": "text/html", "value": message.HTML},
		},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	raw, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("sendgrid API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
}

var (
	senderOnce sync.Once
	senderMu   sync.RWMutex
	global     Sender
	override   Sender
)

func Send(ctx context.Context, message Message) error {
	sender := getSender()
	return sender.Send(ctx, message)
}

func getSender() Sender {
	senderMu.RLock()
	custom := override
	senderMu.RUnlock()
	if custom != nil {
		return custom
	}

	senderOnce.Do(func() {
		global = buildSenderFromEnv()
	})
	if global == nil {
		return &noopSender{}
	}
	return global
}

func buildSenderFromEnv() Sender {
	timeout := time.Duration(envInt("EMAIL_SEND_TIMEOUT_SEC", 8)) * time.Second
	from := strings.TrimSpace(os.Getenv("EMAIL_FROM"))
	if from == "" {
		from = "no-reply@time-tree.local"
	}

	primaryName := strings.ToLower(strings.TrimSpace(os.Getenv("EMAIL_PROVIDER_PRIMARY")))
	fallbackName := strings.ToLower(strings.TrimSpace(os.Getenv("EMAIL_PROVIDER_FALLBACK")))
	if primaryName == "" {
		if strings.TrimSpace(os.Getenv("RESEND_API_KEY")) != "" {
			primaryName = "resend"
		} else if strings.TrimSpace(os.Getenv("SENDGRID_API_KEY")) != "" {
			primaryName = "sendgrid"
		} else {
			primaryName = "noop"
		}
	}

	primary := buildProvider(primaryName, from, timeout)
	fallback := buildProvider(fallbackName, from, timeout)
	if primary == nil {
		if fallback != nil {
			return fallback
		}
		return &noopSender{}
	}
	if fallback == nil || strings.EqualFold(strings.TrimSpace(fallbackName), "noop") {
		return primary
	}
	return &fallbackSender{
		primary:  primary,
		fallback: fallback,
	}
}

func buildProvider(name, from string, timeout time.Duration) Sender {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "noop":
		return &noopSender{}
	case "resend":
		apiKey := strings.TrimSpace(os.Getenv("RESEND_API_KEY"))
		if apiKey == "" {
			return nil
		}
		endpoint := strings.TrimSpace(os.Getenv("RESEND_API_ENDPOINT"))
		if endpoint == "" {
			endpoint = "https://api.resend.com/emails"
		}
		return &resendSender{
			client: &http.Client{
				Timeout: timeout,
			},
			endpoint: endpoint,
			apiKey:   apiKey,
			from:     from,
		}
	case "sendgrid":
		apiKey := strings.TrimSpace(os.Getenv("SENDGRID_API_KEY"))
		if apiKey == "" {
			return nil
		}
		endpoint := strings.TrimSpace(os.Getenv("SENDGRID_API_ENDPOINT"))
		if endpoint == "" {
			endpoint = "https://api.sendgrid.com/v3/mail/send"
		}
		return &sendgridSender{
			client: &http.Client{
				Timeout: timeout,
			},
			endpoint: endpoint,
			apiKey:   apiKey,
			from:     from,
		}
	default:
		return nil
	}
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func SetSenderForTest(sender Sender) {
	senderMu.Lock()
	defer senderMu.Unlock()
	override = sender
}

func ResetForTest() {
	senderMu.Lock()
	override = nil
	senderMu.Unlock()

	senderOnce = sync.Once{}
	global = nil
}

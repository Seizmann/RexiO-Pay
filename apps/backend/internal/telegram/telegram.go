// Package telegram sends alert messages to Sijan's Telegram chat.
// All calls are fire-and-forget (run in a goroutine); a missing or empty
// bot token is a no-op with a warning log.
package telegram

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Alerter sends Telegram alerts.
type Alerter struct {
	botToken string
	chatID   string
	client   *http.Client
}

// New creates an Alerter. If botToken or chatID is empty, all sends are no-ops.
func New(botToken, chatID string) *Alerter {
	return &Alerter{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Send fires a Telegram message asynchronously. Never blocks.
func (a *Alerter) Send(text string) {
	if a.botToken == "" || a.chatID == "" {
		slog.Warn("telegram: bot token or chat ID not configured, skipping alert", "text", text)
		return
	}
	go func() {
		if err := a.send(context.Background(), text); err != nil {
			slog.Error("telegram: failed to send alert", "err", err, "text", text)
		}
	}()
}

func (a *Alerter) send(ctx context.Context, text string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", a.botToken)
	resp, err := a.client.PostForm(endpoint, url.Values{
		"chat_id": {a.chatID},
		"text":    {text},
	})
	if err != nil {
		return fmt.Errorf("telegram.send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram.send: HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

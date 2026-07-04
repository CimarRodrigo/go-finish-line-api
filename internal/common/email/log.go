package email

import (
	"context"
	"log/slog"
)

// LogSender logs messages instead of sending them — the development fallback
// when no provider is configured, so local work needs no API key.
type LogSender struct{}

var _ Sender = (*LogSender)(nil)

func NewLogSender() *LogSender { return &LogSender{} }

func (s *LogSender) Send(_ context.Context, msg Message) error {
	slog.Info("email not sent (log sender)",
		"to", msg.To,
		"subject", msg.Subject,
	)
	return nil
}

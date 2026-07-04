package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

// ResendSender delivers email through the Resend API.
type ResendSender struct {
	client *resend.Client
	from   string
}

var _ Sender = (*ResendSender)(nil)

func NewResendSender(apiKey, from string) *ResendSender {
	return &ResendSender{client: resend.NewClient(apiKey), from: from}
}

func (s *ResendSender) Send(ctx context.Context, msg Message) error {
	_, err := s.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{msg.To},
		Subject: msg.Subject,
		Html:    msg.HTML,
	})
	if err != nil {
		return fmt.Errorf("sending email via resend: %w", err)
	}
	return nil
}

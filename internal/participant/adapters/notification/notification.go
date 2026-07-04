// Package notification implements the participant module's Notifier port by
// building the confirmation email content and delegating delivery to a shared
// email Sender.
package notification

import (
	"context"
	"fmt"

	"finish-line/internal/common/email"
	"finish-line/internal/participant/domain"
	"finish-line/internal/participant/ports"
	racedomain "finish-line/internal/race/domain"
)

type ConfirmationNotifier struct {
	sender email.Sender
}

var _ ports.Notifier = (*ConfirmationNotifier)(nil)

func NewConfirmationNotifier(sender email.Sender) *ConfirmationNotifier {
	return &ConfirmationNotifier{sender: sender}
}

func (n *ConfirmationNotifier) SendConfirmation(ctx context.Context, p *domain.Participant, r *domain.Registration, race *racedomain.Race) error {
	dorsal := "—"
	if r.Dorsal != nil {
		dorsal = fmt.Sprintf("%d", *r.Dorsal)
	}

	msg := email.Message{
		To:      p.Email,
		Subject: fmt.Sprintf("Confirmación de inscripción — %s", race.Name),
		HTML: fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
  <h2>¡Estás inscrito, %s!</h2>
  <p>Tu inscripción a <strong>%s</strong> quedó confirmada.</p>
  <p style="font-size: 20px;">Tu dorsal es <strong>%s</strong>.</p>
  <p>¡Nos vemos en la carrera!</p>
</div>`, p.FirstNames, race.Name, dorsal),
	}

	return n.sender.Send(ctx, msg)
}

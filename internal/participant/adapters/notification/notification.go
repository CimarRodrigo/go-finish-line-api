// Package notification implements the participant module's Notifier port by
// building the confirmation email content and delegating delivery to a shared
// email Sender.
package notification

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"time"

	"finish-line/internal/common/email"
	"finish-line/internal/participant/domain"
	"finish-line/internal/participant/ports"
	racedomain "finish-line/internal/race/domain"
)

//go:embed templates/confirmation.html.tmpl
var confirmationTemplateSource string

var confirmationTemplate = template.Must(template.New("confirmation").Parse(confirmationTemplateSource))

var monthsES = [...]string{
	"enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
}

// confirmationEmailData is the data the confirmation template renders. It is
// html/template (not text/template), so every field is auto-escaped in its
// HTML context — a participant name containing "<" or "&" can't break the
// layout or inject markup.
type confirmationEmailData struct {
	FirstName string
	RaceName  string
	DateLabel string
	Modalidad string
	Dorsal    string
	LogoURL   string
}

type ConfirmationNotifier struct {
	sender          email.Sender
	frontendBaseURL string
}

var _ ports.Notifier = (*ConfirmationNotifier)(nil)

func NewConfirmationNotifier(sender email.Sender, frontendBaseURL string) *ConfirmationNotifier {
	return &ConfirmationNotifier{sender: sender, frontendBaseURL: frontendBaseURL}
}

func (n *ConfirmationNotifier) SendConfirmation(ctx context.Context, p *domain.Participant, r *domain.Registration, race *racedomain.Race) error {
	dorsal := "—"
	if r.Dorsal != nil {
		dorsal = fmt.Sprintf("%d", *r.Dorsal)
	}

	modalidad := r.Modalidad
	if modalidad == "" {
		modalidad = "—"
	}

	data := confirmationEmailData{
		FirstName: p.FirstNames,
		RaceName:  race.Name,
		DateLabel: spanishDateLabel(race.Date),
		Modalidad: modalidad,
		Dorsal:    dorsal,
		LogoURL:   n.frontendBaseURL + "/email/isotype-mark.png",
	}

	var body bytes.Buffer
	if err := confirmationTemplate.Execute(&body, data); err != nil {
		return fmt.Errorf("render confirmation email: %w", err)
	}

	msg := email.Message{
		To:      p.Email,
		Subject: fmt.Sprintf("Confirmación de inscripción — %s", race.Name),
		HTML:    body.String(),
	}

	return n.sender.Send(ctx, msg)
}

// spanishDateLabel formats a date as "16 de agosto de 2026" — Go's time
// package has no Spanish month names built in.
func spanishDateLabel(t time.Time) string {
	return fmt.Sprintf("%d de %s de %d", t.Day(), monthsES[t.Month()-1], t.Year())
}

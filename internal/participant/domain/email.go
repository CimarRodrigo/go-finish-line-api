package domain

import (
	"net/mail"
	"strings"
)

// NormalizeEmail lowercases, trims and validates an email address so the
// same identity always compares equal regardless of how it was typed. Kept
// local to this module on purpose: a little copying beats a dependency on
// another module's domain.
func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return "", ErrEmailInvalid
	}
	return email, nil
}

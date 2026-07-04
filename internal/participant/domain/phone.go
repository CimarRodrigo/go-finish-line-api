package domain

import (
	"regexp"
	"strings"
)

var phonePattern = regexp.MustCompile(`^\+?[0-9]{7,15}$`)

// NormalizePhone strips formatting characters and validates the result:
// an optional leading +, then 7 to 15 digits.
func NormalizePhone(phone string) (string, error) {
	cleaned := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(phone))
	if !phonePattern.MatchString(cleaned) {
		return "", ErrPhoneInvalid
	}
	return cleaned, nil
}

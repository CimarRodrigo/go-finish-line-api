package domain

import (
	"regexp"
	"strings"
)

// documentIDPattern accepts a Bolivian CI (cédula de identidad): 5 to 10
// digits, optionally followed by a dash and a short alphanumeric complement
// (e.g. "1234567" or "1234567-1K"). Loose on purpose — v1 only stores the
// value, it does not validate against RENIEC/SEGIP nor enforce uniqueness
// (see design decision: CI dedup is out of scope, email already covers
// per-race duplicate registration).
var documentIDPattern = regexp.MustCompile(`^[0-9]{5,10}(-[0-9A-Za-z]{1,3})?$`)

// NormalizeDocumentID trims the input and validates its shape.
func NormalizeDocumentID(documentID string) (string, error) {
	cleaned := strings.TrimSpace(documentID)
	if !documentIDPattern.MatchString(cleaned) {
		return "", ErrDocumentIDInvalid
	}
	return cleaned, nil
}

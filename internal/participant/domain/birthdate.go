package domain

import "time"

// ValidateBirthDate rejects birth dates in the future or unrealistically old
// (over 120 years). Minimum-age policy is a pending business decision.
func ValidateBirthDate(birth, now time.Time) error {
	if birth.After(now) {
		return ErrBirthDateInFuture
	}
	if birth.Before(now.AddDate(-120, 0, 0)) {
		return ErrBirthDateUnrealistic
	}
	return nil
}

package domain

import "strings"

type Gender string

const (
	GenderMale   Gender = "M"
	GenderFemale Gender = "F"
	GenderOther  Gender = "X"
)

// ParseGender normalizes raw form input into a valid Gender.
func ParseGender(raw string) (Gender, error) {
	g := Gender(strings.ToUpper(strings.TrimSpace(raw)))
	switch g {
	case GenderMale, GenderFemale, GenderOther:
		return g, nil
	}
	return "", ErrGenderInvalid
}

package domain

import "strings"

// ShirtSize is the shirt size chosen for a single registration. It lives on
// the registration, not the participant: the same runner may pick a different
// size in a later race, and each race keeps its own historical record.
type ShirtSize string

const (
	ShirtSizeXS  ShirtSize = "XS"
	ShirtSizeS   ShirtSize = "S"
	ShirtSizeM   ShirtSize = "M"
	ShirtSizeL   ShirtSize = "L"
	ShirtSizeXL  ShirtSize = "XL"
	ShirtSizeXXL ShirtSize = "XXL"
)

// ParseShirtSize normalizes raw form input into a valid ShirtSize. It is
// optional: an empty input yields an empty size, because a modalidad without a
// shirt has no size to record. A non-empty value must match the size chart, so
// what reaches the report can be counted — the whole point of collecting it.
func ParseShirtSize(raw string) (ShirtSize, error) {
	s := ShirtSize(strings.ToUpper(strings.TrimSpace(raw)))
	switch s {
	case "":
		return "", nil
	case ShirtSizeXS, ShirtSizeS, ShirtSizeM, ShirtSizeL, ShirtSizeXL, ShirtSizeXXL:
		return s, nil
	}
	return "", ErrShirtSizeInvalid
}

package domain

import "finish-line/internal/apperr"

var (
	// ErrAgeRangeInvalid covers a negative age and a min above the max: both
	// describe a set of people that cannot exist.
	ErrAgeRangeInvalid = apperr.New(apperr.KindValidation, "age range is invalid")
)

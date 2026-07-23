package domain

import "finish-line/internal/apperr"

var (
	// ErrDocumentIDRequired keeps its legacy name (see race.go) but the message
	// is CMS-neutral: the value it validates is now the Sanity slug.
	ErrDocumentIDRequired = apperr.New(apperr.KindValidation, "race external id is required")
	ErrNameRequired       = apperr.New(apperr.KindValidation, "race name is required")
	ErrCapacityInvalid    = apperr.New(apperr.KindValidation, "capacity must be greater than zero")
	ErrNotFound           = apperr.New(apperr.KindNotFound, "race not found")
)

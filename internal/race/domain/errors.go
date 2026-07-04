package domain

import "finish-line/internal/apperr"

var (
	ErrStrapiIDRequired = apperr.New(apperr.KindValidation, "strapi id is required")
	ErrNameRequired     = apperr.New(apperr.KindValidation, "race name is required")
	ErrCapacityInvalid  = apperr.New(apperr.KindValidation, "capacity must be greater than zero")
	ErrNotFound         = apperr.New(apperr.KindNotFound, "race not found")
)

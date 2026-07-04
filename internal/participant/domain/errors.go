package domain

import "finish-line/internal/apperr"

var (
	ErrParticipantRequired  = apperr.New(apperr.KindValidation, "participant is required")
	ErrRaceRequired         = apperr.New(apperr.KindValidation, "race is required")
	ErrFirstNamesRequired   = apperr.New(apperr.KindValidation, "first names are required")
	ErrLastNamesRequired    = apperr.New(apperr.KindValidation, "last names are required")
	ErrEmailInvalid         = apperr.New(apperr.KindValidation, "email is invalid")
	ErrPhoneInvalid         = apperr.New(apperr.KindValidation, "phone number is invalid")
	ErrBirthDateInFuture    = apperr.New(apperr.KindValidation, "birth date cannot be in the future")
	ErrBirthDateUnrealistic = apperr.New(apperr.KindValidation, "birth date is not realistic")
	ErrGenderInvalid        = apperr.New(apperr.KindValidation, "gender must be M, F or X")
	ErrReferralRequired     = apperr.New(apperr.KindValidation, "referral source is required")
	ErrTicketRequired       = apperr.New(apperr.KindValidation, "ticket type is required")
	ErrDorsalInvalid        = apperr.New(apperr.KindValidation, "dorsal must be greater than zero")

	ErrAlreadyRegistered = apperr.New(apperr.KindConflict, "already registered for this race")
	ErrRaceFull          = apperr.New(apperr.KindConflict, "race capacity is full")
	ErrAlreadyConfirmed  = apperr.New(apperr.KindConflict, "registration is already confirmed")

	ErrNotFound = apperr.New(apperr.KindNotFound, "registration not found")
)

package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Registration is the N:M link between a Participant and a Race — one row per
// person per race. Dorsal and ConfirmedAt are nil until confirmed.
type Registration struct {
	ID             uuid.UUID
	ParticipantID  uuid.UUID
	RaceID         uuid.UUID
	ReferralSource string
	Modalidad      string
	ShirtSize      ShirtSize
	Status         Status
	Dorsal         *int
	CreatedAt      time.Time
	ConfirmedAt    *time.Time
}

// RegistrationParams are the inputs NewRegistration validates. It is a struct
// so the two adjacent free-text fields (referral source, modalidad) and the
// shirt size are labelled at the call site and cannot be swapped silently.
type RegistrationParams struct {
	ParticipantID  uuid.UUID
	RaceID         uuid.UUID
	ReferralSource string
	Modalidad      string
	ShirtSize      string
}

// Validate checks the inputs that do not depend on a persisted id. The service
// calls it before writing anything, so a rejected registration never leaves a
// participant row behind; NewRegistration calls it too, so the domain remains
// the single authority on what a valid registration is.
func (p RegistrationParams) Validate() error {
	if strings.TrimSpace(p.ReferralSource) == "" {
		return ErrReferralRequired
	}
	if _, err := ParseShirtSize(p.ShirtSize); err != nil {
		return err
	}
	return nil
}

// NewRegistration builds a valid pending registration.
//
// Modalidad records the distance/variant the runner already picked on the
// detail page (e.g. "10K · Con polera") — display data carried through from
// the form, not re-asked here, so it is stored as given without its own
// validation rule.
//
// ShirtSize, in contrast, IS validated against the size chart: it exists to be
// counted when ordering shirts, so free-form values would defeat its purpose.
// It stays optional — a modalidad without a shirt has no size to record — and
// lives here rather than on the participant because the same runner may choose
// a different size in a later race.
func NewRegistration(p RegistrationParams) (*Registration, error) {
	if p.ParticipantID == uuid.Nil {
		return nil, ErrParticipantRequired
	}
	if p.RaceID == uuid.Nil {
		return nil, ErrRaceRequired
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	shirtSize, err := ParseShirtSize(p.ShirtSize)
	if err != nil {
		return nil, err
	}

	return &Registration{
		ID:             uuid.New(),
		ParticipantID:  p.ParticipantID,
		RaceID:         p.RaceID,
		ReferralSource: strings.TrimSpace(p.ReferralSource),
		Modalidad:      strings.TrimSpace(p.Modalidad),
		ShirtSize:      shirtSize,
		Status:         StatusPending,
		CreatedAt:      time.Now().UTC(),
	}, nil
}

// Confirm transitions the registration to confirmed with its dorsal. All the
// business rules of confirmation live here: it refuses to run twice, rejects a
// non-positive dorsal, and rejects a dorsal beyond the race capacity (the race
// is full). The service supplies the candidate dorsal and capacity; the
// decision is the domain's.
func (r *Registration) Confirm(dorsal, capacity int, at time.Time) error {
	if r.Status == StatusConfirmed {
		return ErrAlreadyConfirmed
	}
	if dorsal <= 0 {
		return ErrDorsalInvalid
	}
	if dorsal > capacity {
		return ErrRaceFull
	}

	r.Status = StatusConfirmed
	r.Dorsal = &dorsal
	r.ConfirmedAt = &at
	return nil
}

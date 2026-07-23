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
	Status         Status
	Dorsal         *int
	CreatedAt      time.Time
	ConfirmedAt    *time.Time
}

// NewRegistration builds a valid pending registration. modalidad records the
// distance/variant the runner already picked on the detail page (e.g. "10K ·
// Con polera") — display data carried through from the form, not re-asked
// here, so it is stored as given without its own validation rule.
func NewRegistration(participantID, raceID uuid.UUID, referralSource, modalidad string) (*Registration, error) {
	if participantID == uuid.Nil {
		return nil, ErrParticipantRequired
	}
	if raceID == uuid.Nil {
		return nil, ErrRaceRequired
	}

	referral := strings.TrimSpace(referralSource)
	if referral == "" {
		return nil, ErrReferralRequired
	}

	return &Registration{
		ID:             uuid.New(),
		ParticipantID:  participantID,
		RaceID:         raceID,
		ReferralSource: referral,
		Modalidad:      strings.TrimSpace(modalidad),
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

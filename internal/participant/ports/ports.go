package ports

import (
	"context"

	"github.com/google/uuid"

	"finish-line/internal/participant/domain"
	racedomain "finish-line/internal/race/domain"
)

// ParticipantRepository persists people, deduplicated by email.
type ParticipantRepository interface {
	// UpsertByEmail creates the person or, if the email already exists,
	// refreshes their profile and returns the existing row (stable id).
	UpsertByEmail(ctx context.Context, p *domain.Participant) (*domain.Participant, error)
	ByID(ctx context.Context, id uuid.UUID) (*domain.Participant, error)
}

// RegistrationRepository persists the N:M participations.
type RegistrationRepository interface {
	// Create persists a pending registration. A (race_id, participant_id)
	// unique violation becomes domain.ErrAlreadyRegistered.
	Create(ctx context.Context, r *domain.Registration) error
	ByID(ctx context.Context, id uuid.UUID) (*domain.Registration, error)
	// ConfirmNext atomically reserves the next sequential dorsal for the
	// registration's race and lets the domain decide whether to confirm. The
	// adapter owns only the infrastructure — transaction, per-race lock, and
	// computing the candidate dorsal — then calls confirm with the loaded
	// registration and that candidate. The confirm callback (supplied by the
	// service) applies the domain transition; returning a domain error from it
	// aborts the transaction. No business rule lives in the adapter.
	ConfirmNext(ctx context.Context, id uuid.UUID, confirm func(reg *domain.Registration, nextDorsal int) error) (*domain.Registration, error)
	// ByRace returns a race's registrations joined with their people — the
	// admin report.
	ByRace(ctx context.Context, raceID uuid.UUID) ([]domain.RegistrationDetail, error)
}

// RaceFinder is the narrow view this module needs of the race module.
type RaceFinder interface {
	ByID(ctx context.Context, id uuid.UUID) (*racedomain.Race, error)
}

// Notifier delivers registration notifications (the email module implements
// it). Failures are tolerated by the service: a confirmed registration is
// never rolled back because a notification could not be sent.
type Notifier interface {
	SendConfirmation(ctx context.Context, p *domain.Participant, r *domain.Registration, race *racedomain.Race) error
}

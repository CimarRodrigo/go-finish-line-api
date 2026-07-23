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

// RegistrationRepository persists the N:M participations. Its methods are
// single-purpose infrastructure: the service composes them (and retries on a
// dorsal collision), keeping the confirmation logic in the service and domain.
type RegistrationRepository interface {
	// Create persists a pending registration. A (race_id, participant_id)
	// unique violation becomes domain.ErrAlreadyRegistered.
	Create(ctx context.Context, r *domain.Registration) error
	ByID(ctx context.Context, id uuid.UUID) (*domain.Registration, error)
	// NextDorsal returns the next candidate dorsal for a race: the current
	// highest confirmed dorsal plus one. It does not reserve anything — two
	// concurrent callers may get the same number; the unique (race_id, dorsal)
	// constraint rejects the loser at save time.
	NextDorsal(ctx context.Context, raceID uuid.UUID) (int, error)
	// SaveConfirmation persists a confirmed registration. A (race_id, dorsal)
	// unique violation becomes domain.ErrDorsalTaken so the service can retry.
	SaveConfirmation(ctx context.Context, r *domain.Registration) error
	// ByRace returns a race's registrations joined with their people — the
	// admin report.
	ByRace(ctx context.Context, raceID uuid.UUID) ([]domain.RegistrationDetail, error)
}

// RaceFinder is the narrow view this module needs of the race module.
// Registration comes in keyed by the race documentId (the Sanity slug, the id
// the public form holds), so it resolves via ByDocumentID; ByID resolves the
// internal race for an existing registration during confirmation.
type RaceFinder interface {
	ByID(ctx context.Context, id uuid.UUID) (*racedomain.Race, error)
	ByDocumentID(ctx context.Context, documentID string) (*racedomain.Race, error)
}

// Notifier delivers registration notifications (the email module implements
// it). Failures are tolerated by the service: a confirmed registration is
// never rolled back because a notification could not be sent.
type Notifier interface {
	SendConfirmation(ctx context.Context, p *domain.Participant, r *domain.Registration, race *racedomain.Race) error
}

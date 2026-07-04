package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/participant/domain"
	"finish-line/internal/participant/ports"
	racedomain "finish-line/internal/race/domain"
)

type Service struct {
	participants  ports.ParticipantRepository
	registrations ports.RegistrationRepository
	races         ports.RaceFinder
	notifier      ports.Notifier
}

func New(
	participants ports.ParticipantRepository,
	registrations ports.RegistrationRepository,
	races ports.RaceFinder,
	notifier ports.Notifier,
) *Service {
	return &Service{
		participants:  participants,
		registrations: registrations,
		races:         races,
		notifier:      notifier,
	}
}

// maxConfirmAttempts bounds the optimistic dorsal-assignment retry loop.
const maxConfirmAttempts = 5

// RegisterInput carries the public registration form data.
type RegisterInput struct {
	RaceID         uuid.UUID
	FirstNames     string
	LastNames      string
	Email          string
	Phone          string
	BirthDate      time.Time
	Gender         string
	ReferralSource string
}

// Result is a completed registration: the person, their participation, and
// the race, ready for the response and the notification.
type Result struct {
	Participant  *domain.Participant
	Registration *domain.Registration
	Race         *racedomain.Race
}

// Register turns a form submission into a confirmed registration. It upserts
// the person (deduplicated by email), creates the participation, and — because
// current races are free — confirms it immediately. When payments arrive, the
// Confirm call moves behind the payment flow; everything else stays as is.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*Result, error) {
	race, err := s.races.ByID(ctx, in.RaceID)
	if err != nil {
		return nil, fmt.Errorf("finding race: %w", err)
	}

	person, err := domain.NewParticipant(domain.ParticipantParams{
		FirstNames: in.FirstNames,
		LastNames:  in.LastNames,
		Email:      in.Email,
		Phone:      in.Phone,
		BirthDate:  in.BirthDate,
		Gender:     in.Gender,
	})
	if err != nil {
		return nil, err
	}

	person, err = s.participants.UpsertByEmail(ctx, person)
	if err != nil {
		return nil, fmt.Errorf("upserting participant: %w", err)
	}

	reg, err := domain.NewRegistration(person.ID, in.RaceID, in.ReferralSource)
	if err != nil {
		return nil, err
	}

	if err := s.registrations.Create(ctx, reg); err != nil {
		return nil, fmt.Errorf("creating registration: %w", err)
	}

	return s.confirm(ctx, reg.ID, person, race)
}

// Confirm is the single confirmation point (the payment seam): free
// registrations reach it inline today; the future payment flow will call it
// when a payment succeeds.
func (s *Service) Confirm(ctx context.Context, registrationID uuid.UUID) (*Result, error) {
	reg, err := s.registrations.ByID(ctx, registrationID)
	if err != nil {
		return nil, fmt.Errorf("getting registration: %w", err)
	}

	race, err := s.races.ByID(ctx, reg.RaceID)
	if err != nil {
		return nil, fmt.Errorf("finding race: %w", err)
	}

	person, err := s.participants.ByID(ctx, reg.ParticipantID)
	if err != nil {
		return nil, fmt.Errorf("getting participant: %w", err)
	}

	return s.confirm(ctx, registrationID, person, race)
}

// confirm assigns a dorsal and confirms the registration. It reads the next
// dorsal, lets the domain confirm, and saves; if a concurrent registration
// grabbed that dorsal first, it retries with a fresh number. Correctness comes
// from the unique (race_id, dorsal) constraint — the loser simply tries again.
func (s *Service) confirm(ctx context.Context, registrationID uuid.UUID, person *domain.Participant, race *racedomain.Race) (*Result, error) {
	for range maxConfirmAttempts {
		reg, err := s.registrations.ByID(ctx, registrationID)
		if err != nil {
			return nil, fmt.Errorf("getting registration: %w", err)
		}

		next, err := s.registrations.NextDorsal(ctx, race.ID)
		if err != nil {
			return nil, fmt.Errorf("getting next dorsal: %w", err)
		}

		if err := reg.Confirm(next, race.Capacity, time.Now()); err != nil {
			return nil, err // already confirmed or full — a real error, not a retry
		}

		err = s.registrations.SaveConfirmation(ctx, reg)
		switch {
		case err == nil:
			// Best effort: the registration is confirmed regardless of whether
			// the notification could be delivered.
			if nerr := s.notifier.SendConfirmation(ctx, person, reg, race); nerr != nil {
				slog.Error("sending confirmation notification",
					"registration", reg.ID, "race", race.ID, "error", nerr)
			}
			return &Result{Participant: person, Registration: reg, Race: race}, nil
		case errors.Is(err, domain.ErrDorsalTaken):
			continue // another registration took this dorsal; retry with a fresh number
		default:
			return nil, fmt.Errorf("confirming registration: %w", err)
		}
	}

	// Sustained contention: treat as the race being full.
	return nil, domain.ErrRaceFull
}

// ByRace lists a race's registrations with their people — the admin report.
func (s *Service) ByRace(ctx context.Context, raceID uuid.UUID) ([]domain.RegistrationDetail, error) {
	details, err := s.registrations.ByRace(ctx, raceID)
	if err != nil {
		return nil, fmt.Errorf("listing registrations: %w", err)
	}
	return details, nil
}

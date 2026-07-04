package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/participant/domain"
)

func validParticipantParams() domain.ParticipantParams {
	return domain.ParticipantParams{
		FirstNames: "Amir",
		LastNames:  "Rojas",
		Email:      "amir@example.com",
		Phone:      "+591 71234567",
		BirthDate:  time.Date(2000, 6, 9, 0, 0, 0, 0, time.UTC),
		Gender:     "M",
	}
}

func TestNewParticipant(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*domain.ParticipantParams)
		wantErr error
	}{
		{name: "valid person", mutate: func(*domain.ParticipantParams) {}},
		{name: "missing first names", mutate: func(p *domain.ParticipantParams) { p.FirstNames = " " }, wantErr: domain.ErrFirstNamesRequired},
		{name: "missing last names", mutate: func(p *domain.ParticipantParams) { p.LastNames = "" }, wantErr: domain.ErrLastNamesRequired},
		{name: "invalid email", mutate: func(p *domain.ParticipantParams) { p.Email = "nope" }, wantErr: domain.ErrEmailInvalid},
		{name: "invalid phone", mutate: func(p *domain.ParticipantParams) { p.Phone = "abc" }, wantErr: domain.ErrPhoneInvalid},
		{name: "future birth date", mutate: func(p *domain.ParticipantParams) { p.BirthDate = time.Now().AddDate(0, 6, 0) }, wantErr: domain.ErrBirthDateInFuture},
		{name: "invalid gender", mutate: func(p *domain.ParticipantParams) { p.Gender = "Z" }, wantErr: domain.ErrGenderInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := validParticipantParams()
			tt.mutate(&params)

			p, err := domain.NewParticipant(params)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NewParticipant() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewParticipant() unexpected error: %v", err)
			}
			if p.Email != "amir@example.com" {
				t.Errorf("Email = %q, want normalized", p.Email)
			}
		})
	}
}

func TestNewRegistration(t *testing.T) {
	participantID, raceID := uuid.New(), uuid.New()

	t.Run("valid registration is pending without a dorsal", func(t *testing.T) {
		r, err := domain.NewRegistration(participantID, raceID, "Instagram", "INSCRIPCION")
		if err != nil {
			t.Fatalf("NewRegistration() unexpected error: %v", err)
		}
		if r.Status != domain.StatusPending {
			t.Errorf("Status = %q, want pending", r.Status)
		}
		if r.Dorsal != nil {
			t.Error("a new registration must not have a dorsal")
		}
	})

	tests := []struct {
		name          string
		participantID uuid.UUID
		raceID        uuid.UUID
		referral      string
		ticket        string
		wantErr       error
	}{
		{name: "missing participant", participantID: uuid.Nil, raceID: raceID, referral: "IG", ticket: "T", wantErr: domain.ErrParticipantRequired},
		{name: "missing race", participantID: participantID, raceID: uuid.Nil, referral: "IG", ticket: "T", wantErr: domain.ErrRaceRequired},
		{name: "missing referral", participantID: participantID, raceID: raceID, referral: " ", ticket: "T", wantErr: domain.ErrReferralRequired},
		{name: "missing ticket", participantID: participantID, raceID: raceID, referral: "IG", ticket: "", wantErr: domain.ErrTicketRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewRegistration(tt.participantID, tt.raceID, tt.referral, tt.ticket)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewRegistration() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegistrationConfirm(t *testing.T) {
	newReg := func() *domain.Registration {
		r, _ := domain.NewRegistration(uuid.New(), uuid.New(), "IG", "T")
		return r
	}

	t.Run("assigns dorsal and confirms within capacity", func(t *testing.T) {
		r := newReg()
		if err := r.Confirm(7, 100, time.Now()); err != nil {
			t.Fatalf("Confirm() unexpected error: %v", err)
		}
		if r.Status != domain.StatusConfirmed || r.Dorsal == nil || *r.Dorsal != 7 {
			t.Error("Confirm() did not set status/dorsal")
		}
	})

	t.Run("cannot confirm twice", func(t *testing.T) {
		r := newReg()
		_ = r.Confirm(7, 100, time.Now())
		if err := r.Confirm(8, 100, time.Now()); !errors.Is(err, domain.ErrAlreadyConfirmed) {
			t.Errorf("Confirm() error = %v, want ErrAlreadyConfirmed", err)
		}
	})

	t.Run("dorsal beyond capacity means the race is full", func(t *testing.T) {
		r := newReg()
		if err := r.Confirm(6, 5, time.Now()); !errors.Is(err, domain.ErrRaceFull) {
			t.Errorf("Confirm() error = %v, want ErrRaceFull", err)
		}
	})

	t.Run("rejects non-positive dorsal", func(t *testing.T) {
		r := newReg()
		if err := r.Confirm(0, 5, time.Now()); !errors.Is(err, domain.ErrDorsalInvalid) {
			t.Errorf("Confirm() error = %v, want ErrDorsalInvalid", err)
		}
	})
}

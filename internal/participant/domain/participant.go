package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Participant is a PERSON — stored once, identified by email. The same person
// can register for many races; those participations live in Registration.
type Participant struct {
	ID         uuid.UUID
	FirstNames string
	LastNames  string
	Email      string
	Phone      string
	DocumentID string
	BirthDate  time.Time
	Gender     Gender
	CreatedAt  time.Time
}

// ParticipantParams carries the person fields from the registration form.
type ParticipantParams struct {
	FirstNames string
	LastNames  string
	Email      string
	Phone      string
	DocumentID string
	BirthDate  time.Time
	Gender     string
}

// NewParticipant builds a valid person or reports why it can't. Field rules
// live with their value objects (email.go, phone.go, gender.go, birthdate.go).
func NewParticipant(p ParticipantParams) (*Participant, error) {
	firstNames := strings.TrimSpace(p.FirstNames)
	if firstNames == "" {
		return nil, ErrFirstNamesRequired
	}

	lastNames := strings.TrimSpace(p.LastNames)
	if lastNames == "" {
		return nil, ErrLastNamesRequired
	}

	email, err := NormalizeEmail(p.Email)
	if err != nil {
		return nil, err
	}

	phone, err := NormalizePhone(p.Phone)
	if err != nil {
		return nil, err
	}

	documentID, err := NormalizeDocumentID(p.DocumentID)
	if err != nil {
		return nil, err
	}

	if err := ValidateBirthDate(p.BirthDate, time.Now()); err != nil {
		return nil, err
	}

	gender, err := ParseGender(p.Gender)
	if err != nil {
		return nil, err
	}

	return &Participant{
		ID:         uuid.New(),
		FirstNames: firstNames,
		LastNames:  lastNames,
		Email:      email,
		Phone:      phone,
		DocumentID: documentID,
		BirthDate:  p.BirthDate,
		Gender:     gender,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

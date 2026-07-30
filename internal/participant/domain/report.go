package domain

import (
	"time"

	"github.com/google/uuid"
)

// RegistrationDetail is a read model joining a registration with its person,
// for the admin report — the dorsal lives on the registration, so listing a
// race's participants with their dorsals is a single join.
type RegistrationDetail struct {
	RegistrationID uuid.UUID
	FirstNames     string
	LastNames      string
	Email          string
	Phone          string
	Gender         Gender
	// Modalidad and ShirtSize are what the report exists for on the
	// merchandise side: the modalidad says whether that runner gets a shirt at
	// all, and the size is what the admin counts to order them.
	Modalidad   string
	ShirtSize   ShirtSize
	Status      Status
	Dorsal      *int
	CreatedAt   time.Time
	ConfirmedAt *time.Time
}

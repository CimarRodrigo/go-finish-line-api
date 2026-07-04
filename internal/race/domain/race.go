package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Race is a local reference to a race managed in Strapi. Strapi is the
// source of truth; we keep the minimal snapshot needed for registrations:
// identity, display data, and capacity (cupo validation + dorsal range).
type Race struct {
	ID        uuid.UUID
	StrapiID  string
	Name      string
	Date      time.Time
	Capacity  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// New builds a valid Race or reports why it can't.
func New(strapiID, name string, date time.Time, capacity int) (*Race, error) {
	strapiID = strings.TrimSpace(strapiID)
	if strapiID == "" {
		return nil, ErrStrapiIDRequired
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrNameRequired
	}

	if capacity <= 0 {
		return nil, ErrCapacityInvalid
	}

	now := time.Now().UTC()
	return &Race{
		ID:        uuid.New(),
		StrapiID:  strapiID,
		Name:      name,
		Date:      date,
		Capacity:  capacity,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

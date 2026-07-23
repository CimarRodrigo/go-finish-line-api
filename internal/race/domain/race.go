package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Race is a local reference to a race managed in Sanity (the CMS of
// record). We keep the minimal snapshot needed for registrations: identity,
// display data, and capacity (cupo validation + dorsal range), synced via
// the inbound /webhooks/sanity adapter. DocumentID holds the Sanity slug.
type Race struct {
	ID         uuid.UUID
	DocumentID string
	Name       string
	Date       time.Time
	Capacity   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// New builds a valid Race or reports why it can't.
func New(documentID, name string, date time.Time, capacity int) (*Race, error) {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return nil, ErrDocumentIDRequired
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
		ID:         uuid.New(),
		DocumentID: documentID,
		Name:       name,
		Date:       date,
		Capacity:   capacity,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

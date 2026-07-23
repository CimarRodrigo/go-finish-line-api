package ports

import (
	"context"

	"github.com/google/uuid"

	"finish-line/internal/race/domain"
)

// RaceRepository is the driven port for race persistence. Implementations
// must return domain.ErrNotFound when a race does not exist.
type RaceRepository interface {
	// Upsert inserts the race or, if its DocumentID already exists, updates
	// the snapshot fields (name, date, capacity). Idempotent by design:
	// webhooks may be retried or duplicated.
	Upsert(ctx context.Context, r *domain.Race) (*domain.Race, error)
	DeleteByDocumentID(ctx context.Context, documentID string) error
	ByID(ctx context.Context, id uuid.UUID) (*domain.Race, error)
	// ByDocumentID looks a race up by its external id (currently the Sanity
	// slug) — the id the public registration form holds.
	ByDocumentID(ctx context.Context, documentID string) (*domain.Race, error)
	List(ctx context.Context) ([]domain.Race, error)
}

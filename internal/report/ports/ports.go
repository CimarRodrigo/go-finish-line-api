package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/report/domain"
)

// ReportRepository is the driven port for the dashboard's queries.
//
// Every method takes an optional raceID: nil means "across all races", a value
// means "that race only". That is the same filter the panel offers on each
// widget, so honouring it here keeps the global and per-race views from
// drifting into two different code paths.
//
// Every method counts CONFIRMED registrations only. A pending row is one that
// never completed, so counting it would put a runner who is not racing on the
// chart and order them a shirt. All four answers apply the same rule, so the
// panels reconcile with each other.
//
// The implementation only reads and aggregates; it makes no decisions. Days
// with no registrations may be omitted from RegistrationsTimeline — the
// service fills them in, because that is a presentation rule rather than a
// storage one.
type ReportRepository interface {
	// RegistrationsTimeline counts registrations per day in [from, to],
	// inclusive.
	RegistrationsTimeline(ctx context.Context, raceID *uuid.UUID, from, to time.Time) ([]domain.TimelinePoint, error)
	// ReferralSources counts registrations grouped by how the runner heard
	// about the race, most common first.
	ReferralSources(ctx context.Context, raceID *uuid.UUID) ([]domain.ReferralCount, error)
	// ShirtSizes counts registrations grouped by shirt size. Order does not
	// matter here: the service sorts by the size chart, which the domain owns.
	ShirtSizes(ctx context.Context, raceID *uuid.UUID) ([]domain.ShirtSizeCount, error)
	// Participants returns one page of the participants report plus the total
	// number of rows the filter matches, so the caller can build the pager.
	// The race count per participant must come from this same query: resolving
	// it per row would turn one page into dozens of round trips.
	Participants(ctx context.Context, q domain.ParticipantQuery) (rows []domain.ParticipantRow, total int, err error)
}

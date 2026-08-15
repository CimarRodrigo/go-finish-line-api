package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/report/domain"
	"finish-line/internal/report/ports"
)

// Service answers the admin dashboard's questions. It owns what "today" means
// — the only non-deterministic input these reports have — so the domain can
// stay pure and the repository never has to reach for the clock.
type Service struct {
	repo ports.ReportRepository
	now  func() time.Time
}

func New(repo ports.ReportRepository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// NewWithClock builds a service with a fixed clock, so tests over date maths
// do not depend on the day they run.
func NewWithClock(repo ports.ReportRepository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

// RegistrationsTimeline returns one point per day for the last `days` days,
// ending today. Quiet days are included with a zero count.
func (s *Service) RegistrationsTimeline(ctx context.Context, days int, raceID *uuid.UUID) ([]domain.TimelinePoint, error) {
	from, to := domain.TimelineRange(days, s.now().UTC())

	found, err := s.repo.RegistrationsTimeline(ctx, raceID, from, to)
	if err != nil {
		return nil, fmt.Errorf("counting registrations per day: %w", err)
	}

	return domain.FillTimeline(found, from, to), nil
}

// ReferralSources reports where registrations came from, for one race or all.
func (s *Service) ReferralSources(ctx context.Context, raceID *uuid.UUID) ([]domain.ReferralCount, error) {
	counts, err := s.repo.ReferralSources(ctx, raceID)
	if err != nil {
		return nil, fmt.Errorf("counting referral sources: %w", err)
	}
	return counts, nil
}

// ShirtSizes reports how many shirts of each size a race needs, in the size
// chart's order — whoever is ordering the shirts reads it as a chart, not as
// an alphabetical list.
func (s *Service) ShirtSizes(ctx context.Context, raceID *uuid.UUID) ([]domain.ShirtSizeCount, error) {
	counts, err := s.repo.ShirtSizes(ctx, raceID)
	if err != nil {
		return nil, fmt.Errorf("counting shirt sizes: %w", err)
	}

	domain.SortShirtSizes(counts)
	return counts, nil
}

// Participants returns one page of the participants report. The filter is
// normalized first, so an invalid request fails before it reaches the
// database, and the age bounds are resolved against today here rather than in
// the adapter.
func (s *Service) Participants(ctx context.Context, filter domain.ParticipantFilter) (domain.ParticipantPage, error) {
	normalized, err := filter.Normalize()
	if err != nil {
		return domain.ParticipantPage{}, err
	}

	today := s.now().UTC()

	rows, total, err := s.repo.Participants(ctx, normalized.Resolve(today))
	if err != nil {
		return domain.ParticipantPage{}, fmt.Errorf("listing participants: %w", err)
	}

	// Age is derived, never stored, so it cannot go stale in the database.
	for i := range rows {
		rows[i].Age = domain.AgeAt(rows[i].BirthDate, today)
	}

	return domain.ParticipantPage{
		Items:    rows,
		Total:    total,
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
	}, nil
}

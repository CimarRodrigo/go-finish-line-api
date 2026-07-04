package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/race/domain"
	"finish-line/internal/race/ports"
)

type Service struct {
	repo ports.RaceRepository
}

func New(repo ports.RaceRepository) *Service {
	return &Service{repo: repo}
}

// Sync creates or updates the local snapshot of a race managed in Strapi.
func (s *Service) Sync(ctx context.Context, strapiID, name string, date time.Time, capacity int) (*domain.Race, error) {
	r, err := domain.New(strapiID, name, date, capacity)
	if err != nil {
		return nil, err
	}

	synced, err := s.repo.Upsert(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("upserting race: %w", err)
	}
	return synced, nil
}

// Remove drops the local reference when the race disappears from Strapi.
// Removing a race that was never synced is a no-op, not an error — webhooks
// may arrive out of order or be retried.
func (s *Service) Remove(ctx context.Context, strapiID string) error {
	if err := s.repo.DeleteByStrapiID(ctx, strapiID); err != nil {
		return fmt.Errorf("deleting race: %w", err)
	}
	return nil
}

func (s *Service) ByID(ctx context.Context, id uuid.UUID) (*domain.Race, error) {
	r, err := s.repo.ByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting race by id: %w", err)
	}
	return r, nil
}

func (s *Service) List(ctx context.Context) ([]domain.Race, error) {
	races, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing races: %w", err)
	}
	return races, nil
}

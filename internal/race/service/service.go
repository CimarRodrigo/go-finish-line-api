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

// Sync creates or updates the local snapshot of a race managed in Sanity.
func (s *Service) Sync(ctx context.Context, documentID, name string, date time.Time, capacity int) (*domain.Race, error) {
	r, err := domain.New(documentID, name, date, capacity)
	if err != nil {
		return nil, err
	}

	synced, err := s.repo.Upsert(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("upserting race: %w", err)
	}
	return synced, nil
}

// Remove drops the local reference when the race disappears from Sanity.
// Removing a race that was never synced is a no-op, not an error — webhooks
// may arrive out of order or be retried.
func (s *Service) Remove(ctx context.Context, documentID string) error {
	if err := s.repo.DeleteByDocumentID(ctx, documentID); err != nil {
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

// ByDocumentID resolves a race from its external id (the Sanity slug) — the
// identifier the public registration form speaks in.
func (s *Service) ByDocumentID(ctx context.Context, documentID string) (*domain.Race, error) {
	r, err := s.repo.ByDocumentID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("getting race by document id: %w", err)
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

package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/race/domain"
	"finish-line/internal/race/service"
)

type fakeRepo struct {
	races             map[string]*domain.Race
	deletedDocumentID string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{races: make(map[string]*domain.Race)}
}

func (r *fakeRepo) Upsert(_ context.Context, race *domain.Race) (*domain.Race, error) {
	if existing, ok := r.races[race.DocumentID]; ok {
		existing.Name = race.Name
		existing.Date = race.Date
		existing.Capacity = race.Capacity
		return existing, nil
	}
	r.races[race.DocumentID] = race
	return race, nil
}

func (r *fakeRepo) DeleteByDocumentID(_ context.Context, documentID string) error {
	r.deletedDocumentID = documentID
	delete(r.races, documentID)
	return nil
}

func (r *fakeRepo) ByID(_ context.Context, id uuid.UUID) (*domain.Race, error) {
	for _, race := range r.races {
		if race.ID == id {
			return race, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeRepo) ByDocumentID(_ context.Context, documentID string) (*domain.Race, error) {
	race, ok := r.races[documentID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return race, nil
}

func (r *fakeRepo) List(_ context.Context) ([]domain.Race, error) {
	out := make([]domain.Race, 0, len(r.races))
	for _, race := range r.races {
		out = append(out, *race)
	}
	return out, nil
}

func TestSync(t *testing.T) {
	date := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	t.Run("creates a race on first sync", func(t *testing.T) {
		repo := newFakeRepo()
		svc := service.New(repo)

		r, err := svc.Sync(context.Background(), "doc-1", "Carrera 10K", date, 500)
		if err != nil {
			t.Fatalf("Sync() unexpected error: %v", err)
		}
		if r.Capacity != 500 {
			t.Errorf("Capacity = %d, want 500", r.Capacity)
		}
	})

	t.Run("second sync updates the snapshot, keeps identity", func(t *testing.T) {
		repo := newFakeRepo()
		svc := service.New(repo)

		first, _ := svc.Sync(context.Background(), "doc-1", "Carrera 10K", date, 500)
		second, err := svc.Sync(context.Background(), "doc-1", "Carrera 10K Renombrada", date, 800)
		if err != nil {
			t.Fatalf("Sync() unexpected error: %v", err)
		}
		if second.ID != first.ID {
			t.Error("Sync() changed the race identity on update")
		}
		if second.Capacity != 800 || second.Name != "Carrera 10K Renombrada" {
			t.Error("Sync() did not update the snapshot fields")
		}
		if len(repo.races) != 1 {
			t.Errorf("races count = %d, want 1 (no duplicates)", len(repo.races))
		}
	})

	t.Run("invalid data is rejected by the domain", func(t *testing.T) {
		svc := service.New(newFakeRepo())

		_, err := svc.Sync(context.Background(), "doc-1", "Carrera", date, 0)
		if !errors.Is(err, domain.ErrCapacityInvalid) {
			t.Errorf("Sync() error = %v, want ErrCapacityInvalid", err)
		}
	})
}

func TestRemove(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo)
	date := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	_, _ = svc.Sync(context.Background(), "doc-1", "Carrera 10K", date, 500)
	if err := svc.Remove(context.Background(), "doc-1"); err != nil {
		t.Fatalf("Remove() unexpected error: %v", err)
	}
	if len(repo.races) != 0 {
		t.Error("Remove() did not delete the race")
	}

	// Removing an unknown race is a no-op, not an error.
	if err := svc.Remove(context.Background(), "ghost"); err != nil {
		t.Errorf("Remove() unexpected error for unknown race: %v", err)
	}
}

package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"finish-line/internal/race/domain"
	"finish-line/internal/race/ports"
)

type Repository struct {
	db *gorm.DB
}

var _ ports.RaceRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Upsert inserts by document_id or refreshes the snapshot fields on conflict.
// The original id and created_at are preserved on update, so participant
// references never break when Sanity re-sends a race.
func (r *Repository) Upsert(ctx context.Context, race *domain.Race) (*domain.Race, error) {
	m := toModel(race)

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "document_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"nombre", "fecha", "capacidad", "updated_at"}),
		}).
		Create(&m).Error
	if err != nil {
		return nil, fmt.Errorf("upserting race: %w", err)
	}

	// Re-read: on conflict the row keeps its original id, not the one we
	// generated for the insert attempt.
	var saved raceModel
	if err := r.db.WithContext(ctx).First(&saved, "document_id = ?", race.DocumentID).Error; err != nil {
		return nil, fmt.Errorf("reloading upserted race: %w", err)
	}
	return toDomain(saved), nil
}

func (r *Repository) DeleteByDocumentID(ctx context.Context, documentID string) error {
	if err := r.db.WithContext(ctx).Delete(&raceModel{}, "document_id = ?", documentID).Error; err != nil {
		return fmt.Errorf("deleting race: %w", err)
	}
	return nil
}

func (r *Repository) ByID(ctx context.Context, id uuid.UUID) (*domain.Race, error) {
	var m raceModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("selecting race by id: %w", err)
	}
	return toDomain(m), nil
}

func (r *Repository) ByDocumentID(ctx context.Context, documentID string) (*domain.Race, error) {
	var m raceModel
	if err := r.db.WithContext(ctx).First(&m, "document_id = ?", documentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("selecting race by document id: %w", err)
	}
	return toDomain(m), nil
}

func (r *Repository) List(ctx context.Context) ([]domain.Race, error) {
	var models []raceModel
	if err := r.db.WithContext(ctx).Order("fecha ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("selecting races: %w", err)
	}

	races := make([]domain.Race, 0, len(models))
	for _, m := range models {
		races = append(races, *toDomain(m))
	}
	return races, nil
}

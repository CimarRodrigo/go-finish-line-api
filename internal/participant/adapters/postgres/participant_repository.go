package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"finish-line/internal/participant/domain"
	"finish-line/internal/participant/ports"
)

type ParticipantRepository struct {
	db *gorm.DB
}

var _ ports.ParticipantRepository = (*ParticipantRepository)(nil)

func NewParticipantRepository(db *gorm.DB) *ParticipantRepository {
	return &ParticipantRepository{db: db}
}

// UpsertByEmail inserts the person or refreshes their profile on an email
// conflict, preserving the original id so existing registrations keep
// referencing the same participant. Email is citext, so the match is
// case-insensitive.
func (r *ParticipantRepository) UpsertByEmail(ctx context.Context, p *domain.Participant) (*domain.Participant, error) {
	m := toParticipantModel(p)

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "email"}},
			DoUpdates: clause.AssignmentColumns([]string{"nombres", "apellidos", "telefono", "fecha_nacimiento", "genero"}),
		}).
		Create(&m).Error
	if err != nil {
		return nil, fmt.Errorf("upserting participant: %w", err)
	}

	var saved participantModel
	if err := r.db.WithContext(ctx).First(&saved, "email = ?", p.Email).Error; err != nil {
		return nil, fmt.Errorf("reloading upserted participant: %w", err)
	}
	return toParticipantDomain(saved), nil
}

func (r *ParticipantRepository) ByID(ctx context.Context, id uuid.UUID) (*domain.Participant, error) {
	var m participantModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("selecting participant by id: %w", err)
	}
	return toParticipantDomain(m), nil
}

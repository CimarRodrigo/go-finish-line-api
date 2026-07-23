package postgres

import (
	"time"

	"github.com/google/uuid"

	"finish-line/internal/race/domain"
)

type raceModel struct {
	ID         uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	DocumentID string    `gorm:"column:document_id;type:text;not null;uniqueIndex"`
	Name       string    `gorm:"column:nombre;type:text;not null"`
	Date       time.Time `gorm:"column:fecha;type:date;not null"`
	Capacity   int       `gorm:"column:capacidad;type:integer;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamptz;not null"`
}

func (raceModel) TableName() string { return "races" }

func toModel(r *domain.Race) raceModel {
	return raceModel{
		ID:         r.ID,
		DocumentID: r.DocumentID,
		Name:       r.Name,
		Date:       r.Date,
		Capacity:   r.Capacity,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

func toDomain(m raceModel) *domain.Race {
	return &domain.Race{
		ID:         m.ID,
		DocumentID: m.DocumentID,
		Name:       m.Name,
		Date:       m.Date,
		Capacity:   m.Capacity,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"finish-line/internal/participant/domain"
	"finish-line/internal/participant/ports"
)

type RegistrationRepository struct {
	db *gorm.DB
}

var _ ports.RegistrationRepository = (*RegistrationRepository)(nil)

func NewRegistrationRepository(db *gorm.DB) *RegistrationRepository {
	return &RegistrationRepository{db: db}
}

func (r *RegistrationRepository) Create(ctx context.Context, reg *domain.Registration) error {
	m := toRegistrationModel(reg)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		// A pending registration carries a null dorsal, so the only unique
		// that can fire here is (race_id, participant_id).
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrAlreadyRegistered
		}
		return fmt.Errorf("inserting registration: %w", err)
	}
	return nil
}

func (r *RegistrationRepository) ByID(ctx context.Context, id uuid.UUID) (*domain.Registration, error) {
	var m registrationModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("selecting registration by id: %w", err)
	}
	return toRegistrationDomain(m), nil
}

// NextDorsal returns the next candidate dorsal for a race — the current
// highest confirmed dorsal plus one. Pure infrastructure: it reserves nothing.
func (r *RegistrationRepository) NextDorsal(ctx context.Context, raceID uuid.UUID) (int, error) {
	var maxDorsal int
	if err := r.db.WithContext(ctx).
		Model(&registrationModel{}).
		Where("race_id = ? AND dorsal IS NOT NULL", raceID).
		Select("COALESCE(MAX(dorsal), 0)").
		Scan(&maxDorsal).Error; err != nil {
		return 0, fmt.Errorf("reading max dorsal: %w", err)
	}
	return maxDorsal + 1, nil
}

// SaveConfirmation persists a confirmed registration. If a concurrent
// registration already took this dorsal, the (race_id, dorsal) unique
// constraint fires and we surface ErrDorsalTaken for the service to retry.
func (r *RegistrationRepository) SaveConfirmation(ctx context.Context, reg *domain.Registration) error {
	err := r.db.WithContext(ctx).
		Model(&registrationModel{}).
		Where("id = ?", reg.ID).
		Updates(map[string]any{
			"estado":       string(reg.Status),
			"dorsal":       reg.Dorsal,
			"confirmed_at": reg.ConfirmedAt,
		}).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrDorsalTaken
		}
		return fmt.Errorf("saving confirmation: %w", err)
	}
	return nil
}

// reportRow is the flat shape of the report join.
type reportRow struct {
	RegistrationID uuid.UUID  `gorm:"column:registration_id"`
	FirstNames     string     `gorm:"column:first_names"`
	LastNames      string     `gorm:"column:last_names"`
	Email          string     `gorm:"column:email"`
	Phone          string     `gorm:"column:phone"`
	Gender         string     `gorm:"column:gender"`
	Status         string     `gorm:"column:status"`
	Dorsal         *int       `gorm:"column:dorsal"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	ConfirmedAt    *time.Time `gorm:"column:confirmed_at"`
}

func (r *RegistrationRepository) ByRace(ctx context.Context, raceID uuid.UUID) ([]domain.RegistrationDetail, error) {
	var rows []reportRow
	err := r.db.WithContext(ctx).
		Table("registrations AS r").
		Select(`r.id AS registration_id, p.nombres AS first_names, p.apellidos AS last_names,
			p.email AS email, p.telefono AS phone, p.genero AS gender,
			r.estado AS status, r.dorsal AS dorsal, r.created_at AS created_at, r.confirmed_at AS confirmed_at`).
		Joins("JOIN participants p ON p.id = r.participant_id").
		Where("r.race_id = ?", raceID).
		Order("r.dorsal ASC NULLS LAST, r.created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("selecting race report: %w", err)
	}

	details := make([]domain.RegistrationDetail, 0, len(rows))
	for _, row := range rows {
		details = append(details, domain.RegistrationDetail{
			RegistrationID: row.RegistrationID,
			FirstNames:     row.FirstNames,
			LastNames:      row.LastNames,
			Email:          row.Email,
			Phone:          row.Phone,
			Gender:         domain.Gender(row.Gender),
			Status:         domain.Status(row.Status),
			Dorsal:         row.Dorsal,
			CreatedAt:      row.CreatedAt,
			ConfirmedAt:    row.ConfirmedAt,
		})
	}
	return details, nil
}

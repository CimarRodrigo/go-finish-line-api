// Package postgres implements the dashboard's read-side queries.
//
// Everything here aggregates and returns; no decisions are made. The service
// already resolved what "today" means and turned ages into date bounds, so
// these queries only ever compare values they were handed.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	participantdomain "finish-line/internal/participant/domain"
	"finish-line/internal/report/domain"
	"finish-line/internal/report/ports"
)

type Repository struct {
	db *gorm.DB
}

var _ ports.ReportRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// scopeConfirmed narrows a registrations query to the rows every widget
// counts: confirmed only, and one race when the caller asked for one.
//
// Pending rows are excluded on purpose. A pending registration is one that
// never completed — the race was full, or it was abandoned — so counting it
// would promise a shirt to someone who is not running and put a registration
// on the chart that never happened. Every widget applies this, so the numbers
// on one panel reconcile with the numbers on the next.
func scopeConfirmed(q *gorm.DB, raceID *uuid.UUID) *gorm.DB {
	q = q.Where("r.estado = ?", string(participantdomain.StatusConfirmed))
	if raceID != nil {
		q = q.Where("r.race_id = ?", *raceID)
	}
	return q
}

func (r *Repository) RegistrationsTimeline(ctx context.Context, raceID *uuid.UUID, from, to time.Time) ([]domain.TimelinePoint, error) {
	var rows []struct {
		Date  time.Time `gorm:"column:date"`
		Count int       `gorm:"column:count"`
	}

	// Grouped by UTC day to match the range the service computed: letting the
	// session time zone decide would put registrations on a different day than
	// the window that selected them.
	q := r.db.WithContext(ctx).
		Table("registrations AS r").
		Select(`(r.created_at AT TIME ZONE 'UTC')::date AS date, COUNT(*) AS count`).
		Where("r.created_at >= ?", from).
		Where("r.created_at < ?", to.AddDate(0, 0, 1)).
		Group(`(r.created_at AT TIME ZONE 'UTC')::date`).
		Order("date ASC")

	if err := scopeConfirmed(q, raceID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("counting registrations per day: %w", err)
	}

	points := make([]domain.TimelinePoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.TimelinePoint{Date: row.Date, Count: row.Count})
	}
	return points, nil
}

func (r *Repository) ReferralSources(ctx context.Context, raceID *uuid.UUID) ([]domain.ReferralCount, error) {
	var rows []struct {
		Source string `gorm:"column:source"`
		Count  int    `gorm:"column:count"`
	}

	q := r.db.WithContext(ctx).
		Table("registrations AS r").
		Select("r.como_te_enteraste AS source, COUNT(*) AS count").
		Group("r.como_te_enteraste").
		// Biggest source first; the name breaks ties so equal counts keep a
		// stable order between refreshes.
		Order("count DESC, source ASC")

	if err := scopeConfirmed(q, raceID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("counting referral sources: %w", err)
	}

	counts := make([]domain.ReferralCount, 0, len(rows))
	for _, row := range rows {
		counts = append(counts, domain.ReferralCount{Source: row.Source, Count: row.Count})
	}
	return counts, nil
}

func (r *Repository) ShirtSizes(ctx context.Context, raceID *uuid.UUID) ([]domain.ShirtSizeCount, error) {
	var rows []struct {
		Size  string `gorm:"column:size"`
		Count int    `gorm:"column:count"`
	}

	// Registrations with no shirt collapse into one empty-size bucket instead
	// of being dropped, so the counts still add up to the registration total.
	q := r.db.WithContext(ctx).
		Table("registrations AS r").
		Select("COALESCE(r.talla_polera, '') AS size, COUNT(*) AS count").
		Group("COALESCE(r.talla_polera, '')")

	if err := scopeConfirmed(q, raceID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("counting shirt sizes: %w", err)
	}

	counts := make([]domain.ShirtSizeCount, 0, len(rows))
	for _, row := range rows {
		counts = append(counts, domain.ShirtSizeCount{Size: row.Size, Count: row.Count})
	}
	return counts, nil
}

func (r *Repository) Participants(ctx context.Context, q domain.ParticipantQuery) ([]domain.ParticipantRow, int, error) {
	var total int64
	if err := r.filterParticipants(r.db.WithContext(ctx).Table("participants AS p"), q).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting participants: %w", err)
	}

	var rows []struct {
		ID         uuid.UUID `gorm:"column:id"`
		FirstNames string    `gorm:"column:first_names"`
		LastNames  string    `gorm:"column:last_names"`
		Email      string    `gorm:"column:email"`
		Phone      string    `gorm:"column:phone"`
		DocumentID string    `gorm:"column:document_id"`
		Gender     string    `gorm:"column:gender"`
		BirthDate  time.Time `gorm:"column:birth_date"`
		RacesCount int       `gorm:"column:races_count"`
	}

	// The race count comes from this same statement: resolving it per row
	// would turn one page into as many extra round trips as there are rows.
	//
	// It joins on confirmed registrations only — a pending row means the
	// runner never got in — and the join carries that condition rather than a
	// WHERE, so a participant with no confirmed race still appears, with zero.
	base := r.db.WithContext(ctx).
		Table("participants AS p").
		Select(`p.id AS id, p.nombres AS first_names, p.apellidos AS last_names,
			p.email AS email, p.telefono AS phone,
			COALESCE(p.documento_identidad, '') AS document_id,
			p.genero AS gender, p.fecha_nacimiento AS birth_date,
			COUNT(DISTINCT reg.race_id) AS races_count`).
		Joins("LEFT JOIN registrations reg ON reg.participant_id = p.id AND reg.estado = ?",
			string(participantdomain.StatusConfirmed)).
		Group("p.id").
		// p.id breaks ties so a row cannot drift between pages while the admin
		// is paging through people who share a name.
		Order("p.apellidos ASC, p.nombres ASC, p.id ASC").
		Limit(q.Limit).
		Offset(q.Offset)

	if err := r.filterParticipants(base, q).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("selecting participants: %w", err)
	}

	out := make([]domain.ParticipantRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ParticipantRow{
			ParticipantID: row.ID,
			FirstNames:    row.FirstNames,
			LastNames:     row.LastNames,
			Email:         row.Email,
			Phone:         row.Phone,
			DocumentID:    row.DocumentID,
			Gender:        row.Gender,
			BirthDate:     row.BirthDate,
			RacesCount:    row.RacesCount,
		})
	}
	return out, int(total), nil
}

// filterParticipants applies the report's filters to a participants query. The
// page query and the count query must agree on exactly these conditions, so
// they share one place instead of repeating them.
func (r *Repository) filterParticipants(q *gorm.DB, query domain.ParticipantQuery) *gorm.DB {
	if query.Search != "" {
		like := "%" + query.Search + "%"
		q = q.Where(
			r.db.Where("p.nombres ILIKE ?", like).
				Or("p.apellidos ILIKE ?", like).
				Or("p.email ILIKE ?", like).
				Or("p.documento_identidad ILIKE ?", like),
		)
	}

	if query.Gender != "" {
		q = q.Where("p.genero = ?", query.Gender)
	}

	// Ages already arrived as birth date bounds, both inclusive.
	if query.BirthFrom != nil {
		q = q.Where("p.fecha_nacimiento >= ?", *query.BirthFrom)
	}
	if query.BirthTo != nil {
		q = q.Where("p.fecha_nacimiento <= ?", *query.BirthTo)
	}

	// EXISTS rather than a join condition on purpose: filtering by race must
	// narrow WHICH people are listed without also narrowing the race count
	// beside their name, which should stay their full history.
	//
	// Confirmed only, like every other widget: someone whose registration
	// never completed is not in that race.
	if query.RaceID != nil {
		q = q.Where(`EXISTS (
			SELECT 1 FROM registrations f
			WHERE f.participant_id = p.id AND f.race_id = ? AND f.estado = ?
		)`, *query.RaceID, string(participantdomain.StatusConfirmed))
	}

	return q
}

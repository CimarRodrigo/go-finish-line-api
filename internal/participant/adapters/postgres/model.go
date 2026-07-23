package postgres

import (
	"time"

	"github.com/google/uuid"

	"finish-line/internal/participant/domain"
)

// participantModel is the person, deduplicated by a globally unique email.
//
// DocumentID (CI) is required by the domain for every NEW registration, but
// the column itself stays nullable/additive: historical rows predate this
// field and must not need a backfill (design decision). It is also
// intentionally NOT unique — v1 stores it, it does not dedupe by it (email
// already covers per-race duplicate registration).
type participantModel struct {
	ID         uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	FirstNames string    `gorm:"column:nombres;type:text;not null"`
	LastNames  string    `gorm:"column:apellidos;type:text;not null"`
	Email      string    `gorm:"column:email;type:citext;not null;uniqueIndex"`
	Phone      string    `gorm:"column:telefono;type:text;not null"`
	DocumentID string    `gorm:"column:documento_identidad;type:text"`
	BirthDate  time.Time `gorm:"column:fecha_nacimiento;type:date;not null"`
	Gender     string    `gorm:"column:genero;type:text;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamptz;not null"`
}

func (participantModel) TableName() string { return "participants" }

// registrationModel is the N:M participation. Two composite unique indexes:
//   - (race_id, participant_id) → one registration per person per race
//   - (race_id, dorsal)         → no duplicate dorsals in a race (NULLs
//     distinct, so many pending rows are allowed)
type registrationModel struct {
	ID             uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	ParticipantID  uuid.UUID `gorm:"column:participant_id;type:uuid;not null;uniqueIndex:uq_registration_race_participant"`
	RaceID         uuid.UUID `gorm:"column:race_id;type:uuid;not null;uniqueIndex:uq_registration_race_participant;uniqueIndex:uq_registration_race_dorsal"`
	ReferralSource string    `gorm:"column:como_te_enteraste;type:text;not null"`
	// Modalidad is additive and nullable: the distance/variant chosen on the
	// detail page, display data only, no invariant depends on it.
	Modalidad   string     `gorm:"column:modalidad;type:text"`
	Status      string     `gorm:"column:estado;type:text;not null"`
	Dorsal      *int       `gorm:"column:dorsal;type:integer;uniqueIndex:uq_registration_race_dorsal"`
	CreatedAt   time.Time  `gorm:"column:created_at;type:timestamptz;not null"`
	ConfirmedAt *time.Time `gorm:"column:confirmed_at;type:timestamptz"`
}

func (registrationModel) TableName() string { return "registrations" }

func toParticipantModel(p *domain.Participant) participantModel {
	return participantModel{
		ID:         p.ID,
		FirstNames: p.FirstNames,
		LastNames:  p.LastNames,
		Email:      p.Email,
		Phone:      p.Phone,
		DocumentID: p.DocumentID,
		BirthDate:  p.BirthDate,
		Gender:     string(p.Gender),
		CreatedAt:  p.CreatedAt,
	}
}

func toParticipantDomain(m participantModel) *domain.Participant {
	return &domain.Participant{
		ID:         m.ID,
		FirstNames: m.FirstNames,
		LastNames:  m.LastNames,
		Email:      m.Email,
		Phone:      m.Phone,
		DocumentID: m.DocumentID,
		BirthDate:  m.BirthDate,
		Gender:     domain.Gender(m.Gender),
		CreatedAt:  m.CreatedAt,
	}
}

func toRegistrationModel(r *domain.Registration) registrationModel {
	return registrationModel{
		ID:             r.ID,
		ParticipantID:  r.ParticipantID,
		RaceID:         r.RaceID,
		ReferralSource: r.ReferralSource,
		Modalidad:      r.Modalidad,
		Status:         string(r.Status),
		Dorsal:         r.Dorsal,
		CreatedAt:      r.CreatedAt,
		ConfirmedAt:    r.ConfirmedAt,
	}
}

func toRegistrationDomain(m registrationModel) *domain.Registration {
	return &domain.Registration{
		ID:             m.ID,
		ParticipantID:  m.ParticipantID,
		RaceID:         m.RaceID,
		ReferralSource: m.ReferralSource,
		Modalidad:      m.Modalidad,
		Status:         domain.Status(m.Status),
		Dorsal:         m.Dorsal,
		CreatedAt:      m.CreatedAt,
		ConfirmedAt:    m.ConfirmedAt,
	}
}

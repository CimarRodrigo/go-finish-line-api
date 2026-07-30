package rest

import (
	"time"

	"github.com/google/uuid"

	"finish-line/internal/participant/domain"
	"finish-line/internal/participant/service"
)

const dateLayout = "2006-01-02"

type registerRequest struct {
	RaceDocumentID string `json:"race_document_id" binding:"required"`
	FirstNames     string `json:"first_names" binding:"required"`
	LastNames      string `json:"last_names" binding:"required"`
	Email          string `json:"email" binding:"required"`
	Phone          string `json:"phone" binding:"required"`
	DocumentID     string `json:"document_id" binding:"required"`
	BirthDate      string `json:"birth_date" binding:"required"`
	Gender         string `json:"gender" binding:"required"`
	ReferralSource string `json:"referral_source" binding:"required"`
	// Modalidad is the distance/variant picked on the detail page (e.g. "10K
	// · Con polera"). Display data carried through the form, not re-asked —
	// optional at the transport boundary since it is stored as a nullable
	// column (see design decision).
	Modalidad string `json:"modalidad"`
	// ShirtSize is optional at the transport boundary — a modalidad without a
	// shirt has none. The domain rejects any non-empty value outside the size
	// chart, so the report can count sizes.
	ShirtSize string `json:"shirt_size"`
}

// registrationResponse is what a successful registration returns: the person
// plus their participation state (status + dorsal) for the frontend to show.
type registrationResponse struct {
	RegistrationID uuid.UUID  `json:"registration_id"`
	RaceDocumentID string     `json:"race_document_id"`
	FirstNames     string     `json:"first_names"`
	LastNames      string     `json:"last_names"`
	Email          string     `json:"email"`
	Phone          string     `json:"phone"`
	DocumentID     string     `json:"document_id"`
	Gender         string     `json:"gender"`
	Modalidad      string     `json:"modalidad"`
	ShirtSize      string     `json:"shirt_size"`
	Status         string     `json:"status"`
	Dorsal         *int       `json:"dorsal"`
	CreatedAt      time.Time  `json:"created_at"`
	ConfirmedAt    *time.Time `json:"confirmed_at"`
}

func toRegistrationResponse(res *service.Result) registrationResponse {
	return registrationResponse{
		RegistrationID: res.Registration.ID,
		RaceDocumentID: res.Race.DocumentID,
		FirstNames:     res.Participant.FirstNames,
		LastNames:      res.Participant.LastNames,
		Email:          res.Participant.Email,
		Phone:          res.Participant.Phone,
		DocumentID:     res.Participant.DocumentID,
		Gender:         string(res.Participant.Gender),
		Modalidad:      res.Registration.Modalidad,
		ShirtSize:      string(res.Registration.ShirtSize),
		Status:         string(res.Registration.Status),
		Dorsal:         res.Registration.Dorsal,
		CreatedAt:      res.Registration.CreatedAt,
		ConfirmedAt:    res.Registration.ConfirmedAt,
	}
}

// reportRowResponse is one line of the admin report.
type reportRowResponse struct {
	RegistrationID uuid.UUID  `json:"registration_id"`
	FirstNames     string     `json:"first_names"`
	LastNames      string     `json:"last_names"`
	Email          string     `json:"email"`
	Phone          string     `json:"phone"`
	Gender         string     `json:"gender"`
	Modalidad      string     `json:"modalidad"`
	ShirtSize      string     `json:"shirt_size"`
	Status         string     `json:"status"`
	Dorsal         *int       `json:"dorsal"`
	CreatedAt      time.Time  `json:"created_at"`
	ConfirmedAt    *time.Time `json:"confirmed_at"`
}

func toReportRow(d domain.RegistrationDetail) reportRowResponse {
	return reportRowResponse{
		RegistrationID: d.RegistrationID,
		FirstNames:     d.FirstNames,
		LastNames:      d.LastNames,
		Email:          d.Email,
		Phone:          d.Phone,
		Gender:         string(d.Gender),
		Modalidad:      d.Modalidad,
		ShirtSize:      string(d.ShirtSize),
		Status:         string(d.Status),
		Dorsal:         d.Dorsal,
		CreatedAt:      d.CreatedAt,
		ConfirmedAt:    d.ConfirmedAt,
	}
}

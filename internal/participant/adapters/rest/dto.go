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
	BirthDate      string `json:"birth_date" binding:"required"`
	Gender         string `json:"gender" binding:"required"`
	ReferralSource string `json:"referral_source" binding:"required"`
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
	Gender         string     `json:"gender"`
	Status         string     `json:"status"`
	Dorsal         *int       `json:"dorsal"`
	CreatedAt      time.Time  `json:"created_at"`
	ConfirmedAt    *time.Time `json:"confirmed_at"`
}

func toRegistrationResponse(res *service.Result) registrationResponse {
	return registrationResponse{
		RegistrationID: res.Registration.ID,
		RaceDocumentID: res.Race.StrapiID,
		FirstNames:     res.Participant.FirstNames,
		LastNames:      res.Participant.LastNames,
		Email:          res.Participant.Email,
		Phone:          res.Participant.Phone,
		Gender:         string(res.Participant.Gender),
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
		Status:         string(d.Status),
		Dorsal:         d.Dorsal,
		CreatedAt:      d.CreatedAt,
		ConfirmedAt:    d.ConfirmedAt,
	}
}

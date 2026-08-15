package rest

import (
	"github.com/google/uuid"

	"finish-line/internal/report/domain"
)

const dateLayout = "2006-01-02"

// timelinePointResponse is one day of the registrations chart. The date is a
// plain day, not a timestamp: the chart plots days, and sending an instant
// would invite the browser to shift it into another one by time zone.
type timelinePointResponse struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

func toTimeline(points []domain.TimelinePoint) []timelinePointResponse {
	out := make([]timelinePointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, timelinePointResponse{Date: p.Date.Format(dateLayout), Count: p.Count})
	}
	return out
}

type referralCountResponse struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

func toReferralCounts(counts []domain.ReferralCount) []referralCountResponse {
	out := make([]referralCountResponse, 0, len(counts))
	for _, c := range counts {
		out = append(out, referralCountResponse{Source: c.Source, Count: c.Count})
	}
	return out
}

// shirtSizeCountResponse carries an empty size for the registrations whose
// modalidad includes no shirt, so the counts still add up.
type shirtSizeCountResponse struct {
	Size  string `json:"size"`
	Count int    `json:"count"`
}

func toShirtSizeCounts(counts []domain.ShirtSizeCount) []shirtSizeCountResponse {
	out := make([]shirtSizeCountResponse, 0, len(counts))
	for _, c := range counts {
		out = append(out, shirtSizeCountResponse{Size: c.Size, Count: c.Count})
	}
	return out
}

// participantRowResponse is one line of the participants report.
type participantRowResponse struct {
	ParticipantID uuid.UUID `json:"participant_id"`
	FirstNames    string    `json:"first_names"`
	LastNames     string    `json:"last_names"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	DocumentID    string    `json:"document_id"`
	Gender        string    `json:"gender"`
	BirthDate     string    `json:"birth_date"`
	Age           int       `json:"age"`
	RacesCount    int       `json:"races_count"`
}

// participantPageResponse wraps the rows with what the pager needs.
type participantPageResponse struct {
	Items    []participantRowResponse `json:"items"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

func toParticipantPage(page domain.ParticipantPage) participantPageResponse {
	items := make([]participantRowResponse, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, participantRowResponse{
			ParticipantID: row.ParticipantID,
			FirstNames:    row.FirstNames,
			LastNames:     row.LastNames,
			Email:         row.Email,
			Phone:         row.Phone,
			DocumentID:    row.DocumentID,
			Gender:        row.Gender,
			BirthDate:     row.BirthDate.Format(dateLayout),
			Age:           row.Age,
			RacesCount:    row.RacesCount,
		})
	}

	return participantPageResponse{
		Items:    items,
		Total:    page.Total,
		Page:     page.Page,
		PageSize: page.PageSize,
	}
}

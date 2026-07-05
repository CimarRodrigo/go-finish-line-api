package rest

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/race/domain"
)

// strapiWebhookPayload is the envelope Strapi sends on entry events. Only
// the fields we care about are mapped; the entry carries the race content
// type's fields.
type strapiWebhookPayload struct {
	Event string      `json:"event" binding:"required"`
	Model string      `json:"model"`
	Entry strapiEntry `json:"entry" binding:"required"`
}

type strapiEntry struct {
	ID         int    `json:"id"`
	DocumentID string `json:"documentId"`
	Name       string `json:"nombre"`
	Date       string `json:"fecha"`
	Capacity   int    `json:"capacidad"`
}

// strapiID prefers the stable documentId (Strapi v5) and falls back to the
// numeric id (v4) so both versions are supported.
func (e strapiEntry) strapiID() string {
	if e.DocumentID != "" {
		return e.DocumentID
	}
	if e.ID != 0 {
		return strconv.Itoa(e.ID)
	}
	return ""
}

// raceResponse is one race in the public list: both ids plus the snapshot
// fields the frontend needs. race_id is our internal key (register + report);
// document_id is the Strapi documentId (fetch display content from Strapi).
type raceResponse struct {
	RaceID     uuid.UUID `json:"race_id"`
	DocumentID string    `json:"document_id"`
	Name       string    `json:"name"`
	Date       string    `json:"date"`
	Capacity   int       `json:"capacity"`
}

func toRaceResponse(r domain.Race) raceResponse {
	return raceResponse{
		RaceID:     r.ID,
		DocumentID: r.StrapiID,
		Name:       r.Name,
		Date:       r.Date.Format("2006-01-02"),
		Capacity:   r.Capacity,
	}
}

// parseDate accepts Strapi date ("2006-01-02") and datetime (RFC3339) fields.
func (e strapiEntry) parseDate() (time.Time, error) {
	if t, err := time.Parse("2006-01-02", e.Date); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, e.Date); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized date format %q", e.Date)
}

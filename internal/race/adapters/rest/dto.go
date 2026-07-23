package rest

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/race/domain"
)

// sanityWebhookPayload is the payload expected from a Sanity GROQ-powered
// webhook (Studio: API → Webhooks). Unlike Strapi's fixed entry envelope,
// a Sanity webhook's shape is entirely defined by the GROQ projection
// configured in the Studio webhook UI — this struct assumes a projection
// that flattens the race document's own fields at the top level and adds
// `operation` via Sanity's `delta::operation()` GROQ function, e.g.:
//
//	{
//	  "operation": delta::operation(), // "create" | "update" | "delete"
//	  "slug": slug.current,
//	  "title": title,
//	  "date": date,
//	  "capacity": capacity
//	}
//
// NOTE: this shape is NOT verified against a real Sanity webhook config yet
// (see design doc, open question). Verify field names/nesting against the
// actual Studio webhook setup before this goes live; adjust as needed.
type sanityWebhookPayload struct {
	Operation string `json:"operation" binding:"required"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Date      string `json:"date"`
	Capacity  int    `json:"capacity"`
}

// externalID is the id we sync races by — the Sanity slug. Named generically
// (not sanityID) because it flows straight into RaceService.Sync's
// CMS-agnostic documentID parameter.
func (p sanityWebhookPayload) externalID() string {
	return p.Slug
}

// raceResponse is one race in the public list: both ids plus the snapshot
// fields the frontend needs. race_id is our internal key (register + report);
// document_id is the external CMS id — the Sanity slug — used to fetch
// display content from Sanity.
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
		DocumentID: r.DocumentID,
		Name:       r.Name,
		Date:       r.Date.Format("2006-01-02"),
		Capacity:   r.Capacity,
	}
}

// parseDate accepts Sanity's `date` type ("2006-01-02") and falls back to
// RFC3339 in case a projection ever forwards a full datetime field instead.
func (p sanityWebhookPayload) parseDate() (time.Time, error) {
	if t, err := time.Parse("2006-01-02", p.Date); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, p.Date); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized date format %q", p.Date)
}

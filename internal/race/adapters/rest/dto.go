package rest

import (
	"fmt"
	"strconv"
	"time"
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

package rest

import (
	"context"
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"finish-line/internal/common/httpx"
	"finish-line/internal/race/domain"
)

// RaceService is the consumer-side contract this adapter needs from the race
// application service.
type RaceService interface {
	Sync(ctx context.Context, strapiID, name string, date time.Time, capacity int) (*domain.Race, error)
	Remove(ctx context.Context, strapiID string) error
	List(ctx context.Context) ([]domain.Race, error)
}

type Handler struct {
	svc    RaceService
	secret string
	authMW gin.HandlerFunc
}

func NewHandler(svc RaceService, webhookSecret string, authMW gin.HandlerFunc) *Handler {
	return &Handler{svc: svc, secret: webhookSecret, authMW: authMW}
}

func (h *Handler) RegisterRoutes(r gin.IRouter) {
	// Machine-to-machine endpoint: authenticated by a shared secret header,
	// not by an admin JWT — Strapi is the caller, not a person.
	r.POST("/webhooks/strapi", h.requireSecret, h.handleWebhook)

	// Admin-only: the panel lists races to get the race_id it needs to pull the
	// registrations report. Registration itself uses the Strapi documentId, so
	// the public form never touches this endpoint. Each row carries our internal
	// race_id plus the Strapi documentId.
	r.GET("/races", h.authMW, h.handleList)
}

// requireSecret validates the shared secret in constant time so the check
// leaks no timing information.
func (h *Handler) requireSecret(c *gin.Context) {
	got := c.GetHeader("X-Webhook-Secret")
	if subtle.ConstantTimeCompare([]byte(got), []byte(h.secret)) != 1 {
		httpx.Unauthorized(c, "invalid webhook secret")
		c.Abort()
		return
	}
	c.Next()
}

func (h *Handler) handleWebhook(c *gin.Context) {
	var payload strapiWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		httpx.BadRequest(c, "invalid webhook payload")
		return
	}

	strapiID := payload.Entry.strapiID()
	if strapiID == "" {
		httpx.BadRequest(c, "entry has no id")
		return
	}

	switch payload.Event {
	case "entry.create", "entry.update", "entry.publish":
		date, err := payload.Entry.parseDate()
		if err != nil {
			httpx.BadRequest(c, "invalid or missing fecha")
			return
		}

		race, err := h.svc.Sync(c.Request.Context(), strapiID, payload.Entry.Name, date, payload.Entry.Capacity)
		if err != nil {
			httpx.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "synced", "id": race.ID})

	case "entry.delete", "entry.unpublish":
		if err := h.svc.Remove(c.Request.Context(), strapiID); err != nil {
			httpx.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "removed"})

	default:
		// Unknown events are acknowledged, not failed: Strapi should not
		// retry events we deliberately ignore.
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
	}
}

func (h *Handler) handleList(c *gin.Context) {
	races, err := h.svc.List(c.Request.Context())
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	out := make([]raceResponse, 0, len(races))
	for _, r := range races {
		out = append(out, toRaceResponse(r))
	}
	c.JSON(http.StatusOK, out)
}

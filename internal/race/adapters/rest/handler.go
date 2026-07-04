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
}

type Handler struct {
	svc    RaceService
	secret string
}

func NewHandler(svc RaceService, webhookSecret string) *Handler {
	return &Handler{svc: svc, secret: webhookSecret}
}

func (h *Handler) RegisterRoutes(r gin.IRouter) {
	// Machine-to-machine endpoint: authenticated by a shared secret header,
	// not by an admin JWT — Strapi is the caller, not a person.
	r.POST("/webhooks/strapi", h.requireSecret, h.handleWebhook)
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

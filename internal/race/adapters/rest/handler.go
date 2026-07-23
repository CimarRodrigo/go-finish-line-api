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
// application service. The externalID parameter is the Sanity slug.
type RaceService interface {
	Sync(ctx context.Context, externalID, name string, date time.Time, capacity int) (*domain.Race, error)
	Remove(ctx context.Context, externalID string) error
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
	// not by an admin JWT — Sanity is the caller, not a person. This replaces
	// the former /webhooks/strapi route 1:1; Strapi is fully decommissioned.
	r.POST("/webhooks/sanity", h.requireSecret, h.handleWebhook)

	// Admin-only: the panel lists races to get the race_id it needs to pull the
	// registrations report. Registration itself uses the Sanity slug, so the
	// public form never touches this endpoint. Each row carries our internal
	// race_id plus the Sanity slug (as document_id).
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
	var payload sanityWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		httpx.BadRequest(c, "invalid webhook payload")
		return
	}

	externalID := payload.externalID()
	if externalID == "" {
		httpx.BadRequest(c, "payload has no slug")
		return
	}

	switch payload.Operation {
	case "create", "update":
		date, err := payload.parseDate()
		if err != nil {
			httpx.BadRequest(c, "invalid or missing date")
			return
		}

		race, err := h.svc.Sync(c.Request.Context(), externalID, payload.Title, date, payload.Capacity)
		if err != nil {
			httpx.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "synced", "id": race.ID})

	case "delete":
		if err := h.svc.Remove(c.Request.Context(), externalID); err != nil {
			httpx.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "removed"})

	default:
		// Unknown/unmapped operations are acknowledged, not failed: Sanity
		// should not retry events we deliberately ignore.
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

package rest

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"finish-line/internal/common/httpx"
	"finish-line/internal/report/domain"
)

// ReportService is the consumer-side contract this adapter needs from the
// report application service.
type ReportService interface {
	RegistrationsTimeline(ctx context.Context, days int, raceID *uuid.UUID) ([]domain.TimelinePoint, error)
	ReferralSources(ctx context.Context, raceID *uuid.UUID) ([]domain.ReferralCount, error)
	ShirtSizes(ctx context.Context, raceID *uuid.UUID) ([]domain.ShirtSizeCount, error)
	Participants(ctx context.Context, filter domain.ParticipantFilter) (domain.ParticipantPage, error)
}

type Handler struct {
	svc    ReportService
	authMW gin.HandlerFunc
}

// NewHandler takes the auth middleware: every route here is admin-only.
func NewHandler(svc ReportService, authMW gin.HandlerFunc) *Handler {
	return &Handler{svc: svc, authMW: authMW}
}

// RegisterRoutes mounts the dashboard's widgets as separate endpoints, one per
// question. Each one filters and refreshes on its own, so changing the race on
// one chart does not recompute the others; a caller that wants them in a
// single response composes them upstream.
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	reports := r.Group("/reports")
	reports.Use(h.authMW)

	reports.GET("/registrations-timeline", h.registrationsTimeline)
	reports.GET("/referral-sources", h.referralSources)
	reports.GET("/shirt-sizes", h.shirtSizes)

	// The participants report is a listing of people rather than a metric, so
	// it sits on its own resource path.
	r.GET("/participants", h.authMW, h.participants)
}

func (h *Handler) registrationsTimeline(c *gin.Context) {
	raceID, err := optionalRaceID(c)
	if err != nil {
		httpx.BadRequest(c, "race_id must be a uuid")
		return
	}

	// An absent or unparseable window falls back to the default rather than
	// failing: the domain clamps it, and a dashboard should still render.
	days, _ := strconv.Atoi(c.Query("days"))

	points, err := h.svc.RegistrationsTimeline(c.Request.Context(), days, raceID)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTimeline(points))
}

func (h *Handler) referralSources(c *gin.Context) {
	raceID, err := optionalRaceID(c)
	if err != nil {
		httpx.BadRequest(c, "race_id must be a uuid")
		return
	}

	counts, err := h.svc.ReferralSources(c.Request.Context(), raceID)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toReferralCounts(counts))
}

func (h *Handler) shirtSizes(c *gin.Context) {
	raceID, err := optionalRaceID(c)
	if err != nil {
		httpx.BadRequest(c, "race_id must be a uuid")
		return
	}

	counts, err := h.svc.ShirtSizes(c.Request.Context(), raceID)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toShirtSizeCounts(counts))
}

func (h *Handler) participants(c *gin.Context) {
	raceID, err := optionalRaceID(c)
	if err != nil {
		httpx.BadRequest(c, "race_id must be a uuid")
		return
	}

	ageMin, err := optionalInt(c, "age_min")
	if err != nil {
		httpx.BadRequest(c, "age_min must be a whole number")
		return
	}
	ageMax, err := optionalInt(c, "age_max")
	if err != nil {
		httpx.BadRequest(c, "age_max must be a whole number")
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	// The service normalizes and validates: an impossible age range or an
	// unknown gender comes back as a 400 from the domain.
	result, err := h.svc.Participants(c.Request.Context(), domain.ParticipantFilter{
		Search:   c.Query("search"),
		RaceID:   raceID,
		Gender:   c.Query("gender"),
		AgeMin:   ageMin,
		AgeMax:   ageMax,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toParticipantPage(result))
}

// optionalRaceID reads the shared race filter: absent means "across all
// races", present means it must be a uuid.
func optionalRaceID(c *gin.Context) (*uuid.UUID, error) {
	raw := c.Query("race_id")
	if raw == "" {
		return nil, nil
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// optionalInt distinguishes "not filtered" from "filtered by zero", which a
// plain Atoi would flatten into the same value.
func optionalInt(c *gin.Context, key string) (*int, error) {
	raw := c.Query(key)
	if raw == "" {
		return nil, nil
	}

	v, err := strconv.Atoi(raw)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

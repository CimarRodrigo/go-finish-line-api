package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"finish-line/internal/common/httpx"
	"finish-line/internal/participant/domain"
	"finish-line/internal/participant/service"
)

// RegistrationService is the consumer-side contract this adapter needs from
// the participant application service.
type RegistrationService interface {
	Register(ctx context.Context, in service.RegisterInput) (*service.Result, error)
	ByRace(ctx context.Context, raceID uuid.UUID) ([]domain.RegistrationDetail, error)
}

type Handler struct {
	svc    RegistrationService
	authMW gin.HandlerFunc
}

// NewHandler takes the auth middleware so the admin report route can be
// protected while registration stays public.
func NewHandler(svc RegistrationService, authMW gin.HandlerFunc) *Handler {
	return &Handler{svc: svc, authMW: authMW}
}

func (h *Handler) RegisterRoutes(r gin.IRouter) {
	reg := r.Group("/registrations")
	reg.POST("", h.register)            // public: the registration form
	reg.GET("", h.authMW, h.listByRace) // admin report, requires a token
}

func (h *Handler) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request body")
		return
	}

	birthDate, err := time.Parse(dateLayout, req.BirthDate)
	if err != nil {
		httpx.BadRequest(c, "birth_date must be YYYY-MM-DD")
		return
	}

	res, err := h.svc.Register(c.Request.Context(), service.RegisterInput{
		RaceDocumentID: req.RaceDocumentID,
		FirstNames:     req.FirstNames,
		LastNames:      req.LastNames,
		Email:          req.Email,
		Phone:          req.Phone,
		BirthDate:      birthDate,
		Gender:         req.Gender,
		ReferralSource: req.ReferralSource,
	})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toRegistrationResponse(res))
}

// listByRace is the admin report: every registration for a race, with dorsals.
func (h *Handler) listByRace(c *gin.Context) {
	raceID, err := uuid.Parse(c.Query("race_id"))
	if err != nil {
		httpx.BadRequest(c, "race_id query parameter is required and must be a uuid")
		return
	}

	details, err := h.svc.ByRace(c.Request.Context(), raceID)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	out := make([]reportRowResponse, 0, len(details))
	for _, d := range details {
		out = append(out, toReportRow(d))
	}
	c.JSON(http.StatusOK, out)
}

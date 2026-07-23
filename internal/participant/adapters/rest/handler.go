package rest

import (
	"context"
	"crypto/subtle"
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
	secret string
	authMW gin.HandlerFunc
}

// NewHandler takes the shared service secret (checked on the public
// registration route, see requireSecret) and the auth middleware (used to
// protect the admin report route).
func NewHandler(svc RegistrationService, serviceSecret string, authMW gin.HandlerFunc) *Handler {
	return &Handler{svc: svc, secret: serviceSecret, authMW: authMW}
}

func (h *Handler) RegisterRoutes(r gin.IRouter) {
	reg := r.Group("/registrations")
	// Machine-to-machine endpoint: the only caller is our own Astro BFF, not
	// a browser, so it is authenticated by a shared secret header rather
	// than an admin JWT (mirrors the race module's Sanity webhook guard,
	// see internal/race/adapters/rest/handler.go's requireSecret).
	reg.POST("", h.requireSecret, h.register)
	reg.GET("", h.authMW, h.listByRace) // admin report, requires a token
}

// requireSecret validates the shared BFF↔Go service secret in constant time
// so the check leaks no timing information. Scoped to POST /registrations
// only — the admin GET report keeps using the JWT auth middleware.
func (h *Handler) requireSecret(c *gin.Context) {
	got := c.GetHeader("X-Service-Secret")
	if subtle.ConstantTimeCompare([]byte(got), []byte(h.secret)) != 1 {
		httpx.Unauthorized(c, "invalid service secret")
		c.Abort()
		return
	}
	c.Next()
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
		DocumentID:     req.DocumentID,
		BirthDate:      birthDate,
		Gender:         req.Gender,
		ReferralSource: req.ReferralSource,
		Modalidad:      req.Modalidad,
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

package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"finish-line/internal/participant/adapters/rest"
	"finish-line/internal/participant/domain"
	"finish-line/internal/participant/service"
	racedomain "finish-line/internal/race/domain"
)

type fakeService struct {
	registerErr error
	details     []domain.RegistrationDetail
}

func (s *fakeService) Register(_ context.Context, in service.RegisterInput) (*service.Result, error) {
	if s.registerErr != nil {
		return nil, s.registerErr
	}
	dorsal := 1
	return &service.Result{
		Participant:  &domain.Participant{ID: uuid.New(), FirstNames: in.FirstNames, Email: in.Email},
		Registration: &domain.Registration{ID: uuid.New(), RaceID: in.RaceID, Status: domain.StatusConfirmed, Dorsal: &dorsal},
		Race:         &racedomain.Race{ID: in.RaceID},
	}, nil
}

func (s *fakeService) ByRace(_ context.Context, _ uuid.UUID) ([]domain.RegistrationDetail, error) {
	return s.details, nil
}

func noopMW(c *gin.Context) { c.Next() }

func setupRouter(svc *fakeService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rest.NewHandler(svc, noopMW).RegisterRoutes(r)
	return r
}

func validBody(raceID string) string {
	return `{"race_id":"` + raceID + `","first_names":"Amir","last_names":"Rojas","email":"amir@example.com","phone":"+59171234567","birth_date":"2000-06-09","gender":"M","referral_source":"Instagram"}`
}

func TestRegister(t *testing.T) {
	raceID := uuid.NewString()

	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{name: "valid registration", body: validBody(raceID), wantStatus: http.StatusCreated},
		{name: "malformed json", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "missing fields", body: `{"race_id":"` + raceID + `"}`, wantStatus: http.StatusBadRequest},
		{name: "invalid race id", body: validBody("not-a-uuid"), wantStatus: http.StatusBadRequest},
		{name: "bad birth date", body: strings.Replace(validBody(raceID), "2000-06-09", "09-06-2000", 1), wantStatus: http.StatusBadRequest},
		{name: "duplicate", body: validBody(raceID), serviceErr: domain.ErrAlreadyRegistered, wantStatus: http.StatusConflict},
		{name: "race full", body: validBody(raceID), serviceErr: domain.ErrRaceFull, wantStatus: http.StatusConflict},
		{name: "unknown race", body: validBody(raceID), serviceErr: racedomain.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "domain validation", body: validBody(raceID), serviceErr: domain.ErrBirthDateInFuture, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter(&fakeService{registerErr: tt.serviceErr})

			req := httptest.NewRequest(http.MethodPost, "/registrations", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
		})
	}
}

func TestListByRace(t *testing.T) {
	t.Run("returns the report rows", func(t *testing.T) {
		dorsal := 5
		svc := &fakeService{details: []domain.RegistrationDetail{
			{RegistrationID: uuid.New(), FirstNames: "Amir", Email: "amir@example.com", Status: domain.StatusConfirmed, Dorsal: &dorsal},
		}}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodGet, "/registrations?race_id="+uuid.NewString(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var out []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("not a JSON array: %v", err)
		}
		if len(out) != 1 || out[0]["dorsal"].(float64) != 5 {
			t.Errorf("unexpected report body: %s", rec.Body)
		}
	})

	t.Run("missing race_id is a 400", func(t *testing.T) {
		router := setupRouter(&fakeService{})
		req := httptest.NewRequest(http.MethodGet, "/registrations", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}

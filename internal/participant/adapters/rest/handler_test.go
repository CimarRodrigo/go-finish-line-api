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
	registerErr    error
	registerCalled bool
	details        []domain.RegistrationDetail
}

func (s *fakeService) Register(_ context.Context, in service.RegisterInput) (*service.Result, error) {
	s.registerCalled = true
	if s.registerErr != nil {
		return nil, s.registerErr
	}
	dorsal := 1
	return &service.Result{
		Participant:  &domain.Participant{ID: uuid.New(), FirstNames: in.FirstNames, Email: in.Email},
		Registration: &domain.Registration{ID: uuid.New(), RaceID: uuid.New(), Status: domain.StatusConfirmed, Dorsal: &dorsal},
		Race:         &racedomain.Race{ID: uuid.New(), DocumentID: in.RaceDocumentID},
	}, nil
}

func (s *fakeService) ByRace(_ context.Context, _ uuid.UUID) ([]domain.RegistrationDetail, error) {
	return s.details, nil
}

// testSecret is the shared BFF↔Go service secret used by these tests — same
// naming convention as the race module's testSecret (its equivalent for the
// Sanity webhook secret).
const testSecret = "test-service-secret"

func noopMW(c *gin.Context) { c.Next() }

func setupRouter(svc *fakeService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rest.NewHandler(svc, testSecret, noopMW).RegisterRoutes(r)
	return r
}

func validBody(documentID string) string {
	return `{"race_document_id":"` + documentID + `","first_names":"Amir","last_names":"Rojas","email":"amir@example.com","phone":"+59171234567","document_id":"1234567","birth_date":"2000-06-09","gender":"M","referral_source":"Instagram","modalidad":"10K · Con polera"}`
}

func TestRegister(t *testing.T) {
	documentID := "clx3k9a0b0001abcd"

	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{name: "valid registration", body: validBody(documentID), wantStatus: http.StatusCreated},
		{name: "malformed json", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "missing fields", body: `{"race_document_id":"` + documentID + `"}`, wantStatus: http.StatusBadRequest},
		{name: "missing participant document_id", body: strings.Replace(validBody(documentID), `"document_id":"1234567",`, "", 1), wantStatus: http.StatusBadRequest},
		{name: "modalidad is optional and can be omitted", body: strings.Replace(validBody(documentID), `,"modalidad":"10K · Con polera"`, "", 1), wantStatus: http.StatusCreated},
		{name: "bad birth date", body: strings.Replace(validBody(documentID), "2000-06-09", "09-06-2000", 1), wantStatus: http.StatusBadRequest},
		{name: "duplicate", body: validBody(documentID), serviceErr: domain.ErrAlreadyRegistered, wantStatus: http.StatusConflict},
		{name: "race full", body: validBody(documentID), serviceErr: domain.ErrRaceFull, wantStatus: http.StatusConflict},
		{name: "unknown race", body: validBody(documentID), serviceErr: racedomain.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "domain validation", body: validBody(documentID), serviceErr: domain.ErrBirthDateInFuture, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter(&fakeService{registerErr: tt.serviceErr})

			req := httptest.NewRequest(http.MethodPost, "/registrations", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Service-Secret", testSecret)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
		})
	}
}

// TestRegisterAuth proves the shared BFF↔Go service secret is enforced
// fail-closed on POST /registrations, mirroring the race module's
// TestWebhookAuth for its Sanity webhook secret.
func TestRegisterAuth(t *testing.T) {
	documentID := "clx3k9a0b0001abcd"

	t.Run("missing service secret is rejected", func(t *testing.T) {
		svc := &fakeService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodPost, "/registrations", strings.NewReader(validBody(documentID)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
		if svc.registerCalled {
			t.Error("service was called despite a missing secret")
		}
	})

	t.Run("wrong service secret is rejected", func(t *testing.T) {
		svc := &fakeService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodPost, "/registrations", strings.NewReader(validBody(documentID)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Service-Secret", "wrong-secret")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
		if svc.registerCalled {
			t.Error("service was called despite a wrong secret")
		}
	})

	t.Run("correct service secret proceeds normally", func(t *testing.T) {
		svc := &fakeService{}
		router := setupRouter(svc)

		req := httptest.NewRequest(http.MethodPost, "/registrations", strings.NewReader(validBody(documentID)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Service-Secret", testSecret)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body)
		}
		if !svc.registerCalled {
			t.Error("service was not called despite a valid secret")
		}
	})
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

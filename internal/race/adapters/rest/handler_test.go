package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"finish-line/internal/race/adapters/rest"
	"finish-line/internal/race/domain"
)

type fakeService struct {
	syncErr           error
	syncedDocumentID  string
	syncedName        string
	syncedCapacity    int
	removedDocumentID string
	races             []domain.Race
}

func (s *fakeService) Sync(_ context.Context, externalID, name string, _ time.Time, capacity int) (*domain.Race, error) {
	if s.syncErr != nil {
		return nil, s.syncErr
	}
	s.syncedDocumentID = externalID
	s.syncedName = name
	s.syncedCapacity = capacity
	return &domain.Race{ID: uuid.New(), DocumentID: externalID, Name: name, Capacity: capacity}, nil
}

func (s *fakeService) Remove(_ context.Context, externalID string) error {
	s.removedDocumentID = externalID
	return nil
}

func (s *fakeService) List(_ context.Context) ([]domain.Race, error) {
	return s.races, nil
}

const testSecret = "test-webhook-secret"

func noopMW(c *gin.Context) { c.Next() }

func setupRouter(svc *fakeService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rest.NewHandler(svc, testSecret, noopMW).RegisterRoutes(r)
	return r
}

func post(router *gin.Engine, secret, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/sanity", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-Webhook-Secret", secret)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestWebhookAuth(t *testing.T) {
	validBody := `{"operation":"create","slug":"renacer","title":"Carrera 10K","date":"2026-08-15","capacity":500}`

	t.Run("missing secret is rejected", func(t *testing.T) {
		svc := &fakeService{}
		rec := post(setupRouter(svc), "", validBody)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
		if svc.syncedDocumentID != "" {
			t.Error("service was called despite invalid secret")
		}
	})

	t.Run("wrong secret is rejected", func(t *testing.T) {
		rec := post(setupRouter(&fakeService{}), "wrong-secret", validBody)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}

func TestWebhookEvents(t *testing.T) {
	t.Run("create syncs the race", func(t *testing.T) {
		svc := &fakeService{}
		body := `{"operation":"create","slug":"renacer","title":"Carrera 10K","date":"2026-08-15","capacity":500}`
		rec := post(setupRouter(svc), testSecret, body)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
		}
		if svc.syncedDocumentID != "renacer" || svc.syncedCapacity != 500 {
			t.Errorf("Sync called with (%q, %d), want (renacer, 500)", svc.syncedDocumentID, svc.syncedCapacity)
		}
	})

	t.Run("update also syncs", func(t *testing.T) {
		svc := &fakeService{}
		body := `{"operation":"update","slug":"renacer","title":"Renombrada","date":"2026-08-15","capacity":800}`
		rec := post(setupRouter(svc), testSecret, body)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if svc.syncedName != "Renombrada" {
			t.Errorf("synced name = %q, want Renombrada", svc.syncedName)
		}
	})

	t.Run("delete removes the race", func(t *testing.T) {
		svc := &fakeService{}
		body := `{"operation":"delete","slug":"renacer"}`
		rec := post(setupRouter(svc), testSecret, body)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if svc.removedDocumentID != "renacer" {
			t.Errorf("removed document id = %q, want renacer", svc.removedDocumentID)
		}
	})

	t.Run("unknown operation is acknowledged, not failed", func(t *testing.T) {
		svc := &fakeService{}
		body := `{"operation":"publish","slug":"renacer"}`
		rec := post(setupRouter(svc), testSecret, body)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if svc.syncedDocumentID != "" || svc.removedDocumentID != "" {
			t.Error("service should not be called for ignored operations")
		}
	})

	t.Run("invalid date is a 400", func(t *testing.T) {
		body := `{"operation":"create","slug":"renacer","title":"Carrera","date":"15/08/2026","capacity":100}`
		rec := post(setupRouter(&fakeService{}), testSecret, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("payload without slug is a 400", func(t *testing.T) {
		body := `{"operation":"create","title":"Carrera","date":"2026-08-15","capacity":100}`
		rec := post(setupRouter(&fakeService{}), testSecret, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("missing operation is a 400", func(t *testing.T) {
		body := `{"slug":"renacer","title":"Carrera","date":"2026-08-15","capacity":100}`
		rec := post(setupRouter(&fakeService{}), testSecret, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("domain validation surfaces as 400", func(t *testing.T) {
		svc := &fakeService{syncErr: domain.ErrCapacityInvalid}
		body := `{"operation":"create","slug":"renacer","title":"Carrera","date":"2026-08-15","capacity":0}`
		rec := post(setupRouter(svc), testSecret, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body)
		}
	})
}

func TestListRaces(t *testing.T) {
	t.Run("returns both ids per race", func(t *testing.T) {
		id := uuid.New()
		svc := &fakeService{races: []domain.Race{
			{ID: id, DocumentID: "renacer", Name: "Carrera 10K", Date: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), Capacity: 500},
		}}
		req := httptest.NewRequest(http.MethodGet, "/races", nil)
		rec := httptest.NewRecorder()
		setupRouter(svc).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var out []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("not a JSON array: %v", err)
		}
		if len(out) != 1 || out[0]["race_id"] != id.String() || out[0]["document_id"] != "renacer" || out[0]["date"] != "2026-08-15" {
			t.Errorf("unexpected race list body: %s", rec.Body)
		}
	})

	t.Run("empty list is a JSON array, not null", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/races", nil)
		rec := httptest.NewRecorder()
		setupRouter(&fakeService{}).ServeHTTP(rec, req)

		if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
			t.Errorf("empty list body = %q, want []", body)
		}
	})
}

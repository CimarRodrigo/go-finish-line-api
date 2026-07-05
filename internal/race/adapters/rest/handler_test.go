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
	syncErr         error
	syncedStrapiID  string
	syncedName      string
	syncedCapacity  int
	removedStrapiID string
	races           []domain.Race
}

func (s *fakeService) Sync(_ context.Context, strapiID, name string, _ time.Time, capacity int) (*domain.Race, error) {
	if s.syncErr != nil {
		return nil, s.syncErr
	}
	s.syncedStrapiID = strapiID
	s.syncedName = name
	s.syncedCapacity = capacity
	return &domain.Race{ID: uuid.New(), StrapiID: strapiID, Name: name, Capacity: capacity}, nil
}

func (s *fakeService) Remove(_ context.Context, strapiID string) error {
	s.removedStrapiID = strapiID
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
	req := httptest.NewRequest(http.MethodPost, "/webhooks/strapi", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-Webhook-Secret", secret)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestWebhookAuth(t *testing.T) {
	validBody := `{"event":"entry.create","model":"race","entry":{"documentId":"doc-1","nombre":"Carrera 10K","fecha":"2026-08-15","capacidad":500}}`

	t.Run("missing secret is rejected", func(t *testing.T) {
		svc := &fakeService{}
		rec := post(setupRouter(svc), "", validBody)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
		if svc.syncedStrapiID != "" {
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
		body := `{"event":"entry.create","entry":{"documentId":"doc-1","nombre":"Carrera 10K","fecha":"2026-08-15","capacidad":500}}`
		rec := post(setupRouter(svc), testSecret, body)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
		}
		if svc.syncedStrapiID != "doc-1" || svc.syncedCapacity != 500 {
			t.Errorf("Sync called with (%q, %d), want (doc-1, 500)", svc.syncedStrapiID, svc.syncedCapacity)
		}
	})

	t.Run("update also syncs", func(t *testing.T) {
		svc := &fakeService{}
		body := `{"event":"entry.update","entry":{"documentId":"doc-1","nombre":"Renombrada","fecha":"2026-08-15","capacidad":800}}`
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
		body := `{"event":"entry.delete","entry":{"documentId":"doc-1"}}`
		rec := post(setupRouter(svc), testSecret, body)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if svc.removedStrapiID != "doc-1" {
			t.Errorf("removed strapi id = %q, want doc-1", svc.removedStrapiID)
		}
	})

	t.Run("numeric v4 id works as fallback", func(t *testing.T) {
		svc := &fakeService{}
		body := `{"event":"entry.create","entry":{"id":42,"nombre":"Carrera","fecha":"2026-08-15","capacidad":100}}`
		rec := post(setupRouter(svc), testSecret, body)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if svc.syncedStrapiID != "42" {
			t.Errorf("synced strapi id = %q, want 42", svc.syncedStrapiID)
		}
	})

	t.Run("unknown event is acknowledged, not failed", func(t *testing.T) {
		svc := &fakeService{}
		body := `{"event":"media.create","entry":{"documentId":"doc-9"}}`
		rec := post(setupRouter(svc), testSecret, body)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if svc.syncedStrapiID != "" || svc.removedStrapiID != "" {
			t.Error("service should not be called for ignored events")
		}
	})

	t.Run("invalid date is a 400", func(t *testing.T) {
		body := `{"event":"entry.create","entry":{"documentId":"doc-1","nombre":"Carrera","fecha":"15/08/2026","capacidad":100}}`
		rec := post(setupRouter(&fakeService{}), testSecret, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("entry without id is a 400", func(t *testing.T) {
		body := `{"event":"entry.create","entry":{"nombre":"Carrera","fecha":"2026-08-15","capacidad":100}}`
		rec := post(setupRouter(&fakeService{}), testSecret, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("domain validation surfaces as 400", func(t *testing.T) {
		svc := &fakeService{syncErr: domain.ErrCapacityInvalid}
		body := `{"event":"entry.create","entry":{"documentId":"doc-1","nombre":"Carrera","fecha":"2026-08-15","capacidad":0}}`
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
			{ID: id, StrapiID: "doc-1", Name: "Carrera 10K", Date: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), Capacity: 500},
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
		if len(out) != 1 || out[0]["race_id"] != id.String() || out[0]["document_id"] != "doc-1" || out[0]["date"] != "2026-08-15" {
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

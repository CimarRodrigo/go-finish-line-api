package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	participantdomain "finish-line/internal/participant/domain"
	"finish-line/internal/report/adapters/rest"
	"finish-line/internal/report/domain"
)

type fakeService struct {
	page domain.ParticipantPage
	err  error

	// what the service was asked for
	gotFilter domain.ParticipantFilter
	gotDays   int
	gotRaceID *uuid.UUID
	called    bool
}

func (s *fakeService) RegistrationsTimeline(_ context.Context, days int, raceID *uuid.UUID) ([]domain.TimelinePoint, error) {
	s.called, s.gotDays, s.gotRaceID = true, days, raceID
	if s.err != nil {
		return nil, s.err
	}
	return []domain.TimelinePoint{
		{Date: time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC), Count: 3},
	}, nil
}

func (s *fakeService) ReferralSources(_ context.Context, raceID *uuid.UUID) ([]domain.ReferralCount, error) {
	s.called, s.gotRaceID = true, raceID
	if s.err != nil {
		return nil, s.err
	}
	return []domain.ReferralCount{{Source: "Instagram", Count: 12}}, nil
}

func (s *fakeService) ShirtSizes(_ context.Context, raceID *uuid.UUID) ([]domain.ShirtSizeCount, error) {
	s.called, s.gotRaceID = true, raceID
	if s.err != nil {
		return nil, s.err
	}
	return []domain.ShirtSizeCount{{Size: "M", Count: 8}}, nil
}

func (s *fakeService) Participants(_ context.Context, filter domain.ParticipantFilter) (domain.ParticipantPage, error) {
	s.called, s.gotFilter = true, filter
	return s.page, s.err
}

func noopMW(c *gin.Context) { c.Next() }

func setupRouter(svc *fakeService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rest.NewHandler(svc, noopMW).RegisterRoutes(r)
	return r
}

func get(router *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRegistrationsTimeline(t *testing.T) {
	t.Run("serves plain days, not timestamps", func(t *testing.T) {
		rec := get(setupRouter(&fakeService{}), "/reports/registrations-timeline")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var out []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("not a JSON array: %v", err)
		}
		// A timestamp here would let the browser shift a registration into the
		// previous day by time zone.
		if out[0]["date"] != "2026-07-30" {
			t.Errorf("date = %v, want 2026-07-30", out[0]["date"])
		}
	})

	t.Run("passes the window and the race filter through", func(t *testing.T) {
		raceID := uuid.New()
		svc := &fakeService{}

		get(setupRouter(svc), "/reports/registrations-timeline?days=30&race_id="+raceID.String())

		if svc.gotDays != 30 {
			t.Errorf("days = %d, want 30", svc.gotDays)
		}
		if svc.gotRaceID == nil || *svc.gotRaceID != raceID {
			t.Errorf("raceID = %v, want %v", svc.gotRaceID, raceID)
		}
	})

	t.Run("no race filter means across all races", func(t *testing.T) {
		svc := &fakeService{}
		get(setupRouter(svc), "/reports/registrations-timeline")
		if svc.gotRaceID != nil {
			t.Errorf("raceID = %v, want nil", svc.gotRaceID)
		}
	})

	t.Run("an unreadable window falls back instead of failing", func(t *testing.T) {
		svc := &fakeService{}
		rec := get(setupRouter(svc), "/reports/registrations-timeline?days=abc")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 — a bad window must not blank the dashboard", rec.Code)
		}
		if svc.gotDays != 0 {
			t.Errorf("days = %d, want 0 so the domain applies its default", svc.gotDays)
		}
	})

	t.Run("a malformed race id is a 400", func(t *testing.T) {
		svc := &fakeService{}
		rec := get(setupRouter(svc), "/reports/registrations-timeline?race_id=not-a-uuid")

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		if svc.called {
			t.Error("the service must not be called for a malformed race id")
		}
	})
}

func TestReferralAndShirtSizes(t *testing.T) {
	t.Run("referral sources come back as an array", func(t *testing.T) {
		rec := get(setupRouter(&fakeService{}), "/reports/referral-sources")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var out []referral
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("not a JSON array: %v", err)
		}
		if len(out) != 1 || out[0].Source != "Instagram" || out[0].Count != 12 {
			t.Errorf("unexpected body: %s", rec.Body)
		}
	})

	t.Run("shirt sizes come back as an array", func(t *testing.T) {
		rec := get(setupRouter(&fakeService{}), "/reports/shirt-sizes")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if body := rec.Body.String(); body == "null" {
			t.Error("an empty report must serialize as [], not null")
		}
	})
}

type referral struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

func TestParticipants(t *testing.T) {
	t.Run("serves the page with its pager fields", func(t *testing.T) {
		svc := &fakeService{page: domain.ParticipantPage{
			Items: []domain.ParticipantRow{{
				ParticipantID: uuid.New(), FirstNames: "Amir", LastNames: "Rojas",
				Email: "amir@example.com", Gender: string(participantdomain.GenderMale),
				BirthDate: time.Date(2000, 6, 9, 0, 0, 0, 0, time.UTC), Age: 26, RacesCount: 3,
			}},
			Total: 130, Page: 3, PageSize: 25,
		}}

		rec := get(setupRouter(svc), "/participants?page=3&page_size=25")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var out struct {
			Items []struct {
				BirthDate  string `json:"birth_date"`
				Age        int    `json:"age"`
				RacesCount int    `json:"races_count"`
			} `json:"items"`
			Total    int `json:"total"`
			Page     int `json:"page"`
			PageSize int `json:"page_size"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("not the page shape: %v", err)
		}
		if out.Total != 130 || out.Page != 3 || out.PageSize != 25 {
			t.Errorf("pager = %d/%d/%d, want 130/3/25", out.Total, out.Page, out.PageSize)
		}
		if out.Items[0].RacesCount != 3 || out.Items[0].Age != 26 {
			t.Errorf("row = %+v, want 3 races and age 26", out.Items[0])
		}
		if out.Items[0].BirthDate != "2000-06-09" {
			t.Errorf("birth_date = %q, want a plain day", out.Items[0].BirthDate)
		}
	})

	t.Run("forwards every filter the panel offers", func(t *testing.T) {
		raceID := uuid.New()
		svc := &fakeService{}

		get(setupRouter(svc), "/participants?search=rojas&gender=F&age_min=25&age_max=30&race_id="+raceID.String())

		f := svc.gotFilter
		if f.Search != "rojas" || f.Gender != "F" {
			t.Errorf("search/gender = %q/%q, want rojas/F", f.Search, f.Gender)
		}
		if f.AgeMin == nil || *f.AgeMin != 25 || f.AgeMax == nil || *f.AgeMax != 30 {
			t.Errorf("age range = %v..%v, want 25..30", f.AgeMin, f.AgeMax)
		}
		if f.RaceID == nil || *f.RaceID != raceID {
			t.Errorf("raceID = %v, want %v", f.RaceID, raceID)
		}
	})

	t.Run("an absent age filter stays absent rather than becoming zero", func(t *testing.T) {
		svc := &fakeService{}
		get(setupRouter(svc), "/participants")

		// Flattening "not filtered" into 0 would quietly exclude nobody but
		// still look like a filter to everything downstream.
		if svc.gotFilter.AgeMin != nil || svc.gotFilter.AgeMax != nil {
			t.Errorf("age range = %v..%v, want both nil", svc.gotFilter.AgeMin, svc.gotFilter.AgeMax)
		}
	})

	t.Run("age zero is a real filter, not an absent one", func(t *testing.T) {
		svc := &fakeService{}
		get(setupRouter(svc), "/participants?age_min=0")

		if svc.gotFilter.AgeMin == nil || *svc.gotFilter.AgeMin != 0 {
			t.Errorf("AgeMin = %v, want a pointer to 0", svc.gotFilter.AgeMin)
		}
	})

	t.Run("a non-numeric age is a 400", func(t *testing.T) {
		svc := &fakeService{}
		rec := get(setupRouter(svc), "/participants?age_min=veinte")

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		if svc.called {
			t.Error("the service must not be called for an unparseable age")
		}
	})

	t.Run("a domain rejection surfaces as its own status", func(t *testing.T) {
		svc := &fakeService{err: domain.ErrAgeRangeInvalid}
		rec := get(setupRouter(svc), "/participants?age_min=50&age_max=20")

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}

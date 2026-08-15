package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	participantdomain "finish-line/internal/participant/domain"
	"finish-line/internal/report/domain"
	"finish-line/internal/report/service"
)

// fixedNow pins "today" so the date maths does not depend on the day the suite
// runs.
var fixedNow = time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

type fakeRepo struct {
	timeline []domain.TimelinePoint
	referral []domain.ReferralCount
	shirts   []domain.ShirtSizeCount
	rows     []domain.ParticipantRow
	total    int
	err      error

	// what the repository was asked for
	gotQuery       domain.ParticipantQuery
	gotFrom, gotTo time.Time
	gotRaceID      *uuid.UUID
	called         bool
}

func (f *fakeRepo) RegistrationsTimeline(_ context.Context, raceID *uuid.UUID, from, to time.Time) ([]domain.TimelinePoint, error) {
	f.called, f.gotRaceID, f.gotFrom, f.gotTo = true, raceID, from, to
	return f.timeline, f.err
}

func (f *fakeRepo) ReferralSources(_ context.Context, raceID *uuid.UUID) ([]domain.ReferralCount, error) {
	f.called, f.gotRaceID = true, raceID
	return f.referral, f.err
}

func (f *fakeRepo) ShirtSizes(_ context.Context, raceID *uuid.UUID) ([]domain.ShirtSizeCount, error) {
	f.called, f.gotRaceID = true, raceID
	return f.shirts, f.err
}

func (f *fakeRepo) Participants(_ context.Context, q domain.ParticipantQuery) ([]domain.ParticipantRow, int, error) {
	f.called, f.gotQuery = true, q
	return f.rows, f.total, f.err
}

func newService(repo *fakeRepo) *service.Service {
	return service.NewWithClock(repo, func() time.Time { return fixedNow })
}

func ptr[T any](v T) *T { return &v }

func TestRegistrationsTimeline(t *testing.T) {
	t.Run("asks for the right window and fills the quiet days", func(t *testing.T) {
		repo := &fakeRepo{timeline: []domain.TimelinePoint{
			{Date: time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC), Count: 5},
		}}

		got, err := newService(repo).RegistrationsTimeline(context.Background(), 14, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(got) != 14 {
			t.Fatalf("points = %d, want 14 — quiet days must still be plotted", len(got))
		}
		if last := got[13]; last.Count != 5 {
			t.Errorf("today's count = %d, want 5", last.Count)
		}
		if got[0].Count != 0 {
			t.Errorf("a day with no registrations = %d, want 0", got[0].Count)
		}
		if !repo.gotFrom.Equal(time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("from = %v, want 2026-07-17", repo.gotFrom)
		}
	})

	t.Run("passes the race filter straight through", func(t *testing.T) {
		raceID := uuid.New()
		repo := &fakeRepo{}

		if _, err := newService(repo).RegistrationsTimeline(context.Background(), 7, &raceID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotRaceID == nil || *repo.gotRaceID != raceID {
			t.Errorf("raceID = %v, want %v", repo.gotRaceID, raceID)
		}
	})

	t.Run("a repository failure surfaces", func(t *testing.T) {
		repo := &fakeRepo{err: errors.New("db down")}
		if _, err := newService(repo).RegistrationsTimeline(context.Background(), 14, nil); err == nil {
			t.Error("expected the repository error to surface")
		}
	})
}

func TestParticipants(t *testing.T) {
	t.Run("derives each row's age from today", func(t *testing.T) {
		repo := &fakeRepo{
			rows: []domain.ParticipantRow{
				{FirstNames: "Amir", BirthDate: time.Date(2000, 1, 10, 0, 0, 0, 0, time.UTC)},
				{FirstNames: "Ana", BirthDate: time.Date(2000, 12, 10, 0, 0, 0, 0, time.UTC)},
			},
			total: 2,
		}

		page, err := newService(repo).Participants(context.Background(), domain.ParticipantFilter{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Items[0].Age != 26 {
			t.Errorf("Amir's age = %d, want 26 (birthday passed)", page.Items[0].Age)
		}
		if page.Items[1].Age != 25 {
			t.Errorf("Ana's age = %d, want 25 (birthday ahead)", page.Items[1].Age)
		}
	})

	t.Run("reports the page it actually served", func(t *testing.T) {
		repo := &fakeRepo{total: 130}

		page, err := newService(repo).Participants(context.Background(), domain.ParticipantFilter{Page: 3, PageSize: 25})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Total != 130 || page.Page != 3 || page.PageSize != 25 {
			t.Errorf("page = %+v, want total 130, page 3, size 25", page)
		}
		if repo.gotQuery.Offset != 50 {
			t.Errorf("offset = %d, want 50", repo.gotQuery.Offset)
		}
	})

	t.Run("normalizes before querying, so defaults reach the repository", func(t *testing.T) {
		repo := &fakeRepo{}

		if _, err := newService(repo).Participants(context.Background(), domain.ParticipantFilter{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotQuery.Limit != domain.DefaultPageSize || repo.gotQuery.Offset != 0 {
			t.Errorf("limit/offset = %d/%d, want %d/0", repo.gotQuery.Limit, repo.gotQuery.Offset, domain.DefaultPageSize)
		}
	})

	t.Run("an invalid filter never reaches the database", func(t *testing.T) {
		repo := &fakeRepo{}

		_, err := newService(repo).Participants(context.Background(), domain.ParticipantFilter{AgeMin: ptr(50), AgeMax: ptr(20)})
		if !errors.Is(err, domain.ErrAgeRangeInvalid) {
			t.Errorf("error = %v, want ErrAgeRangeInvalid", err)
		}
		if repo.called {
			t.Error("the repository must not be queried for an invalid filter")
		}
	})

	t.Run("an invalid gender never reaches the database", func(t *testing.T) {
		repo := &fakeRepo{}

		_, err := newService(repo).Participants(context.Background(), domain.ParticipantFilter{Gender: "Z"})
		if !errors.Is(err, participantdomain.ErrGenderInvalid) {
			t.Errorf("error = %v, want ErrGenderInvalid", err)
		}
		if repo.called {
			t.Error("the repository must not be queried for an invalid filter")
		}
	})
}

func TestReferralAndShirtSizes(t *testing.T) {
	t.Run("referral sources pass through with their race filter", func(t *testing.T) {
		raceID := uuid.New()
		repo := &fakeRepo{referral: []domain.ReferralCount{{Source: "Instagram", Count: 12}}}

		got, err := newService(repo).ReferralSources(context.Background(), &raceID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0].Count != 12 {
			t.Errorf("counts = %+v, want Instagram 12", got)
		}
		if repo.gotRaceID == nil || *repo.gotRaceID != raceID {
			t.Errorf("raceID = %v, want %v", repo.gotRaceID, raceID)
		}
	})

	t.Run("shirt sizes pass through", func(t *testing.T) {
		repo := &fakeRepo{shirts: []domain.ShirtSizeCount{
			{Size: string(participantdomain.ShirtSizeM), Count: 8},
		}}

		got, err := newService(repo).ShirtSizes(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0].Size != "M" || got[0].Count != 8 {
			t.Errorf("counts = %+v, want M 8", got)
		}
	})
}

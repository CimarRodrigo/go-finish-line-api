package domain_test

import (
	"errors"
	"testing"
	"time"

	participantdomain "finish-line/internal/participant/domain"
	"finish-line/internal/report/domain"
)

func ptr[T any](v T) *T { return &v }

func TestParticipantFilterNormalize(t *testing.T) {
	t.Run("defaults an unset page and size", func(t *testing.T) {
		got, err := domain.ParticipantFilter{}.Normalize()
		if err != nil {
			t.Fatalf("Normalize() unexpected error: %v", err)
		}
		if got.Page != 1 || got.PageSize != domain.DefaultPageSize {
			t.Errorf("page/size = %d/%d, want 1/%d", got.Page, got.PageSize, domain.DefaultPageSize)
		}
	})

	t.Run("clamps an oversized page size instead of failing", func(t *testing.T) {
		got, err := domain.ParticipantFilter{PageSize: 5000}.Normalize()
		if err != nil {
			t.Fatalf("Normalize() unexpected error: %v", err)
		}
		if got.PageSize != domain.MaxPageSize {
			t.Errorf("PageSize = %d, want %d", got.PageSize, domain.MaxPageSize)
		}
	})

	t.Run("normalizes gender through the participant rules", func(t *testing.T) {
		got, err := domain.ParticipantFilter{Gender: " f "}.Normalize()
		if err != nil {
			t.Fatalf("Normalize() unexpected error: %v", err)
		}
		if got.Gender != "F" {
			t.Errorf("Gender = %q, want F", got.Gender)
		}
	})

	t.Run("rejects a gender outside the chart", func(t *testing.T) {
		_, err := domain.ParticipantFilter{Gender: "Z"}.Normalize()
		if !errors.Is(err, participantdomain.ErrGenderInvalid) {
			t.Errorf("error = %v, want ErrGenderInvalid", err)
		}
	})

	t.Run("an empty gender means any gender, not an error", func(t *testing.T) {
		got, err := domain.ParticipantFilter{}.Normalize()
		if err != nil || got.Gender != "" {
			t.Errorf("empty gender must pass through: got %q, err %v", got.Gender, err)
		}
	})

	ageTests := []struct {
		name     string
		min, max *int
		wantErr  bool
	}{
		{name: "min above max is rejected", min: ptr(40), max: ptr(20), wantErr: true},
		{name: "negative min is rejected", min: ptr(-1), wantErr: true},
		{name: "negative max is rejected", max: ptr(-5), wantErr: true},
		{name: "equal bounds are a valid single age", min: ptr(30), max: ptr(30)},
		{name: "only a minimum is valid", min: ptr(18)},
		{name: "no bounds at all is valid"},
	}
	for _, tt := range ageTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.ParticipantFilter{AgeMin: tt.min, AgeMax: tt.max}.Normalize()
			if tt.wantErr && !errors.Is(err, domain.ErrAgeRangeInvalid) {
				t.Errorf("error = %v, want ErrAgeRangeInvalid", err)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParticipantFilterResolve(t *testing.T) {
	// A fixed "today" so the date maths does not depend on the day the suite runs.
	today := time.Date(2026, 7, 30, 15, 4, 5, 0, time.UTC)

	t.Run("turns a page into limit and offset", func(t *testing.T) {
		f, _ := domain.ParticipantFilter{Page: 3, PageSize: 25}.Normalize()
		q := f.Resolve(today)
		if q.Limit != 25 || q.Offset != 50 {
			t.Errorf("limit/offset = %d/%d, want 25/50", q.Limit, q.Offset)
		}
	})

	// The bounds are the whole point of the age filter: an off-by-one day here
	// silently includes or drops everyone born on a birthday boundary.
	t.Run("a minimum age bounds the latest birth date", func(t *testing.T) {
		f, _ := domain.ParticipantFilter{AgeMin: ptr(25)}.Normalize()
		q := f.Resolve(today)

		want := time.Date(2001, 7, 30, 0, 0, 0, 0, time.UTC) // turns 25 exactly today
		if q.BirthTo == nil || !q.BirthTo.Equal(want) {
			t.Errorf("BirthTo = %v, want %v", q.BirthTo, want)
		}
		if q.BirthFrom != nil {
			t.Errorf("BirthFrom = %v, want nil when no maximum is set", q.BirthFrom)
		}
	})

	t.Run("a maximum age bounds the earliest birth date one day after the birthday", func(t *testing.T) {
		f, _ := domain.ParticipantFilter{AgeMax: ptr(30)}.Normalize()
		q := f.Resolve(today)

		// Born 1995-07-30 turns 31 today and must be excluded; born 1995-07-31
		// is still 30 and must be included.
		want := time.Date(1995, 7, 31, 0, 0, 0, 0, time.UTC)
		if q.BirthFrom == nil || !q.BirthFrom.Equal(want) {
			t.Errorf("BirthFrom = %v, want %v", q.BirthFrom, want)
		}
	})

	t.Run("someone whose birthday is today counts as the new age", func(t *testing.T) {
		f, _ := domain.ParticipantFilter{AgeMin: ptr(18), AgeMax: ptr(18)}.Normalize()
		q := f.Resolve(today)

		birthdayToday := time.Date(2008, 7, 30, 0, 0, 0, 0, time.UTC) // turns 18 today
		if birthdayToday.Before(*q.BirthFrom) || birthdayToday.After(*q.BirthTo) {
			t.Errorf("a runner turning 18 today must fall inside [%v, %v]", q.BirthFrom, q.BirthTo)
		}
	})
}

func TestTimelineRange(t *testing.T) {
	today := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)

	t.Run("covers exactly the requested number of days, ending today", func(t *testing.T) {
		from, to := domain.TimelineRange(14, today)

		if !to.Equal(time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("to = %v, want today truncated", to)
		}
		// 14 days INCLUDING today means 13 days back, not 14.
		if !from.Equal(time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("from = %v, want 2026-07-17", from)
		}
	})

	t.Run("falls back to the default window when days is not positive", func(t *testing.T) {
		from, to := domain.TimelineRange(0, today)
		if days := int(to.Sub(from).Hours()/24) + 1; days != domain.DefaultTimelineDays {
			t.Errorf("days = %d, want %d", days, domain.DefaultTimelineDays)
		}
	})

	t.Run("clamps an unbounded window", func(t *testing.T) {
		from, to := domain.TimelineRange(100000, today)
		if days := int(to.Sub(from).Hours()/24) + 1; days != domain.MaxTimelineDays {
			t.Errorf("days = %d, want %d", days, domain.MaxTimelineDays)
		}
	})
}

package domain_test

import (
	"testing"
	"time"

	"finish-line/internal/report/domain"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestFillTimeline(t *testing.T) {
	from, to := day(2026, 7, 1), day(2026, 7, 5)

	t.Run("quiet days appear with a zero count", func(t *testing.T) {
		got := domain.FillTimeline([]domain.TimelinePoint{
			{Date: day(2026, 7, 1), Count: 3},
			{Date: day(2026, 7, 4), Count: 7},
		}, from, to)

		if len(got) != 5 {
			t.Fatalf("points = %d, want 5 (one per day in range)", len(got))
		}
		want := []int{3, 0, 0, 7, 0}
		for i, w := range want {
			if got[i].Count != w {
				t.Errorf("day %s count = %d, want %d", got[i].Date.Format("2006-01-02"), got[i].Count, w)
			}
		}
	})

	t.Run("points come back in chronological order", func(t *testing.T) {
		got := domain.FillTimeline(nil, from, to)
		for i := 1; i < len(got); i++ {
			if !got[i].Date.After(got[i-1].Date) {
				t.Fatalf("points out of order at %d: %v then %v", i, got[i-1].Date, got[i].Date)
			}
		}
	})

	t.Run("a range with no registrations is all zeros, never empty", func(t *testing.T) {
		got := domain.FillTimeline(nil, from, to)
		if len(got) != 5 {
			t.Fatalf("points = %d, want 5", len(got))
		}
		for _, p := range got {
			if p.Count != 0 {
				t.Errorf("day %s = %d, want 0", p.Date.Format("2006-01-02"), p.Count)
			}
		}
	})

	t.Run("a timestamp within a day still matches that day", func(t *testing.T) {
		got := domain.FillTimeline([]domain.TimelinePoint{
			{Date: time.Date(2026, 7, 2, 18, 30, 0, 0, time.UTC), Count: 4},
		}, from, to)

		if got[1].Count != 4 {
			t.Errorf("2026-07-02 count = %d, want 4 — a mid-day timestamp must land on its day", got[1].Count)
		}
	})

	t.Run("a single-day range yields a single point", func(t *testing.T) {
		got := domain.FillTimeline(nil, from, from)
		if len(got) != 1 {
			t.Errorf("points = %d, want 1", len(got))
		}
	})
}

func TestSortShirtSizes(t *testing.T) {
	t.Run("orders by the size chart, not the alphabet", func(t *testing.T) {
		counts := []domain.ShirtSizeCount{
			{Size: "XXL", Count: 1}, {Size: "S", Count: 2}, {Size: "XL", Count: 3},
			{Size: "XS", Count: 4}, {Size: "L", Count: 5}, {Size: "M", Count: 6},
		}
		domain.SortShirtSizes(counts)

		want := []string{"XS", "S", "M", "L", "XL", "XXL"}
		for i, w := range want {
			if counts[i].Size != w {
				t.Fatalf("position %d = %q, want %q (got %v)", i, counts[i].Size, w, counts)
			}
		}
	})

	t.Run("the no-shirt bucket sorts last", func(t *testing.T) {
		counts := []domain.ShirtSizeCount{
			{Size: "", Count: 9}, {Size: "M", Count: 1},
		}
		domain.SortShirtSizes(counts)

		if counts[len(counts)-1].Size != "" {
			t.Errorf("last = %q, want the empty size", counts[len(counts)-1].Size)
		}
	})

	t.Run("sorting an empty report is not a crash", func(t *testing.T) {
		domain.SortShirtSizes(nil)
	})
}

func TestAgeAt(t *testing.T) {
	tests := []struct {
		name      string
		birth, at time.Time
		want      int
	}{
		{name: "birthday already passed this year", birth: day(2000, 1, 10), at: day(2026, 7, 30), want: 26},
		{name: "birthday still ahead this year", birth: day(2000, 12, 10), at: day(2026, 7, 30), want: 25},
		{name: "birthday is exactly today", birth: day(2000, 7, 30), at: day(2026, 7, 30), want: 26},
		{name: "the day before a birthday is still the old age", birth: day(2000, 7, 31), at: day(2026, 7, 30), want: 25},
		{name: "a future birth date never yields a negative age", birth: day(2030, 1, 1), at: day(2026, 7, 30), want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.AgeAt(tt.birth, tt.at); got != tt.want {
				t.Errorf("AgeAt() = %d, want %d", got, tt.want)
			}
		})
	}
}

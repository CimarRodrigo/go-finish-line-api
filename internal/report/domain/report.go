// Package domain holds the read models behind the admin dashboard.
//
// This is the read side: these types answer questions about registrations that
// already happened, so they carry no invariants and nothing here can be
// "invalid" — the rules live on the write side (the participant module). What
// DOES live here is the logic that turns what an admin asked for (ages, a page
// number, a window of days) into what a query needs (date ranges, limit and
// offset), because that translation is a decision, not plumbing.
package domain

import (
	"sort"
	"time"

	"github.com/google/uuid"

	participantdomain "finish-line/internal/participant/domain"
)

// TimelinePoint is one day of the registrations timeline. Days with no
// registrations are present with a zero count — see FillTimeline.
type TimelinePoint struct {
	Date  time.Time
	Count int
}

// ReferralCount is how many registrations named one referral source.
type ReferralCount struct {
	Source string
	Count  int
}

// ShirtSizeCount is how many shirts of one size a race needs. Registrations
// with no size (a modalidad without a shirt) are reported under an empty size
// rather than dropped, so the totals still add up to the registration count.
type ShirtSizeCount struct {
	Size  string
	Count int
}

// ParticipantRow is one line of the participants report: the person plus how
// many races they have run.
type ParticipantRow struct {
	ParticipantID uuid.UUID
	FirstNames    string
	LastNames     string
	Email         string
	Phone         string
	DocumentID    string
	Gender        string
	BirthDate     time.Time
	Age           int
	RacesCount    int
}

// ParticipantPage is one page of the participants report, with the total row
// count so the panel can render the pager.
type ParticipantPage struct {
	Items    []ParticipantRow
	Total    int
	Page     int
	PageSize int
}

// shirtSizeOrder is the size chart's running order. It is derived from the
// participant module's chart so there is still only one place to add a size.
var shirtSizeOrder = func() map[string]int {
	chart := []participantdomain.ShirtSize{
		participantdomain.ShirtSizeXS,
		participantdomain.ShirtSizeS,
		participantdomain.ShirtSizeM,
		participantdomain.ShirtSizeL,
		participantdomain.ShirtSizeXL,
		participantdomain.ShirtSizeXXL,
	}
	order := make(map[string]int, len(chart))
	for i, size := range chart {
		order[string(size)] = i
	}
	return order
}()

// SortShirtSizes puts the counts in the size chart's order, in place.
//
// Sorting them in the query would mean spelling the chart out in SQL as well,
// so adding a size would need a migration to a report. Sorting here keeps the
// chart in one place. Alphabetical order is not an option: it reads L, M, S,
// XL, XS, XXL, which is useless to whoever is ordering the shirts. Sizes off
// the chart — including the empty "no shirt" bucket — sort last.
func SortShirtSizes(counts []ShirtSizeCount) {
	sort.SliceStable(counts, func(i, j int) bool {
		return shirtSizeRank(counts[i].Size) < shirtSizeRank(counts[j].Size)
	})
}

func shirtSizeRank(size string) int {
	if rank, ok := shirtSizeOrder[size]; ok {
		return rank
	}
	return len(shirtSizeOrder)
}

// AgeAt returns the participant's age in completed years at the given day.
// Derived rather than stored, so it can never drift out of date.
func AgeAt(birthDate, at time.Time) int {
	age := at.Year() - birthDate.Year()
	// Not yet had this year's birthday. Compared by month and day rather than
	// day-of-year: a leap year shifts every date after February by one, which
	// would take a year off everyone born in the second half of one.
	if at.Month() < birthDate.Month() ||
		(at.Month() == birthDate.Month() && at.Day() < birthDate.Day()) {
		age--
	}
	if age < 0 {
		return 0
	}
	return age
}

// FillTimeline returns one point per day in [from, to], taking counts from
// found and using zero for the days it does not mention.
//
// The query only returns days that have registrations, but a chart with the
// empty days missing is a chart that lies: it draws a flat line between two
// busy days instead of the dip that actually happened. Filling the gaps is the
// read model's job, not the database's.
func FillTimeline(found []TimelinePoint, from, to time.Time) []TimelinePoint {
	counts := make(map[string]int, len(found))
	for _, p := range found {
		counts[dayKey(p.Date)] = p.Count
	}

	from, to = truncateDay(from), truncateDay(to)
	out := make([]TimelinePoint, 0)
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		out = append(out, TimelinePoint{Date: day, Count: counts[dayKey(day)]})
	}
	return out
}

func dayKey(t time.Time) string {
	return truncateDay(t).Format("2006-01-02")
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	participantdomain "finish-line/internal/participant/domain"
)

const (
	// DefaultPageSize is what the panel gets when it asks for no size.
	DefaultPageSize = 25
	// MaxPageSize caps how much one request can pull: without a ceiling a
	// caller could ask for every participant at once and turn a report into an
	// outage.
	MaxPageSize = 100

	// DefaultTimelineDays matches the dashboard's default window.
	DefaultTimelineDays = 14
	// MaxTimelineDays bounds the window for the same reason as MaxPageSize.
	MaxTimelineDays = 365
)

// ParticipantFilter is what the admin asked for: search text, optional race,
// gender and age bounds, and which page. It is the panel's vocabulary.
type ParticipantFilter struct {
	Search   string
	RaceID   *uuid.UUID
	Gender   string
	AgeMin   *int
	AgeMax   *int
	Page     int
	PageSize int
}

// ParticipantQuery is what the database needs: the same request with ages
// already turned into birth date bounds and the page turned into limit and
// offset. Keeping the two apart means the adapter never does arithmetic on
// dates and never needs to know what "today" is.
type ParticipantQuery struct {
	Search    string
	RaceID    *uuid.UUID
	Gender    string
	BirthFrom *time.Time
	BirthTo   *time.Time
	Limit     int
	Offset    int
}

// Normalize validates the filter and fills in defaults, returning the cleaned
// copy. Pagination is clamped rather than rejected — a page of 0 or a size of
// 5000 is a caller being sloppy, not a request worth failing. A malformed
// gender or an inverted age range IS rejected: those mean the admin asked for
// something we cannot answer, and silently returning everything would be a lie.
func (f ParticipantFilter) Normalize() (ParticipantFilter, error) {
	f.Search = strings.TrimSpace(f.Search)

	if f.Gender != "" {
		g, err := participantdomain.ParseGender(f.Gender)
		if err != nil {
			return ParticipantFilter{}, err
		}
		f.Gender = string(g)
	}

	if f.AgeMin != nil && *f.AgeMin < 0 {
		return ParticipantFilter{}, ErrAgeRangeInvalid
	}
	if f.AgeMax != nil && *f.AgeMax < 0 {
		return ParticipantFilter{}, ErrAgeRangeInvalid
	}
	if f.AgeMin != nil && f.AgeMax != nil && *f.AgeMin > *f.AgeMax {
		return ParticipantFilter{}, ErrAgeRangeInvalid
	}

	if f.Page < 1 {
		f.Page = 1
	}
	switch {
	case f.PageSize < 1:
		f.PageSize = DefaultPageSize
	case f.PageSize > MaxPageSize:
		f.PageSize = MaxPageSize
	}

	return f, nil
}

// Resolve turns a normalized filter into the query the repository runs.
//
// The age bounds become birth date bounds, both inclusive. Someone is N years
// old from the day they turn N until the day before they turn N+1, so:
//   - "at least AgeMin" means born on or before today minus AgeMin years;
//   - "at most AgeMax" means born on or after the day after today minus
//     (AgeMax + 1) years — one day later, or we would also catch the people who
//     turned AgeMax+1 exactly today.
func (f ParticipantFilter) Resolve(today time.Time) ParticipantQuery {
	today = truncateDay(today)

	q := ParticipantQuery{
		Search: f.Search,
		RaceID: f.RaceID,
		Gender: f.Gender,
		Limit:  f.PageSize,
		Offset: (f.Page - 1) * f.PageSize,
	}

	if f.AgeMin != nil {
		latest := today.AddDate(-*f.AgeMin, 0, 0)
		q.BirthTo = &latest
	}
	if f.AgeMax != nil {
		earliest := today.AddDate(-(*f.AgeMax+1), 0, 0).AddDate(0, 0, 1)
		q.BirthFrom = &earliest
	}

	return q
}

// TimelineRange returns the inclusive day range the timeline covers, ending
// today. days is clamped the same way page size is.
func TimelineRange(days int, today time.Time) (from, to time.Time) {
	switch {
	case days < 1:
		days = DefaultTimelineDays
	case days > MaxTimelineDays:
		days = MaxTimelineDays
	}

	to = truncateDay(today)
	from = to.AddDate(0, 0, -(days - 1))
	return from, to
}

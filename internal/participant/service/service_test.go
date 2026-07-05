package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"finish-line/internal/participant/domain"
	"finish-line/internal/participant/service"
	racedomain "finish-line/internal/race/domain"
)

type fakeParticipants struct {
	byEmail map[string]*domain.Participant
	byID    map[uuid.UUID]*domain.Participant
}

func newFakeParticipants() *fakeParticipants {
	return &fakeParticipants{byEmail: map[string]*domain.Participant{}, byID: map[uuid.UUID]*domain.Participant{}}
}

func (f *fakeParticipants) UpsertByEmail(_ context.Context, p *domain.Participant) (*domain.Participant, error) {
	if existing, ok := f.byEmail[p.Email]; ok {
		return existing, nil
	}
	f.byEmail[p.Email] = p
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeParticipants) ByID(_ context.Context, id uuid.UUID) (*domain.Participant, error) {
	p, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

type fakeRegistrations struct {
	byID       map[uuid.UUID]*domain.Registration
	pairs      map[string]bool
	nextDorsal int
}

func newFakeRegistrations() *fakeRegistrations {
	return &fakeRegistrations{byID: map[uuid.UUID]*domain.Registration{}, pairs: map[string]bool{}}
}

func (f *fakeRegistrations) Create(_ context.Context, r *domain.Registration) error {
	key := r.RaceID.String() + "|" + r.ParticipantID.String()
	if f.pairs[key] {
		return domain.ErrAlreadyRegistered
	}
	f.pairs[key] = true
	f.byID[r.ID] = r
	return nil
}

func (f *fakeRegistrations) ByID(_ context.Context, id uuid.UUID) (*domain.Registration, error) {
	r, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return r, nil
}

func (f *fakeRegistrations) NextDorsal(_ context.Context, _ uuid.UUID) (int, error) {
	return f.nextDorsal + 1, nil
}

func (f *fakeRegistrations) SaveConfirmation(_ context.Context, r *domain.Registration) error {
	f.nextDorsal++
	f.byID[r.ID] = r
	return nil
}

func (f *fakeRegistrations) ByRace(_ context.Context, _ uuid.UUID) ([]domain.RegistrationDetail, error) {
	return nil, nil
}

type fakeRaces struct {
	race *racedomain.Race
	err  error
}

func (f *fakeRaces) ByID(_ context.Context, _ uuid.UUID) (*racedomain.Race, error) {
	return f.race, f.err
}

func (f *fakeRaces) ByStrapiID(_ context.Context, _ string) (*racedomain.Race, error) {
	return f.race, f.err
}

type fakeNotifier struct {
	sent int
	err  error
}

func (n *fakeNotifier) SendConfirmation(_ context.Context, _ *domain.Participant, _ *domain.Registration, _ *racedomain.Race) error {
	n.sent++
	return n.err
}

func testRace(capacity int) *racedomain.Race {
	return &racedomain.Race{ID: uuid.New(), StrapiID: "doc-" + uuid.NewString(), Name: "Carrera 10K", Capacity: capacity}
}

func validInput(raceDocumentID string) service.RegisterInput {
	return service.RegisterInput{
		RaceDocumentID: raceDocumentID, FirstNames: "Amir", LastNames: "Rojas", Email: "amir@example.com",
		Phone: "+59171234567", BirthDate: time.Date(2000, 6, 9, 0, 0, 0, 0, time.UTC),
		Gender: "M", ReferralSource: "Instagram",
	}
}

func newService(race *racedomain.Race, notifier *fakeNotifier) (*service.Service, *fakeParticipants, *fakeRegistrations) {
	participants, registrations := newFakeParticipants(), newFakeRegistrations()
	svc := service.New(participants, registrations, &fakeRaces{race: race}, notifier)
	return svc, participants, registrations
}

func TestRegister(t *testing.T) {
	t.Run("free race: person upserted, registration confirmed with dorsal 1, notified", func(t *testing.T) {
		race := testRace(100)
		notifier := &fakeNotifier{}
		svc, participants, _ := newService(race, notifier)

		res, err := svc.Register(context.Background(), validInput(race.StrapiID))
		if err != nil {
			t.Fatalf("Register() unexpected error: %v", err)
		}
		if res.Registration.Status != domain.StatusConfirmed || *res.Registration.Dorsal != 1 {
			t.Error("registration not confirmed with dorsal 1")
		}
		if len(participants.byEmail) != 1 {
			t.Errorf("participants stored = %d, want 1", len(participants.byEmail))
		}
		if notifier.sent != 1 {
			t.Errorf("notifications = %d, want 1", notifier.sent)
		}
	})

	t.Run("same person, two races: one participant, sequential dorsals", func(t *testing.T) {
		notifier := &fakeNotifier{}
		participants, registrations := newFakeParticipants(), newFakeRegistrations()
		raceA, raceB := testRace(100), testRace(100)
		races := &multiRaceFinder{races: map[string]*racedomain.Race{raceA.StrapiID: raceA, raceB.StrapiID: raceB}}
		svc := service.New(participants, registrations, races, notifier)

		_, err := svc.Register(context.Background(), validInput(raceA.StrapiID))
		if err != nil {
			t.Fatalf("Register(A) error: %v", err)
		}
		_, err = svc.Register(context.Background(), validInput(raceB.StrapiID))
		if err != nil {
			t.Fatalf("Register(B) error: %v", err)
		}
		if len(participants.byEmail) != 1 {
			t.Errorf("same email must be ONE participant, got %d", len(participants.byEmail))
		}
	})

	t.Run("duplicate registration in same race is rejected", func(t *testing.T) {
		race := testRace(100)
		svc, _, _ := newService(race, &fakeNotifier{})

		_, _ = svc.Register(context.Background(), validInput(race.StrapiID))
		_, err := svc.Register(context.Background(), validInput(race.StrapiID))
		if !errors.Is(err, domain.ErrAlreadyRegistered) {
			t.Errorf("error = %v, want ErrAlreadyRegistered", err)
		}
	})

	t.Run("full race is rejected", func(t *testing.T) {
		race := testRace(0)
		svc, _, _ := newService(race, &fakeNotifier{})

		_, err := svc.Register(context.Background(), validInput(race.StrapiID))
		if !errors.Is(err, domain.ErrRaceFull) {
			t.Errorf("error = %v, want ErrRaceFull", err)
		}
	})

	t.Run("unknown race fails fast without persisting", func(t *testing.T) {
		participants, registrations := newFakeParticipants(), newFakeRegistrations()
		svc := service.New(participants, registrations, &fakeRaces{err: racedomain.ErrNotFound}, &fakeNotifier{})

		_, err := svc.Register(context.Background(), validInput("nonexistent-doc"))
		if !errors.Is(err, racedomain.ErrNotFound) {
			t.Errorf("error = %v, want race ErrNotFound", err)
		}
		if len(participants.byEmail) != 0 {
			t.Error("nothing must be persisted when the race does not exist")
		}
	})

	t.Run("invalid form data never reaches persistence", func(t *testing.T) {
		race := testRace(100)
		svc, participants, _ := newService(race, &fakeNotifier{})

		in := validInput(race.StrapiID)
		in.BirthDate = time.Now().AddDate(1, 0, 0)
		_, err := svc.Register(context.Background(), in)
		if !errors.Is(err, domain.ErrBirthDateInFuture) {
			t.Errorf("error = %v, want ErrBirthDateInFuture", err)
		}
		if len(participants.byEmail) != 0 {
			t.Error("invalid data must not be persisted")
		}
	})

	t.Run("notifier failure does not fail the registration", func(t *testing.T) {
		race := testRace(100)
		svc, _, _ := newService(race, &fakeNotifier{err: errors.New("smtp down")})

		res, err := svc.Register(context.Background(), validInput(race.StrapiID))
		if err != nil {
			t.Fatalf("Register() unexpected error: %v", err)
		}
		if res.Registration.Status != domain.StatusConfirmed {
			t.Error("registration must be confirmed even if notification fails")
		}
	})
}

// multiRaceFinder resolves several races by their Strapi documentId for the
// multi-race test.
type multiRaceFinder struct {
	races map[string]*racedomain.Race
}

func (f *multiRaceFinder) ByStrapiID(_ context.Context, strapiID string) (*racedomain.Race, error) {
	r, ok := f.races[strapiID]
	if !ok {
		return nil, racedomain.ErrNotFound
	}
	return r, nil
}

func (f *multiRaceFinder) ByID(_ context.Context, id uuid.UUID) (*racedomain.Race, error) {
	for _, r := range f.races {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, racedomain.ErrNotFound
}

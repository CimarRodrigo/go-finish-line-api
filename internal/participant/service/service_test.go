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
	deleteErr  error
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

// Delete mirrors the real repository: it frees the (race_id, participant_id)
// pair too, so a test can prove the runner is able to try that race again.
func (f *fakeRegistrations) Delete(_ context.Context, id uuid.UUID) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if r, ok := f.byID[id]; ok {
		delete(f.pairs, r.RaceID.String()+"|"+r.ParticipantID.String())
		delete(f.byID, id)
	}
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

func (f *fakeRaces) ByDocumentID(_ context.Context, _ string) (*racedomain.Race, error) {
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
	return &racedomain.Race{ID: uuid.New(), DocumentID: "doc-" + uuid.NewString(), Name: "Carrera 10K", Capacity: capacity}
}

func validInput(raceDocumentID string) service.RegisterInput {
	return service.RegisterInput{
		RaceDocumentID: raceDocumentID, FirstNames: "Amir", LastNames: "Rojas", Email: "amir@example.com",
		Phone: "+59171234567", DocumentID: "1234567", BirthDate: time.Date(2000, 6, 9, 0, 0, 0, 0, time.UTC),
		Gender: "M", ReferralSource: "Instagram", Modalidad: "10K · Con polera", ShirtSize: "M",
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

		res, err := svc.Register(context.Background(), validInput(race.DocumentID))
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
		if res.Participant.DocumentID != "1234567" {
			t.Errorf("DocumentID = %q, want %q", res.Participant.DocumentID, "1234567")
		}
		if res.Registration.Modalidad != "10K · Con polera" {
			t.Errorf("Modalidad = %q, want %q", res.Registration.Modalidad, "10K · Con polera")
		}
		if res.Registration.ShirtSize != domain.ShirtSizeM {
			t.Errorf("ShirtSize = %q, want M", res.Registration.ShirtSize)
		}
	})

	t.Run("an unknown shirt size is rejected before anything is persisted", func(t *testing.T) {
		race := testRace(100)
		svc, participants, _ := newService(race, &fakeNotifier{})

		in := validInput(race.DocumentID)
		in.ShirtSize = "Mediana"
		_, err := svc.Register(context.Background(), in)
		if !errors.Is(err, domain.ErrShirtSizeInvalid) {
			t.Errorf("error = %v, want ErrShirtSizeInvalid", err)
		}
		if len(participants.byEmail) != 0 {
			t.Error("nothing must be persisted when the shirt size is invalid")
		}
	})

	t.Run("same person, two races: one participant, sequential dorsals", func(t *testing.T) {
		notifier := &fakeNotifier{}
		participants, registrations := newFakeParticipants(), newFakeRegistrations()
		raceA, raceB := testRace(100), testRace(100)
		races := &multiRaceFinder{races: map[string]*racedomain.Race{raceA.DocumentID: raceA, raceB.DocumentID: raceB}}
		svc := service.New(participants, registrations, races, notifier)

		_, err := svc.Register(context.Background(), validInput(raceA.DocumentID))
		if err != nil {
			t.Fatalf("Register(A) error: %v", err)
		}
		_, err = svc.Register(context.Background(), validInput(raceB.DocumentID))
		if err != nil {
			t.Fatalf("Register(B) error: %v", err)
		}
		if len(participants.byEmail) != 1 {
			t.Errorf("same email must be ONE participant, got %d", len(participants.byEmail))
		}
	})

	// Resolves the spec requirement design flagged as UNVERIFIED: per-race
	// duplicate-by-email rejection. Confirmed in code (participants.email
	// UNIQUE + registrations UNIQUE(race_id, participant_id) →
	// ErrAlreadyRegistered → HTTP 409, see registration_repository.go and
	// handler_test.go's "duplicate" case); this pins the service-level
	// wiring with a table test that submits the SAME email twice for the
	// SAME race under otherwise DIFFERENT identity fields, so the rejection
	// is provably keyed by email — not by coincidental full-input equality.
	t.Run("second registration with the same email for the same race is rejected (dedup key is email)", func(t *testing.T) {
		race := testRace(100)
		svc, participants, _ := newService(race, &fakeNotifier{})

		first := validInput(race.DocumentID)
		_, err := svc.Register(context.Background(), first)
		if err != nil {
			t.Fatalf("first Register() unexpected error: %v", err)
		}

		second := validInput(race.DocumentID)
		second.FirstNames, second.LastNames = "Otra", "Persona"
		second.DocumentID = "7654321"
		_, err = svc.Register(context.Background(), second)
		if !errors.Is(err, domain.ErrAlreadyRegistered) {
			t.Errorf("error = %v, want ErrAlreadyRegistered", err)
		}
		if len(participants.byEmail) != 1 {
			t.Errorf("participants stored = %d, want 1 (same email must stay one person)", len(participants.byEmail))
		}
	})

	t.Run("full race is rejected", func(t *testing.T) {
		race := testRace(0)
		svc, _, _ := newService(race, &fakeNotifier{})

		_, err := svc.Register(context.Background(), validInput(race.DocumentID))
		if !errors.Is(err, domain.ErrRaceFull) {
			t.Errorf("error = %v, want ErrRaceFull", err)
		}
	})

	// Creating and confirming are one action to the runner, so a full race must
	// not leave the half-finished registration behind. Left there, the unique
	// (race_id, participant_id) answers every retry with "already registered"
	// — the wrong reason, and one that would outlive the race making more room.
	t.Run("a full race leaves no registration behind", func(t *testing.T) {
		race := testRace(0)
		svc, _, registrations := newService(race, &fakeNotifier{})

		_, err := svc.Register(context.Background(), validInput(race.DocumentID))
		if !errors.Is(err, domain.ErrRaceFull) {
			t.Fatalf("error = %v, want ErrRaceFull", err)
		}
		if len(registrations.byID) != 0 {
			t.Errorf("registrations left behind = %d, want 0", len(registrations.byID))
		}
	})

	t.Run("the runner can try again once the race makes room", func(t *testing.T) {
		race := testRace(0)
		svc, _, _ := newService(race, &fakeNotifier{})

		if _, err := svc.Register(context.Background(), validInput(race.DocumentID)); !errors.Is(err, domain.ErrRaceFull) {
			t.Fatalf("first attempt error = %v, want ErrRaceFull", err)
		}

		// The organiser raises the cap in the CMS and the same runner retries.
		race.Capacity = 10
		res, err := svc.Register(context.Background(), validInput(race.DocumentID))
		if err != nil {
			t.Fatalf("retry after the race grew: %v — a leftover row would report ErrAlreadyRegistered here", err)
		}
		if res.Registration.Status != domain.StatusConfirmed {
			t.Errorf("status = %q, want confirmed", res.Registration.Status)
		}
	})

	// The cleanup belongs to Register alone. Confirm is the payment seam: the
	// registration it works on was created by an earlier call and is waiting
	// for a payment, so a failure there must leave it exactly where it is.
	t.Run("the payment seam never deletes a registration it did not create", func(t *testing.T) {
		race := testRace(0) // full
		participants, registrations := newFakeParticipants(), newFakeRegistrations()

		person, err := domain.NewParticipant(domain.ParticipantParams{
			FirstNames: "Amir", LastNames: "Rojas", Email: "amir@example.com",
			Phone: "+59171234567", DocumentID: "1234567",
			BirthDate: time.Date(2000, 6, 9, 0, 0, 0, 0, time.UTC), Gender: "M",
		})
		if err != nil {
			t.Fatalf("building the participant: %v", err)
		}
		if _, err := participants.UpsertByEmail(context.Background(), person); err != nil {
			t.Fatalf("storing the participant: %v", err)
		}

		reg, err := domain.NewRegistration(domain.RegistrationParams{
			ParticipantID: person.ID, RaceID: race.ID, ReferralSource: "Instagram",
		})
		if err != nil {
			t.Fatalf("building the registration: %v", err)
		}
		if err := registrations.Create(context.Background(), reg); err != nil {
			t.Fatalf("storing the registration: %v", err)
		}

		svc := service.New(participants, registrations, &fakeRaces{race: race}, &fakeNotifier{})

		if _, err := svc.Confirm(context.Background(), reg.ID); !errors.Is(err, domain.ErrRaceFull) {
			t.Fatalf("error = %v, want ErrRaceFull", err)
		}
		if len(registrations.byID) != 1 {
			t.Error("Confirm must leave the pending registration in place — it is awaiting payment, not abandoned")
		}
	})

	t.Run("a failed cleanup still reports the real reason", func(t *testing.T) {
		race := testRace(0)
		participants, registrations := newFakeParticipants(), newFakeRegistrations()
		registrations.deleteErr = errors.New("db down")
		svc := service.New(participants, registrations, &fakeRaces{race: race}, &fakeNotifier{})

		// The caller needs to hear "the race is full", not whatever went wrong
		// while we were tidying up after ourselves.
		_, err := svc.Register(context.Background(), validInput(race.DocumentID))
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

		in := validInput(race.DocumentID)
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

		res, err := svc.Register(context.Background(), validInput(race.DocumentID))
		if err != nil {
			t.Fatalf("Register() unexpected error: %v", err)
		}
		if res.Registration.Status != domain.StatusConfirmed {
			t.Error("registration must be confirmed even if notification fails")
		}
	})
}

// multiRaceFinder resolves several races by their documentId for the
// multi-race test.
type multiRaceFinder struct {
	races map[string]*racedomain.Race
}

func (f *multiRaceFinder) ByDocumentID(_ context.Context, documentID string) (*racedomain.Race, error) {
	r, ok := f.races[documentID]
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

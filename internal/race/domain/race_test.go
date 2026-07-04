package domain_test

import (
	"errors"
	"testing"
	"time"

	"finish-line/internal/race/domain"
)

func TestNew(t *testing.T) {
	date := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		strapiID string
		raceName string
		capacity int
		wantErr  error
	}{
		{name: "valid race", strapiID: "doc-abc123", raceName: "Carrera 10K", capacity: 500},
		{name: "missing strapi id", strapiID: "  ", raceName: "Carrera 10K", capacity: 500, wantErr: domain.ErrStrapiIDRequired},
		{name: "missing name", strapiID: "doc-abc123", raceName: "", capacity: 500, wantErr: domain.ErrNameRequired},
		{name: "zero capacity", strapiID: "doc-abc123", raceName: "Carrera 10K", capacity: 0, wantErr: domain.ErrCapacityInvalid},
		{name: "negative capacity", strapiID: "doc-abc123", raceName: "Carrera 10K", capacity: -10, wantErr: domain.ErrCapacityInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := domain.New(tt.strapiID, tt.raceName, date, tt.capacity)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("New() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("New() unexpected error: %v", err)
			}
			if r.ID.String() == "00000000-0000-0000-0000-000000000000" {
				t.Error("New() did not assign an ID")
			}
		})
	}
}

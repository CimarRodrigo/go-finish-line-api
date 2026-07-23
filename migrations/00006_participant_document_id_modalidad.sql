-- +goose Up
-- Additive only, both columns nullable — no backfill for existing rows.
-- documento_identidad (CI) is required by the domain for every NEW
-- registration but is NOT unique: v1 stores it, it does not dedupe by it
-- (participants.email already enforces per-race duplicate rejection).
-- modalidad records the distance/variant chosen on the detail page.
ALTER TABLE participants ADD COLUMN documento_identidad text;
ALTER TABLE registrations ADD COLUMN modalidad text;

-- +goose Down
ALTER TABLE registrations DROP COLUMN modalidad;
ALTER TABLE participants DROP COLUMN documento_identidad;

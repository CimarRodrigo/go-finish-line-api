-- +goose Up
-- Additive and nullable — no backfill: existing registrations predate the
-- shirt size and legitimately have none.
-- talla_polera lives on the registration, not the participant: the size is
-- chosen per race, so the same runner may pick a different one next time and
-- each race keeps its own historical record.
-- The value set (XS..XXL) is enforced by the domain, not a CHECK constraint,
-- so the size chart can change without a migration.
ALTER TABLE registrations ADD COLUMN talla_polera text;

-- +goose Down
ALTER TABLE registrations DROP COLUMN talla_polera;

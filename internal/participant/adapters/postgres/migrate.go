package postgres

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate creates the participants and registrations tables and their foreign
// keys. RESTRICT (not CASCADE) protects registration data: a person or race
// that still has registrations cannot be deleted out from under them.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&participantModel{}, &registrationModel{}); err != nil {
		return fmt.Errorf("migrating participant tables: %w", err)
	}

	fks := []struct{ name, sql string }{
		{
			name: "fk_registrations_participant",
			sql: `ALTER TABLE registrations
				ADD CONSTRAINT fk_registrations_participant
				FOREIGN KEY (participant_id) REFERENCES participants(id) ON DELETE RESTRICT`,
		},
		{
			name: "fk_registrations_race",
			sql: `ALTER TABLE registrations
				ADD CONSTRAINT fk_registrations_race
				FOREIGN KEY (race_id) REFERENCES races(id) ON DELETE RESTRICT`,
		},
	}

	for _, fk := range fks {
		stmt := fmt.Sprintf(
			`DO $$ BEGIN
			  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = '%s') THEN
			    %s;
			  END IF;
			END $$;`, fk.name, fk.sql)
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("adding %s: %w", fk.name, err)
		}
	}
	return nil
}

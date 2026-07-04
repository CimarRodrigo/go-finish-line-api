package postgres

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate creates the tables this module owns.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&raceModel{}); err != nil {
		return fmt.Errorf("migrating races table: %w", err)
	}
	return nil
}

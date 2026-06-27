package migrations

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// AddTransferChecks creates the transfer_checks table for config comparison runs.
func AddTransferChecks() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "015_add_transfer_checks",
		Migrate: func(tx *gorm.DB) error {
			fmt.Println("Running migration 015: Adding transfer checks table...")

			if err := tx.Exec(`CREATE TABLE IF NOT EXISTS transfer_checks (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				config_id INTEGER NOT NULL,
				status VARCHAR(20) NOT NULL DEFAULT 'pending',
				started_at DATETIME,
				completed_at DATETIME,
				differences INTEGER NOT NULL DEFAULT 0,
				missing_on_source INTEGER NOT NULL DEFAULT 0,
				missing_on_dest INTEGER NOT NULL DEFAULT 0,
				error_message TEXT,
				created_by INTEGER NOT NULL,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY (config_id) REFERENCES transfer_configs(id),
				FOREIGN KEY (created_by) REFERENCES users(id)
			)`).Error; err != nil {
				return fmt.Errorf("failed to create transfer_checks table: %w", err)
			}

			fmt.Println("Migration 015 completed successfully.")
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			fmt.Println("Rolling back migration 015: Removing transfer checks...")

			if err := tx.Exec(`DROP TABLE IF EXISTS transfer_checks`).Error; err != nil {
				return fmt.Errorf("failed to drop transfer_checks table: %w", err)
			}

			return nil
		},
	}
}

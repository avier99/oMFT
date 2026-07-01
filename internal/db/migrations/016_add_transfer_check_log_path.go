package migrations

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// AddTransferCheckLogPath adds a log_path column to transfer_checks for rclone output.
func AddTransferCheckLogPath() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "016_add_transfer_check_log_path",
		Migrate: func(tx *gorm.DB) error {
			fmt.Println("Running migration 016: Adding log_path to transfer_checks...")

			if err := tx.Exec(`ALTER TABLE transfer_checks ADD COLUMN log_path TEXT DEFAULT ''`).Error; err != nil {
				return fmt.Errorf("failed to add log_path column: %w", err)
			}

			fmt.Println("Migration 016 completed successfully.")
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			fmt.Println("Rolling back migration 016: Removing log_path from transfer_checks...")

			if err := tx.Exec(`ALTER TABLE transfer_checks DROP COLUMN log_path`).Error; err != nil {
				return fmt.Errorf("failed to drop log_path column: %w", err)
			}

			return nil
		},
	}
}

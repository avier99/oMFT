package migrations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func TestAddTransferChecksMigration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "omft-transfer-checks-migration-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	gormDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	priorMigrations := []*gormigrate.Migration{
		InitialSchema(),
		UpdateGDriveType(),
		Add2FA(),
		AddAuditLogs(),
		AddDefaultRoles(),
		AddTimestampsToJobHistories(),
		AddNotificationServices(),
		AddUserNotifications(),
		AddRcloneTables(),
		AddRcloneCommandToConfig(),
		AddAuthProviders(),
		AlterBooleanDefaults(),
		RecoverTransferConfigsRename(),
		RecoverNotificationServicesRename(),
		RecoverAuthProvidersRename(),
		CleanupInvalidBooleans(),
		AddMachines(),
	}
	if err := gormigrate.New(gormDB, gormigrate.DefaultOptions, priorMigrations).Migrate(); err != nil {
		t.Fatalf("failed to run prior migrations: %v", err)
	}

	if err := AddTransferChecks().Migrate(gormDB); err != nil {
		t.Fatalf("migration 015 failed: %v", err)
	}

	var tableCount int
	if err := gormDB.Raw("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'transfer_checks'").Scan(&tableCount).Error; err != nil {
		t.Fatalf("failed to inspect transfer_checks table: %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("expected transfer_checks table to exist, got count %d", tableCount)
	}

	if err := gormDB.Exec(`INSERT INTO users (id, email, password_hash, created_at, updated_at) VALUES (1, 'test@example.com', 'hash', datetime('now'), datetime('now'))`).Error; err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	if err := gormDB.Exec(`INSERT INTO transfer_configs (
		id, name, source_type, source_path, destination_type, destination_path, created_by, created_at, updated_at
	) VALUES (1, 'cfg-a', 'local', '/src', 'local', '/dest', 1, datetime('now'), datetime('now'))`).Error; err != nil {
		t.Fatalf("failed to insert transfer config: %v", err)
	}
	if err := gormDB.Exec(`INSERT INTO transfer_checks (config_id, created_by, created_at, updated_at) VALUES (1, 1, datetime('now'), datetime('now'))`).Error; err != nil {
		t.Fatalf("failed to insert transfer check: %v", err)
	}

	var status string
	var differences, missingOnSource, missingOnDest int
	if err := gormDB.Table("transfer_checks").
		Select("status", "differences", "missing_on_source", "missing_on_dest").
		Where("config_id = ?", 1).
		Row().
		Scan(&status, &differences, &missingOnSource, &missingOnDest); err != nil {
		t.Fatalf("failed to read transfer check defaults: %v", err)
	}
	if status != "pending" || differences != 0 || missingOnSource != 0 || missingOnDest != 0 {
		t.Fatalf("unexpected defaults: status=%q differences=%d missing_on_source=%d missing_on_dest=%d", status, differences, missingOnSource, missingOnDest)
	}

	if err := AddTransferChecks().Rollback(gormDB); err != nil {
		t.Fatalf("migration 015 rollback failed: %v", err)
	}

	if err := gormDB.Raw("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'transfer_checks'").Scan(&tableCount).Error; err != nil {
		t.Fatalf("failed to inspect transfer_checks table after rollback: %v", err)
	}
	if tableCount != 0 {
		t.Fatalf("expected transfer_checks table to be dropped, got count %d", tableCount)
	}
}

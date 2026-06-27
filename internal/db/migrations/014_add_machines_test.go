package migrations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func TestAddMachinesMigration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "omft-migration-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	gormDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Run migrations 001-013 only
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
	}
	if err := gormigrate.New(gormDB, gormigrate.DefaultOptions, priorMigrations).Migrate(); err != nil {
		t.Fatalf("failed to run prior migrations: %v", err)
	}

	// Seed user and transfer configs
	if err := gormDB.Exec(`INSERT INTO users (id, email, password_hash, created_at, updated_at) VALUES (1, 'test@example.com', 'hash', datetime('now'), datetime('now'))`).Error; err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	if err := gormDB.Exec(`INSERT INTO transfer_configs (
		id, name, source_type, source_path, source_host, source_port, source_user,
		destination_type, destination_path, dest_host, dest_port, dest_user, created_by, created_at, updated_at
	) VALUES
	(1, 'cfg-a', 'sftp', '/src', 'host1.example.com', 22, 'alice', 'local', '/dest', '', 0, '', 1, datetime('now'), datetime('now')),
	(2, 'cfg-b', 'sftp', '/src2', 'host1.example.com', 22, 'alice', 'sftp', '/dest2', 'host2.example.com', 22, 'bob', 1, datetime('now'), datetime('now')),
	(3, 'cfg-local', 'local', '/local', '', 0, '', 'local', '/local2', '', 0, '', 1, datetime('now'), datetime('now'))`).Error; err != nil {
		t.Fatalf("failed to insert transfer configs: %v", err)
	}

	// Run migration 014
	if err := AddMachines().Migrate(gormDB); err != nil {
		t.Fatalf("migration 014 failed: %v", err)
	}

	var machineCount int64
	gormDB.Table("machines").Count(&machineCount)
	if machineCount != 2 {
		t.Fatalf("expected 2 machines (shared source fingerprint + unique dest), got %d", machineCount)
	}

	var cfg1SourceMachineID, cfg1DestMachineID *uint
	gormDB.Table("transfer_configs").Where("id = 1").Select("source_machine_id", "dest_machine_id").Row().Scan(&cfg1SourceMachineID, &cfg1DestMachineID)
	if cfg1SourceMachineID == nil {
		t.Fatal("config 1 should have source_machine_id set")
	}
	if cfg1DestMachineID != nil {
		t.Fatal("config 1 with local dest should not have dest_machine_id")
	}

	var cfg2SourceMachineID, cfg2DestMachineID *uint
	gormDB.Table("transfer_configs").Where("id = 2").Select("source_machine_id", "dest_machine_id").Row().Scan(&cfg2SourceMachineID, &cfg2DestMachineID)
	if cfg2SourceMachineID == nil || cfg2DestMachineID == nil {
		t.Fatal("config 2 should have both machine IDs set")
	}
	if *cfg1SourceMachineID != *cfg2SourceMachineID {
		t.Fatal("configs sharing source fingerprint should reference the same machine")
	}

	var cfg3SourceMachineID, cfg3DestMachineID *uint
	gormDB.Table("transfer_configs").Where("id = 3").Select("source_machine_id", "dest_machine_id").Row().Scan(&cfg3SourceMachineID, &cfg3DestMachineID)
	if cfg3SourceMachineID != nil || cfg3DestMachineID != nil {
		t.Fatal("local-only config should not have machine IDs")
	}
}

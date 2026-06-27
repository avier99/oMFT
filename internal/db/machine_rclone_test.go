package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupMachineRcloneTestDB(t *testing.T) *DB {
	t.Helper()

	gormDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := gormDB.AutoMigrate(&User{}, &Machine{}, &TransferConfig{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return &DB{DB: gormDB}
}

func TestGenerateRcloneConfigWithMachinesMaterializesFromExistingConfig(t *testing.T) {
	database := setupMachineRcloneTestDB(t)
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)

	sourceMachineID := uint(1)
	destMachineID := uint(2)
	if err := database.Create(&Machine{ID: sourceMachineID, Name: "source", Type: "sftp"}).Error; err != nil {
		t.Fatalf("failed to create source machine: %v", err)
	}
	if err := database.Create(&Machine{ID: destMachineID, Name: "dest", Type: "s3"}).Error; err != nil {
		t.Fatalf("failed to create destination machine: %v", err)
	}

	config := &TransferConfig{
		ID:              10,
		Name:            "config",
		SourceType:      "sftp",
		SourcePath:      "/incoming",
		SourceMachineID: &sourceMachineID,
		DestinationType: "s3",
		DestinationPath: "/outgoing",
		DestMachineID:   &destMachineID,
	}

	configPath := database.GetConfigRclonePath(config)
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	legacyConfig := `[source_10]
type = sftp
host = source.example.com
user = alice
pass = obscured-source

[dest_10]
type = s3
provider = AWS
access_key_id = key
secret_access_key = obscured-dest
`
	if err := os.WriteFile(configPath, []byte(legacyConfig), 0600); err != nil {
		t.Fatalf("failed to write legacy config: %v", err)
	}

	if err := database.GenerateRcloneConfig(config); err != nil {
		t.Fatalf("GenerateRcloneConfig failed: %v", err)
	}

	sourceMachineConfig, err := os.ReadFile(database.GetMachineRclonePath(&Machine{ID: sourceMachineID}))
	if err != nil {
		t.Fatalf("failed to read source machine config: %v", err)
	}
	if !strings.Contains(string(sourceMachineConfig), "[machine_1]") || !strings.Contains(string(sourceMachineConfig), "pass = obscured-source") {
		t.Fatalf("source machine config was not materialized from legacy config:\n%s", sourceMachineConfig)
	}

	destMachineConfig, err := os.ReadFile(database.GetMachineRclonePath(&Machine{ID: destMachineID}))
	if err != nil {
		t.Fatalf("failed to read destination machine config: %v", err)
	}
	if !strings.Contains(string(destMachineConfig), "[machine_2]") || !strings.Contains(string(destMachineConfig), "secret_access_key = obscured-dest") {
		t.Fatalf("destination machine config was not materialized from legacy config:\n%s", destMachineConfig)
	}

	runtimeConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read runtime config: %v", err)
	}
	runtime := string(runtimeConfig)
	if !strings.Contains(runtime, "[source_10]") || !strings.Contains(runtime, "[dest_10]") {
		t.Fatalf("runtime config should preserve source/dest remote names:\n%s", runtime)
	}
	if strings.Contains(runtime, "[machine_1]") || strings.Contains(runtime, "[machine_2]") {
		t.Fatalf("runtime config should not expose machine remote names:\n%s", runtime)
	}
}

func TestEnsureMachineConfigFromExistingRemoteIsIdempotent(t *testing.T) {
	database := setupMachineRcloneTestDB(t)
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)

	machine := &Machine{ID: 3, Name: "source", Type: "sftp"}

	// Pre-populate machine_3.conf with the correct section.
	machinePath := database.GetMachineRclonePath(machine)
	if err := os.MkdirAll(filepath.Dir(machinePath), 0755); err != nil {
		t.Fatalf("failed to create machine config dir: %v", err)
	}
	existingMachineConfig := `[machine_3]
type = sftp
host = source.example.com
user = alice
pass = original-secret
`
	if err := os.WriteFile(machinePath, []byte(existingMachineConfig), 0600); err != nil {
		t.Fatalf("failed to write existing machine config: %v", err)
	}

	// A different legacy config section that must NOT overwrite the existing machine config.
	legacyConfig := `[source_99]
type = sftp
host = different.example.com
user = bob
pass = different-secret
`

	if err := database.ensureMachineConfigFromExistingRemote(machine, legacyConfig, "source_99"); err != nil {
		t.Fatalf("ensureMachineConfigFromExistingRemote failed: %v", err)
	}

	after, err := os.ReadFile(machinePath)
	if err != nil {
		t.Fatalf("failed to read machine config: %v", err)
	}
	if string(after) != existingMachineConfig {
		t.Fatalf("existing machine config should be left untouched when section already present:\n got:\n%s\nwant:\n%s", after, existingMachineConfig)
	}
}

func TestGenerateRcloneConfigMixedMachineAndLegacy(t *testing.T) {
	database := setupMachineRcloneTestDB(t)
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)

	sourceMachineID := uint(5)
	if err := database.Create(&Machine{ID: sourceMachineID, Name: "source", Type: "sftp"}).Error; err != nil {
		t.Fatalf("failed to create source machine: %v", err)
	}

	// Pre-materialize the source machine config so no rclone binary is required.
	sourceMachinePath := database.GetMachineRclonePath(&Machine{ID: sourceMachineID})
	if err := os.MkdirAll(filepath.Dir(sourceMachinePath), 0755); err != nil {
		t.Fatalf("failed to create machine config dir: %v", err)
	}
	sourceMachineConfig := `[machine_5]
type = sftp
host = source.example.com
user = alice
pass = source-secret
`
	if err := os.WriteFile(sourceMachinePath, []byte(sourceMachineConfig), 0600); err != nil {
		t.Fatalf("failed to write source machine config: %v", err)
	}

	// Destination is legacy (local) — no machine FK, no rclone binary needed.
	config := &TransferConfig{
		ID:              20,
		Name:            "mixed",
		SourceType:      "sftp",
		SourcePath:      "/incoming",
		SourceMachineID: &sourceMachineID,
		DestinationType: "local",
		DestinationPath: "/outgoing",
	}

	if err := database.GenerateRcloneConfig(config); err != nil {
		t.Fatalf("GenerateRcloneConfig failed: %v", err)
	}

	runtimeConfig, err := os.ReadFile(database.GetConfigRclonePath(config))
	if err != nil {
		t.Fatalf("failed to read runtime config: %v", err)
	}
	runtime := string(runtimeConfig)

	if !strings.Contains(runtime, "[source_20]") || !strings.Contains(runtime, "pass = source-secret") {
		t.Fatalf("runtime config should contain machine-backed source remote:\n%s", runtime)
	}
	if !strings.Contains(runtime, "[dest_20]") || !strings.Contains(runtime, "type = local") {
		t.Fatalf("runtime config should contain legacy local destination remote:\n%s", runtime)
	}
	if strings.Contains(runtime, "[machine_5]") {
		t.Fatalf("runtime config should not expose machine remote name:\n%s", runtime)
	}
}

func TestGenerateMachineRcloneConfigLocal(t *testing.T) {
	database := &DB{}
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)

	machine := &Machine{ID: 7, Name: "local", Type: "local"}
	if err := database.GenerateMachineRcloneConfig(machine); err != nil {
		t.Fatalf("GenerateMachineRcloneConfig failed: %v", err)
	}

	content, err := os.ReadFile(database.GetMachineRclonePath(machine))
	if err != nil {
		t.Fatalf("failed to read machine config: %v", err)
	}
	if !strings.Contains(string(content), "[machine_7]") || !strings.Contains(string(content), "type = local") {
		t.Fatalf("unexpected local machine config:\n%s", content)
	}
}

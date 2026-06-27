package migrations

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

type machineRecord struct {
	ID          uint `gorm:"primarykey"`
	Name        string
	Type        string
	Host        string
	Port        int
	User        string
	KeyFile     string
	Region      string
	AccessKey   string
	Endpoint    string
	Bucket      string
	Domain      string
	Share       string
	PassiveMode *bool
	ClientID    string
	DriveID     string
	TeamDrive   string
	CreatedBy   uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (machineRecord) TableName() string {
	return "machines"
}

type transferConfigMigrationRow struct {
	ID              uint
	CreatedBy       uint
	SourceType      string
	SourceHost      string
	SourcePort      int
	SourceUser      string
	SourceKeyFile   string
	SourceBucket    string
	SourceRegion    string
	SourceAccessKey string
	SourceEndpoint  string
	SourceShare     string
	SourceDomain    string
	SourcePassiveMode *bool
	SourceClientID  string
	SourceDriveID   string
	SourceTeamDrive string
	DestinationType string
	DestHost        string
	DestPort        int
	DestUser        string
	DestKeyFile     string
	DestBucket      string
	DestRegion      string
	DestAccessKey   string
	DestEndpoint    string
	DestShare       string
	DestDomain      string
	DestPassiveMode *bool
	DestClientID    string
	DestDriveID     string
	DestTeamDrive   string
}

type machineFingerprint struct {
	Type string
	Host string
	Port int
	User string
}

func machineFingerprintKey(fp machineFingerprint) string {
	return fmt.Sprintf("%s|%s|%d|%s", fp.Type, fp.Host, fp.Port, fp.User)
}

func defaultMachineName(fp machineFingerprint) string {
	if fp.User != "" && fp.Host != "" {
		return fmt.Sprintf("%s (%s@%s)", fp.Type, fp.User, fp.Host)
	}
	if fp.Host != "" {
		return fmt.Sprintf("%s (%s)", fp.Type, fp.Host)
	}
	return fp.Type
}

func machineFromSource(config transferConfigMigrationRow) machineRecord {
	now := time.Now()
	return machineRecord{
		Name:        defaultMachineName(machineFingerprint{Type: config.SourceType, Host: config.SourceHost, Port: config.SourcePort, User: config.SourceUser}),
		Type:        config.SourceType,
		Host:        config.SourceHost,
		Port:        config.SourcePort,
		User:        config.SourceUser,
		KeyFile:     config.SourceKeyFile,
		Region:      config.SourceRegion,
		AccessKey:   config.SourceAccessKey,
		Endpoint:    config.SourceEndpoint,
		Bucket:      config.SourceBucket,
		Domain:      config.SourceDomain,
		Share:       config.SourceShare,
		PassiveMode: config.SourcePassiveMode,
		ClientID:    config.SourceClientID,
		DriveID:     config.SourceDriveID,
		TeamDrive:   config.SourceTeamDrive,
		CreatedBy:   config.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func machineFromDest(config transferConfigMigrationRow) machineRecord {
	now := time.Now()
	return machineRecord{
		Name:        defaultMachineName(machineFingerprint{Type: config.DestinationType, Host: config.DestHost, Port: config.DestPort, User: config.DestUser}),
		Type:        config.DestinationType,
		Host:        config.DestHost,
		Port:        config.DestPort,
		User:        config.DestUser,
		KeyFile:     config.DestKeyFile,
		Region:      config.DestRegion,
		AccessKey:   config.DestAccessKey,
		Endpoint:    config.DestEndpoint,
		Bucket:      config.DestBucket,
		Domain:      config.DestDomain,
		Share:       config.DestShare,
		PassiveMode: config.DestPassiveMode,
		ClientID:    config.DestClientID,
		DriveID:     config.DestDriveID,
		TeamDrive:   config.DestTeamDrive,
		CreatedBy:   config.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func getOrCreateMachine(tx *gorm.DB, fp machineFingerprint, record machineRecord, cache map[string]uint) (uint, error) {
	key := machineFingerprintKey(fp)
	if id, ok := cache[key]; ok {
		return id, nil
	}

	if err := tx.Create(&record).Error; err != nil {
		return 0, fmt.Errorf("failed to create machine for fingerprint %s: %w", key, err)
	}

	cache[key] = record.ID
	return record.ID, nil
}

func migrateTransferConfigsToMachines(tx *gorm.DB) error {
	var configs []transferConfigMigrationRow
	if err := tx.Table("transfer_configs").Find(&configs).Error; err != nil {
		return fmt.Errorf("failed to load transfer configs: %w", err)
	}

	machineCache := make(map[string]uint)

	for _, config := range configs {
		updates := map[string]interface{}{}

		if config.SourceType != "local" {
			fp := machineFingerprint{
				Type: config.SourceType,
				Host: config.SourceHost,
				Port: config.SourcePort,
				User: config.SourceUser,
			}
			machineID, err := getOrCreateMachine(tx, fp, machineFromSource(config), machineCache)
			if err != nil {
				return err
			}
			updates["source_machine_id"] = machineID
		}

		if config.DestinationType != "local" {
			fp := machineFingerprint{
				Type: config.DestinationType,
				Host: config.DestHost,
				Port: config.DestPort,
				User: config.DestUser,
			}
			machineID, err := getOrCreateMachine(tx, fp, machineFromDest(config), machineCache)
			if err != nil {
				return err
			}
			updates["dest_machine_id"] = machineID
		}

		if len(updates) > 0 {
			if err := tx.Table("transfer_configs").Where("id = ?", config.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("failed to update transfer config %d: %w", config.ID, err)
			}
		}
	}

	return nil
}

// AddMachines creates the machines table, links transfer configs to machines, and backfills existing configs.
func AddMachines() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "014_add_machines",
		Migrate: func(tx *gorm.DB) error {
			fmt.Println("Running migration 014: Adding machines table and machine references...")

			if err := tx.Exec(`CREATE TABLE IF NOT EXISTS machines (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name VARCHAR(255) NOT NULL,
				type VARCHAR(255) NOT NULL,
				host VARCHAR(255),
				port INTEGER DEFAULT 22,
				user VARCHAR(255),
				key_file TEXT,
				region VARCHAR(255),
				access_key VARCHAR(255),
				endpoint VARCHAR(255),
				bucket VARCHAR(255),
				domain VARCHAR(255),
				share VARCHAR(255),
				passive_mode BOOLEAN DEFAULT TRUE,
				client_id VARCHAR(255),
				drive_id VARCHAR(255),
				team_drive VARCHAR(255),
				created_by INTEGER NOT NULL,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY (created_by) REFERENCES users(id)
			)`).Error; err != nil {
				return fmt.Errorf("failed to create machines table: %w", err)
			}

			var sourceColExists int
			tx.Raw("SELECT count(*) FROM pragma_table_info('transfer_configs') WHERE name='source_machine_id'").Scan(&sourceColExists)
			if sourceColExists == 0 {
				if err := tx.Exec(`ALTER TABLE transfer_configs ADD COLUMN source_machine_id INTEGER REFERENCES machines(id)`).Error; err != nil {
					return fmt.Errorf("failed to add source_machine_id column: %w", err)
				}
			}

			var destColExists int
			tx.Raw("SELECT count(*) FROM pragma_table_info('transfer_configs') WHERE name='dest_machine_id'").Scan(&destColExists)
			if destColExists == 0 {
				if err := tx.Exec(`ALTER TABLE transfer_configs ADD COLUMN dest_machine_id INTEGER REFERENCES machines(id)`).Error; err != nil {
					return fmt.Errorf("failed to add dest_machine_id column: %w", err)
				}
			}

			if err := migrateTransferConfigsToMachines(tx); err != nil {
				return err
			}

			fmt.Println("Migration 014 completed successfully.")
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			fmt.Println("Rolling back migration 014: Removing machines...")

			var sourceColExists int
			tx.Raw("SELECT count(*) FROM pragma_table_info('transfer_configs') WHERE name='source_machine_id'").Scan(&sourceColExists)
			if sourceColExists > 0 {
				if err := tx.Exec(`ALTER TABLE transfer_configs DROP COLUMN source_machine_id`).Error; err != nil {
					return fmt.Errorf("failed to drop source_machine_id column: %w", err)
				}
			}

			var destColExists int
			tx.Raw("SELECT count(*) FROM pragma_table_info('transfer_configs') WHERE name='dest_machine_id'").Scan(&destColExists)
			if destColExists > 0 {
				if err := tx.Exec(`ALTER TABLE transfer_configs DROP COLUMN dest_machine_id`).Error; err != nil {
					return fmt.Errorf("failed to drop dest_machine_id column: %w", err)
				}
			}

			if err := tx.Exec(`DROP TABLE IF EXISTS machines`).Error; err != nil {
				return fmt.Errorf("failed to drop machines table: %w", err)
			}

			return nil
		},
	}
}

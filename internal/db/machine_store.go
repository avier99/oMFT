package db

import (
	"fmt"
	"os"
	"path/filepath"
)

// --- Machine Store Methods ---

// CreateMachine creates a new machine record.
func (db *DB) CreateMachine(machine *Machine) error {
	return db.Create(machine).Error
}

// GetMachine retrieves a single machine by ID.
func (db *DB) GetMachine(id uint) (*Machine, error) {
	var machine Machine
	err := db.First(&machine, id).Error
	if err != nil {
		return nil, err
	}
	return &machine, nil
}

// GetMachines retrieves all machines for a user.
func (db *DB) GetMachines(userID uint) ([]Machine, error) {
	var machines []Machine
	err := db.Where("created_by = ?", userID).Find(&machines).Error
	return machines, err
}

// UpdateMachine updates an existing machine record.
func (db *DB) UpdateMachine(machine *Machine) error {
	return db.Save(machine).Error
}

// DeleteMachine deletes a machine after verifying no transfer configs reference it.
func (db *DB) DeleteMachine(id uint) error {
	var count int64
	if err := db.Model(&TransferConfig{}).
		Where("source_machine_id = ? OR dest_machine_id = ?", id, id).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check for dependent configs: %v", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete machine: %d configs are using this machine", count)
	}

	return db.Delete(&Machine{}, id).Error
}

// GetMachineRclonePath returns the path to the rclone config file for a given machine.
func (db *DB) GetMachineRclonePath(machine *Machine) string {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	return filepath.Join(dataDir, "machines", fmt.Sprintf("machine_%d.conf", machine.ID))
}

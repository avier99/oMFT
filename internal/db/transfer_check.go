package db

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const defaultTransferCheckHistoryLimit = 50

// TransferCheck records a source/destination comparison run for a transfer config.
type TransferCheck struct {
	ID              uint           `gorm:"primarykey"`
	ConfigID        uint           `gorm:"not null"`
	Config          TransferConfig `gorm:"foreignkey:ConfigID"`
	Status          string         `gorm:"not null;default:pending"`
	StartedAt       *time.Time
	CompletedAt     *time.Time
	Differences     int `gorm:"not null;default:0"`
	MissingOnSource int `gorm:"not null;default:0"`
	MissingOnDest   int `gorm:"not null;default:0"`
	ErrorMessage    string
	LogPath         string
	CreatedBy       uint `gorm:"not null"`
	Creator         User `gorm:"foreignkey:CreatedBy"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateTransferCheck creates a new transfer check record.
func (db *DB) CreateTransferCheck(check *TransferCheck) error {
	return db.Create(check).Error
}

// UpdateTransferCheck updates only the fields the check owns, avoiding GORM's
// FullSaveAssociations behavior which would upsert the preloaded Config/Creator
// and could silently revert config edits made during a long-running check.
func (db *DB) UpdateTransferCheck(check *TransferCheck) error {
	return db.Model(check).Updates(map[string]interface{}{
		"status":            check.Status,
		"completed_at":      check.CompletedAt,
		"differences":       check.Differences,
		"missing_on_source": check.MissingOnSource,
		"missing_on_dest":   check.MissingOnDest,
		"error_message":     check.ErrorMessage,
		"log_path":          check.LogPath,
	}).Error
}

// GetTransferCheck retrieves a single transfer check by ID.
func (db *DB) GetTransferCheck(id uint) (*TransferCheck, error) {
	var check TransferCheck
	err := db.Preload("Config").Preload("Creator").First(&check, id).Error
	if err != nil {
		return nil, err
	}
	return &check, nil
}

// GetTransferChecksForConfig retrieves transfer checks for a config, newest first.
// limit caps how many rows are returned; non-positive values use defaultTransferCheckHistoryLimit.
func (db *DB) GetTransferChecksForConfig(configID uint, limit int) ([]TransferCheck, error) {
	if limit <= 0 {
		limit = defaultTransferCheckHistoryLimit
	}

	var checks []TransferCheck
	err := db.Preload("Config").
		Where("config_id = ?", configID).
		Order("created_at DESC").
		Limit(limit).
		Find(&checks).Error
	return checks, err
}

// GetLatestTransferCheck retrieves the newest transfer check for a config.
func (db *DB) GetLatestTransferCheck(configID uint) (*TransferCheck, error) {
	var check TransferCheck
	err := db.Preload("Config").
		Where("config_id = ?", configID).
		Order("created_at DESC").
		First(&check).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &check, nil
}

package db

import (
	"time"
)

// Machine holds reusable connection credentials for a remote storage endpoint.
// Secrets are form-only (gorm:"-"); obscured credentials live in machine_N.conf via rclone.
type Machine struct {
	ID   uint   `gorm:"primarykey"`
	Name string `gorm:"not null" form:"name"`
	Type string `gorm:"not null" form:"type"` // sftp, s3, wasabi, minio, b2, ftp, smb, webdav, nextcloud, gdrive, gphotos, local, hetzner

	// Non-secret (stored in DB)
	Host        string `form:"host"`
	Port        int    `gorm:"default:22" form:"port"`
	User        string `form:"user"`
	KeyFile     string `form:"key_file"`
	Region      string `form:"region"`
	AccessKey   string `form:"access_key"`
	Endpoint    string `form:"endpoint"`
	Bucket      string `form:"bucket"`
	Domain      string `form:"domain"`
	Share       string `form:"share"`
	PassiveMode *bool  `gorm:"default:true" form:"passive_mode"`
	ClientID    string `form:"client_id"`
	DriveID     string `form:"drive_id"`
	TeamDrive   string `form:"team_drive"`

	// Secret (form only, not stored in DB)
	Password     string `form:"password" gorm:"-"`
	SecretKey    string `form:"secret_key" gorm:"-"`
	ClientSecret string `form:"client_secret" gorm:"-"`

	CreatedBy uint
	Creator   User `gorm:"foreignkey:CreatedBy"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

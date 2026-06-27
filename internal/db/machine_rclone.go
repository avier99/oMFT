package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type rcloneRemoteConfig struct {
	Type         string
	Host         string
	Port         int
	User         string
	Password     string
	KeyFile      string
	Region       string
	AccessKey    string
	SecretKey    string
	Endpoint     string
	Domain       string
	PassiveMode  *bool
	ClientID     string
	ClientSecret string
	DriveID      string
	TeamDrive    string
}

func machineRemoteName(machine *Machine) string {
	return fmt.Sprintf("machine_%d", machine.ID)
}

func sourceRemoteName(config *TransferConfig) string {
	return fmt.Sprintf("source_%d", config.ID)
}

func destRemoteName(config *TransferConfig) string {
	return fmt.Sprintf("dest_%d", config.ID)
}

func boolValue(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func rclonePath() string {
	path := os.Getenv("RCLONE_PATH")
	if path == "" {
		return "rclone"
	}
	return path
}

func remoteConfigFromMachine(machine *Machine) rcloneRemoteConfig {
	return rcloneRemoteConfig{
		Type:         machine.Type,
		Host:         machine.Host,
		Port:         machine.Port,
		User:         machine.User,
		Password:     machine.Password,
		KeyFile:      machine.KeyFile,
		Region:       machine.Region,
		AccessKey:    machine.AccessKey,
		SecretKey:    machine.SecretKey,
		Endpoint:     machine.Endpoint,
		Domain:       machine.Domain,
		PassiveMode:  machine.PassiveMode,
		ClientID:     machine.ClientID,
		ClientSecret: machine.ClientSecret,
		DriveID:      machine.DriveID,
		TeamDrive:    machine.TeamDrive,
	}
}

func sourceRemoteConfig(config *TransferConfig) rcloneRemoteConfig {
	return rcloneRemoteConfig{
		Type:         config.SourceType,
		Host:         config.SourceHost,
		Port:         config.SourcePort,
		User:         config.SourceUser,
		Password:     config.SourcePassword,
		KeyFile:      config.SourceKeyFile,
		Region:       config.SourceRegion,
		AccessKey:    config.SourceAccessKey,
		SecretKey:    config.SourceSecretKey,
		Endpoint:     config.SourceEndpoint,
		Domain:       config.SourceDomain,
		PassiveMode:  config.SourcePassiveMode,
		ClientID:     config.SourceClientID,
		ClientSecret: config.SourceClientSecret,
		DriveID:      config.SourceDriveID,
		TeamDrive:    config.SourceTeamDrive,
	}
}

func destRemoteConfig(config *TransferConfig) rcloneRemoteConfig {
	return rcloneRemoteConfig{
		Type:         config.DestinationType,
		Host:         config.DestHost,
		Port:         config.DestPort,
		User:         config.DestUser,
		Password:     config.DestPassword,
		KeyFile:      config.DestKeyFile,
		Region:       config.DestRegion,
		AccessKey:    config.DestAccessKey,
		SecretKey:    config.DestSecretKey,
		Endpoint:     config.DestEndpoint,
		Domain:       config.DestDomain,
		PassiveMode:  config.DestPassiveMode,
		ClientID:     config.DestClientID,
		ClientSecret: config.DestClientSecret,
		DriveID:      config.DestDriveID,
		TeamDrive:    config.DestTeamDrive,
	}
}

func appendIfSet(args []string, key, value string) []string {
	if value == "" {
		return args
	}
	return append(args, key, value)
}

func primaryRcloneType(providerType string) string {
	switch providerType {
	case "hetzner":
		return "sftp"
	case "minio", "wasabi":
		return "s3"
	case "gphotos":
		return "google photos"
	default:
		return providerType
	}
}

func webdavRemoteURL(remote rcloneRemoteConfig) (string, error) {
	parsedURL, err := url.Parse(remote.Host)
	if err != nil {
		return "", fmt.Errorf("failed to parse %s URL %q: %w", remote.Type, remote.Host, err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid %s URL %q: must include scheme and host", remote.Type, remote.Host)
	}

	remoteURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
	if remote.Type == "nextcloud" {
		remoteURL = fmt.Sprintf("%s/remote.php/dav/files/%s/", remoteURL, remote.User)
	}

	return remoteURL, nil
}

func createRcloneRemote(configPath, remoteName string, remote rcloneRemoteConfig) error {
	if remote.Type == "local" {
		return appendRcloneSection(configPath, fmt.Sprintf("[%s]\ntype = local\nnounc = true\n", remoteName))
	}

	args := []string{
		"config", "create", remoteName, primaryRcloneType(remote.Type),
		"--non-interactive",
		"--config", configPath,
		"--log-level", "ERROR",
	}

	switch remote.Type {
	case "sftp", "hetzner":
		args = appendIfSet(args, "host", remote.Host)
		args = appendIfSet(args, "user", remote.User)
		if remote.Port != 0 {
			args = append(args, "port", strconv.Itoa(remote.Port))
		}
		args = appendIfSet(args, "pass", remote.Password)
		args = appendIfSet(args, "key_file", remote.KeyFile)
	case "s3":
		args = append(args, "provider", "AWS", "env_auth", "false")
		args = appendIfSet(args, "access_key_id", remote.AccessKey)
		args = appendIfSet(args, "secret_access_key", remote.SecretKey)
		args = appendIfSet(args, "region", remote.Region)
		args = appendIfSet(args, "endpoint", remote.Endpoint)
	case "wasabi":
		args = append(args, "provider", "Wasabi", "env_auth", "false")
		args = appendIfSet(args, "access_key_id", remote.AccessKey)
		args = appendIfSet(args, "secret_access_key", remote.SecretKey)
		args = appendIfSet(args, "region", remote.Region)
		endpoint := remote.Endpoint
		if endpoint == "" {
			endpoint = "s3.wasabisys.com"
		}
		args = append(args, "endpoint", endpoint)
	case "minio":
		args = append(args, "provider", "Minio", "env_auth", "false")
		args = appendIfSet(args, "access_key_id", remote.AccessKey)
		args = appendIfSet(args, "secret_access_key", remote.SecretKey)
		args = appendIfSet(args, "endpoint", remote.Endpoint)
		args = appendIfSet(args, "region", remote.Region)
	case "b2":
		args = appendIfSet(args, "account", remote.AccessKey)
		args = appendIfSet(args, "key", remote.SecretKey)
		args = appendIfSet(args, "endpoint", remote.Endpoint)
	case "ftp":
		args = appendIfSet(args, "host", remote.Host)
		args = appendIfSet(args, "user", remote.User)
		if remote.Port != 0 {
			args = append(args, "port", strconv.Itoa(remote.Port))
		}
		args = appendIfSet(args, "pass", remote.Password)
		args = append(args, "passive_mode", strconv.FormatBool(boolValue(remote.PassiveMode, true)))
		args = append(args, "explicit_tls", "true")
	case "smb":
		args = appendIfSet(args, "host", remote.Host)
		args = appendIfSet(args, "user", remote.User)
		if remote.Port != 0 {
			args = append(args, "port", strconv.Itoa(remote.Port))
		}
		args = appendIfSet(args, "pass", remote.Password)
		args = appendIfSet(args, "domain", remote.Domain)
	case "webdav", "nextcloud":
		vendor := "other"
		if remote.Type == "nextcloud" {
			vendor = "nextcloud"
		}
		remoteURL, err := webdavRemoteURL(remote)
		if err != nil {
			return err
		}
		args = append(args, "url", remoteURL)
		args = append(args, "vendor", vendor)
		args = appendIfSet(args, "user", remote.User)
		args = appendIfSet(args, "pass", remote.Password)
	case "gdrive":
		args = append(args, "scope", "drive")
		args = appendIfSet(args, "client_id", remote.ClientID)
		args = appendIfSet(args, "client_secret", remote.ClientSecret)
		args = appendIfSet(args, "root_folder_id", remote.DriveID)
		args = appendIfSet(args, "team_drive", remote.TeamDrive)
	case "gphotos":
		args = appendIfSet(args, "client_id", remote.ClientID)
		args = appendIfSet(args, "client_secret", remote.ClientSecret)
	default:
		return fmt.Errorf("unsupported machine type for rclone config generation: %s", remote.Type)
	}

	cmd := exec.Command(rclonePath(), args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create rclone remote %s (%s): %v\nOutput: %s", remoteName, remote.Type, err, output)
	}

	return nil
}

func appendRcloneSection(configPath, section string) error {
	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open rclone config for appending: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(section); err != nil {
		return fmt.Errorf("failed to append rclone config section: %w", err)
	}
	if !strings.HasSuffix(section, "\n") {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to append rclone config newline: %w", err)
		}
	}
	return nil
}

func extractRcloneSection(content, sectionName string) (string, bool) {
	header := "[" + sectionName + "]"
	lines := strings.Split(content, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == header {
			start = i
			break
		}
	}
	if start == -1 {
		return "", false
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			end = i
			break
		}
	}

	section := strings.Join(lines[start:end], "\n")
	if !strings.HasSuffix(section, "\n") {
		section += "\n"
	}
	return section, true
}

func renameRcloneSection(section, oldName, newName string) string {
	return strings.Replace(section, "["+oldName+"]", "["+newName+"]", 1)
}

func writeRcloneSection(configPath, section string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create rclone config directory: %w", err)
	}
	return os.WriteFile(configPath, []byte(section), 0600)
}

func copyRcloneSection(sourcePath, sourceName, destPath, destName string) error {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read rclone config %s: %w", sourcePath, err)
	}

	section, ok := extractRcloneSection(string(content), sourceName)
	if !ok {
		return fmt.Errorf("rclone config %s does not contain section %s", sourcePath, sourceName)
	}

	return appendRcloneSection(destPath, renameRcloneSection(section, sourceName, destName))
}

// GenerateMachineRcloneConfig creates or replaces the rclone config file for a machine.
func (db *DB) GenerateMachineRcloneConfig(machine *Machine) error {
	configPath := db.GetMachineRclonePath(machine)
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create machines config directory: %w", err)
	}
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to replace machine rclone config: %w", err)
	}

	return createRcloneRemote(configPath, machineRemoteName(machine), remoteConfigFromMachine(machine))
}

func (db *DB) ensureMachineConfigFromExistingRemote(machine *Machine, existingConfigContent, existingRemoteName string) error {
	machinePath := db.GetMachineRclonePath(machine)
	if content, err := os.ReadFile(machinePath); err == nil {
		if _, ok := extractRcloneSection(string(content), machineRemoteName(machine)); ok {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read machine rclone config: %w", err)
	}

	if section, ok := extractRcloneSection(existingConfigContent, existingRemoteName); ok {
		return writeRcloneSection(machinePath, renameRcloneSection(section, existingRemoteName, machineRemoteName(machine)))
	}

	return db.GenerateMachineRcloneConfig(machine)
}

func (db *DB) appendMachineRemoteToConfig(configPath string, machine *Machine, remoteName string) error {
	machinePath := db.GetMachineRclonePath(machine)
	return copyRcloneSection(machinePath, machineRemoteName(machine), configPath, remoteName)
}

// TestMachineConnection runs rclone lsd against a temporary config built from machine fields.
// testPath is optional; empty string tests the root listing.
func (db *DB) TestMachineConnection(machine *Machine, testPath string) (bool, string, error) {
	tempDir, err := os.MkdirTemp("", "omft-machine-test-")
	if err != nil {
		return false, "Failed to create temp directory", err
	}
	defer os.RemoveAll(tempDir)

	tempConfig := filepath.Join(tempDir, "machine_test.conf")
	if err := createRcloneRemote(tempConfig, "test", remoteConfigFromMachine(machine)); err != nil {
		return false, fmt.Sprintf("Failed to build rclone config: %v", err), err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{
		"--config", tempConfig,
		"lsd",
		"test:" + testPath,
		"--low-level-retries", "1",
		"--retries", "1",
	}

	var stderr strings.Builder
	cmd := exec.CommandContext(ctx, rclonePath(), args...)
	cmd.Stderr = &stderr

	err = cmd.Run()
	stderrStr := stderr.String()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return false, "Connection test timed out after 30 seconds.", ctx.Err()
		}
		msg := fmt.Sprintf("Connection failed: %v", err)
		switch {
		case strings.Contains(stderrStr, "connect: connection refused"):
			msg = "Connection refused by host."
		case strings.Contains(stderrStr, "no such host"), strings.Contains(stderrStr, "name resolution error"):
			msg = "Hostname not found or DNS resolution error."
		case strings.Contains(stderrStr, "authentication failed"), strings.Contains(stderrStr, "login incorrect"), strings.Contains(stderrStr, "permission denied"):
			msg = "Authentication failed — check credentials."
		case strings.Contains(stderrStr, "directory not found"):
			msg = "Path not found."
		}
		return false, msg, err
	}

	return true, "Connection test successful!", nil
}

func (db *DB) generateRcloneConfigWithMachines(config *TransferConfig, configPath, existingConfigContent string) error {
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to replace transfer rclone config: %w", err)
	}

	if config.SourceMachineID != nil {
		machine, err := db.GetMachine(*config.SourceMachineID)
		if err != nil {
			return fmt.Errorf("failed to load source machine %d: %w", *config.SourceMachineID, err)
		}
		if err := db.ensureMachineConfigFromExistingRemote(machine, existingConfigContent, sourceRemoteName(config)); err != nil {
			return fmt.Errorf("failed to prepare source machine config: %w", err)
		}
		if err := db.appendMachineRemoteToConfig(configPath, machine, sourceRemoteName(config)); err != nil {
			return fmt.Errorf("failed to add source machine remote: %w", err)
		}
	} else if err := createRcloneRemote(configPath, sourceRemoteName(config), sourceRemoteConfig(config)); err != nil {
		return fmt.Errorf("failed to create source config: %w", err)
	}

	if config.DestMachineID != nil {
		machine, err := db.GetMachine(*config.DestMachineID)
		if err != nil {
			return fmt.Errorf("failed to load destination machine %d: %w", *config.DestMachineID, err)
		}
		if err := db.ensureMachineConfigFromExistingRemote(machine, existingConfigContent, destRemoteName(config)); err != nil {
			return fmt.Errorf("failed to prepare destination machine config: %w", err)
		}
		if err := db.appendMachineRemoteToConfig(configPath, machine, destRemoteName(config)); err != nil {
			return fmt.Errorf("failed to add destination machine remote: %w", err)
		}
	} else if err := createRcloneRemote(configPath, destRemoteName(config), destRemoteConfig(config)); err != nil {
		return fmt.Errorf("failed to create destination config: %w", err)
	}

	return nil
}

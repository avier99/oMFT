package rclone_service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/avier99/oMFT/internal/db"
)

// CheckResult holds parsed output from an rclone check run.
type CheckResult struct {
	Differences     int
	MissingOnSource int
	MissingOnDest   int
	ErrorMessage    string
}

var (
	reMissingOnRemote  = regexp.MustCompile(`Missing on remote:\s+(\d+)`)
	reMissingOnLocal  = regexp.MustCompile(`Missing on local:\s+(\d+)`)
	reDifferencesFound = regexp.MustCompile(`(\d+) differences found`)
)

func checkSourcePath(config *db.TransferConfig) string {
	if config.SourceBucket != "" {
		return fmt.Sprintf("source_%d:%s/%s", config.ID, config.SourceBucket, config.SourcePath)
	}
	return fmt.Sprintf("source_%d:%s", config.ID, config.SourcePath)
}

func checkDestPath(config *db.TransferConfig) string {
	if config.DestBucket != "" {
		return fmt.Sprintf("dest_%d:%s/%s", config.ID, config.DestBucket, config.DestinationPath)
	}
	return fmt.Sprintf("dest_%d:%s", config.ID, config.DestinationPath)
}

func parseCheckOutput(output string) (CheckResult, bool) {
	var result CheckResult
	parsed := false

	if m := reMissingOnRemote.FindStringSubmatch(output); len(m) > 1 {
		result.MissingOnDest, _ = strconv.Atoi(m[1])
		parsed = true
	}
	if m := reMissingOnLocal.FindStringSubmatch(output); len(m) > 1 {
		result.MissingOnSource, _ = strconv.Atoi(m[1])
		parsed = true
	}
	if m := reDifferencesFound.FindStringSubmatch(output); len(m) > 1 {
		result.Differences, _ = strconv.Atoi(m[1])
		parsed = true
	}

	return result, parsed
}

// RunTransferCheck compares source and destination paths for a transfer config using rclone check.
// When logPath is non-empty, rclone output is written to that file for live polling.
func RunTransferCheck(config *db.TransferConfig, configPath string, logPath string) (CheckResult, error) {
	rclonePath := os.Getenv("RCLONE_PATH")
	if rclonePath == "" {
		rclonePath = "rclone"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	args := []string{
		"check",
		checkSourcePath(config),
		checkDestPath(config),
		"--config", configPath,
		"--log-level", "NOTICE",
	}

	if logPath != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
			return CheckResult{}, fmt.Errorf("failed to create log directory: %w", err)
		}
		args = append(args, "--log-file", logPath)
	}

	cmd := execCommandContext(ctx, rclonePath, args...)

	var outputStr string
	var err error
	if logPath != "" {
		err = cmdRun(cmd)
		if output, readErr := os.ReadFile(logPath); readErr == nil {
			outputStr = string(output)
		}
	} else {
		var output []byte
		output, err = cmdCombinedOutput(cmd)
		outputStr = string(output)
	}

	result, parsed := parseCheckOutput(outputStr)
	if parsed {
		return result, nil
	}

	if err != nil {
		result.ErrorMessage = outputStr
		if result.ErrorMessage == "" {
			result.ErrorMessage = err.Error()
		}
		return result, err
	}

	return result, nil
}

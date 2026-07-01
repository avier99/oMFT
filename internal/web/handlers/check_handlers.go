package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/avier99/oMFT/components"
	"github.com/avier99/oMFT/internal/db"
	"github.com/avier99/oMFT/internal/rclone_service"
)

func (h *Handlers) userCanAccessConfig(c *gin.Context, config *db.TransferConfig) bool {
	userID := c.GetUint("userID")
	if config.CreatedBy == userID {
		return true
	}
	isAdmin, exists := c.Get("isAdmin")
	return exists && isAdmin == true
}

func (h *Handlers) userCanAccessCheck(c *gin.Context, check *db.TransferCheck) bool {
	return h.userCanAccessConfig(c, &check.Config)
}

// HandleRunCheck handles POST /configs/:id/check.
func (h *Handlers) HandleRunCheck(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/configs?error=Invalid+config+ID")
		return
	}

	userID := c.GetUint("userID")

	var config db.TransferConfig
	if err := h.DB.First(&config, id).Error; err != nil {
		c.Redirect(http.StatusFound, "/configs?error=Config+not+found")
		return
	}

	if !h.userCanAccessConfig(c, &config) {
		c.Redirect(http.StatusFound, "/configs?error=Permission+denied")
		return
	}

	if err := h.DB.GenerateRcloneConfig(&config); err != nil {
		c.Redirect(http.StatusFound, "/configs?error="+url.QueryEscape("Failed to generate rclone config")+"&details="+url.QueryEscape(err.Error()))
		return
	}

	configPath := h.DB.GetConfigRclonePath(&config)

	now := time.Now()
	check := db.TransferCheck{
		ConfigID:  config.ID,
		Status:    "running",
		StartedAt: &now,
		CreatedBy: userID,
	}
	if err := h.DB.CreateTransferCheck(&check); err != nil {
		log.Printf("HandleRunCheck: failed to create transfer check: %v", err)
		c.Redirect(http.StatusFound, "/configs?error="+url.QueryEscape("Failed to start check"))
		return
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	logPath := filepath.Join(dataDir, "checks", fmt.Sprintf("check_%d.log", check.ID))
	check.LogPath = logPath
	if err := h.DB.UpdateTransferCheck(&check); err != nil {
		log.Printf("HandleRunCheck: failed to save log path for check %d: %v", check.ID, err)
	}

	go func(checkID uint, cfg db.TransferConfig, cfgPath, logPath string) {
		result, runErr := rclone_service.RunTransferCheck(&cfg, cfgPath, logPath)

		checkRecord, dbErr := h.DB.GetTransferCheck(checkID)
		if dbErr != nil {
			log.Printf("HandleRunCheck: failed to load transfer check %d: %v", checkID, dbErr)
			return
		}

		completedAt := time.Now()
		checkRecord.CompletedAt = &completedAt

		if runErr != nil {
			checkRecord.Status = "failed"
			checkRecord.ErrorMessage = result.ErrorMessage
			if checkRecord.ErrorMessage == "" {
				checkRecord.ErrorMessage = runErr.Error()
			}
		} else {
			checkRecord.Status = "completed"
			checkRecord.Differences = result.Differences
			checkRecord.MissingOnSource = result.MissingOnSource
			checkRecord.MissingOnDest = result.MissingOnDest
		}

		if updateErr := h.DB.UpdateTransferCheck(checkRecord); updateErr != nil {
			log.Printf("HandleRunCheck: failed to update transfer check %d: %v", checkID, updateErr)
		}
	}(check.ID, config, configPath, logPath)

	if c.GetHeader("HX-Request") != "" {
		c.Header("HX-Redirect", fmt.Sprintf("/checks/%d", check.ID))
		c.Status(http.StatusOK)
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("/checks/%d", check.ID))
}

// HandleCheckResult handles GET /checks/:id.
func (h *Handlers) HandleCheckResult(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid check ID")
		return
	}

	check, err := h.DB.GetTransferCheck(uint(id))
	if err != nil {
		c.String(http.StatusNotFound, "Check not found")
		return
	}

	if !h.userCanAccessCheck(c, check) {
		c.String(http.StatusForbidden, "You do not have permission to view this check")
		return
	}

	checks, _ := h.DB.GetTransferChecksForConfig(check.ConfigID, 10)

	var logContent string
	if check.LogPath != "" {
		if b, err := os.ReadFile(check.LogPath); err == nil {
			logContent = string(b)
		}
	}

	ctx := components.CreateTemplateContext(c)
	data := components.CheckResultData{
		Check:      *check,
		Config:     check.Config,
		Checks:     checks,
		LogContent: logContent,
	}
	_ = components.CheckResult(ctx, data).Render(ctx, c.Writer)
}

// HandleCheckStatus handles GET /checks/:id/status.
func (h *Handlers) HandleCheckStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid check ID")
		return
	}

	check, err := h.DB.GetTransferCheck(uint(id))
	if err != nil {
		c.String(http.StatusNotFound, "Check not found")
		return
	}

	if !h.userCanAccessCheck(c, check) {
		c.String(http.StatusForbidden, "You do not have permission to view this check")
		return
	}

	c.Header("Content-Type", "text/html")
	_ = components.CheckStatusPartial(*check).Render(c.Request.Context(), c.Writer)
}

// HandleConfigCheckBadge handles GET /configs/:id/check-badge.
func (h *Handlers) HandleConfigCheckBadge(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid config ID")
		return
	}

	var config db.TransferConfig
	if err := h.DB.First(&config, id).Error; err != nil {
		c.String(http.StatusNotFound, "Config not found")
		return
	}

	if !h.userCanAccessConfig(c, &config) {
		c.String(http.StatusForbidden, "Permission denied")
		return
	}

	latestCheck, _ := h.DB.GetLatestTransferCheck(uint(id))

	c.Header("Content-Type", "text/html")
	_ = components.CheckLatestBadge(latestCheck, uint(id)).Render(c.Request.Context(), c.Writer)
}

// HandleCheckLog handles GET /checks/:id/log.
func (h *Handlers) HandleCheckLog(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid check ID")
		return
	}

	check, err := h.DB.GetTransferCheck(uint(id))
	if err != nil {
		c.String(http.StatusNotFound, "Check not found")
		return
	}

	if !h.userCanAccessCheck(c, check) {
		c.String(http.StatusForbidden, "You do not have permission to view this check")
		return
	}

	var content string
	if check.LogPath != "" {
		if b, err := os.ReadFile(check.LogPath); err == nil {
			content = string(b)
		}
	}

	c.Header("Content-Type", "text/html")
	_ = components.CheckLogPartial(content, *check).Render(c.Request.Context(), c.Writer)
}

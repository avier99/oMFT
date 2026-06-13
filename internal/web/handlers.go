package web

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/avier99/oMFT/internal/config"
	"github.com/avier99/oMFT/internal/db"
	"github.com/avier99/oMFT/internal/email"
	"github.com/avier99/oMFT/internal/scheduler"
	"github.com/avier99/oMFT/internal/web/handlers"
)

// Handler is a wrapper around the handlers package
type Handler struct {
	handlers *handlers.Handlers
}

// Global handlers instance for access from other packages
var globalHandlersInstance *handlers.Handlers

// GetHandlersInstance returns the global handlers instance and a boolean indicating if it's initialized
func GetHandlersInstance() (*handlers.Handlers, bool) {
	return globalHandlersInstance, globalHandlersInstance != nil
}

// NewHandler creates a new Handler instance that delegates to the handlers package
func NewHandler(database *db.DB, scheduler *scheduler.Scheduler, jwtSecret string, dbPath string, backupDir string, cfg *config.Config) (*Handler, error) {
	// Create email service instance
	emailService := email.NewService(cfg)

	// Use logs directory from config
	logsDir := filepath.Join(cfg.DataDir, "logs")

	// Create handlers instance
	handlersInstance := handlers.NewHandlers(database, scheduler, jwtSecret, dbPath, backupDir, logsDir, emailService)

	// Store the handlers instance globally
	globalHandlersInstance = handlersInstance

	return &Handler{
		handlers: handlersInstance,
	}, nil
}

// InitializeRoutes delegates route registration to the handlers package
func (h *Handler) InitializeRoutes(router *gin.Engine) {
	// Register all routes through the handlers package
	h.handlers.RegisterRoutes(router)
}

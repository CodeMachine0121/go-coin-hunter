package main

import (
	"net/http"

	"github.com/CodeMachine0121/go-coin-hunter/internal/config"
	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// registerRoutes is the composition root: the only place that knows every concrete type.
func registerRoutes(engine *gin.Engine, database *gorm.DB, applicationConfig config.ApplicationConfig) {
	engine.GET("/health", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{"status": "Healthy"})
	})
}

// backgroundJobsFor returns no jobs when the global switch is off, which is how StartAll is disabled.
func backgroundJobsFor(database *gorm.DB, applicationConfig config.ApplicationConfig) []domaininterface.IBackgroundJob {
	if !applicationConfig.BackgroundJobsEnabled {
		return nil
	}

	return []domaininterface.IBackgroundJob{}
}

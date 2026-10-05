package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/config"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/persistence"
	"github.com/CodeMachine0121/go-coin-hunter/internal/job"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const shutdownGracePeriod = 30 * time.Second

func main() {
	if loadError := godotenv.Load(); loadError != nil {
		log.Println("no .env file loaded, falling back to process environment")
	}

	applicationConfig := config.Load()

	database, databaseError := persistence.NewDatabase(applicationConfig.SqliteDatabasePath)
	if databaseError != nil {
		log.Fatalf("failed to initialize database: %v", databaseError)
	}
	if migrateError := persistence.NewSchemaMigrator(database).Migrate(); migrateError != nil {
		log.Fatalf("failed to migrate schema: %v", migrateError)
	}

	builtApplications := applicationsFor(database, applicationConfig)
	// Nothing can be running yet, so any run still marked running was cut short by the last shutdown.
	interruptedCount, sweepError := builtApplications.pipelineRun.FailInterruptedPipelineRuns(context.Background())
	if sweepError != nil {
		log.Printf("failed to clear pipeline runs interrupted by restart: %v", sweepError)
	} else if interruptedCount > 0 {
		log.Printf("marked %d pipeline run(s) interrupted by restart as failed", interruptedCount)
	}

	shutdownSignalled, stopListeningForSignals := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopListeningForSignals()

	engine := gin.Default()
	registerRoutes(engine, builtApplications)

	backgroundJobManager := job.NewBackgroundJobManager(backgroundJobsFor(applicationConfig, builtApplications))
	backgroundJobManager.StartAll(shutdownSignalled)

	server := &http.Server{Addr: applicationConfig.ServerAddress, Handler: engine}
	go func() {
		if serveError := server.ListenAndServe(); serveError != nil && !errors.Is(serveError, http.ErrServerClosed) {
			log.Fatalf("failed to serve: %v", serveError)
		}
	}()

	<-shutdownSignalled.Done()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer cancelShutdown()

	backgroundJobManager.StopAll()
	if shutdownError := server.Shutdown(shutdownContext); shutdownError != nil {
		log.Printf("failed to shut down http server cleanly: %v", shutdownError)
	}
	if !backgroundJobManager.WaitAll(shutdownContext) {
		log.Println("background jobs did not finish within the grace period")
	}
}

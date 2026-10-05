package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/CodeMachine0121/go-coin-hunter/internal/config"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/persistence"
	"github.com/CodeMachine0121/go-coin-hunter/internal/job"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if loadError := godotenv.Load(); loadError != nil {
		log.Println("no .env file loaded, falling back to process environment")
	}

	applicationConfig := config.Load()

	database, databaseError := persistence.NewDatabase(applicationConfig.Database.DataSourceName())
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
	registerRoutes(engine, builtApplications, shutdownSignalled.Done())

	// Jobs get their own context: on shutdown they are first asked to stop at a step boundary, and only abandoned if the
	// grace period runs out.
	jobsContext, abandonJobs := context.WithCancel(context.Background())
	defer abandonJobs()
	backgroundJobManager := job.NewBackgroundJobManager(backgroundJobsFor(applicationConfig, builtApplications))
	backgroundJobManager.StartAll(jobsContext)

	server := &http.Server{Addr: applicationConfig.ServerAddress, Handler: engine}
	go func() {
		if serveError := server.ListenAndServe(); serveError != nil && !errors.Is(serveError, http.ErrServerClosed) {
			log.Fatalf("failed to serve: %v", serveError)
		}
	}()

	<-shutdownSignalled.Done()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), applicationConfig.ShutdownGracePeriod)
	defer cancelShutdown()

	backgroundJobManager.StopAll()
	if shutdownError := server.Shutdown(shutdownContext); shutdownError != nil {
		log.Printf("failed to shut down http server cleanly: %v", shutdownError)
	}
	if !backgroundJobManager.WaitAll(shutdownContext) {
		log.Println("background jobs did not finish within the grace period; abandoning them")
		abandonJobs()
	}
}

package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/job"
)

// shutDown stops jobs from starting further steps, lets requests and the steps in flight finish within the grace
// period, and abandons the jobs only if the grace period runs out; it reports whether they had to be abandoned.
func shutDown(server *http.Server, backgroundJobManager *job.BackgroundJobManager, gracePeriod time.Duration, abandonJobs func()) bool {
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), gracePeriod)
	defer cancelShutdown()

	backgroundJobManager.StopAll()
	if shutdownError := server.Shutdown(shutdownContext); shutdownError != nil {
		log.Printf("failed to shut down http server cleanly: %v", shutdownError)
	}
	if backgroundJobManager.WaitAll(shutdownContext) {
		return false
	}
	log.Println("background jobs did not finish within the grace period; abandoning them")
	abandonJobs()

	return true
}

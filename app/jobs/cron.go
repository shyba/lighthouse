package jobs

import (
	"github.com/jasonlvhit/gocron"
	"github.com/sirupsen/logrus"
)

var cronRunning chan bool
var scheduler *gocron.Scheduler

// Start starts the jobs that run in the background after initialization
func Start() {
	scheduler = gocron.NewScheduler()
	var channels *string
	scheduler.Every(1).Minutes().Do(SyncClaims, channels)

	cronRunning = scheduler.Start()
}

// Shutdown is used to shutdown the background jobs.
func Shutdown() {
	logrus.Debug("Shutting down cron jobs...")
	scheduler.Clear()
	close(cronRunning)
}

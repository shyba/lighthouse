package jobs

import (
	"github.com/jasonlvhit/gocron"
	"github.com/sirupsen/logrus"
)

var cronRunning chan bool
var scheduler *gocron.Scheduler

func Start() {
	scheduler = gocron.NewScheduler()

	scheduler.Every(1).Minutes().Do(SyncClaims)

	cronRunning = scheduler.Start()
}

func Shutdown() {
	logrus.Debug("Shutting down cron jobs...")
	scheduler.Clear()
	close(cronRunning)
}

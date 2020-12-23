package internalapis

import (
	"time"

	"github.com/lbryio/lighthouse/app/internal/metrics"
)

// APIURL is the url for internal-apis to be used by lighthouse
var APIURL string

// APIToken is the token allowed to access the api used for internal-apis
var APIToken string

var incSyncRunning bool

const batchSize = 1000

// Sync synchronizes view and subscription counts from internal-apis
func Sync() {
	if incSyncRunning {
		return
	}
	metrics.JobLoad.WithLabelValues("internalapis_sync").Inc()
	defer metrics.JobLoad.WithLabelValues("internalapis_sync").Dec()
	defer metrics.Job(time.Now(), "internalapis_sync")
	incSyncRunning = true
	defer endIncSync()
	syncSubCounts()
	syncViewCounts()

}

func endIncSync() {
	incSyncRunning = false
}

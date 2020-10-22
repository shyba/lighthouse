package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ServerErrors metric to capture errors from api
	ServerErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "lighthouse",
		Subsystem: "search",
		Name:      "errors",
		Help:      "The error count per api",
	})

	// SearchDuration metric to capture the duration of each search request
	SearchDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "lighthouse",
		Subsystem: "search",
		Name:      "duration",
		Help:      "The duration for search by type and term count",
	}, []string{"type", "term_count"})

	// AutoCompleteDuration metric to capture the duration of each auto complete request
	AutoCompleteDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lighthouse",
		Subsystem: "auto_complete",
		Name:      "duration",
		Help:      "The duration for auto_complete by type and term count",
	})

	jobs = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "lighthouse",
		Subsystem: "jobs",
		Name:      "duration",
		Help:      "The durations of the individual job processing",
	}, []string{"job"})

	// JobLoad metric for number of active calls by job
	JobLoad = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "lighthouse",
		Subsystem: "jobs",
		Name:      "job_load",
		Help:      "Number of active calls by job",
	}, []string{"job"})
)

//Job helper function to make tracking metric one line deferral
func Job(start time.Time, name string) {
	duration := time.Since(start).Seconds()
	jobs.WithLabelValues(name).Observe(duration)
}

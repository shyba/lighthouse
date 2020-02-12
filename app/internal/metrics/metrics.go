package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ServerErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "lighthouse",
		Subsystem: "search",
		Name:      "errors",
		Help:      "The error count per api",
	})

	SearchDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "lighthouse",
		Subsystem: "search",
		Name:      "duration",
		Help:      "The duration for search by type and term count",
	}, []string{"type", "term_count"})

	AutoCompleteDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lighthouse",
		Subsystem: "auto_complete",
		Name:      "duration",
		Help:      "The duration for auto_complete by type and term count",
	})
)

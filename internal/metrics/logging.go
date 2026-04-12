package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Logging metrics.
var (
	// LogMessagesTotal counts log messages by level.
	LogMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "scuffinger",
			Subsystem: "log",
			Name:      "messages_total",
			Help:      "Total number of log messages by level.",
		},
		[]string{"level"},
	)
)

// IncLogMessage increments the log message counter for the given level.
func IncLogMessage(level string) {
	LogMessagesTotal.WithLabelValues(level).Inc()
}

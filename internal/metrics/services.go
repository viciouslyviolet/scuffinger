package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Service lifecycle metrics.
var (
	// ServiceHealthChecksTotal counts the number of health-check pings per service.
	ServiceHealthChecksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "scuffinger",
			Subsystem: "service",
			Name:      "health_checks_total",
			Help:      "Total number of health-check pings per service.",
		},
		[]string{"service"},
	)

	// ServiceHealthCheckFailuresTotal counts the number of failed health-check pings per service.
	ServiceHealthCheckFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "scuffinger",
			Subsystem: "service",
			Name:      "health_check_failures_total",
			Help:      "Total number of failed health-check pings per service.",
		},
		[]string{"service"},
	)
)

// IncHealthCheck records a health-check ping for the named service.
func IncHealthCheck(service string) {
	ServiceHealthChecksTotal.WithLabelValues(service).Inc()
}

// IncHealthCheckFailure records a failed health-check ping for the named service.
func IncHealthCheckFailure(service string) {
	ServiceHealthCheckFailuresTotal.WithLabelValues(service).Inc()
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Health-related metrics.
var (
	// HealthStatus reports the overall application health (1 = healthy, 0 = unhealthy).
	HealthStatus = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "health",
			Name:      "status",
			Help:      "Overall application health status (1 = healthy, 0 = unhealthy).",
		},
	)

	// HealthServiceStatus reports per-service health (1 = healthy, 0 = unhealthy).
	HealthServiceStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "health",
			Name:      "service_status",
			Help:      "Per-service health status (1 = healthy, 0 = unhealthy).",
		},
		[]string{"service"},
	)
)

// SetOverallHealth sets the overall health gauge.
func SetOverallHealth(healthy bool) {
	if healthy {
		HealthStatus.Set(1)
	} else {
		HealthStatus.Set(0)
	}
}

// SetServiceHealth sets the health gauge for a specific service.
func SetServiceHealth(name string, healthy bool) {
	if healthy {
		HealthServiceStatus.WithLabelValues(name).Set(1)
	} else {
		HealthServiceStatus.WithLabelValues(name).Set(0)
	}
}

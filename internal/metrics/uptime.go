package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// UptimeSeconds reports how long the application has been running.
var UptimeSeconds = promauto.NewGaugeFunc(
	prometheus.GaugeOpts{
		Namespace: "scuffinger",
		Name:      "uptime_seconds",
		Help:      "Number of seconds since the application started.",
	},
	func() float64 { return time.Since(startTime).Seconds() },
)

var startTime = time.Now()

// RecordStartTime sets the reference point for uptime.
// Call this once at the very beginning of the serve command.
func RecordStartTime() {
	startTime = time.Now()
}

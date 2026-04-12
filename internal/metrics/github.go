package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// GitHub API metrics.
var (
	// GitHubAPICallsTotal counts GitHub API calls by endpoint.
	GitHubAPICallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "api_calls_total",
			Help:      "Total number of GitHub API calls.",
		},
		[]string{"endpoint"},
	)

	// GitHubAPIErrorsTotal counts GitHub API errors by endpoint.
	GitHubAPIErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "api_errors_total",
			Help:      "Total number of GitHub API errors.",
		},
		[]string{"endpoint"},
	)

	// GitHubAPIDuration observes GitHub API call latency by endpoint.
	GitHubAPIDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "api_duration_seconds",
			Help:      "GitHub API call duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// GitHubRateLimitRemaining reports the last-seen GitHub core rate-limit remaining count per credential.
	GitHubRateLimitRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "scuffinger",
			Subsystem: "github",
			Name:      "rate_limit_remaining",
			Help:      "GitHub API core rate limit remaining per credential.",
		},
		[]string{"credential"},
	)
)

// ObserveGitHubCall records a GitHub API call's duration and success/failure.
func ObserveGitHubCall(endpoint string, duration time.Duration, err error) {
	GitHubAPICallsTotal.WithLabelValues(endpoint).Inc()
	GitHubAPIDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
	if err != nil {
		GitHubAPIErrorsTotal.WithLabelValues(endpoint).Inc()
	}
}

// SetGitHubRateLimit sets the current rate-limit remaining gauge for a credential.
func SetGitHubRateLimit(credential string, remaining int) {
	GitHubRateLimitRemaining.WithLabelValues(credential).Set(float64(remaining))
}

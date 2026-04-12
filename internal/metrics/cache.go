package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Cache hit/miss metrics for the two-tier GitHub data cache.
var (
	// CacheHitsTotal counts cache hits by tier (valkey, postgres) and resource type.
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "scuffinger",
			Subsystem: "cache",
			Name:      "hits_total",
			Help:      "Total cache hits by tier and resource.",
		},
		[]string{"tier", "resource"},
	)

	// CacheMissesTotal counts cache misses by tier and resource type.
	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "scuffinger",
			Subsystem: "cache",
			Name:      "misses_total",
			Help:      "Total cache misses by tier and resource.",
		},
		[]string{"tier", "resource"},
	)
)

// RecordCacheHit increments the cache hit counter.
func RecordCacheHit(tier, resource string) {
	CacheHitsTotal.WithLabelValues(tier, resource).Inc()
}

// RecordCacheMiss increments the cache miss counter.
func RecordCacheMiss(tier, resource string) {
	CacheMissesTotal.WithLabelValues(tier, resource).Inc()
}

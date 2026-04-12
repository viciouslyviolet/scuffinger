// Package metrics defines all Prometheus metrics for the scuffinger application.
// Each metric domain lives in its own file for modularity.
// All collectors use promauto, which registers them with the default
// prometheus registry — the same one served by promhttp.Handler().
package metrics

package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"scuffinger/internal/metrics"
)

// HealthChecker is satisfied by any type that can report aggregate service health.
// services.Manager implements this implicitly.
type HealthChecker interface {
	IsHealthy() bool
	Statuses() map[string]bool
}

// LiveHandler responds with 200 OK — the process is alive.
func LiveHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// ReadyHandler returns 200 when all services are healthy, 503 otherwise.
func ReadyHandler(hc HealthChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hc != nil && hc.IsHealthy() {
			metrics.SetOverallHealth(true)
			for name, healthy := range hc.Statuses() {
				metrics.SetServiceHealth(name, healthy)
			}
			c.JSON(http.StatusOK, gin.H{
				"status":   "ok",
				"services": hc.Statuses(),
			})
			return
		}
		metrics.SetOverallHealth(false)
		for name, healthy := range safeStatuses(hc) {
			metrics.SetServiceHealth(name, healthy)
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unavailable",
			"services": safeStatuses(hc),
		})
	}
}

func safeStatuses(hc HealthChecker) map[string]bool {
	if hc == nil {
		return map[string]bool{}
	}
	return hc.Statuses()
}

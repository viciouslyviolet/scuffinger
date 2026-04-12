package server

import (
	"scuffinger/internal/config"
	"scuffinger/internal/metrics"

	"github.com/gin-gonic/gin"
)

// RouteRegistrar can register additional routes on the Gin engine.
// Implement this interface to add new API groups without changing NewRouter's signature.
type RouteRegistrar interface {
	RegisterRoutes(r *gin.Engine)
}

// NewRouter creates and configures a new Gin engine with all routes registered.
// hc may be nil (health/ready will return 503 until services are wired).
// Additional route groups are added via the variadic registrars.
func NewRouter(cfg *config.Config, hc HealthChecker, registrars ...RouteRegistrar) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// Prometheus HTTP request metrics (must come before route registration)
	r.Use(metrics.GinMiddleware())

	// Health endpoints
	health := r.Group("/health")
	{
		health.GET("/live", LiveHandler)
		health.GET("/ready", ReadyHandler(hc))
	}

	// Prometheus metrics
	RegisterMetrics(r)

	// API documentation (Swagger UI)
	RegisterDocs(r)

	// Additional route groups (GitHub, etc.)
	for _, reg := range registrars {
		reg.RegisterRoutes(r)
	}

	return r
}

package server_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"scuffinger/internal/config"
	"scuffinger/internal/server"
)

// ── mock health checker ──────────────────────────────────────────────────────

type mockHealthChecker struct {
	healthy  bool
	statuses map[string]bool
}

func (m *mockHealthChecker) IsHealthy() bool           { return m.healthy }
func (m *mockHealthChecker) Statuses() map[string]bool { return m.statuses }

// ── tests ────────────────────────────────────────────────────────────────────

var _ = Describe("Server", func() {
	var (
		router http.Handler
		hc     *mockHealthChecker
	)

	BeforeEach(func() {
		hc = &mockHealthChecker{
			healthy: true,
			statuses: map[string]bool{
				"cache":    true,
				"database": true,
			},
		}
		cfg := &config.Config{
			Server: config.ServerConfig{Host: "0.0.0.0", Port: 8080},
			Log:    config.LogConfig{Level: "info"},
			App:    config.AppConfig{Name: "scuffinger", Version: "test"},
		}
		router = server.NewRouter(cfg, hc)
	})

	Describe("Health endpoints", func() {
		Context("GET /health/live", func() {
			It("should always return 200 OK", func() {
				req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"ok"`))
			})
		})

		Context("GET /health/ready", func() {
			It("should return 200 OK when all services are healthy", func() {
				req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"ok"`))
				Expect(w.Body.String()).To(ContainSubstring(`"cache":true`))
				Expect(w.Body.String()).To(ContainSubstring(`"database":true`))
			})

			It("should return 503 when a service is unhealthy", func() {
				hc.healthy = false
				hc.statuses["database"] = false

				req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusServiceUnavailable))
				Expect(w.Body.String()).To(ContainSubstring(`"status":"unavailable"`))
				Expect(w.Body.String()).To(ContainSubstring(`"database":false`))
			})
		})
	})

	Describe("Metrics endpoint", func() {
		Context("GET /metrics", func() {
			It("should return 200 OK with Prometheus metrics", func() {
				req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring("go_goroutines"))
			})
		})
	})

	Describe("Unknown routes", func() {
		It("should return 404 for unknown paths", func() {
			req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})
})

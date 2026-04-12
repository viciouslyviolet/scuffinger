package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"scuffinger/internal/i18n"
	"scuffinger/internal/logging"
	"scuffinger/internal/metrics"
)

// Service defines the interface that all services must implement.
// Implement this interface to add new services to the application.
type Service interface {
	// Name returns the unique identifier for the service.
	Name() string
	// Connect establishes a connection to the backing resource.
	Connect(ctx context.Context) error
	// SelfTest runs a one-time startup self-test to verify correct operation.
	SelfTest(ctx context.Context) error
	// Ping performs a lightweight health check used during periodic monitoring.
	Ping(ctx context.Context) error
	// Close cleanly shuts down the service connection.
	Close() error
}

// ServiceStatus holds the health state of a single service.
type ServiceStatus struct {
	Healthy   bool
	LastCheck time.Time
	Error     string
}

// Manager orchestrates the lifecycle of multiple services:
// connection, self-testing, periodic health monitoring, and shutdown.
type Manager struct {
	services []Service
	statuses map[string]*ServiceStatus
	mu       sync.RWMutex
	cancel   context.CancelFunc
	log      *logging.Logger
}

// NewManager creates a new Manager for the given services.
func NewManager(log *logging.Logger, services ...Service) *Manager {
	m := &Manager{
		services: services,
		statuses: make(map[string]*ServiceStatus, len(services)),
		log:      log,
	}
	for _, svc := range services {
		m.statuses[svc.Name()] = &ServiceStatus{}
	}
	return m
}

// AddService registers an additional service with the manager.
// Use this for services that must be created after ConnectAll (e.g. the GitHub collector).
func (m *Manager) AddService(svc Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services = append(m.services, svc)
	m.statuses[svc.Name()] = &ServiceStatus{}
}

// ConnectAll connects to every registered service sequentially.
// Returns immediately on the first failure.
func (m *Manager) ConnectAll(ctx context.Context) error {
	for _, svc := range m.services {
		m.log.Info(i18n.Get(i18n.MsgManagerConnecting), "service", svc.Name())
		if err := svc.Connect(ctx); err != nil {
			m.log.Error(i18n.Get(i18n.ErrManagerConnect), "service", svc.Name(), "error", err)
			return fmt.Errorf("%s [%s]: %w", i18n.Get(i18n.ErrManagerConnect), svc.Name(), err)
		}
		m.log.Info(i18n.Get(i18n.MsgManagerConnected), "service", svc.Name())
	}
	return nil
}

// RunSelfTests executes the one-time self-test for every registered service.
// Each service is marked healthy/unhealthy based on the result.
func (m *Manager) RunSelfTests(ctx context.Context) error {
	for _, svc := range m.services {
		m.log.Info(i18n.Get(i18n.MsgManagerSelfTest), "service", svc.Name())
		if err := svc.SelfTest(ctx); err != nil {
			m.mu.Lock()
			m.statuses[svc.Name()] = &ServiceStatus{
				Healthy:   false,
				LastCheck: time.Now(),
				Error:     err.Error(),
			}
			m.mu.Unlock()
			m.log.Error(i18n.Get(i18n.ErrManagerSelfTest), "service", svc.Name(), "error", err)
			return fmt.Errorf("%s [%s]: %w", i18n.Get(i18n.ErrManagerSelfTest), svc.Name(), err)
		}
		m.mu.Lock()
		m.statuses[svc.Name()] = &ServiceStatus{
			Healthy:   true,
			LastCheck: time.Now(),
		}
		m.mu.Unlock()
		m.log.Info(i18n.Get(i18n.MsgManagerSelfTestPassed), "service", svc.Name())
	}
	return nil
}

// StartHealthChecks begins a background goroutine that pings every service
// at the given interval. Call CloseAll to stop it.
func (m *Manager) StartHealthChecks(interval time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.CheckHealth(ctx)
			}
		}
	}()
}

// CheckHealth runs a synchronous health check (Ping) on every service
// and updates statuses. Exported so tests and callers can trigger it directly.
func (m *Manager) CheckHealth(ctx context.Context) {
	for _, svc := range m.services {
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := svc.Ping(checkCtx)
		cancel()

		metrics.IncHealthCheck(svc.Name())

		m.mu.Lock()
		if err != nil {
			m.statuses[svc.Name()] = &ServiceStatus{
				Healthy:   false,
				LastCheck: time.Now(),
				Error:     err.Error(),
			}
			metrics.IncHealthCheckFailure(svc.Name())
			metrics.SetServiceHealth(svc.Name(), false)
			m.log.Warn(i18n.Get(i18n.WarnManagerHealthFailed), "service", svc.Name(), "error", err)
		} else {
			m.statuses[svc.Name()] = &ServiceStatus{
				Healthy:   true,
				LastCheck: time.Now(),
			}
			metrics.SetServiceHealth(svc.Name(), true)
		}
		m.mu.Unlock()
	}

	// Update the overall health gauge so Prometheus always reflects the
	// aggregate status, even when nothing is polling /ready.
	metrics.SetOverallHealth(m.IsHealthy())
}

// IsHealthy returns true only when ALL registered services are healthy.
func (m *Manager) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.statuses) == 0 {
		return false
	}
	for _, s := range m.statuses {
		if !s.Healthy {
			return false
		}
	}
	return true
}

// Statuses returns a snapshot of each service's health.
func (m *Manager) Statuses() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]bool, len(m.statuses))
	for name, s := range m.statuses {
		out[name] = s.Healthy
	}
	return out
}

// ServiceByName returns the Service with the given name, or nil if not found.
func (m *Manager) ServiceByName(name string) Service {
	for _, svc := range m.services {
		if svc.Name() == name {
			return svc
		}
	}
	return nil
}

// CloseAll stops health checks and closes every service.
func (m *Manager) CloseAll() error {
	if m.cancel != nil {
		m.cancel()
	}
	m.log.Debug(i18n.Get(i18n.MsgManagerClosing))
	var errs []error
	for _, svc := range m.services {
		if err := svc.Close(); err != nil {
			m.log.Error(i18n.Get(i18n.ErrManagerClose), "service", svc.Name(), "error", err)
			errs = append(errs, fmt.Errorf("%s [%s]: %w", i18n.Get(i18n.ErrManagerClose), svc.Name(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s: %v", i18n.Get(i18n.ErrManagerShutdown), errs)
	}
	return nil
}

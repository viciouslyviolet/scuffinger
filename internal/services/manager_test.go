package services_test

import (
	"context"
	"errors"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"scuffinger/internal/config"
	"scuffinger/internal/logging"
	"scuffinger/internal/services"
)

// ── mock service ─────────────────────────────────────────────────────────────

type mockService struct {
	name        string
	connectErr  error
	selfTestErr error
	pingErr     error
	closeErr    error
}

func (m *mockService) Name() string                     { return m.name }
func (m *mockService) Connect(_ context.Context) error  { return m.connectErr }
func (m *mockService) SelfTest(_ context.Context) error { return m.selfTestErr }
func (m *mockService) Ping(_ context.Context) error     { return m.pingErr }
func (m *mockService) Close() error                     { return m.closeErr }

// ── test logger (discards output) ────────────────────────────────────────────

func testLogger() *logging.Logger {
	return logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "json"}, io.Discard)
}

// ── tests ────────────────────────────────────────────────────────────────────

var _ = Describe("Manager", func() {
	var (
		ctx        context.Context
		svc1, svc2 *mockService
		log        *logging.Logger
	)

	BeforeEach(func() {
		ctx = context.Background()
		svc1 = &mockService{name: "alpha"}
		svc2 = &mockService{name: "beta"}
		log = testLogger()
	})

	Describe("ConnectAll", func() {
		It("succeeds when all services connect", func() {
			mgr := services.NewManager(log, svc1, svc2)
			Expect(mgr.ConnectAll(ctx)).To(Succeed())
		})

		It("returns an error if any service fails to connect", func() {
			svc2.connectErr = errors.New("connection refused")
			mgr := services.NewManager(log, svc1, svc2)
			err := mgr.ConnectAll(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("beta"))
			Expect(err.Error()).To(ContainSubstring("connection refused"))
		})
	})

	Describe("RunSelfTests", func() {
		It("marks all services healthy on success", func() {
			mgr := services.NewManager(log, svc1, svc2)
			Expect(mgr.ConnectAll(ctx)).To(Succeed())
			Expect(mgr.RunSelfTests(ctx)).To(Succeed())
			Expect(mgr.IsHealthy()).To(BeTrue())
			Expect(mgr.Statuses()).To(Equal(map[string]bool{
				"alpha": true,
				"beta":  true,
			}))
		})

		It("marks the failed service unhealthy and returns an error", func() {
			svc2.selfTestErr = errors.New("self-test boom")
			mgr := services.NewManager(log, svc1, svc2)
			Expect(mgr.ConnectAll(ctx)).To(Succeed())

			err := mgr.RunSelfTests(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("self-test boom"))
			Expect(mgr.IsHealthy()).To(BeFalse())
			Expect(mgr.Statuses()["alpha"]).To(BeTrue())
			Expect(mgr.Statuses()["beta"]).To(BeFalse())
		})
	})

	Describe("CheckHealth", func() {
		It("updates statuses based on ping results", func() {
			mgr := services.NewManager(log, svc1, svc2)
			Expect(mgr.ConnectAll(ctx)).To(Succeed())
			Expect(mgr.RunSelfTests(ctx)).To(Succeed())
			Expect(mgr.IsHealthy()).To(BeTrue())

			// Simulate svc1 going down
			svc1.pingErr = errors.New("timeout")
			mgr.CheckHealth(ctx)

			Expect(mgr.IsHealthy()).To(BeFalse())
			Expect(mgr.Statuses()["alpha"]).To(BeFalse())
			Expect(mgr.Statuses()["beta"]).To(BeTrue())
		})

		It("recovers when a service comes back", func() {
			mgr := services.NewManager(log, svc1, svc2)
			Expect(mgr.ConnectAll(ctx)).To(Succeed())
			Expect(mgr.RunSelfTests(ctx)).To(Succeed())

			// Goes down
			svc1.pingErr = errors.New("timeout")
			mgr.CheckHealth(ctx)
			Expect(mgr.IsHealthy()).To(BeFalse())

			// Comes back
			svc1.pingErr = nil
			mgr.CheckHealth(ctx)
			Expect(mgr.IsHealthy()).To(BeTrue())
		})
	})

	Describe("IsHealthy", func() {
		It("returns false when no services are registered", func() {
			mgr := services.NewManager(log)
			Expect(mgr.IsHealthy()).To(BeFalse())
		})

		It("returns false before self-tests have run", func() {
			mgr := services.NewManager(log, svc1)
			Expect(mgr.IsHealthy()).To(BeFalse())
		})
	})

	Describe("CloseAll", func() {
		It("closes all services", func() {
			mgr := services.NewManager(log, svc1, svc2)
			Expect(mgr.CloseAll()).To(Succeed())
		})

		It("aggregates close errors", func() {
			svc1.closeErr = errors.New("close fail")
			mgr := services.NewManager(log, svc1, svc2)
			err := mgr.CloseAll()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("close fail"))
		})
	})
})

package logging_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"scuffinger/internal/config"
	"scuffinger/internal/logging"
)

var _ = Describe("Logger", func() {
	var buf *bytes.Buffer

	BeforeEach(func() {
		buf = new(bytes.Buffer)
	})

	// ── Format: JSON ─────────────────────────────────────────────────────

	Describe("JSON format", func() {
		It("should output structured JSON", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "json"}, buf)

			logger.Info("hello world", "key", "value")

			out := buf.String()
			Expect(out).To(ContainSubstring(`"msg":"hello world"`))
			Expect(out).To(ContainSubstring(`"key":"value"`))
		})

		It("should default to JSON when format is empty", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "info", Format: ""}, buf)

			logger.Info("default format")

			Expect(buf.String()).To(ContainSubstring(`"msg":"default format"`))
		})
	})

	// ── Format: Plain ────────────────────────────────────────────────────

	Describe("Plain format", func() {
		It("should output slog text format", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "plain"}, buf)

			logger.Info("hello plain", "count", 42)

			out := buf.String()
			Expect(out).To(ContainSubstring("hello plain"))
			Expect(out).To(ContainSubstring("count=42"))
		})
	})

	// ── Format: YAML ─────────────────────────────────────────────────────

	Describe("YAML format", func() {
		It("should output YAML documents", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "yaml"}, buf)

			logger.Info("yaml entry", "service", "cache")

			out := buf.String()
			Expect(out).To(ContainSubstring("---"))
			Expect(out).To(ContainSubstring(`msg: "yaml entry"`))
			Expect(out).To(ContainSubstring(`service: "cache"`))
			Expect(out).To(ContainSubstring("level: INFO"))
		})

		It("should handle integer values", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "yaml"}, buf)

			logger.Info("with int", "port", 8080)

			Expect(buf.String()).To(ContainSubstring("port: 8080"))
		})

		It("should handle boolean values", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "yaml"}, buf)

			logger.Info("with bool", "ready", true)

			Expect(buf.String()).To(ContainSubstring("ready: true"))
		})
	})

	// ── Debug caller tracking ────────────────────────────────────────────

	Describe("Debug with caller info", func() {
		It("should include function, file, and line number", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "json"}, buf)

			logger.Debug("trace me")

			out := buf.String()
			Expect(out).To(ContainSubstring(`"caller.function"`))
			Expect(out).To(ContainSubstring(`"caller.file"`))
			Expect(out).To(ContainSubstring(`"caller.line"`))
			// file should be the test file itself
			Expect(out).To(ContainSubstring("logger_test.go"))
		})

		It("should include extra key/value metadata", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "json"}, buf)

			logger.Debug("cache miss", "cache_key", "user:42", "latency_ms", 12)

			out := buf.String()
			Expect(out).To(ContainSubstring(`"cache_key":"user:42"`))
			Expect(out).To(ContainSubstring(`"latency_ms":12`))
			// caller info still present alongside custom metadata
			Expect(out).To(ContainSubstring(`"caller.function"`))
		})

		It("should include caller info in YAML format", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "yaml"}, buf)

			logger.Debug("yaml debug", "extra", "data")

			out := buf.String()
			Expect(out).To(ContainSubstring("caller.function:"))
			Expect(out).To(ContainSubstring("caller.file:"))
			Expect(out).To(ContainSubstring("caller.line:"))
			Expect(out).To(ContainSubstring(`extra: "data"`))
		})
	})

	// ── Level filtering ──────────────────────────────────────────────────

	Describe("Log level filtering", func() {
		It("should suppress messages below the configured level", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "warn", Format: "json"}, buf)

			logger.Debug("nope")
			logger.Info("nope")
			logger.Warn("yes warn")
			logger.Error("yes error")

			out := buf.String()
			Expect(out).NotTo(ContainSubstring("nope"))
			Expect(out).To(ContainSubstring("yes warn"))
			Expect(out).To(ContainSubstring("yes error"))
		})

		It("should allow all messages at debug level", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "json"}, buf)

			logger.Debug("d")
			logger.Info("i")
			logger.Warn("w")
			logger.Error("e")

			out := buf.String()
			Expect(out).To(ContainSubstring(`"msg":"d"`))
			Expect(out).To(ContainSubstring(`"msg":"i"`))
			Expect(out).To(ContainSubstring(`"msg":"w"`))
			Expect(out).To(ContainSubstring(`"msg":"e"`))
		})
	})

	// ── With (child logger) ──────────────────────────────────────────────

	Describe("With", func() {
		It("should create a child logger that always includes given fields", func() {
			logger := logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "json"}, buf)
			child := logger.With("component", "cache")

			child.Info("hit")

			out := buf.String()
			Expect(out).To(ContainSubstring(`"component":"cache"`))
			Expect(out).To(ContainSubstring(`"msg":"hit"`))
		})
	})
})

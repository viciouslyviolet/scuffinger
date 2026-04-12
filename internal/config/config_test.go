package config_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"scuffinger/internal/config"
)

var _ = Describe("Config", func() {
	var tmpDir string
	var configFile string

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
		configFile = filepath.Join(tmpDir, "config.yaml")
	})

	Context("when loading from a YAML file", func() {
		BeforeEach(func() {
			content := []byte(`
server:
  host: "127.0.0.1"
  port: 9090
log:
  level: "debug"
app:
  name: "test-app"
  version: "1.0.0"
`)
			Expect(os.WriteFile(configFile, content, 0644)).To(Succeed())
		})

		It("should load values from the YAML file", func() {
			cfg, err := config.Load(configFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Server.Host).To(Equal("127.0.0.1"))
			Expect(cfg.Server.Port).To(Equal(9090))
			Expect(cfg.Log.Level).To(Equal("debug"))
			Expect(cfg.App.Name).To(Equal("test-app"))
			Expect(cfg.App.Version).To(Equal("1.0.0"))
		})

		It("should return the correct address", func() {
			cfg, err := config.Load(configFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Address()).To(Equal("127.0.0.1:9090"))
		})
	})

	Context("when environment variables override config", func() {
		BeforeEach(func() {
			content := []byte(`
server:
  host: "127.0.0.1"
  port: 9090
log:
  level: "debug"
app:
  name: "test-app"
  version: "1.0.0"
`)
			Expect(os.WriteFile(configFile, content, 0644)).To(Succeed())
		})

		It("should override server port from env var", func() {
			os.Setenv("SCUFFINGER_SERVER_PORT", "3000")
			defer os.Unsetenv("SCUFFINGER_SERVER_PORT")

			cfg, err := config.Load(configFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Server.Port).To(Equal(3000))
		})

		It("should override log level from env var", func() {
			os.Setenv("SCUFFINGER_LOG_LEVEL", "warn")
			defer os.Unsetenv("SCUFFINGER_LOG_LEVEL")

			cfg, err := config.Load(configFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Log.Level).To(Equal("warn"))
		})
	})

	Context("when the config file does not exist", func() {
		It("should fall back to defaults", func() {
			cfg, err := config.Load(filepath.Join(tmpDir, "nonexistent.yaml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Server.Host).To(Equal("0.0.0.0"))
			Expect(cfg.Server.Port).To(Equal(8080))
			Expect(cfg.Log.Level).To(Equal("info"))
		})
	})
})

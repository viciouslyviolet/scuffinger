package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Log      LogConfig      `mapstructure:"log"`
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	Cache    CacheConfig    `mapstructure:"cache"`
	GitHub   GitHubConfig   `mapstructure:"github"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

type CacheConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	GitHubCacheTTL string `mapstructure:"github_cache_ttl"`
}

type GitHubConfig struct {
	// ── Multi-credential auth (preferred) ────────────────────────────
	Tokens       []string          `mapstructure:"tokens"`
	Applications []GitHubAppConfig `mapstructure:"applications"`

	// ── Legacy single-credential auth (deprecated, auto-migrated) ────
	Token          string `mapstructure:"token"`
	ClientID       string `mapstructure:"client_id"`
	AppID          int64  `mapstructure:"app_id"`
	InstallationID int64  `mapstructure:"installation_id"`
	PrivateKeyPath string `mapstructure:"private_key_path"`

	// ── Shared settings ──────────────────────────────────────────────
	Organization       string   `mapstructure:"organization"`
	RateLimitThreshold int      `mapstructure:"rate_limit_threshold"`
	Repositories       []string `mapstructure:"repositories"`
	CollectorInterval  string   `mapstructure:"collector_interval"`
	MaxRecentRuns      int      `mapstructure:"max_recent_runs"`
	FetchAnnotations   bool     `mapstructure:"fetch_annotations"`
}

// GitHubAppConfig holds credentials for a single GitHub App installation.
type GitHubAppConfig struct {
	ClientID       string `mapstructure:"client_id"`
	AppID          int64  `mapstructure:"app_id"`
	InstallationID int64  `mapstructure:"installation_id"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
}

// Migrate merges legacy single-value fields into the new slice fields.
// Call this after unmarshalling.
func (g *GitHubConfig) Migrate() {
	// Migrate legacy single token → tokens slice
	if g.Token != "" {
		found := false
		for _, t := range g.Tokens {
			if t == g.Token {
				found = true
				break
			}
		}
		if !found {
			g.Tokens = append([]string{g.Token}, g.Tokens...)
		}
		g.Token = "" // clear legacy field
	}

	// Migrate legacy single app → applications slice
	if g.AppID != 0 {
		found := false
		for _, a := range g.Applications {
			if a.AppID == g.AppID {
				found = true
				break
			}
		}
		if !found {
			g.Applications = append([]GitHubAppConfig{{
				ClientID:       g.ClientID,
				AppID:          g.AppID,
				InstallationID: g.InstallationID,
				PrivateKeyPath: g.PrivateKeyPath,
			}}, g.Applications...)
		}
		g.AppID = 0 // clear legacy fields
		g.ClientID = ""
		g.InstallationID = 0
		g.PrivateKeyPath = ""
	}
}

// Enabled returns true if at least one credential is configured.
func (g *GitHubConfig) Enabled() bool {
	return len(g.Tokens) > 0 || len(g.Applications) > 0
}

// FirstClientID returns the client_id from the first application, if any.
// Used by the OAuth device flow CLI and server handler.
func (g *GitHubConfig) FirstClientID() string {
	for _, a := range g.Applications {
		if a.ClientID != "" {
			return a.ClientID
		}
	}
	return ""
}

// CollectorDuration parses the collector_interval string (e.g. "5m") into a time.Duration.
// Falls back to 5 minutes on parse errors.
func (g *GitHubConfig) CollectorDuration() time.Duration {
	d, err := time.ParseDuration(g.CollectorInterval)
	if err != nil || d <= 0 {
		return 5 * time.Minute
	}
	return d
}

// DSN returns the PostgreSQL connection string.
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// Address returns the host:port string for the cache.
func (c *CacheConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GitHubCacheDuration parses the github_cache_ttl string into a time.Duration.
// Falls back to 5 minutes on parse errors.
func (c *CacheConfig) GitHubCacheDuration() time.Duration {
	d, err := time.ParseDuration(c.GitHubCacheTTL)
	if err != nil || d <= 0 {
		return 5 * time.Minute
	}
	return d
}

// Load reads configuration from config/config.yaml and environment variables.
// Environment variables are prefixed with SCUFFINGER and override YAML values.
// e.g. SCUFFINGER_SERVER_PORT=9090 overrides server.port
func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(configPath)

	// Defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("app.name", "scuffinger")
	v.SetDefault("app.version", "0.1.0")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "scuffinger")
	v.SetDefault("database.password", "scuffinger")
	v.SetDefault("database.name", "scuffinger")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("cache.host", "localhost")
	v.SetDefault("cache.port", 6379)
	v.SetDefault("cache.github_cache_ttl", "5m")
	v.SetDefault("github.token", "")
	v.SetDefault("github.tokens", []string{})
	v.SetDefault("github.client_id", "")
	v.SetDefault("github.app_id", 0)
	v.SetDefault("github.installation_id", 0)
	v.SetDefault("github.private_key_path", "")
	v.SetDefault("github.applications", []GitHubAppConfig{})
	v.SetDefault("github.organization", "")
	v.SetDefault("github.rate_limit_threshold", 100)
	v.SetDefault("github.repositories", []string{})
	v.SetDefault("github.collector_interval", "5m")
	v.SetDefault("github.max_recent_runs", 5)
	v.SetDefault("github.fetch_annotations", true)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// Config file not found is not fatal – we have defaults + env
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// If the file exists but can't be read, still try to continue
			fmt.Printf("Warning: could not read config file: %v\n", err)
		}
	}

	// Environment variable overrides
	v.SetEnvPrefix("SCUFFINGER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to unmarshal config: %w", err)
	}

	// Merge legacy single-value fields into the new slice-based fields.
	cfg.GitHub.Migrate()

	return &cfg, nil
}

// Address returns the host:port string for the server.
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

# Scuffinger — Code Formatting Guide

This document describes the coding conventions used throughout the Scuffinger codebase. All contributors should follow these patterns to keep the code consistent and reviewable.

---

## Table of Contents

1. [General Principles](#1-general-principles)
2. [Go Formatting](#2-go-formatting)
3. [Naming Conventions](#3-naming-conventions)
4. [File Organisation](#4-file-organisation)
5. [Package Layout](#5-package-layout)
6. [Imports](#6-imports)
7. [Comments and Documentation](#7-comments-and-documentation)
8. [Section Dividers](#8-section-dividers)
9. [Error Handling](#9-error-handling)
10. [Interfaces](#10-interfaces)
11. [Struct Patterns](#11-struct-patterns)
12. [Testing](#12-testing)
13. [SQL Files](#13-sql-files)
14. [YAML and Configuration](#14-yaml-and-configuration)
15. [Metrics](#15-metrics)
16. [Commit Messages](#16-commit-messages)

---

## 1. General Principles

- Run `gofmt` (or `goimports`) on every file before committing. The project follows the standard Go formatting rules without exception.
- Favour clarity over cleverness. A new contributor should be able to read any function and understand it without referring to external documentation.
- Keep functions short. If a function exceeds ~40 lines, consider extracting helpers.
- Every public symbol must have a doc comment.

---

## 2. Go Formatting

### Tabs, not spaces

Go uses tabs for indentation. This is enforced by `gofmt` and is non-negotiable.

### Line length

There is no hard line-length limit (Go does not enforce one), but aim to keep lines under **100 characters** where practical. Break long function signatures, struct literals, and chained calls across multiple lines.

### Trailing newline

Every file ends with a single trailing newline (blank line after the last closing brace). This is the convention used throughout the project.

### Blank lines

Use a single blank line to separate logical sections within a function. Do not use multiple consecutive blank lines.

---

## 3. Naming Conventions

### Packages

- All lowercase, single-word where possible: `config`, `metrics`, `auth`, `vault`, `server`, `services`.
- Avoid stuttering: the `config` package exports `Config`, not `ConfigConfig`.

### Types

- PascalCase for exported types: `GitHubService`, `CacheService`, `DatabaseConfig`.
- Acronyms are fully capitalised only when they start the name: `DSN()`, `SSLMode`.
- Interface names describe the capability, not the implementation: `Service`, `Store`, `HealthChecker`, `ClientProvider`, `RouteRegistrar`.

### Functions and methods

- PascalCase for exported, camelCase for unexported.
- Constructors are named `New<Type>`: `NewManager`, `NewGitHubHandler`, `NewCacheService`.
- Boolean-returning methods start with `Is`, `Has`, or `Enabled`: `IsHealthy()`, `HasCredentials()`, `Enabled()`.
- Getters omit the `Get` prefix: `Client()`, `Pool()`, `Organization()`, `ActiveLabel()`.

### Variables

- Package-level Prometheus metrics are PascalCase exported vars: `HealthStatus`, `HTTPRequestsTotal`, `RepoStars`.
- Loop variables use short names: `svc`, `cfg`, `err`, `ctx`.
- Avoid single-letter names except for `i`, `j` (loop indices), `c` (Gin context), `r` (Gin engine or HTTP request), and `w` (writer).

### Constants

- PascalCase for exported, camelCase for unexported.
- i18n message keys use dot-separated paths: `MsgServerStarting`, `ErrDbPing`, `WarnGhRateLow`.
- Prefixes: `Msg` (info), `Err` (error), `Warn` (warning), `Cmd` (CLI description).

### Files

- Lowercase with underscores: `github_collector.go`, `yaml_handler.go`, `cmd_suite_test.go`.
- Test files use the `_test.go` suffix.
- Suite bootstrap files follow the pattern `<package>_suite_test.go`.
- Platform-specific files use build-tag naming: `vault_darwin.go`, `vault_linux.go`, `vault_windows.go`.

---

## 4. File Organisation

Within a single `.go` file, content is ordered as follows:

1. **Package clause** and **doc comment** (if this is the package's primary file).
2. **Imports** (grouped, see [§6](#6-imports)).
3. **Constants**.
4. **Package-level variables** (e.g. Prometheus metric definitions).
5. **Type definitions** (interfaces first, then structs).
6. **Constructor functions** (`New…`).
7. **Exported methods** (grouped by receiver type).
8. **Unexported methods and helper functions** (preceded by a section divider comment).

Example structure:

```go
package services

import ( … )

// Service defines the interface …
type Service interface { … }

// Manager orchestrates …
type Manager struct { … }

// NewManager creates …
func NewManager(…) *Manager { … }

// ConnectAll connects …
func (m *Manager) ConnectAll(…) error { … }

// ── helpers ──────────────────────────────────────────

func (m *Manager) someHelper(…) { … }
```

---

## 5. Package Layout

The project follows the Go standard layout with `internal/` for private packages:

```
main.go                          ← entry point
cmd/                             ← CLI commands (Cobra)
config/                          ← YAML configuration file
internal/
├── auth/                        ← GitHub OAuth device flow
├── config/                      ← configuration loading (Viper)
├── i18n/                        ← internationalisation messages
├── logging/                     ← structured logging (slog)
├── metrics/                     ← Prometheus metric definitions
├── server/                      ← HTTP handlers (Gin)
├── services/                    ← service lifecycle management
└── vault/                       ← cross-platform secrets storage
database/
├── github/                      ← embedded SQL for GitHub data cache
└── self_test/                   ← embedded SQL for startup self-tests
monitoring/                      ← Prometheus, Grafana, Loki, Promtail configs
charts/                          ← Helm chart
docs/                            ← documentation
```

**Rules:**

- Business logic lives in `internal/`. Nothing outside the module should import these packages.
- SQL lives in `database/` as embedded `.sql` files, loaded via `//go:embed`.
- Each `internal/` package has a single responsibility.
- Avoid circular dependencies. The dependency graph flows downward: `cmd` → `server` → `services` → `metrics`, `config`, `i18n`, `logging`.

---

## 6. Imports

Imports are grouped into three blocks separated by blank lines:

1. **Standard library** (`context`, `fmt`, `net/http`, etc.)
2. **Third-party** (`github.com/gin-gonic/gin`, `github.com/prometheus/…`, etc.)
3. **Internal** (`scuffinger/internal/config`, `scuffinger/internal/metrics`, etc.)

Example:

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"

    "scuffinger/internal/config"
    "scuffinger/internal/i18n"
    "scuffinger/internal/logging"
)
```

Use named imports only when there is a conflict or when the package name differs from the last path segment:

```go
import (
    dbgithub "scuffinger/database/github"
    selftest "scuffinger/database/self_test"
)
```

Dot imports are permitted **only** in test files for Ginkgo/Gomega:

```go
import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)
```

---

## 7. Comments and Documentation

### Doc comments

Every exported type, function, method, and package-level variable must have a doc comment. The comment starts with the symbol name:

```go
// Manager orchestrates the lifecycle of multiple services:
// connection, self-testing, periodic health monitoring, and shutdown.
type Manager struct { … }

// ConnectAll connects to every registered service sequentially.
// Returns immediately on the first failure.
func (m *Manager) ConnectAll(ctx context.Context) error { … }
```

### Inline comments

Use inline comments sparingly, only to explain **why**, not **what**:

```go
// +1 because callerFields is itself one frame.
pc, file, line, ok := runtime.Caller(skip + 1)
```

### TODO / FIXME

Use `// TODO:` or `// FIXME:` for items that need future attention. Include context:

```go
// TODO: add retry logic with exponential backoff
```

---

## 8. Section Dividers

The codebase uses Unicode box-drawing section dividers to visually separate logical blocks within a file. These are **74 characters wide** (including the `// ` prefix):

```go
// ── Section title ────────────────────────────────────────────────────────
```

For subsections:

```go
// ── helpers ──────────────────────────────────────────────────────────────
```

These are used consistently throughout the project in:

- `bootstrap.go` — to label each lifecycle phase (Register, Connect, Self-tests, Health checks, Collector).
- `github.go` — to separate PAT client building from App client building.
- `config.go` — to label config struct groups.
- `messages.go` — to label message key groups.
- `serve.go` — to label the route registrar setup phases.

Use them whenever a file has distinct logical sections that benefit from visual separation.

---

## 9. Error Handling

### Wrap with context

Always wrap errors with context using `fmt.Errorf` or the `i18n.Err` helper:

```go
// Using i18n.Err for user-facing errors:
return i18n.Err(i18n.ErrDbPing, err)
// → "Database did not respond to ping: <original error>"

// Using fmt.Errorf for internal errors:
return fmt.Errorf("connect to %s: %w", svc.Name(), err)
```

### Never ignore errors silently

If an error is intentionally ignored, assign it to `_` with a comment:

```go
_ = mgr.CloseAll() // best-effort cleanup
```

### Early returns

Prefer early returns over deep nesting:

```go
func (s *GitHubService) Connect(ctx context.Context) error {
    if !s.cfg.Enabled() {
        return errors.New(i18n.Get(i18n.ErrGhNotConfigured))
    }
    // … main logic
}
```

---

## 10. Interfaces

### Keep interfaces small

Interfaces should have 1–5 methods. The `Service` interface (5 methods) is the upper bound:

```go
type Service interface {
    Name() string
    Connect(ctx context.Context) error
    SelfTest(ctx context.Context) error
    Ping(ctx context.Context) error
    Close() error
}
```

### Define interfaces where they are used

Interfaces are defined in the **consumer** package, not the provider. For example:

- `HealthChecker` is defined in `internal/server/` (where it's needed), not in `internal/services/` (where it's implemented).
- `ClientProvider` is defined in `internal/server/github.go`, not in `internal/services/github.go`.

### Implicit satisfaction

Go interfaces are satisfied implicitly. Document this when it matters:

```go
// HealthChecker is satisfied by any type that can report aggregate service health.
// services.Manager implements this implicitly.
type HealthChecker interface { … }
```

---

## 11. Struct Patterns

### Constructor functions

Every struct with dependencies gets a constructor. Constructors accept only what the struct needs:

```go
func NewDatabaseService(cfg config.DatabaseConfig, log *logging.Logger) *DatabaseService {
    return &DatabaseService{cfg: cfg, log: log}
}
```

### Unexported fields

Struct fields are unexported by default. Provide accessor methods where external access is needed:

```go
type DatabaseService struct {
    cfg  config.DatabaseConfig  // unexported
    pool *pgxpool.Pool          // unexported
    log  *logging.Logger        // unexported
}

// Pool returns the underlying connection pool for reuse by other packages.
func (s *DatabaseService) Pool() *pgxpool.Pool {
    return s.pool
}
```

### Concurrency

Structs with shared mutable state use `sync.RWMutex`:

```go
type Manager struct {
    services []Service
    statuses map[string]*ServiceStatus
    mu       sync.RWMutex  // protects statuses
    // …
}
```

Read operations use `RLock` / `RUnlock`; write operations use `Lock` / `Unlock`.

---

## 12. Testing

### Framework

The project uses [Ginkgo v2](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/) for BDD-style tests. Standard library `testing` is also used for simple unit tests (e.g. table-driven tests in `debug_test.go`).

### Test file naming

- BDD suite bootstrap: `<package>_suite_test.go`
- BDD specs: `<module>_test.go` (e.g. `manager_test.go`, `config_test.go`)
- Standard tests: `<module>_test.go` (e.g. `debug_test.go`)

### Suite bootstrap

Each package using Ginkgo needs a suite file:

```go
package services_test

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestServices(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Services Suite")
}
```

### BDD spec structure

Use `Describe` / `Context` / `It` with clear descriptions:

```go
var _ = Describe("Manager", func() {
    Describe("ConnectAll", func() {
        It("succeeds when all services connect", func() { … })
        It("returns an error if any service fails to connect", func() { … })
    })
})
```

### Table-driven tests

For pure functions without side effects, standard table-driven tests are preferred:

```go
func TestClampInt(t *testing.T) {
    tests := []struct {
        s    string
        lo   int
        hi   int
        want int
    }{
        {"50", 1, 500, 50},
        {"0", 1, 500, 1},
        // …
    }
    for _, tt := range tests {
        got := clampInt(tt.s, tt.lo, tt.hi)
        if got != tt.want {
            t.Errorf("clampInt(%q, %d, %d) = %d, want %d", tt.s, tt.lo, tt.hi, got, tt.want)
        }
    }
}
```

### Mocks

Mocks are defined inline in the test file, implementing only the interface needed:

```go
type mockService struct {
    name        string
    connectErr  error
    selfTestErr error
    pingErr     error
    closeErr    error
}

func (m *mockService) Name() string                     { return m.name }
func (m *mockService) Connect(_ context.Context) error  { return m.connectErr }
// …
```

### Test loggers

When a logger is needed in tests, use `logging.NewWithWriter` writing to `io.Discard`:

```go
func testLogger() *logging.Logger {
    return logging.NewWithWriter(config.LogConfig{Level: "debug", Format: "json"}, io.Discard)
}
```

---

## 13. SQL Files

- Each query lives in its own `.sql` file under `database/<domain>/`.
- Use `CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS` for idempotent DDL.
- SQL keywords are UPPERCASE: `SELECT`, `INSERT INTO`, `WHERE`, `ORDER BY`.
- Table and column names are lowercase with underscores: `github_workflow_runs`, `fetched_at`.
- Parameterised queries use `$1`, `$2`, etc. (PostgreSQL syntax).
- Comments at the top of the file describe the query's purpose.
- Go files in the same directory use `//go:embed` to load the SQL at compile time.

---

## 14. YAML and Configuration

### Config file

- Keys are `snake_case`: `server.host`, `github.collector_interval`, `cache.github_cache_ttl`.
- Inline comments use `#` with two spaces of padding after the value.
- Sections are separated by a blank line.

### Helm values

Follow the same `snake_case` convention. Kubernetes-specific keys (e.g. `replicaCount`) follow the Helm community convention.

---

## 15. Metrics

### Naming

Follow the [Prometheus naming best practices](https://prometheus.io/docs/practices/naming/):

- Namespace: `scuffinger`
- Subsystem: `http`, `health`, `github`, `cache`, `service`, `log`
- Name: descriptive, ending with the unit: `_seconds`, `_total`, `_bytes`, `_status`

### Registration

All metrics use `promauto.New*` for automatic registration with the default Prometheus registry. Define them as package-level `var` blocks:

```go
var HTTPRequestsTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "scuffinger",
        Subsystem: "http",
        Name:      "requests_total",
        Help:      "Total number of HTTP requests.",
    },
    []string{"method", "path", "status"},
)
```

### Helper functions

Provide helper functions to avoid metric manipulation spreading across the codebase:

```go
func ObserveGitHubCall(endpoint string, duration time.Duration, err error) { … }
func SetOverallHealth(healthy bool) { … }
func IncLogMessage(level string) { … }
```

---

## 16. Commit Messages

Use the conventional commit format:

```
<type>(<scope>): <description>

[optional body]
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `ci`.

**Scopes:** `server`, `services`, `metrics`, `config`, `auth`, `i18n`, `logging`, `cmd`, `db`, `docker`, `helm`, `grafana`.

**Examples:**

```
feat(services): add new GitHubService for API interactions
fix(health): update overall gauge in background health check loop
docs: add architecture and formatting guides
test(services): add manager recovery test
refactor(metrics): extract stale series cleanup into helper
chore(docker): bump Grafana to 11.6.0
```

Keep the subject line under 72 characters. Use the body for motivation and context when the change is non-trivial.


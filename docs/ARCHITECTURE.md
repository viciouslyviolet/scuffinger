# Scuffinger — Technical Documentation

This document provides an in-depth look at the theory of operation, architecture, feature set, and individual modules that make up Scuffinger.

---

## Table of Contents

1. [Theory of Operation](#1-theory-of-operation)
2. [Architecture Overview](#2-architecture-overview)
3. [Feature Summary](#3-feature-summary)
4. [Module Reference](#4-module-reference)
   - 4.1 [Entry Point — `main.go`](#41-entry-point--maingo)
   - 4.2 [CLI Layer — `cmd/`](#42-cli-layer--cmd)
   - 4.3 [Configuration — `internal/config/`](#43-configuration--internalconfig)
   - 4.4 [Service Manager — `internal/services/`](#44-service-manager--internalservices)
   - 4.5 [HTTP Server — `internal/server/`](#45-http-server--internalserver)
   - 4.6 [Metrics — `internal/metrics/`](#46-metrics--internalmetrics)
   - 4.7 [Logging — `internal/logging/`](#47-logging--internallogging)
   - 4.8 [Internationalisation — `internal/i18n/`](#48-internationalisation--internali18n)
   - 4.9 [Authentication — `internal/auth/`](#49-authentication--internalauth)
   - 4.10 [Vault — `internal/vault/`](#410-vault--internalvault)
   - 4.11 [Database Layer — `database/`](#411-database-layer--database)
   - 4.12 [Monitoring Stack — `monitoring/`](#412-monitoring-stack--monitoring)
   - 4.13 [Deployment — `Dockerfile`, `docker-compose.yml`, `charts/`](#413-deployment--dockerfile-docker-composeyml-charts)
5. [Data Flow](#5-data-flow)
6. [Health Model](#6-health-model)
7. [Database Schema](#8-database-schema)
8. [Prometheus Metric Catalogue](#9-prometheus-metric-catalogue)
9. [Grafana Dashboards](#10-grafana-dashboards)

---

## 1. Theory of Operation

Scuffinger operates as a **continuous monitoring bridge** between the GitHub API and a Prometheus/Grafana observability stack. Its lifecycle works as follows:

### Startup Phase

1. **Configuration loading** — Viper reads `config/config.yaml`, applies defaults, then overlays any `SCUFFINGER_*` environment variables.

2. **Service bootstrap** — the `Manager` connects to each backing service sequentially (Valkey → PostgreSQL → GitHub). If any connection fails, startup is aborted.

3. **Self-testing** — every service runs a one-time self-test:
   - **Cache (Valkey):** writes an `init` key with the current timestamp and reads it back.
   - **Database (PostgreSQL):** creates a temporary `testdb` database, performs full CRUD (create table → insert → read → update → read again), then drops the test database.
   - **GitHub:** verifies authentication by fetching the authenticated user, then checks the API rate limit against the configured threshold.
   - **GitHub Collector:** validates the repository list configuration and trial-fetches the first repository's metadata.

4. **Health check loop** — a background goroutine pings every service at a configurable interval (default 10 s) and updates both per-service and overall health gauges.

5. **Collector start** — the GitHub collector runs an immediate first cycle, then repeats on the configured `collector_interval` (default 5 m).

6. **HTTP server** — Gin starts listening on the configured address with all route groups registered.

### Steady-State Phase

The application enters a loop where three concurrent activities run:

- **Health checks** — every 10 seconds, the manager pings all services and updates Prometheus gauges.
- **GitHub collection** — every `collector_interval`, the collector fetches repository metadata, workflow runs, job steps, and failure annotations for every configured repository, publishing the data as Prometheus gauges and persisting raw JSON to PostgreSQL.
- **HTTP request serving** — the Gin router handles incoming API requests, health probes, and Prometheus scrapes.

### Shutdown Phase

On receiving `SIGINT` or `SIGTERM`:

1. The HTTP server is gracefully drained with a 5-second timeout.
2. The health-check loop is cancelled.
3. The collector loop is cancelled.
4. All service connections are closed (database pool, cache client).

---

## 2. Architecture Overview

```
    ┌──────────────────────────────────────────────────────────────┐
    │                          CLI (Cobra)                         │
    │  scuffinger serve | version | github auth | status | logout  │
    └──────────────────────────────┬───────────────────────────────┘
                                   │
                           PersistentPreRunE
                                   │
                      ┌────────────▼─────────────┐
                      │   Configuration (Viper)  │
                      │   config/config.yaml     │
                      │   + SCUFFINGER_* env     │
                      └────────────┬─────────────┘
                                   │
                      ┌────────────▼─────────────┐
                      │   Service Bootstrap      │
                      │   services.Bootstrap()   │
                      └──┬─────────┬──────────┬──┘
                         │         │          │
                  ┌──────▼───┐ ┌───▼─────┐ ┌──▼──────┐
                  │  Cache   │ │ GitHub  │ │   DB    │
                  │ (Valkey) │ │ Service │ │  (Pg)   │
                  └──────────┘ └────┬────┘ └─────────┘
                                    │          
                           ┌────────▼─────────┐
                           │  GitHub          │
                           │  Collector       │
                           │  (background)    │
                           └────────┬─────────┘
                                    │
         ┌──────────────────────────▼───────────────────────────┐
         │                    HTTP Server (Gin)                 │
         │                                                      │
         │  /health/live          Liveness probe                │
         │  /health/ready         Readiness probe               │
         │  /metrics              Prometheus scrape endpoint    │
         │  /api/github/*         GitHub API proxy endpoints    │
         │  /api/auth             OAuth device flow endpoint    │
         │  /api/debug/pg/*       PostgreSQL debug browser      │
         │  /api/debug/cache/*    ValKey debug browser          │
         └───────────────────────┬──────────────────────────────┘
                                 │
         ┌───────────────────────▼──────────────────────────────┐
         │              Observability Stack                     │
         │  Prometheus ← scrapes /metrics                       │
         │  Loki ← receives logs via Promtail                   │
         │  Grafana ← queries Prometheus + Loki                 │
         │    ├─ Scuffinger Overview                            │
         │    ├─ GitHub Workflows                               │
         │    └─ Job State Timeline                             │
         └──────────────────────────────────────────────────────┘
```

### Key Design Principles

- **Service interface pattern** — every backing resource (database, cache, GitHub) implements a uniform `Service` interface (`Connect`, `SelfTest`, `Ping`, `Close`). The `Manager` orchestrates their lifecycle generically.
- **RouteRegistrar pattern** — HTTP route groups are injected into the router via a `RouteRegistrar` interface, keeping `NewRouter` decoupled from specific endpoint logic.
- **Metric-per-file modularity** — Prometheus metrics are defined in individual files under `internal/metrics/`, one per domain (HTTP, health, GitHub, logging, cache, uptime), all auto-registered via `promauto`.
- **Embedded SQL** — all database queries are stored in `.sql` files and loaded at compile time with `//go:embed`, keeping SQL out of Go code.
- **i18n-first messaging** — every user-facing string is a key in `internal/i18n/messages.go`, enabling full localisation without touching business logic.

---

## 3. Feature Summary

### GitHub Monitoring

| Feature | Description |
|---|---|
| Repository metadata | Stars, forks, open issues, size, language, default branch, archived status |
| Workflow runs | Duration, conclusion/status, per-workflow tracking |
| Workflow jobs | State-timeline data (status code, start time, duration) for each job in a run |
| Step-level timings | Start time and duration for every step within a job — enables Gantt chart visualisation |
| Failure annotations | Check-run annotations (level, title, path) collected from failed jobs |
| Rate limit tracking | Per-credential remaining count exposed as a gauge with threshold warnings |

### Observability

| Feature | Description |
|---|---|
| Prometheus metrics | 30+ metrics covering HTTP, health, GitHub API, collector, logging, cache, and uptime |
| Grafana dashboards | Three pre-provisioned dashboards: Overview, GitHub Workflows, Job State Timeline |
| Loki integration | Structured JSON logs shipped via Promtail, queryable in Grafana |
| Log level counters | Prometheus counters for debug/info/warn/error log messages |

### Operations

| Feature | Description |
|---|---|
| Health endpoints | `/health/live` (process liveness) and `/health/ready` (aggregate service health) |
| Self-tests | Startup CRUD verification for database, round-trip for cache, auth+rate-limit for GitHub |
| Debug browser | REST API to inspect PostgreSQL tables and ValKey keys at runtime |
| Graceful shutdown | Signal handling with 5 s HTTP drain timeout and orderly service teardown |
| Configuration | YAML file + environment variable overrides with automatic legacy field migration |
| Secrets management | Cross-platform vault (macOS Keychain, Windows Credential Manager, Linux file store) |
| GitHub OAuth | Device flow authentication via CLI (`scuffinger github auth`) or HTTP endpoint |
| Kubernetes | Helm chart with configurable values |
| Docker | Multi-stage Dockerfile + full Docker Compose stack |
| i18n | 10 languages (EN, DE, ES, FI, FR, JA, MT, NO, SV, ZH) |

---

## 4. Module Reference

### 4.1 Entry Point — `main.go`

The entire application entry point. It simply calls `cmd.Execute()` which delegates to Cobra.

```go
func main() {
    cmd.Execute()
}
```

### 4.2 CLI Layer — `cmd/`

Built on [Cobra](https://github.com/spf13/cobra). The root command's `PersistentPreRunE` hook runs before every subcommand and handles:

1. Loading configuration via Viper.
2. Creating the structured logger.
3. Bootstrapping all services (unless the command opts out with `skipBootstrap: "true"` annotation).

`PersistentPostRunE` ensures `Manager.CloseAll()` is called on exit.

| File | Command | Description |
|---|---|---|
| `root.go` | `scuffinger` | Root command; loads config, logger, and bootstraps services |
| `serve.go` | `scuffinger serve` | Starts the HTTP server with all route registrars |
| `version.go` | `scuffinger version` | Prints the version string (skips bootstrap) |
| `github.go` | `scuffinger github` | Parent command for GitHub subcommands |
| `github_auth.go` | `scuffinger github auth` | Runs the GitHub OAuth device flow |
| `github_status.go` | `scuffinger github status` | Shows current authentication status |
| `github_logout.go` | `scuffinger github logout` | Removes stored credentials from the vault |

**Subcommand registration:**

```
scuffinger
├── serve
├── version
└── github
    ├── auth
    ├── status
    └── logout
```

### 4.3 Configuration — `internal/config/`

Uses [Viper](https://github.com/spf13/viper) for configuration management.

**`Config` struct hierarchy:**

```
Config
├── Server   (host, port)
├── Log      (level, format)
├── App      (name, version)
├── Database (host, port, user, password, name, sslmode)
├── Cache    (host, port, github_cache_ttl)
└── GitHub
    ├── Tokens []string              — PATs
    ├── Applications []GitHubAppConfig — App credentials
    ├── Organization                 — optional org scope
    ├── RateLimitThreshold           — minimum remaining requests
    ├── Repositories []string        — repos to monitor (owner/repo)
    ├── CollectorInterval            — e.g. "5m"
    ├── MaxRecentRuns                — per-workflow
    └── FetchAnnotations             — boolean
```

**Configuration precedence (highest wins):**

1. Environment variables (`SCUFFINGER_SERVER_PORT`, etc.)
2. Config file (`config/config.yaml`)
3. Viper defaults (hardcoded in `Load()`)

**Legacy migration:** `GitHubConfig.Migrate()` automatically merges the deprecated single-value fields (`token`, `app_id`, etc.) into the new slice-based fields (`tokens`, `applications`), ensuring backward compatibility.

**Helper methods:**

- `DatabaseConfig.DSN()` — builds the PostgreSQL connection string.
- `CacheConfig.Address()` — returns `host:port` for the cache.
- `CacheConfig.GitHubCacheDuration()` — parses the TTL string into a `time.Duration`.
- `GitHubConfig.CollectorDuration()` — parses the interval string into a `time.Duration`.
- `GitHubConfig.Enabled()` — returns true if at least one credential is configured.
- `GitHubConfig.FirstClientID()` — finds the first OAuth App client ID for the device flow.

### 4.4 Service Manager — `internal/services/`

The service layer follows a **uniform lifecycle interface**:

```go
type Service interface {
    Name() string
    Connect(ctx context.Context) error
    SelfTest(ctx context.Context) error
    Ping(ctx context.Context) error
    Close() error
}
```

The `Manager` struct orchestrates all registered services:

| Method | Description |
|---|---|
| `ConnectAll(ctx)` | Connects to every service sequentially; fails fast on first error |
| `RunSelfTests(ctx)` | Runs each service's one-time startup verification |
| `StartHealthChecks(interval)` | Spawns a background goroutine that calls `CheckHealth` at the given interval |
| `CheckHealth(ctx)` | Pings every service, updates per-service and overall Prometheus health gauges |
| `IsHealthy()` | Returns `true` only when ALL services are healthy |
| `Statuses()` | Returns a `map[string]bool` snapshot of each service's health |
| `AddService(svc)` | Registers a service after initial construction (used for the collector) |
| `CloseAll()` | Stops health checks, then closes every service |

#### CacheService (`cache.go`)

Manages the Valkey (Redis-compatible) connection via `go-redis`.

- **Connect:** creates a `redis.Client` with configurable timeouts and pings it.
- **SelfTest:** writes an `init` key with the current RFC 3339 timestamp, reads it back, and verifies the round-trip.
- **Ping:** `PING` command.
- **Exports:** `Client()` returns the underlying `*redis.Client` for reuse.

#### DatabaseService (`database.go`)

Manages the PostgreSQL connection via `pgxpool`.

- **Connect:** creates a connection pool and pings the database.
- **SelfTest:** 
  1. Terminates any leftover connections to `testdb` and drops it (cleanup from a previous crash).
  2. Creates a fresh `testdb`.
  3. Connects to `testdb`.
  4. Runs full CRUD: `CREATE TABLE` → `INSERT` → `SELECT` (verify value) → `UPDATE` → `SELECT` (verify updated value).
  5. Closes the test connection and drops `testdb`.
- **Ping:** `pool.Ping()`.
- **Exports:** `Pool()` returns the `*pgxpool.Pool` for reuse.


#### GitHubCollectorService (`github_collector.go`)

Periodically fetches data from the GitHub API and publishes Prometheus metrics.

- **Connect:** no-op (reuses the GitHubService client).
- **SelfTest:** validates that repositories are configured and trial-fetches the first one.
- **Start:** kicks off the background loop (`collect` → `collectRepo` → `collectRun` → `collectAnnotations`).
- **Ping:** returns the error from the last collection cycle (`nil` = healthy).

**Collection cycle:**

1. `ResetTrackedWorkflowMetrics()` — deletes stale gauge series from the previous cycle.
2. For each configured repository:
   - Fetch repository metadata → set `RepoStars`, `RepoForks`, `RepoOpenIssues`, `RepoSize`, `RepoInfo` gauges.
   - Fetch recent workflow runs (up to `max_recent_runs`) → for each run:
     - Record run duration and conclusion as Prometheus gauges.
     - Upsert the run's raw JSON to `github_workflow_runs` in PostgreSQL.
     - Fetch jobs for the run → for each job:
       - Record job status, start time, and duration (for state-timeline).
       - Upsert the job's raw JSON to `github_workflow_jobs`.
       - For each step: record step start time and duration (for Gantt charts).
       - If `fetch_annotations` is enabled and the job failed: fetch check-run annotations and record them.
3. Increment the `collector_cycles_total` counter.

#### GitHub App Transport (`ghauth.go`)

Implements `http.RoundTripper` for GitHub App authentication. It generates a JWT signed with the app's RSA private key and exchanges it for an installation access token, caching the token until 1 minute before expiry.

#### Bootstrap (`bootstrap.go`)

`Bootstrap()` is the orchestration function called during startup:

1. Creates `CacheService` and `DatabaseService`.
2. Checks for GitHub tokens in the config; falls back to the system vault.
3. If GitHub is enabled, creates `GitHubService`.
4. Creates the `Manager` and calls `ConnectAll`.
5. Creates the GitHub cache tables in PostgreSQL (idempotent `CREATE TABLE IF NOT EXISTS`).
6. If GitHub is enabled and repositories are configured, creates and connects the `GitHubCollectorService`.
7. Runs `RunSelfTests` on all services.
8. Starts periodic health checks.
9. Starts the collector's background loop.

### 4.5 HTTP Server — `internal/server/`

Built on [Gin](https://github.com/gin-gonic/gin) in release mode.

**`NewRouter` assembles the engine:**

1. Applies `gin.Logger()` and `gin.Recovery()` middleware.
2. Applies the Prometheus HTTP metrics middleware (`metrics.GinMiddleware()`).
3. Registers health endpoints (`/health/live`, `/health/ready`).
4. Registers the Prometheus `/metrics` endpoint.
5. Iterates over all `RouteRegistrar` implementations and calls `RegisterRoutes(r)`.

#### Health Handlers (`health.go`)

| Endpoint | Behaviour |
|---|---|
| `GET /health/live` | Always returns `200 {"status":"ok"}` — the process is alive |
| `GET /health/ready` | Returns `200` when all services are healthy, `503` otherwise; updates per-service and overall health gauges |

The `HealthChecker` interface (`IsHealthy()`, `Statuses()`) is satisfied by `services.Manager`.

#### GitHub Handler (`github.go`)

Registers `/api/github/*` endpoints that proxy to the GitHub API:

| Endpoint | GitHub API call |
|---|---|
| `GET /api/github/users/:username` | `Users.Get` |
| `GET /api/github/orgs/:org` | `Organizations.Get` |
| `GET /api/github/repos/:owner/:repo` | `Repositories.Get` |
| `GET /api/github/repos/:owner/:repo/branches` | `Repositories.ListBranches` (paginated) |
| `GET /api/github/repos/:owner/:repo/workflows` | `Actions.ListWorkflows` (paginated) |
| `GET /api/github/repos/:owner/:repo/workflows/:workflow_id/runs` | `Actions.ListWorkflowRunsByID` (paginated) |
| `GET /api/github/rate-limit` | `RateLimit.Get` |

Every call is instrumented with `metrics.ObserveGitHubCall` (counter, histogram, error counter). GitHub 404s are mapped to HTTP 404; other errors return 502.

#### Auth Handler (`auth.go`)

Registers `POST /api/auth`, which initiates the GitHub OAuth device flow and returns the verification URI and user code. This allows web-based clients to trigger authentication without the CLI.

#### Debug Handler (`debug.go`)

Registers `/api/debug/*` endpoints for development-time inspection:

**PostgreSQL:**

| Endpoint | Description |
|---|---|
| `GET /api/debug/pg/databases` | Lists non-template databases |
| `GET /api/debug/pg/tables` | Lists tables in a schema (default `public`) |
| `GET /api/debug/pg/tables/:table/columns` | Column metadata for a table |
| `GET /api/debug/pg/tables/:table/rows` | Query rows with fuzzy search (`?q=`), date filtering (`?from=`, `?to=`), sorting (`?sort=`, `?order=`), and pagination (`?limit=`, `?offset=`) |

**ValKey:**

| Endpoint | Description |
|---|---|
| `GET /api/debug/cache/keys` | Scan keys with glob pattern, fuzzy filter, type filter, and cursor pagination |
| `GET /api/debug/cache/keys/:key` | Key value, type, TTL, and memory usage (supports all Redis data types) |
| `GET /api/debug/cache/stats` | Parsed Redis `INFO` sections + `DBSIZE` |

Security note: the debug endpoints have no authentication and are intended for development only. In production, restrict access via network policy or remove the registrar.

#### Metrics Endpoint (`metrics.go`)

Registers `GET /metrics` using `promhttp.Handler()`, which exposes all Prometheus metrics registered with the default registry.

### 4.6 Metrics — `internal/metrics/`

All metrics are registered with `promauto` (automatic default registry). Each domain has its own file:

| File | Metrics |
|---|---|
| `uptime.go` | `scuffinger_uptime_seconds` (GaugeFunc) |
| `health.go` | `scuffinger_health_status`, `scuffinger_health_service_status{service}` |
| `http.go` | `scuffinger_http_requests_total{method,path,status}`, `scuffinger_http_request_duration_seconds{method,path}`, `scuffinger_http_requests_in_flight` |
| `github.go` | `scuffinger_github_api_calls_total{endpoint}`, `scuffinger_github_api_errors_total{endpoint}`, `scuffinger_github_api_duration_seconds{endpoint}`, `scuffinger_github_rate_limit_remaining{credential}` |
| `github_collector.go` | Repository gauges, workflow run/job/step gauges, annotation gauges, collector cycle/error counters |
| `logging.go` | `scuffinger_log_messages_total{level}` |
| `cache.go` | `scuffinger_cache_hits_total{tier,resource}`, `scuffinger_cache_misses_total{tier,resource}` |
| `services.go` | `scuffinger_service_health_checks_total{service}`, `scuffinger_service_health_check_failures_total{service}` |

**Stale series management:** the collector uses `ResetTrackedWorkflowMetrics()` at the start of each cycle to delete gauge series from the previous cycle. All emitted label sets are tracked in slices (`trackedStepKeys`, `trackedRunKeys`, etc.) and deleted before re-population.

**Gin middleware:** `GinMiddleware()` uses `c.FullPath()` (the registered route pattern, e.g. `/api/github/users/:username`) as the path label to avoid high-cardinality explosions from path parameters.

### 4.7 Logging — `internal/logging/`

Built on the standard library's `log/slog` package.

**Supported formats:**

| Format | Handler | Notes |
|---|---|---|
| `json` (default) | `slog.JSONHandler` | Structured JSON, ideal for Loki ingestion |
| `plain` | `slog.TextHandler` | Key=value pairs, suitable for local development |
| `yaml` | Custom `YAMLHandler` | YAML documents delimited by `---` |

**Caller tracking:** `Debug()` calls capture the caller's function name, file path, and line number via `runtime.Caller`. File paths are trimmed to project-relative form (e.g. `cmd/serve.go`).

**Metric integration:** every log call increments `scuffinger_log_messages_total{level}`, enabling Grafana alerting on error rates.

**YAMLHandler:** a custom `slog.Handler` that formats log records as YAML documents. Each record is separated by `---` and includes `time`, `level`, `msg`, and all key/value attributes.

### 4.8 Internationalisation — `internal/i18n/`

All user-facing strings are defined as constants of type `Key` in `messages.go`. The English translation map (`En`) is the default.

**Available translations:**

| File | Language |
|---|---|
| `messages.go` | English (default) |
| `de.go` | German |
| `es.go` | Spanish |
| `fi.go` | Finnish |
| `fr.go` | French |
| `ja.go` | Japanese |
| `mt.go` | Maltese |
| `no.go` | Norwegian |
| `sv.go` | Swedish |
| `zh.go` | Chinese |

**API:**

- `i18n.Get(key)` — returns the translated string, falling back to the key name.
- `i18n.Set(messages)` — switches the active language.
- `i18n.Err(key, cause)` — creates a formatted `error` wrapping a translated message prefix around an underlying error.

### 4.9 Authentication — `internal/auth/`

Implements the **GitHub OAuth Device Flow** (RFC 8628):

1. `RequestDeviceCode(clientID, scopes)` — `POST https://github.com/login/device/code` to get a `device_code`, `user_code`, and `verification_uri`.
2. The user navigates to `verification_uri` and enters the `user_code`.
3. `PollForToken(clientID, deviceCode, interval, expiresIn)` — polls `POST https://github.com/login/oauth/access_token` at the specified interval until the user authorises, or the code expires. Handles `slow_down`, `expired_token`, and `access_denied` responses.

**Credential management functions:**

- `SaveToken(store, token)` / `LoadToken(store)` — persist/retrieve the OAuth token from the vault.
- `SaveUser(store, username)` / `LoadUser(store)` — persist/retrieve the GitHub username.
- `ClearCredentials(store)` — remove all stored credentials.
- `HasCredentials(store)` — check if a token exists.

### 4.10 Vault — `internal/vault/`

Cross-platform secrets storage abstraction.

```go
type Store interface {
    Set(key, value string) error
    Get(key string) (string, error)
    Delete(key string) error
}
```

| Platform | Backend | File |
|---|---|---|
| macOS | `security` CLI (system Keychain) | `vault_darwin.go` |
| Windows | PowerShell (Credential Manager) | `vault_windows.go` |
| Linux | Plain files under `~/.scuffinger/` | `vault_linux.go` |
| Other | In-memory (lost on exit) | `vault_other.go` |

`vault.New()` returns the appropriate `Store` for the current `GOOS`.

### 4.11 Database Layer — `database/`

SQL queries are stored in `.sql` files and embedded at compile time with `//go:embed`.

#### `database/self_test/`

SQL used during the startup self-test:

| File | Purpose |
|---|---|
| `create_database.sql` | `CREATE DATABASE %s` |
| `drop_database.sql` | `DROP DATABASE IF EXISTS %s` |
| `terminate_connections.sql` | Terminates active connections to a database |
| `create_self_test_table.sql` | Creates the self-test table |
| `insert_self_test.sql` | Inserts a test record |
| `select_self_test.sql` | Reads a test record by key |
| `update_self_test.sql` | Updates a test record |

#### `database/github/`

SQL for the GitHub data cache:

| File | Purpose |
|---|---|
| `create_tables.sql` | Idempotent `CREATE TABLE IF NOT EXISTS` for all GitHub tables |
| `upsert_repo.sql` | Insert or update repository metadata |
| `select_repo.sql` | Fetch repository by name |
| `upsert_workflow_run.sql` | Insert or update a workflow run |
| `select_recent_runs.sql` | Fetch recent runs for a repository |
| `upsert_workflow_job.sql` | Insert or update a workflow job |
| `select_jobs_for_run.sql` | Fetch jobs for a given run |
| `insert_annotation.sql` | Insert a check-run annotation |
| `select_annotations_for_job.sql` | Fetch annotations for a job |
| `delete_annotations_for_job.sql` | Delete annotations for a job (before re-insertion) |

### 4.12 Monitoring Stack — `monitoring/`

Pre-configured observability services deployed via Docker Compose:

| Directory | Service | Purpose |
|---|---|---|
| `prometheus/` | Prometheus v3.4.0 | Scrapes `/metrics` from the app, retains 7 days of data |
| `grafana/` | Grafana 11.6.0 | Dashboards and alerting; auto-provisioned datasources and dashboards |
| `loki/` | Loki 3.5.0 | Log aggregation backend |
| `promtail/` | Promtail 3.5.0 | Ships container logs from Docker to Loki |

**Grafana provisioning:**

- `grafana/provisioning/datasources/` — auto-configures Prometheus and Loki datasources.
- `grafana/provisioning/dashboards/dashboards.yml` — tells Grafana where to find JSON dashboard files.
- `grafana/provisioning/dashboards/json/` — three pre-built dashboards.

### 4.13 Deployment — `Dockerfile`, `docker-compose.yml`, `charts/`

#### Dockerfile

Multi-stage build:

1. **Builder stage:** `golang:1.26-alpine` — downloads dependencies, copies source, and compiles with `-ldflags="-s -w"` (stripped, no debug info).
2. **Runtime stage:** `alpine:3.21` — copies the binary and config, exposes port 8080, runs `scuffinger serve`.

#### Docker Compose

Defines 7 services:

| Service | Image | Ports |
|---|---|---|
| `valkey` | `valkey/valkey:8-alpine` | 6379 |
| `postgres` | `postgres:17-alpine` | 5432 |
| `app` | Built from Dockerfile | 8080 |
| `prometheus` | `prom/prometheus:v3.4.0` | 9090 |
| `loki` | `grafana/loki:3.5.0` | 3100 |
| `promtail` | `grafana/promtail:3.5.0` | — |
| `grafana` | `grafana/grafana:11.6.0` | 3000 |

Health checks ensure the app doesn't start until Valkey and PostgreSQL are ready.

#### Helm Chart (`charts/scuffinger/`)

A Helm 3 chart for Kubernetes deployment. See `charts/scuffinger/Chart.yaml` for metadata and `values.yaml` for configurable parameters.

---

## 5. Data Flow

```
GitHub API
    │
    │  (REST API calls, rate-limited)
    ▼
GitHubService ─── credentials
    │
    │  Client()
    ▼
GitHubCollectorService
    │
    ├── Repository metadata ──► Prometheus gauges (stars, forks, issues, size)
    │                        └─► PostgreSQL (github_repos)
    │
    ├── Workflow runs ────────► Prometheus gauges (duration, conclusion)
    │                        └─► PostgreSQL (github_workflow_runs)
    │
    ├── Workflow jobs ────────► Prometheus gauges (status, start, duration)
    │                        └─► PostgreSQL (github_workflow_jobs)
    │
    ├── Workflow steps ───────► Prometheus gauges (start time, duration)
    │
    └── Annotations ──────────► Prometheus gauges (level, count)
                             └─► PostgreSQL (github_annotations)

Prometheus ◄── scrapes /metrics every 15s ──► Grafana queries
Loki ◄── Promtail ships container logs ──► Grafana log panel
```

---

## 6. Health Model

Health is evaluated at two levels:

### Per-Service Health

Each service's `Ping()` method is called every 10 seconds. The result updates:

- `scuffinger_health_service_status{service="cache"|"database"|"github"|"github_collector"}` (1 = UP, 0 = DOWN)
- `scuffinger_service_health_checks_total{service}` (counter)
- `scuffinger_service_health_check_failures_total{service}` (counter)

### Overall Health

`scuffinger_health_status` is set to `1` only when **all** registered services report healthy. It is updated:

- After every background health check cycle (`Manager.CheckHealth`).
- When the `/health/ready` endpoint is hit.

The `/health/ready` endpoint also returns a JSON response with the per-service breakdown:

```json
{
  "status": "ok",
  "services": {
    "cache": true,
    "database": true,
    "github": true,
    "github_collector": true
  }
}
```

---

## 7. Database Schema

### `github_repos`

| Column | Type | Description |
|---|---|---|
| `repo` | `TEXT PRIMARY KEY` | Full name (`owner/repo`) |
| `data` | `JSONB` | Raw GitHub API JSON |
| `fetched_at` | `TIMESTAMPTZ` | When the data was fetched |

### `github_workflow_runs`

| Column | Type | Description |
|---|---|---|
| `repo` | `TEXT` | Full name |
| `run_id` | `BIGINT PRIMARY KEY` | GitHub run ID |
| `workflow` | `TEXT` | Workflow name |
| `conclusion` | `TEXT` | Conclusion/status string |
| `data` | `JSONB` | Raw JSON |
| `updated_at` | `TIMESTAMPTZ` | Run's last update time |
| `fetched_at` | `TIMESTAMPTZ` | When fetched |

Indexes: `(repo)`, `(repo, updated_at DESC)`.

### `github_workflow_jobs`

| Column | Type | Description |
|---|---|---|
| `repo` | `TEXT` | Full name |
| `run_id` | `BIGINT` | Parent run |
| `job_id` | `BIGINT PRIMARY KEY` | GitHub job ID |
| `data` | `JSONB` | Raw JSON |
| `fetched_at` | `TIMESTAMPTZ` | When fetched |

Index: `(run_id)`.

### `github_annotations`

| Column | Type | Description |
|---|---|---|
| `id` | `BIGSERIAL PRIMARY KEY` | Auto-increment |
| `repo` | `TEXT` | Full name |
| `run_id` | `BIGINT` | Parent run |
| `job_id` | `BIGINT` | Parent job |
| `annotation_level` | `TEXT` | Level (failure, warning, notice) |
| `title` | `TEXT` | Annotation title |
| `message` | `TEXT` | Annotation message |
| `path` | `TEXT` | File path |
| `start_line` | `INT` | Start line in file |
| `end_line` | `INT` | End line in file |
| `fetched_at` | `TIMESTAMPTZ` | When fetched |

Indexes: `(job_id)`, `(run_id)`.

---

## 8. Prometheus Metric Catalogue

### System

| Metric | Type | Labels | Description |
|---|---|---|---|
| `scuffinger_uptime_seconds` | Gauge | — | Seconds since process start |
| `scuffinger_health_status` | Gauge | — | Overall health (1 = healthy) |
| `scuffinger_health_service_status` | Gauge | `service` | Per-service health (1 = UP) |

### HTTP

| Metric | Type | Labels | Description |
|---|---|---|---|
| `scuffinger_http_requests_total` | Counter | `method`, `path`, `status` | Total HTTP requests |
| `scuffinger_http_request_duration_seconds` | Histogram | `method`, `path` | Request latency |
| `scuffinger_http_requests_in_flight` | Gauge | — | Currently processing |

### GitHub API

| Metric | Type | Labels | Description |
|---|---|---|---|
| `scuffinger_github_api_calls_total` | Counter | `endpoint` | API call count |
| `scuffinger_github_api_errors_total` | Counter | `endpoint` | API error count |
| `scuffinger_github_api_duration_seconds` | Histogram | `endpoint` | API call latency |
| `scuffinger_github_rate_limit_remaining` | Gauge | `credential` | Remaining rate limit |

### GitHub Collector

| Metric | Type | Labels | Description |
|---|---|---|---|
| `scuffinger_github_repo_stars` | Gauge | `repo` | Star count |
| `scuffinger_github_repo_forks` | Gauge | `repo` | Fork count |
| `scuffinger_github_repo_open_issues` | Gauge | `repo` | Open issue count |
| `scuffinger_github_repo_size_kb` | Gauge | `repo` | Repo size in KB |
| `scuffinger_github_repo_info` | Gauge | `repo`, `language`, `default_branch`, `archived` | Metadata info (always 1) |
| `scuffinger_github_workflow_run_duration_seconds` | Gauge | `repo`, `workflow`, `run_id`, `conclusion` | Run duration |
| `scuffinger_github_workflow_run_status` | Gauge | `repo`, `workflow`, `conclusion` | Latest run status (1 = current) |
| `scuffinger_github_workflow_step_duration_seconds` | Gauge | `repo`, `workflow`, `run_id`, `job`, `step`, `conclusion` | Step duration |
| `scuffinger_github_workflow_step_start_time` | Gauge | `repo`, `workflow`, `run_id`, `job`, `step`, `conclusion` | Step start (Unix) |
| `scuffinger_github_workflow_job_status` | Gauge | `repo`, `workflow`, `run_id`, `job` | Job status code |
| `scuffinger_github_workflow_job_started_at` | Gauge | `repo`, `workflow`, `run_id`, `job` | Job start (Unix) |
| `scuffinger_github_workflow_job_duration_seconds` | Gauge | `repo`, `workflow`, `run_id`, `job` | Job duration |
| `scuffinger_github_workflow_annotation_total` | Gauge | `repo`, `workflow`, `run_id`, `job`, `annotation_level` | Annotation count by level |
| `scuffinger_github_workflow_annotation_info` | Gauge | `repo`, `workflow`, `run_id`, `job`, `annotation_level`, `title`, `path` | Annotation detail (always 1) |
| `scuffinger_github_collector_cycles_total` | Counter | — | Completed collection cycles |
| `scuffinger_github_collector_errors_total` | Counter | `repo` | Collection errors by repo |

### Services

| Metric | Type | Labels | Description |
|---|---|---|---|
| `scuffinger_service_health_checks_total` | Counter | `service` | Health check pings |
| `scuffinger_service_health_check_failures_total` | Counter | `service` | Failed health checks |

### Logging

| Metric | Type | Labels | Description |
|---|---|---|---|
| `scuffinger_log_messages_total` | Counter | `level` | Log messages by level |

### Cache

| Metric | Type | Labels | Description |
|---|---|---|---|
| `scuffinger_cache_hits_total` | Counter | `tier`, `resource` | Cache hits |
| `scuffinger_cache_misses_total` | Counter | `tier`, `resource` | Cache misses |

---

## 9. Grafana Dashboards

Three dashboards are auto-provisioned:

### Scuffinger — Overview (`scuffinger-overview`)

The main operational dashboard with six rows:

1. **⚡ System Status** — uptime, overall health, in-flight requests, rate limit remaining, per-service health table.
2. **📦 Repository & CI** — stars, forks, open issues, repo size, collection cycles, CI failure annotations.
3. **🌐 HTTP Performance** — request rate by path, latency percentiles (p50/p95/p99), requests by status code, error rate (4xx + 5xx).
4. **🐙 GitHub API** — API call rate by endpoint, API error rate, API p95 latency, rate limit over time.
5. **🏥 Service Health** — health check rate and failure rate per service.
6. **📋 Logs** — log messages by level (bar chart), error + warning counters, live Loki log stream, error-only Loki stream.

### Scuffinger — GitHub Workflows (`scuffinger-github-workflows`)

Detailed workflow run and step timing visualisations, including Gantt-style step breakdowns.

### Scuffinger — Job State Timeline (`scuffinger-job-state-timeline`)

State-timeline panels showing job status transitions over time, colour-coded by conclusion (success, failure, in_progress, queued, skipped, cancelled).


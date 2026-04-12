# Scuffinger

Scuffinger is a lightweight, self-hosted GitHub monitoring service. It periodically collects repository metadata, workflow runs, job step timings, and check-run annotations from the GitHub API and exposes them as Prometheus metrics, pre-built Grafana dashboards, and a RESTful JSON API — all from a single Go binary backed by PostgreSQL and ValKey.

## Features

- **GitHub data collection** — background collector fetches repo stats, CI workflow runs, job step durations, and failure annotations on a configurable interval.
- **Full observability stack** — Prometheus metrics, Grafana dashboards (Overview, Workflows, Job State Timeline), Loki log aggregation via Promtail — all provisioned out of the box.
- **Health management** — startup self-tests (database CRUD, cache round-trip, GitHub auth verification), periodic health checks, and `/health/live` + `/health/ready` endpoints.
- **Debug browser** — REST endpoints to inspect PostgreSQL tables and ValKey keys at runtime.
- **GitHub OAuth device flow** — CLI and HTTP-based authentication; tokens stored in the OS keychain (macOS Keychain, Windows Credential Manager, or `~/.scuffinger/` on Linux).
- **Internationalisation** — all user-facing strings live in a central message catalogue with translations for DE, ES, FI, FR, JA, MT, NO, SV, and ZH.
- **Structured logging** — JSON, plain text, or YAML output formats with automatic caller tracking on debug messages and per-level Prometheus counters.
- **Kubernetes-ready** — ships with a Helm chart (`charts/scuffinger/`) and a multi-stage Dockerfile.

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.26 |
| CLI framework | [Cobra](https://github.com/spf13/cobra) + [Viper](https://github.com/spf13/viper) |
| HTTP framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | [PostgreSQL 17](https://www.postgresql.org/) via [pgx](https://github.com/jackc/pgx) |
| Cache | [Valkey 8](https://valkey.io/) (Redis-compatible) via [go-redis](https://github.com/redis/go-redis) |
| GitHub API | [go-github](https://github.com/google/go-github) + [oauth2](https://pkg.go.dev/golang.org/x/oauth2) |
| Metrics | [Prometheus client_golang](https://github.com/prometheus/client_golang) |
| Monitoring | Prometheus · Grafana · Loki · Promtail |
| Testing | [Ginkgo](https://onsi.github.io/ginkgo/) + [Gomega](https://onsi.github.io/gomega/) |
| Secrets | OS keychain (macOS/Windows) or file-based vault (Linux) |
| Container | Docker (multi-stage) · Docker Compose |
| Orchestration | Helm 3 chart |

## Prerequisites

- **Go** ≥ 1.26
- **Docker** and **Docker Compose** (for the full stack)
- A **GitHub Personal Access Token** (or GitHub App credentials) for API access

## Quick Start

### Running with Docker Compose (recommended)

```bash
# Clone the repository
git clone https://github.com/your-org/scuffinger.git
cd scuffinger

# Add your GitHub PAT to config/config.yaml (github.tokens)
# Then start everything:
make run
```

This builds the Go binary, starts PostgreSQL, ValKey, the app, Prometheus, Loki, Promtail, and Grafana. Services are available at:

| Service | URL |
|---|---|
| Scuffinger API | <http://localhost:8080> |
| Grafana | <http://localhost:3000> (admin / scuffinger) |
| Prometheus | <http://localhost:9090> |

### Running locally (without Docker)

Make sure PostgreSQL and ValKey are running on their default ports, then:

```bash
# Build the binary
make build

# Run the server
./bin/scuffinger serve --config config/config.yaml
```

### CLI Commands

```bash
scuffinger version                   # Print version
scuffinger serve --config <path>     # Start the HTTP server
scuffinger github auth               # Authenticate via GitHub OAuth device flow
scuffinger github status             # Show current authentication status
scuffinger github logout             # Remove stored credentials
```

### Configuration

Scuffinger reads configuration from a YAML file (default `config/config.yaml`). Every value can be overridden with environment variables prefixed with `SCUFFINGER_` using underscore-separated paths:

```bash
# Example: override the server port and database host
export SCUFFINGER_SERVER_PORT=9090
export SCUFFINGER_DATABASE_HOST=db.example.com
```

See [`config/config.yaml`](config/config.yaml) for all available options.

## Makefile Targets

```
make build   — Compile the Go binary to bin/scuffinger
make test    — Run all tests (go test ./... -v)
make run     — Build and start all services with Docker Compose
make stop    — Stop and remove all Docker Compose services
make clean   — Remove build artifacts and data volumes
make help    — Show available targets
```

## Debugging

### Delve (CLI)

```bash
# Start the server under Delve
dlv debug . -- serve --config config/config.yaml

# Or attach to a running process
dlv attach $(pgrep scuffinger)
```

### GoLand / VS Code

1. Create a **Go Build** run configuration (GoLand) or a `launch.json` entry (VS Code).
2. Set the program arguments to `serve --config config/config.yaml`.
3. Set the working directory to the project root.
4. Start with the debugger (breakpoints, stepping, and variable inspection work as usual).

### Debug API Endpoints

While the server is running, the debug browser is available at:

```
GET /api/debug/pg/databases          — List PostgreSQL databases
GET /api/debug/pg/tables             — List tables in a schema
GET /api/debug/pg/tables/:table/columns — Describe a table's columns
GET /api/debug/pg/tables/:table/rows — Query rows (supports ?q=, ?from=, ?to=, ?sort=, ?limit=)
GET /api/debug/cache/keys            — Scan ValKey keys
GET /api/debug/cache/keys/:key       — Get a key's value, type, TTL, and memory usage
GET /api/debug/cache/stats           — ValKey server statistics
```

## Contributing

Contributions are welcome! To get started:

1. **Fork** the repository and create a feature branch from `main`.
2. **Install dependencies** — run `go mod download` (vendored in `vendor/`).
3. **Read the docs** — see [`docs/`](docs/) for architecture, module details, and the formatting guide.
4. **Write tests** — the project uses [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/). Run the suite with `make test`.
5. **Follow the style guide** — see [`docs/FORMATTING.md`](docs/FORMATTING.md) for code conventions.
6. **Commit clearly** — use concise, descriptive commit messages (e.g. `fix(health): update overall gauge in background loop`).
7. **Open a Pull Request** — describe what changed and why. Link to any related issues.

### Adding a New Service

1. Implement the `Service` interface in `internal/services/`.
2. Register the service in `Bootstrap()` (`internal/services/bootstrap.go`).
3. The manager automatically handles connection, self-testing, health checks, and shutdown.

### Adding a New Language

1. Create a new file in `internal/i18n/` (e.g. `pt.go`).
2. Copy the `En` Messages map and translate every value.
3. Call `i18n.Set(yourMap)` before the application starts.

## License

See the [LICENSE](LICENSE) file for details.


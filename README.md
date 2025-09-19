# Atlas

<p align="center">
  <img src="./docs/assets/logo.jpeg" alt="ATLAS logo" />
</p>

Atlas is a batteries-included Go service template optimised for developer experience and production-readiness. It bundles HTTP & gRPC servers, dependency injection (Fx), Bun-backed persistence, Redis caching, messaging + worker orchestration, structured logging, distributed tracing, metrics, and a CLI that drives the full lifecycle.

## Features

- **Layers & DI** – Opinionated domain layering (`entity` → `repository` → `service` → `transport`) wired through Uber Fx modules.
- **HTTP + gRPC** – Echo HTTP server with health/metrics endpoints and OTEL tracing; gRPC infrastructure ready to be plugged back in.
- **Persistence** – Bun ORM with read/write splitting, goose migrations, and seed helpers.
- **Caching** – Pluggable cache module with Redis or noop backends.
- **Messaging & Workers** – Kafka client abstraction plus worker engine with configurable concurrency.
- **Observability** – Zap logging, OpenTelemetry tracing, Prometheus metrics (`/metrics`), stdout/OTLP exporters configurable via env.
- **CLI (`atlas`)** – UX-centric Cobra CLI for running the service, migrations, seeds, worker engine, and scaffolding.

## Quick Start

1. **Clone & install deps**
   ```bash
   git clone https://github.com/Additional-Code/atlas.git && cd atlas
   go mod tidy
   ```

2. **Bootstrap environment**
   ```bash
   cp .example.env .env
   # Update connection strings for Postgres, Redis, Kafka, OTLP collector, etc.
   ```

3. **Run database migrations & seed data**
   ```bash
   go run main.go migrate up
   go run main.go seed
   ```

4. **Start the HTTP API**
   ```bash
   go run main.go run
   # Health check:  curl http://localhost:8080/health
   # Metrics:       curl http://localhost:8080/metrics
   ```

5. **Start workers (Kafka consumers/jobs)**
   ```bash
   go run main.go worker run
   ```

## CLI Reference

| Command | Description |
|---------|-------------|
| `go run main.go run` | Starts the HTTP server (Echo) with tracing and metrics middleware. |
| `go run main.go migrate up` | Applies goose migrations from `db/migrations/sql`. |
| `go run main.go migrate down --steps 1` | Rolls back the latest migration (use `--all` to drop back to baseline). |
| `go run main.go seed` | Inserts sample seed data using the Bun seeder. |
| `go run main.go worker run` | Boots the worker engine wired to the messaging client. |
| `go run main.go module create <name>` | Placeholder for future code generation scaffolding. |

## Configuration

Configuration is read from environment variables (with `.env` automatically loaded via `godotenv`). Key variables are documented in `.example.env`:

### HTTP / gRPC
- `HTTP_HOST` / `HTTP_PORT`
- `GRPC_HOST` / `GRPC_PORT`

### Database & Cache
- `DB_DRIVER`, `DB_WRITER_DSN`, `DB_READER_DSN`
- `CACHE_ENABLED`, `CACHE_DRIVER`, `REDIS_ADDR`, `CACHE_DEFAULT_TTL`

### Messaging & Workers
- `MESSAGING_ENABLED`, `MESSAGING_DRIVER`
- Kafka specifics: `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_CONSUMER_GROUP`, etc.
- Worker knobs: `WORKER_ENABLED`, `WORKER_CONCURRENCY`, `WORKER_POLL_INTERVAL`

### Observability
- Logging: `OBS_LOG_LEVEL` (`debug`, `info`, `warn`, ...), `OBS_LOG_ENCODING` (`json`|`console`)
- Traces: `OBS_ENABLE_TRACING`, `OBS_TRACE_EXPORTER` (`stdout`|`otlp`), `OBS_OTLP_ENDPOINT`, `OBS_OTLP_INSECURE`
- Metrics: `OBS_ENABLE_METRICS`, `OBS_METRICS_EXPORTER` (`prometheus`|`stdout`), `OBS_PROMETHEUS_PATH`

## Observability Stack

- **Logging** – Zap is configured via env vars and enriches logs with service + environment labels.
- **Tracing** – OpenTelemetry tracer provider supports stdout or OTLP exporters. Echo requests are instrumented automatically when tracing is enabled.
- **Metrics** – Prometheus exporter registers at `OBS_PROMETHEUS_PATH` (default `/metrics`). A stdout exporter is also available for local debugging.

## Project Layout

```
internal/
  app/              Fx wiring (Core/HTTP/Worker bundles)
  cache/            Redis/noop cache drivers
  cli/              Cobra command tree powering the atlas CLI
  config/           Config loader + env helpers
  database/         Bun connection management
  entity/           Domain models
  migration/        Goose migrator wrapper
  messaging/        Kafka client abstraction
  observability/    OTEL tracing & metrics manager
  repository/       Persistence repositories
  service/          Domain services (business logic)
  server/http/      Echo server lifecycle & middleware
  presentation/http HTTP handlers (orders, metrics)
  worker/           Worker engine + order event example

cmd/
  atlas/            Separate main for building the CLI binary

main.go             Root entrypoint delegating to the CLI
.db/...             Goose SQL migrations
.example.env        Environment variable template
```

## Development Workflow

- **Migrations** – Add new Goose migrations under `db/migrations/sql` (`00002_<name>.sql`) using `-- +goose Up/Down` markers. Run `go run main.go migrate up` to apply.
- **Seeding** – Extend `internal/seeder` to add fixtures; execute with `go run main.go seed`.
- **Workers** – Register new handlers by adding `worker.HandlerRegistration` in packages like `internal/worker/<domain>`.
- **Testing** – Standard Go testing (`go test ./...`). Add integration tests per service/repository once backing services are available.

## Next Steps

- Fill in additional domain modules using the provided layering.
- Extend module scaffolding command to generate entity/repository/service stubs.
- Wire gRPC transport back in once proto definitions are available.
- Add Docker Compose / Helm charts for local + production orchestration.

---
Happy hacking! Contributions & feedback are welcome.

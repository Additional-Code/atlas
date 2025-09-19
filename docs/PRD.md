# Product Requirements Document (PRD)

## Product Overview
A Go boilerplate that maximizes developer experience and operational scalability. It provides a ready-to-use architecture with DI, web APIs, gRPC, database, caching, messaging, background jobs, observability, and developer tooling. It is designed for building high-performance, distributed systems with minimal setup friction.

## Goals
- Deliver the most developer-friendly Go boilerplate.
- Support both request-response and event-driven architectures.
- Provide battle-tested integrations for databases, caching, messaging, workers, and observability.
- Optimize developer workflows with scaffolding, CLI tooling, and seamless local/prod parity.

## Target Users
- Backend engineers building microservices and event-driven systems.
- Teams requiring high reliability and observability in production.
- Organizations standardizing Go-based service development.

## Core Features

### Framework & DX
- Uber Fx for DI and lifecycle management.
- Echo for REST APIs with structured middleware.
- gRPC server with interceptors for auth, logging, tracing.
- Hot reload for development.
- Scaffolding CLI for services, modules, migrations, workers.

### Persistence
- Bun ORM with MySQL, PostgreSQL, SQLite support.
- Read/write DB split.
- Migration and seeding framework.

### Caching
- Redis for caching, session, rate limiting, and pub-sub.
- Cache decorators for repository queries.

### Messaging & Workers
- Kafka integration for event streaming.
- Pub/Sub interface with pluggable backends (Kafka, Redis, Google Pub/Sub).
- Background workers with retry, scheduling, and dead-letter handling.
- Worker CLI to run standalone or embedded workers.

### Auth & Security
- JWT authentication with refresh flow.
- RBAC authorization middleware.
- Config-driven roles and permissions.
- Rate limiting, CSRF, and CORS defaults.

### Observability
- Zap structured logging.
- OpenTelemetry tracing and metrics.
- Prometheus metrics endpoint.
- Health, readiness, and debug endpoints.

### Tooling & DevOps
- Unified CLI (`atlas`) for scaffolding, migrations, seeding, worker management, testing.
- Config loader with env + file support.
- Docker + docker-compose for local development (DB, Redis, Kafka).
- Helm charts and GitHub Actions CI/CD templates.
- Devcontainer for instant environment setup.

### Testing
- Unit, integration, and E2E test harness.
- Testcontainers for DB, Redis, Kafka.
- Mock generation for gRPC and REST services.
- Auto-generated OpenAPI and gRPC clients for E2E tests.

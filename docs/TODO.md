# Implementation TODO

- [x] Scaffold core service layout (Fx app, HTTP server, config loader)
- [x] Add gRPC server skeleton with interceptors
- [x] Implement shared error handling (errorbank, response builder)
- [x] Wire database module (Bun, read/write pools)
- [x] Integrate Redis cache module
- [x] Provide messaging + worker engine abstraction
- [x] Implement CLI (`atlas`) commands (migrate, seed, module, worker)
- [x] Implement migrations and seeders
- [x] Set up observability stack (Zap, OTEL, Prometheus endpoints)
- [ ] Create Docker/devcontainer assets
- [ ] Add CI/CD pipelines and Helm chart
- [ ] Flesh out testing harness (unit, integration, E2E)

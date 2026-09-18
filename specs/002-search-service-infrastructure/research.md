# Research: Search Service Infrastructure

**Date**: 2026-09-15
**Feature**: [spec.md](./spec.md)

## Technical Analysis & Decisions

### 1. Go Runtime & Build Strategy (`search-back`)

- **Decision**: Use `golang:1.24-alpine` (or `golang:1.24-alpine`) base image with Alpine package manager, supporting non-root host UID/GID mapping (UID 1000) and entrypoint `CMD ["go", "run", "./cmd/api/main.go"]`.
- **Rationale**:
  - Provides a lightweight (~300MB), fast container startup suitable for local development.
  - Aligns with the project's Go microservice pattern established in the `auth` service.
  - Standard `go run ./cmd/api/main.go` compiles and runs the application cleanly from live mounted `/app` directory, allowing developers to restart the container (`docker compose restart search-back`) to reload changes without image rebuilds.
- **Alternatives Considered**:
  - *Air live-reloader*: Rejected during clarification session (Option B selected for standard Go execution and container restart).
  - *Debian/Ubuntu Go image*: Heavier footprint and slower pull/build times compared to Alpine.

### 2. Search Engine & Storage Architecture (`search-engine`)

- **Decision**: Use official `elasticsearch:8.17.2` (or 8.x) Docker image with `discovery.type=single-node`, `xpack.security.enabled=false`, memory capped at 256MB heap (`ES_JAVA_OPTS: "-Xms256m -Xmx256m"`), and named persistent volume `search-engine-data:/usr/share/elasticsearch/data`.
- **Rationale**:
  - Follows Clarification Session answers (Option A for disabled security in dev network and Option C for 256MB JVM heap limit).
  - Keeps local development resource consumption minimal and prevents out-of-memory container crashes on developer workstations.
  - Ensures search indexes, mappings, and indexed documents persist across `docker compose down` and `docker compose up` cycles.
- **Alternatives Considered**:
  - *OpenSearch*: Elasticsearch was explicitly specified in user requirements.
  - *Shared Elasticsearch cluster*: Violates Constitution Principle I (Domain Autonomy & Storage Isolation).

### 3. Client Libraries & Drivers

- **Decision**:
  - Elasticsearch client: Official `github.com/elastic/go-elasticsearch/v8` package.
  - Message Broker client: `github.com/rabbitmq/amqp091-go` (matching `services/auth`).
  - HTTP routing: Go standard library `net/http` with structured JSON handlers.
- **Rationale**:
  - Standard library `net/http` provides maximum performance, minimal external dependency bloat, and full native compatibility.
  - Official Elasticsearch v8 client provides typed query builders, connection pooling, cluster health pinging, and retry policies.
  - `amqp091-go` is the standard official RabbitMQ Go client used across the repository.

### 4. Network Topology & Isolation Boundaries

- **Decision**:
  - `search` network (isolated bridge): Encapsulates `search-back` and `search-engine`. `search-engine` has ZERO external network attachments.
  - `frontend` network: Shared with `search-back` on port 8080 for incoming API traffic from frontend/gateway.
  - `broker` network: Shared with `search-back` for RabbitMQ connection at `broker:5672`.
- **Rationale**:
  - Fulfills Constitution Principle I (Microservice Autonomy & Domain Isolation) and Principle VI (Zero-Trust Security & Boundary Auth).
  - Prevents other services (`auth`, `catalog`, `order`, `notification`) from directly querying or accessing the Elasticsearch instance.

### 5. Health Check & Observability Contract

- **Decision**: Implement `/api/health` endpoint verifying both Elasticsearch (`client.Ping()`) and RabbitMQ broker connectivity (`conn.IsClosed()`). Return HTTP 200 with `{"status":"ok","search_engine":"connected","broker":"connected"}` or HTTP 503 if any dependency is unreachable.
- **Rationale**:
  - Fulfills Clarification Session Question 4 and Constitution Principle V (End-to-End Observability & Tracing).
  - Allows platform readiness probes to distinguish between application crashes and dependency outages.

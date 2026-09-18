# Implementation Plan: Search Service Infrastructure

**Branch**: `002-search-service-infrastructure` | **Date**: 2026-09-15 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from [`specs/002-search-service-infrastructure/spec.md`](./spec.md)

## Summary

Build the complete containerized infrastructure and initial Go service skeleton for the autonomous "Search" microservice. This includes configuring a dedicated Dockerfile under `./infrastructure/search/` with non-root host permission mapping (UID/GID 1000), provisioning Elasticsearch 8.17 in `docker-compose.yaml` with persistent volume storage (`search-engine-data`) and single-node development configuration (256MB heap, `xpack.security.enabled=false`), establishing an isolated `search` network with selective attachments to `frontend` and `broker`, and implementing an initial Go service skeleton in `./services/search` with an `/api/health` endpoint verifying both Elasticsearch and RabbitMQ broker connectivity.

## Technical Context

**Language/Version**: Go 1.24+ (Alpine Linux)  
**Primary Dependencies**: `github.com/elastic/go-elasticsearch/v8`, `github.com/rabbitmq/amqp091-go`  
**Storage**: Elasticsearch 8.17 (`search-engine`, persistent volume `search-engine-data`)  
**Testing**: Go standard testing package (`testing`), automated container health validation  
**Target Platform**: Linux containers (Docker Compose orchestration)  
**Project Type**: Containerized Microservice Web Backend & Search Engine  
**Performance Goals**: Sub-300ms health check response time; JVM heap capped at 256MB  
**Constraints**: Zero cross-service datastore sharing; isolated `search` network; direct HTTP on port 8080 (no Nginx reverse proxy)  
**Scale/Scope**: 2 Docker containers (`search-back`, `search-engine`), 1 Dockerfile, 1 Go service skeleton  

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Requirement | Compliance Status | Notes |
| :--- | :--- | :--- | :--- |
| **I. Microservice Autonomy & Domain Isolation** | Standalone service, isolated search engine datastore | **PASS** | Dedicated `search-engine` container on private `search` network with dedicated volume `search-engine-data`. |
| **II. Broker-Only Inter-Service Comm** | Async broker integration via RabbitMQ | **PASS** | `search-back` joins `broker` network to produce and consume events. |
| **III. Frontend & Gateway Decoupling** | Frontend communicates via HTTP on port 8080 | **PASS** | `search-back` joins `frontend` network on port 8080. |
| **IV. Test-Driven & Contract Verification** | Verifiable endpoints & contracts | **PASS** | `/api/health` JSON contract specified in `contracts/health-api.json`. |
| **V. End-to-End Observability** | Structured health endpoints | **PASS** | `/api/health` verifies both Elasticsearch and RabbitMQ readiness. |
| **VI. Zero-Trust Security & Boundary Auth** | Least-privilege networking | **PASS** | `search-engine` has zero external network exposure; isolated exclusively to `search` network. |

## Project Structure

### Documentation (this feature)

```text
specs/002-search-service-infrastructure/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── health-api.json
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code Layout

```text
infrastructure/search/
└── Dockerfile           # Go 1.24 Alpine with UID/GID mapping and entrypoint

services/search/
├── cmd/
│   └── api/
│       └── main.go      # HTTP server entrypoint & router initialization
├── internal/
│   ├── config/
│   │   └── config.go    # Environment configuration loader
│   ├── handler/
│   │   ├── health.go    # /api/health handler verifying ES & Broker
│   │   └── search.go    # /api/search baseline query handler
│   ├── broker/
│   │   └── rabbitmq.go  # RabbitMQ connection & event handling
│   └── search/
│       └── client.go    # Elasticsearch v8 client wrapper & ping
├── go.mod               # Go module definition
└── go.sum

docker-compose.yaml      # Appended with search-back and search-engine services, search network, search-engine-data volume
```

**Structure Decision**: Standalone Go microservice topology consistent with `services/auth`, with direct HTTP server on port 8080 and dedicated Elasticsearch engine in an isolated network.

## Complexity Tracking

> **Constitution Check passed with zero violations.** No complexity exemptions required.

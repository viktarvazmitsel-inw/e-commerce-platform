# Implementation Plan: Order Service Infrastructure

**Branch**: `001-order-service-infrastructure` | **Date**: 2026-09-15 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from [`specs/001-order-service-infrastructure/spec.md`](./spec.md)

## Summary

Build the complete containerized infrastructure and initial Laravel skeleton for the autonomous "Order" microservice. This includes configuring Dockerfiles for PHP-FPM 8.5 and Nginx under `./infrastructure/order/`, provisioning PostgreSQL 18 and Redis 8.10 in `docker-compose.yaml` within an isolated `order` network, establishing local host permission mapping (UID/GID 1000), configuring port 8080 routing to PHP-FPM, and establishing an initial Laravel service skeleton with `/api/health` connectivity verification.

## Technical Context

**Language/Version**: PHP 8.5 (CLI & FPM on Alpine 3.23) / Composer 2.x  
**Primary Dependencies**: Laravel Framework 12/13, `ext-pdo_pgsql`, `ext-redis`, `ext-bcmath`, `ext-gd`, `ext-zip`  
**Storage**: PostgreSQL 18 (`shop_order_db`, persistent volume `order-db-data`), Redis 8.10 (in-memory cache)  
**Testing**: PHPUnit / Pest & automated container health checks  
**Target Platform**: Linux containers (Docker Compose orchestration)  
**Project Type**: Containerized Microservice Web Backend  
**Performance Goals**: Sub-200ms health check response time; zero volume permission locks on host  
**Constraints**: Zero cross-service database access; isolated `order` network; Nginx listening on port 8080  
**Scale/Scope**: 4 Docker containers (`order-back`, `order-server`, `order-db`, `order-cache`), 2 Dockerfiles, 1 Laravel service skeleton  

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Requirement | Compliance Status | Notes |
| :--- | :--- | :--- | :--- |
| **I. Microservice Autonomy & Domain Isolation** | Standalone service, isolated database & cache | **PASS** | Dedicated `shop_order_db` and `order-cache` on private `order` network. |
| **II. Broker-Only Inter-Service Comm** | Async broker integration via RabbitMQ | **PASS** | Connected to `broker` network for future event publishing/consuming. |
| **III. Frontend & Gateway Decoupling** | Frontend communicates via HTTP on port 8080 | **PASS** | Connected to `frontend` network on port 8080. |
| **IV. Test-Driven & Contract Verification** | Verifiable endpoints & contracts | **PASS** | `/api/health` JSON contract specified in `contracts/health-api.json`. |
| **V. End-to-End Observability** | Structured health endpoints | **PASS** | `/api/health` endpoint tests DB and Redis readiness. |
| **VI. Zero-Trust Security & Boundary Auth** | Least-privilege networking | **PASS** | DB and Cache are non-exposed to external networks. |

## Project Structure

### Documentation (this feature)

```text
specs/001-order-service-infrastructure/
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
infrastructure/order/
├── nginx/
│   ├── Dockerfile       # Nginx Alpine unprivileged image definition
│   └── default.conf     # Virtual host config (port 8080, FastCGI forward)
└── php/
    └── Dockerfile       # PHP 8.5 FPM with pdo_pgsql, redis, and host UID mapping

services/order/
├── app/
│   ├── Http/Controllers/HealthController.php
│   └── Providers/
├── bootstrap/
│   └── app.php
├── config/
│   ├── app.php
│   ├── database.php
│   └── cache.php
├── database/
│   └── migrations/
├── public/
│   └── index.php
├── routes/
│   └── api.php          # Health route definition (/api/health)
├── .env.example
├── artisan
└── composer.json

docker-compose.yaml      # Appended with order-back, order-server, order-db, order-cache
.env                     # Appended with ORDER_PG_USER, ORDER_PG_PASSWORD, ORDER_REDIS_PASSWORD
```

**Structure Decision**: Microservice multi-container topology matching the existing `catalog` service architecture pattern.

## Complexity Tracking

> **Constitution Check passed with zero violations.** No complexity exemptions required.

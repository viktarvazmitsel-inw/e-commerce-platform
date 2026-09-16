# Tasks: Search Service Infrastructure

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Branch**: `002-search-service-infrastructure`

---

## Phase 1: Setup (Directory Structure & Environment Configuration)

**Purpose**: Project directory scaffolding and shared root environment variables

- [x] T001 [P] Create directory structure for infrastructure and Go service in `infrastructure/search/` and `services/search/` (including `cmd/api`, `internal/config`, `internal/handler`, `internal/broker`, `internal/search`)
- [x] T002 [P] Configure environment variables for Search service (`SEARCH_ES_HOST`, `SEARCH_BROKER_URL`, `SEARCH_PORT`) in `.env`

---

## Phase 2: Foundational (Go Module & Configuration Skeleton)

**Purpose**: Core application files and configuration required for the Go backend service

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T003 Initialize Go module definition and dependencies for Elasticsearch v8 and RabbitMQ in `services/search/go.mod`
- [x] T004 [P] Implement environment configuration loader in `services/search/internal/config/config.go`
- [x] T005 [P] Setup environment configuration template in `services/search/.env.example`

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Local Development & Bidirectional File Permissions (Priority: P1) 🎯 MVP

**Goal**: Configure Go Alpine container with host user ID mapping (UID/GID 1000) and two-way volume binding to prevent host-container permission conflicts.

**Independent Test**: Create and edit a file in `services/search` from the host machine and verify ownership (UID/GID 1000) inside `search-back`.

- [x] T006 [P] [US1] Create Go Alpine Dockerfile with UID/GID mapping and `go run` entrypoint in `infrastructure/search/Dockerfile`
- [x] T007 [US1] Define `search-back` container service and volume bindings (`./services/search:/app`) in `docker-compose.yaml`

**Checkpoint**: At this point, the backend Go container is buildable and editable with bidirectional host file permissions.

---

## Phase 4: User Story 2 - HTTP Search API Ingress & Health Observability (Priority: P1)

**Goal**: Configure standalone Go HTTP server listening on port 8080 with `/api/health` and `/api/search` handlers.

**Independent Test**: Send HTTP GET to `http://<search-back>:8080/api/health` and verify HTTP 200 response with JSON health status.

- [x] T008 [P] [US2] Implement health check handler in `services/search/internal/handler/health.go` returning JSON matching `contracts/health-api.json`
- [x] T009 [P] [US2] Implement baseline search query handler in `services/search/internal/handler/search.go`
- [x] T010 [US2] Implement HTTP server entrypoint and route registrations in `services/search/cmd/api/main.go`

**Checkpoint**: At this point, HTTP requests on port 8080 route to Go backend and return health status.

---

## Phase 5: User Story 3 - Dedicated Search Engine Datastore & Index Isolation (Priority: P2)

**Goal**: Provision dedicated Elasticsearch 8.17 container in `docker-compose.yaml` and implement Go Elasticsearch v8 client connection and pinging.

**Independent Test**: Verify Elasticsearch cluster connectivity via `search-back` and data retention across container restarts.

- [x] T011 [P] [US3] Implement Elasticsearch v8 client connection wrapper and ping helper in `services/search/internal/search/client.go`
- [x] T012 [P] [US3] Define `search-engine` Elasticsearch container with 256MB heap, single-node mode, disabled xpack security, and `search-engine-data` volume in `docker-compose.yaml`

**Checkpoint**: At this point, Elasticsearch runs in single-node mode with 256MB heap and is accessible by the Go backend.

---

## Phase 6: User Story 4 - Asynchronous Event Publishing & Ingestion via Message Broker (Priority: P2)

**Goal**: Connect Search service to RabbitMQ broker to consume catalog domain events and produce search events.

**Independent Test**: Publish test message to RabbitMQ and verify `search-back` consumes the event.

- [x] T013 [P] [US4] Implement RabbitMQ connection management and event consumer/publisher in `services/search/internal/broker/rabbitmq.go`
- [x] T014 [US4] Integrate broker consumer and publisher lifecycle into `services/search/cmd/api/main.go`

**Checkpoint**: At this point, RabbitMQ connection is managed and monitored by the Go backend.

---

## Phase 7: User Story 5 - Network Isolation & Boundary Security (Priority: P3)

**Goal**: Isolate `search-engine` to the `search` network while attaching `search-back` to `search`, `frontend`, and `broker` networks.

**Independent Test**: Inspect `docker compose config` ensuring Elasticsearch has zero external network overlap.

- [x] T015 [US5] Configure isolated `search` network definition and network attachments for `search-back` and `search-engine` in `docker-compose.yaml`

**Checkpoint**: All Search containers run with strict network boundary isolation.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Validation and verification across all infrastructure components

- [x] T016 [P] Validate container build and startup across `search-back` and `search-engine` using `quickstart.md`
- [x] T017 [P] Verify `/api/health` response schema compliance against `specs/002-search-service-infrastructure/contracts/health-api.json`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3 - P1)**: Depends on Foundational phase
- **User Story 2 (Phase 4 - P1)**: Depends on User Story 1
- **User Story 3 (Phase 5 - P2)**: Depends on User Story 1 & 2
- **User Story 4 (Phase 6 - P2)**: Depends on User Story 1 & 2
- **User Story 5 (Phase 7 - P3)**: Finalizes network attachments across compose
- **Polish (Phase 8)**: Depends on all user stories complete

### Parallel Opportunities

- `T001` and `T002` in Setup can run in parallel
- `T004` and `T005` in Foundational can run in parallel
- `T006` [US1], `T008` [US2], `T009` [US2], `T011` [US3], and `T013` [US4] can be authored in parallel
- `T016` and `T017` in Polish can run in parallel

---

## Implementation Strategy

### MVP First (User Story 1 & 2)
1. Complete Setup (Phase 1) & Foundational (Phase 2)
2. Implement Go Alpine Dockerfile with UID/GID mapping (User Story 1)
3. Implement Go HTTP server listening on port 8080 with `/api/health` (User Story 2)
4. Validate HTTP ingress and bidirectional host file permissions (MVP achieved!)

### Full Feature Increment
5. Add Elasticsearch 8.17 container and Go client (User Story 3)
6. Add RabbitMQ broker connection and event worker (User Story 4)
7. Enforce isolated `search` network boundaries (User Story 5)
8. Run end-to-end quickstart validation (Phase 8)

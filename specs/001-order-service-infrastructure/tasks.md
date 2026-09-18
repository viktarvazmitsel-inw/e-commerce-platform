# Tasks: Order Service Infrastructure

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Branch**: `001-order-service-infrastructure`

---

## Phase 1: Setup (Directory Structure & Environment Configuration)

**Purpose**: Project directory scaffolding and shared root environment variables

- [X] T001 [P] Create directory structure for infrastructure in `infrastructure/order/php` and `infrastructure/order/nginx`
- [X] T002 [P] Configure environment variables for Order service in `.env`

---

## Phase 2: Foundational (Laravel Skeleton Initialization)

**Purpose**: Core application files and configuration required for the backend service

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Create Laravel skeleton `composer.json` and `artisan` CLI script in `services/order/`
- [X] T004 [P] Setup Laravel bootstrapping and app configuration in `services/order/bootstrap/app.php` and `services/order/config/app.php`
- [X] T005 [P] Configure public entrypoint in `services/order/public/index.php`
- [X] T006 Setup environment template in `services/order/.env.example`

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Local Development & Code Hot-Reloading (Priority: P1) 🎯 MVP

**Goal**: Configure PHP-FPM 8.5 container with host user ID mapping (UID/GID 1000) and two-way volume binding to prevent host-container permission conflicts.

**Independent Test**: Create/edit a file in `services/order` from the host machine and verify ownership (UID/GID 1000) inside `order-back`.

- [X] T007 [P] [US1] Create PHP-FPM Alpine Dockerfile with UID/GID mapping and extensions in `infrastructure/order/php/Dockerfile`
- [X] T008 [US1] Define `order-back` container service and volume bindings in `docker-compose.yaml`

**Checkpoint**: At this point, the backend PHP container is buildable with bidirectional host file permissions.

---

## Phase 4: User Story 2 - HTTP Request Routing & Observability via Web Server (Priority: P1)

**Goal**: Configure unprivileged Nginx web server listening on port 8080 proxying to `order-back:9000`, with an `/api/health` Laravel route.

**Independent Test**: Send HTTP GET to `http://<order-server>:8080/api/health` and verify HTTP 200 response with JSON health status.

- [X] T009 [P] [US2] Create Nginx configuration file in `infrastructure/order/nginx/default.conf`
- [X] T010 [P] [US2] Create Nginx Dockerfile in `infrastructure/order/nginx/Dockerfile`
- [X] T011 [US2] Define `order-server` container in `docker-compose.yaml` with port 8080 mapping and dependency on `order-back`
- [X] T012 [P] [US2] Implement HealthController in `services/order/app/Http/Controllers/HealthController.php`
- [X] T013 [US2] Register `/api/health` route in `services/order/routes/api.php`

**Checkpoint**: At this point, HTTP requests on port 8080 route to Laravel and return health status.

---

## Phase 5: User Story 3 - Isolated Data Persistence & Caching (Priority: P2)

**Goal**: Provision dedicated PostgreSQL 18 and Redis 8.10 containers and configure Laravel database and cache settings.

**Independent Test**: Run `php artisan migrate --force` and execute Redis read/write in `order-back`.

- [X] T014 [P] [US3] Configure database connections in `services/order/config/database.php` and cache in `services/order/config/cache.php`
- [X] T015 [P] [US3] Define `order-db` PostgreSQL 18 container and `order-db-data` volume in `docker-compose.yaml`
- [X] T016 [P] [US3] Define `order-cache` Redis 8.10 container in `docker-compose.yaml`
- [X] T017 [US3] Add baseline database migrations in `services/order/database/migrations/0001_01_01_000000_create_users_table.php`

**Checkpoint**: At this point, PostgreSQL and Redis services are isolated, running, and accessible to Laravel.

---

## Phase 6: User Story 4 - Inter-Service Network Boundary & Security (Priority: P3)

**Goal**: Isolate database and cache to `order` network while attaching web server and backend to `order`, `frontend`, and `broker` networks.

**Independent Test**: Inspect `docker compose config` ensuring database and cache are unreachable from external networks.

- [X] T018 [US4] Configure `order` network definition and network attachments for all Order services in `docker-compose.yaml`

**Checkpoint**: All 4 Order containers run with correct network boundaries.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Validation and verification across all infrastructure components

- [X] T019 [P] Validate container build and startup across all 4 containers using `quickstart.md`
- [X] T020 [P] Verify health check contract validation against `specs/001-order-service-infrastructure/contracts/health-api.json`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3 - P1)**: Depends on Foundational phase
- **User Story 2 (Phase 4 - P1)**: Depends on User Story 1 (for PHP-FPM upstream)
- **User Story 3 (Phase 5 - P2)**: Depends on User Story 1 & 2
- **User Story 4 (Phase 6 - P3)**: Finalizes network attachments across compose
- **Polish (Phase 7)**: Depends on all user stories complete

### Parallel Opportunities

- `T001` and `T002` in Setup can run in parallel
- `T004` and `T005` in Foundational can run in parallel
- `T007` [US1] and `T009`, `T010`, `T012` [US2] can be authored in parallel
- `T014`, `T015`, `T016` [US3] can be authored in parallel
- `T019` and `T020` in Polish can run in parallel

---

## Implementation Strategy

### MVP First (User Story 1 & 2)
1. Complete Setup (Phase 1) & Foundational (Phase 2)
2. Implement PHP-FPM container with UID mapping (User Story 1)
3. Implement Nginx web server on port 8080 with `/api/health` (User Story 2)
4. Validate HTTP ingress and file permissions (MVP achieved!)

### Full Feature Increment
5. Add PostgreSQL 18 & Redis 8.10 datastores (User Story 3)
6. Enforce isolated `order` network boundaries (User Story 4)
7. Run end-to-end quickstart validation (Phase 7)

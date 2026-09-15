# Feature Specification: Order Service Infrastructure

**Feature Branch**: `feature/TASK-1-infrastructure-setup-global`

**Created**: 2026-09-15

**Status**: Ready for Planning

**Input**: User description: "Build infrastructure for \"Order\" service. Modify docker-compose.yaml to add necessary containers. Service must use laravel for backend, nginx as server, postgreSQL as database and redis for caching. All containers must run in separate network to avoid intersections with other services. Nginx must listen port 8080, wait api requests from frontend container and redirect it to laravel. Put all necessary Dockerfiles inside ./infrastructure/order folder. And all necessary service's code inside ./services/order folder. Be sure that I have permissions to modify files inside laravel container from local machine. You can use \"Catalog\" service as an example. Ask questions if it's needed."

## Clarifications

### Session 2026-09-15

- Q: Should the initial Order service skeleton include a pre-configured `/api/health` endpoint that validates PostgreSQL and Redis connectivity? → A: Option A (Include `/api/health` returning JSON status with PostgreSQL and Redis connection verification).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Local Development & Code Hot-Reloading (Priority: P1)

As a backend engineer, I need the Order service to run locally within Docker containers with two-way volume binding and proper host user permissions so that I can modify Laravel source code and configuration files directly on my host machine without file permission errors or needing container rebuilds.

**Why this priority**: Core developer experience and productivity require instant code modification reflection and unblocked read/write permissions on the workspace files.

**Independent Test**: Can be fully tested by creating/editing a file in `services/order` from the host machine and inside the container to verify bidirectional editability and correct ownership (UID/GID 1000).

**Acceptance Scenarios**:

1. **Given** the Order containers are up and running, **When** a developer edits or creates a file in `./services/order` on the local machine, **Then** the file is immediately accessible inside the backend container with matching permissions and no permission denied errors occur.
2. **Given** the Laravel backend generates runtime logs, caches, or artisan files inside `/var/www/html/storage`, **When** inspected from the host machine, **Then** the files retain user-level ownership without requiring `sudo` to inspect or clean up.

---

### User Story 2 - HTTP Request Routing & Observability via Web Server (Priority: P1)

As a frontend client or API gateway, I need to send HTTP API requests to the Order web server on port 8080 so that requests are seamlessly proxied to the Laravel FastCGI backend and standard JSON responses are returned, including a health check endpoint for monitoring system status.

**Why this priority**: Essential communication ingress for all client and gateway interactions targeting the Order microservice and operational observability.

**Independent Test**: Can be fully tested by issuing an HTTP GET request to `http://<order-server>:8080/api/health` and receiving an HTTP 200 JSON payload verifying application, database, and cache health.

**Acceptance Scenarios**:

1. **Given** the frontend container is connected to the shared frontend network, **When** it issues an API request to `http://shop-order-server:8080/api/health`, **Then** the Nginx server forwards the request to the PHP-FPM backend and returns HTTP 200 OK with `{"status":"ok","database":"connected","cache":"connected"}`.
2. **Given** an invalid URI is requested, **When** sent to the Nginx server on port 8080, **Then** the request is forwarded to Laravel's front controller (`index.php`) and handled by standard routing fallback.

---

### User Story 3 - Isolated Data Persistence & Caching (Priority: P2)

As the Order microservice, I need dedicated PostgreSQL database and Redis caching containers in an isolated network so that order transactions and cache operations occur securely without risk of cross-service contamination or shared datastore conflicts.

**Why this priority**: Enforces microservice domain isolation, database encapsulation, and reliable cache management per the project constitution.

**Independent Test**: Can be fully tested by running Laravel database migrations and executing cache read/write operations from the Laravel container against the dedicated PostgreSQL and Redis instances.

**Acceptance Scenarios**:

1. **Given** the Order database container is initialized, **When** Laravel runs database migrations, **Then** tables are created in the dedicated `shop_order_db` PostgreSQL database.
2. **Given** the Order Redis instance is running, **When** Laravel writes or reads cache keys, **Then** the cache operation succeeds through the authenticated Redis connection.

---

### User Story 4 - Inter-Service Network Boundary & Security (Priority: P3)

As a system architect, I need the Order service containers to run in a dedicated Docker network while selectively exposing only the required network interfaces to the frontend and message broker networks.

**Why this priority**: Adheres to the principle of least privilege networking and domain isolation.

**Independent Test**: Can be verified by inspecting container network attachments ensuring database and cache are unreachable from external service networks while web server and backend can access required shared boundaries.

**Acceptance Scenarios**:

1. **Given** the multi-container environment is running, **When** network topology is inspected, **Then** the Order database and Redis containers belong exclusively to the `order` network.
2. **Given** the Order server and backend containers are running, **When** inter-service communication is evaluated, **Then** they can interact on the `frontend` network for client traffic and `broker` network for asynchronous messaging.

---

### Edge Cases

- **UID/GID Mismatch**: Handling systems where the host user is not UID/GID 1000 through customizable build arguments (`WWWUSER`/`WWWGROUP`).
- **Container Startup Order**: Backend or database not fully ready when Nginx starts, handled through proper container dependency configuration and reconnection resilience.
- **Database/Cache Unavailability on Health Check**: When database or Redis is unreachable during `/api/health` ping, return HTTP 503 Service Unavailable with descriptive status.
- **Persistent Volume Retention**: Preserving PostgreSQL and cache volume data across `docker-compose down` and `docker-compose up` cycles.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide dedicated Dockerfiles for PHP-FPM (Laravel backend) and Nginx under `./infrastructure/order/`.
- **FR-002**: System MUST include a complete Laravel project skeleton under `./services/order/` configured for PostgreSQL and Redis.
- **FR-003**: System MUST define the Order service containers (`order-back`, `order-server`, `order-db`, `order-cache`) in `docker-compose.yaml`.
- **FR-004**: System MUST configure an isolated `order` network in `docker-compose.yaml` to encapsulate internal Order service traffic.
- **FR-005**: System MUST join `order-server` and `order-back` to `order`, `frontend`, and `broker` networks to facilitate client ingress and asynchronous event publishing.
- **FR-006**: System MUST expose Nginx listening on port 8080 configured to proxy PHP requests to `order-back:9000`.
- **FR-007**: System MUST configure `order-back` with PHP extensions for PostgreSQL (`pdo_pgsql`, `pgsql`), Redis, and required Laravel runtime dependencies.
- **FR-008**: System MUST map host user permissions (UID/GID) inside the PHP container to prevent host-container file permission conflicts.
- **FR-009**: System MUST provision a dedicated PostgreSQL 18 container with persistent volume storage (`order-db-data`) and database `shop_order_db`.
- **FR-010**: System MUST provision a dedicated Redis container configured with memory policies and password authentication via environment variables.
- **FR-011**: System MUST provide an `/api/health` endpoint in the Laravel Order service verifying application, PostgreSQL, and Redis connectivity.

### Key Entities

- **Order Service**: The autonomous domain service encapsulating order lifecycle, processing, and checkout workflows.
- **Order Data Store**: Dedicated relational database (`shop_order_db`) containing order entities, items, payments, and state histories.
- **Order Cache Store**: Dedicated in-memory cache for fast session lookup, idempotency keys, and temporary transaction state.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Order service stack starts successfully with zero startup errors via `docker compose up -d`.
- **SC-002**: HTTP GET request to `http://localhost:8080/api/health` (or container network alias) routed via Nginx reaches Laravel and returns HTTP 200 OK with verified DB and Redis connectivity within 200ms in local environment.
- **SC-003**: Database migrations and Redis connection tests complete successfully within the Laravel container on initial setup.
- **SC-004**: Host files modified by developers in `./services/order` are immediately readable and writable by both host and container without `permission denied` errors or `root` ownership locks.
- **SC-005**: Zero network or datastore overlap with Catalog, Auth, or Notification services.

## Assumptions

- The host developer environment is Linux/macOS with standard user UID 1000 and GID 1000 (with support for overriding via `WWWUSER` and `WWWGROUP`).
- Standard environment variables `ORDER_PG_USER`, `ORDER_PG_PASSWORD`, and `ORDER_REDIS_PASSWORD` will follow project conventions in `.env`.
- Asynchronous messaging will integrate with RabbitMQ (`broker` network) when Order domain events are introduced.

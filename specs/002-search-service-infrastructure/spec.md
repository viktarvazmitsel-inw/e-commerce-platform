# Feature Specification: Search Service Infrastructure

**Feature Branch**: `feature/TASK-1-infrastructure-setup-global`

**Created**: 2026-09-15

**Status**: Ready for Planning

**Input**: User description: "Build infrastructure for \"Search\" service. Modify docker-compose.yaml to add all necessary containers. Service must use golang for backend, Elasticsearch as search engine. \"Search\" service must be isolated in separate network. Service must be able to accept api requests from frontend and produce and consume broker events. Use ./infrastructure/search folder for all necessary dockerfiles. Put service code inside ./services/search folder. Be sure I have permissions to modify files inside golang container from local machine."

## Clarifications

### Session 2026-09-15

- Q: How should Elasticsearch security and authentication be configured for the local development environment? (FR-011) → A: Option A (Disable security with `xpack.security.enabled=false` and use plaintext HTTP inside the isolated `search` Docker network).
- Q: Should the Go Search backend container use an automatic live-reload tool (like Air) or standard Go command execution for local development? (FR-001) → A: Option B (Use standard `go run ./cmd/api/main.go` and restart the container via `docker compose restart search-back` to apply changes).
- Q: Should the Go Search backend serve HTTP traffic directly on port 8080 or sit behind a dedicated Nginx reverse proxy container? (FR-004) → A: Option A (Expose HTTP on port 8080 directly from the Go backend container `search-back` without an Nginx reverse proxy).
- Q: What dependencies should the Search service `/api/health` endpoint verify during health checks? (FR-010) → A: Option A (Validate both Elasticsearch datastore and RabbitMQ message broker connectivity, returning `{"status":"ok","search_engine":"connected","broker":"connected"}`).
- Q: What JVM heap memory limit should be configured for Elasticsearch in `docker-compose.yaml` for local development? (FR-011) → A: Option C (Constrain JVM heap to 256MB via `ES_JAVA_OPTS: "-Xms256m -Xmx256m"`).


## User Scenarios & Testing *(mandatory)*

### User Story 1 - Local Development & Bidirectional File Permissions (Priority: P1)

As a backend engineer, I need the Search service Go backend to run locally in a Docker container with live volume mounts and user permission mapping so that I can create, edit, and compile Go files directly from my host machine without file permission conflicts, root lockouts, or container rebuild delays.

**Why this priority**: Core developer productivity and rapid feedback loop require unhindered two-way file access and immediate compilation/execution of Go code.

**Independent Test**: Can be fully tested by creating and editing files in `./services/search` from the local host and inside the running container, verifying identical UID/GID ownership and zero `permission denied` errors.

**Acceptance Scenarios**:

1. **Given** the Search service containers are running, **When** a developer edits or creates a Go file in `./services/search` on the host machine, **Then** the changes are instantly available inside the container at `/app` and take effect upon restarting the container (`docker compose restart search-back`) without requiring image rebuilds.
2. **Given** the Go backend writes files, caches, or build artifacts inside the container, **When** checked on the host machine, **Then** those files are owned by the local host user (e.g., UID/GID 1000) and can be modified or deleted without `sudo`.

---

### User Story 2 - HTTP Search API Ingress & Health Observability (Priority: P1)

As a frontend application or API gateway, I need to send HTTP API requests (including search queries and health status checks) to the Search service so that search results and service readiness information are delivered in standard JSON format.

**Why this priority**: Primary ingress interface for client search interactions and platform observability.

**Independent Test**: Can be fully tested by issuing an HTTP GET request to `http://<search-service>:8080/api/health` and receiving an HTTP 200 JSON payload verifying application health and Elasticsearch engine connectivity.

**Acceptance Scenarios**:

1. **Given** the Search service is running and connected to the `frontend` network, **When** a client sends an HTTP GET request to `/api/health`, **Then** the service responds with HTTP 200 OK and JSON `{"status":"ok","search_engine":"connected","broker":"connected"}`.
2. **Given** the Search service is running, **When** a search query request is received on port 8080, **Then** the Go backend processes the request against the search engine and returns matching results in structured JSON format.

---

### User Story 3 - Dedicated Search Engine Datastore & Index Isolation (Priority: P2)

As the Search microservice, I need a dedicated Elasticsearch instance running in an isolated network with persistent data volume storage so that search indexes, mappings, and document ingestions are securely encapsulated without cross-service data sharing.

**Why this priority**: Enforces microservice domain autonomy and persistence isolation in accordance with the project constitution.

**Independent Test**: Can be fully tested by indexing documents and executing search queries against the Elasticsearch instance from the Go backend container, verifying data persistence across container restarts.

**Acceptance Scenarios**:

1. **Given** the Elasticsearch container is initialized with persistent volume storage, **When** the Go service inserts or updates search index documents, **Then** the documents are indexed and retrievable via search queries.
2. **Given** indexed documents exist in Elasticsearch, **When** the containers are stopped and restarted via `docker compose down` and `docker compose up`, **Then** the search indexes and documents persist without data loss.

---

### User Story 4 - Asynchronous Event Publishing & Ingestion via Message Broker (Priority: P2)

As the Search microservice, I need to connect to the central message broker (RabbitMQ) to consume domain events (such as catalog product updates) and produce search events so that search indexes remain synchronized asynchronously with catalog changes.

**Why this priority**: Fulfills Constitution Principle II (Broker-Only Inter-Service Communication) for asynchronous event-driven synchronization across decoupled microservices.

**Independent Test**: Can be fully tested by publishing test messages to the message broker and verifying that the Search service consumes and processes them into index updates, and publishes confirmation/audit events.

**Acceptance Scenarios**:

1. **Given** the message broker is active on the `broker` network, **When** the Search service initializes, **Then** it establishes a persistent connection to the broker and binds consumer queues for catalog update events.
2. **Given** an event is published to the broker, **When** received by the Search service consumer, **Then** the Search service processes the payload and updates the corresponding search index.

---

### User Story 5 - Network Isolation & Boundary Security (Priority: P3)

As a system architect, I need the Search service infrastructure to run in a dedicated Docker network while selectively exposing interfaces only to the `frontend` and `broker` networks.

**Why this priority**: Adheres to the principle of least privilege networking and domain isolation.

**Independent Test**: Can be verified by inspecting Docker network topologies to ensure Elasticsearch is unreachable from other microservice networks (auth, order, catalog, note) and only accessible by the Search backend in the `search` network.

**Acceptance Scenarios**:

1. **Given** the multi-container environment is running, **When** network topology is inspected, **Then** the Elasticsearch container belongs exclusively to the isolated `search` network.
2. **Given** the Search Go backend is running, **When** inter-service ingress and egress are evaluated, **Then** the backend joins `search`, `frontend`, and `broker` networks to communicate with its datastore, client ingress, and event messaging.

---

### Edge Cases

- **Host UID/GID Mismatch**: Local developer user UID/GID may differ from 1000:1000; resolved using configurable build arguments (`WWWUSER`/`WWWGROUP` or environment variables) in the Dockerfile.
- **Search Engine Initialization Delay**: Elasticsearch requires more time to become healthy upon boot than the Go container; backend must implement reconnection backoff/retries when initializing index connections.
- **Single-Node Resource Configuration**: Elasticsearch in development requires single-node cluster settings (`discovery.type=single-node`) and JVM heap allocation constrained to 256MB (`ES_JAVA_OPTS: "-Xms256m -Xmx256m"`) to prevent out-of-memory container crashes and excessive host RAM consumption.
- **Broker Unavailability**: Temporary message broker connection drops must trigger automatic reconnection mechanisms in the Go background worker.
- **Health Check Degradation**: If Elasticsearch is unreachable during an `/api/health` probe, the endpoint must return HTTP 503 Service Unavailable with structured error details rather than crashing.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a dedicated `Dockerfile` for the Go Search backend under `./infrastructure/search/`.
- **FR-002**: System MUST configure the Go backend Dockerfile to support host user UID/GID build arguments to ensure read/write file permissions on the local machine.
- **FR-003**: System MUST create a Go service skeleton under `./services/search/` with module definition (`go.mod`), application entrypoint (`cmd/api/main.go`), and internal packages for HTTP routing, Elasticsearch integration, and broker messaging.
- **FR-004**: System MUST define the Search backend container (`search-back`) in `docker-compose.yaml` with volume mount `./services/search:/app`, serving HTTP traffic directly on port 8080 without an Nginx reverse proxy.
- **FR-005**: System MUST provision a dedicated Elasticsearch container (`search-engine`) in `docker-compose.yaml` with persistent volume storage (`search-engine-data`).
- **FR-006**: System MUST define an isolated `search` network in `docker-compose.yaml` dedicated to the Search service components.
- **FR-007**: System MUST connect the Search backend container to `search`, `frontend`, and `broker` networks.
- **FR-008**: System MUST isolate the Elasticsearch container exclusively inside the `search` network with no direct external or cross-service network attachments.
- **FR-009**: System MUST configure the Search backend to listen for HTTP API requests on port 8080.
- **FR-010**: System MUST provide an `/api/health` endpoint in the Go backend that verifies service readiness and active connectivity to both Elasticsearch and the RabbitMQ message broker, returning JSON `{"status":"ok","search_engine":"connected","broker":"connected"}` with HTTP 200 (or HTTP 503 if any dependency is unreachable).
- **FR-011**: System MUST configure Elasticsearch environment settings for local development with `discovery.type=single-node`, `xpack.security.enabled=false`, and JVM heap capped at 256MB (`ES_JAVA_OPTS: "-Xms256m -Xmx256m"`), connecting via plaintext HTTP within the isolated `search` Docker network.
- **FR-012**: System MUST configure environment variable parameters for Elasticsearch host, port, and broker credentials in `.env` and `docker-compose.yaml`.

### Key Entities

- **Search Service Backend**: Golang service handling HTTP search queries, document indexing, broker event consumption, and query validation.
- **Search Engine Datastore**: Dedicated Elasticsearch instance storing search indexes, document schemas, and inverted index mappings.
- **Search Index**: Structured data representation of searchable domain objects (e.g. catalog products, tags, categories).
- **Domain Event Message**: Asynchronous payload transmitted via RabbitMQ containing entity creation, modification, or deletion triggers.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Search service containers start cleanly without errors via `docker compose up -d`.
- **SC-002**: HTTP GET request to `http://localhost:8080/api/health` (or container network alias) returns HTTP 200 OK with verified Elasticsearch and RabbitMQ broker connectivity within 300ms in local environment.
- **SC-003**: Host developer can create, edit, and delete files inside `./services/search` without encountering permission errors or root-locked files.
- **SC-004**: Search index creation and document query operations execute successfully between the Go backend and Elasticsearch container.
- **SC-005**: Elasticsearch container is completely isolated from other microservice networks (auth, order, catalog, note) with zero network overlap outside the designated `search` network.

## Assumptions

- Standard local developer environment is Linux/macOS with default UID 1000 and GID 1000, configurable via build args/env vars.
- Central message broker is RabbitMQ available on the shared `broker` network at `broker:5672`.
- Elasticsearch 8.x (or 7.17.x compatible image) will be used with single-node development configuration.
- Search service accepts incoming HTTP traffic on port 8080 within the `frontend` network for direct frontend/gateway integration.

<!--
SYNC IMPACT REPORT
Version change: 1.0.0 → 1.1.1
Modified principles:
- Principle 1: I. Microservice Boundary & Domain Isolation → I. Microservice Autonomy & Domain Isolation (Strengthened self-contained application requirements and zero cross-service dependencies)
- Principle 2: II. Contract-First & Asynchronous Messaging → II. Broker-Only Inter-Service Communication (Mandated all inter-service communication to route exclusively through the message broker)
- Principle 3: III. Test-Driven & Contract Verification (NON-NEGOTIABLE) (Retained)
- Principle 4: IV. End-to-End Observability & Tracing (Retained)
- Principle 5: V. Zero-Trust Security & Boundary Auth (Retained)
Added sections:
- Principle III: Frontend & Gateway Decoupling
- Architecture & Infrastructure Standards: Frontend & API Gateway Separation
Removed sections: None
Follow-up TODOs: None
-->

# Shop Constitution

## Core Principles

### I. Microservice Autonomy & Domain Isolation
Every microservice (Auth, Catalog, Order, Search, Notification, Analytics) MUST be a completely self-contained, independently deployable application. Services MUST NOT depend directly on other services. Cross-service direct database access or shared datastores are strictly forbidden; each service manages its own isolated storage lifecycle.

### II. Broker-Only Inter-Service Communication
All communication, data exchange, and event propagation between microservices MUST occur exclusively through the central message broker. Direct synchronous service-to-service calls (e.g. direct HTTP/RPC between internal backend services) are strictly prohibited. All broker messages and events MUST adhere to versioned schemas with idempotent consumers and dead-letter queue (DLQ) handling.

### III. Frontend & Gateway Decoupling
The frontend client application is strictly separated from the backend microservices. The frontend MUST communicate with the backend exclusively via the API Gateway. Direct exposure or direct network access from the frontend to internal microservices is possible only for developing and testing concrete microservice.

### IV. Test-Driven & Contract Verification (NON-NEGOTIABLE)
All service capabilities, message contracts, event schemas, and gateway endpoints MUST be covered by automated test suites prior to merging. Unit tests, contract verification, and asynchronous message flow tests MUST pass in continuous integration before deployment.

### V. End-to-End Observability & Tracing
Every service, gateway, and message broker publisher/consumer MUST emit structured JSON logs and propagate correlation/trace IDs across gateway and broker boundaries. Every service MUST expose standardized health and readiness endpoints.

### VI. Zero-Trust Security & Boundary Auth
Authentication tokens MUST be validated at the API Gateway and service boundaries. Secrets MUST never be committed in plaintext, and services MUST operate under least-privilege networking.

## Architecture & Infrastructure Standards
- **Containers & Orchestration**: All modules (frontend, gateway, broker, services) MUST run by single `docker-compose.yaml`. Each service (analytics, auth, catalog, notification, order, search) MUST be able to run separately with frontend module.
- **Gateway & Ingress**: The API Gateway acts as the single entry point for all external client traffic, handling SSL termination, request routing, rate limiting, but not for authentication.
- **Message Broker**: Serves as the backbone for all asynchronous inter-service coordination, publish/subscribe events, and command distribution.
- **Data Persistence**: Each service encapsulates its dedicated data store with zero shared persistence layers.

## Development Workflow & Quality Gates
- **Branching & Pull Requests**: Feature development branches from `develop` and MUST be called like `feature/some-new-feature`. All pull requests require passing automated test suites, linter verification, and peer review.
- **Spec Kit Compliance**: All feature implementations MUST follow the Spec-Driven Development lifecycle (`spec` -> `plan` -> `tasks` -> `implement`).

## Governance
- This constitution is the authoritative foundation for all architectural and design decisions, superseding informal guidelines.
- Amendments require a documented pull request, architectural justification, and semantic version bump:
  - **MAJOR**: Breaking changes to foundational principles or architectural paradigms.
  - **MINOR**: Additions of new principles, structural constraints, or materially expanded guidance.
  - **PATCH**: Clarifications, wording refinements, and non-semantic corrections.
- All code reviews, design docs, and CI checks MUST verify compliance with these principles.

**Version**: 1.1.0 | **Ratified**: 2026-09-07 | **Last Amended**: 2026-09-07

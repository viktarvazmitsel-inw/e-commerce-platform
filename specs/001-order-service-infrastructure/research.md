# Research: Order Service Infrastructure

**Date**: 2026-09-15
**Feature**: [spec.md](./spec.md)

## Technical Analysis & Decisions

### 1. PHP-FPM & Laravel Runtime (`order-back`)

- **Decision**: Use `php:8.5-fpm-alpine3.23` with `shadow` package to configure UID/GID 1000 on `www-data`, matching `infrastructure/catalog/php/Dockerfile`.
- **Rationale**:
  - Ensures seamless parity with existing Catalog service.
  - Resolves file permission issues when writing/editing files from host machine without root locks.
  - Installs required extensions (`pdo_pgsql`, `pgsql`, `redis`, `bcmath`, `gd`, `zip`) directly on Alpine for minimal image size and maximum performance.
- **Alternatives Considered**:
  - *Standard Debian PHP image*: Heavier footprint (>400MB vs ~80MB Alpine), longer build times.
  - *Root execution inside container*: Causes permission denied errors on the host machine when artisan/composer generates files.

### 2. Web Server Ingress (`order-server`)

- **Decision**: Use unprivileged Nginx (`nginxinc/nginx-unprivileged:alpine`) listening on port 8080, forwarding PHP requests to `order-back:9000`.
- **Rationale**:
  - Standardizes port 8080 listening across microservices in the Shop project.
  - Runs unprivileged for improved container security posture.
  - FastCGI configuration forwards all dynamic route requests to `/var/www/html/public/index.php`.
- **Alternatives Considered**:
  - *PHP built-in server (`php artisan serve`)*: Not suitable for multi-process concurrency or production-like setups.
  - *Apache mod_php*: Heavier resource usage and less flexible FastCGI process decoupling.

### 3. Isolated Storage & Caching Layer (`order-db` and `order-cache`)

- **Decision**:
  - Database: `postgres:18.6-alpine` running on private network `order`, persistent volume `order-db-data`, database name `shop_order_db`.
  - Cache: `redis:8.10` on private network `order`, LRU eviction (`allkeys-lru`, 256MB limit), password protected.
- **Rationale**:
  - Strictly follows Constitution Principle I (Microservice Autonomy & Domain Isolation) and Principle VI (Least Privilege Networking).
  - Database and Cache containers are strictly attached to `order` network and isolated from `frontend`, `catalog`, `auth`, `note`, and `broker`.
- **Alternatives Considered**:
  - *Shared Postgres cluster with multiple schemas*: Violates microservice autonomy and independent lifecycle management.

### 4. Network Topology & Boundaries

- **Decision**:
  - `order` network: Private to `order-back`, `order-server`, `order-db`, `order-cache`.
  - `frontend` network: Shared with `order-server` (and `order-back` for ingress/routing) for client/gateway API access.
  - `broker` network: Shared with `order-back` for asynchronous messaging with RabbitMQ.
- **Rationale**: Prevents cross-service database access while enabling required external HTTP ingress and broker events.

### 5. Health Check Endpoint Implementation

- **Decision**: Provide `/api/health` in `routes/api.php` utilizing `DB::connection()->getPdo()` and `Redis::ping()`.
- **Rationale**: Enables immediate verification of end-to-end container health, Nginx proxying, database connection, and Redis availability.

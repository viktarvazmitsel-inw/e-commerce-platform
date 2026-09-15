# Quickstart & Validation Guide: Order Service Infrastructure

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

This guide details how to spin up and validate the Order microservice infrastructure locally.

---

## 1. Prerequisites

- Docker and Docker Compose installed and daemon running.
- Host user with UID 1000 and GID 1000 (or `WWWUSER`/`WWWGROUP` exported in environment).
- Root `.env` file configured with `ORDER_PG_USER`, `ORDER_PG_PASSWORD`, and `ORDER_REDIS_PASSWORD`.

---

## 2. Infrastructure Setup & Startup

### Step 1: Initialize Service Environment
```bash
cp services/order/.env.example services/order/.env
```

### Step 2: Build and Start Containers
```bash
docker compose up -d order-back order-server order-db order-cache
```

Verify that all 4 containers are in `running` status:
```bash
docker compose ps | grep order
```

---

## 3. End-to-End Validation Steps

### Test 1: Web Server & PHP-FPM Ingress (Port 8080)
Execute an HTTP GET request to the health endpoint via the web server container:
```bash
# From host via mapped port or via docker exec
docker compose exec order-server curl -s http://localhost:8080/api/health
```
**Expected Output**:
```json
{
  "status": "ok",
  "service": "order-service",
  "database": "connected",
  "cache": "connected"
}
```

### Test 2: Database Migration Execution
Run Laravel database migrations inside the backend container:
```bash
docker compose exec order-back php artisan migrate --force
```
**Expected Output**:
```text
Migration table created successfully.
Migrating: 0001_01_01_000000_create_users_table
...
Migrated: (success)
```

### Test 3: Cache Verification
Test Redis connection and read/write from artisan tinker or PHP command:
```bash
docker compose exec order-back php -r "Illuminate\Support\Facades\Redis::set('order:test', 'ready'); echo Illuminate\Support\Facades\Redis::get('order:test');"
```
**Expected Output**:
```text
ready
```

### Test 4: Host File Permissions & Hot-Reloading
Create and modify a test file from the host machine and verify inside the container without permission errors:
```bash
echo "<?php // permission test" > services/order/test_perm.php
docker compose exec order-back ls -la /var/www/html/test_perm.php
rm services/order/test_perm.php
```
**Expected Output**: File is readable and owned by `www-data:www-data` (UID 1000).

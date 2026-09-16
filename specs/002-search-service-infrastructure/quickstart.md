# Quickstart & Validation Guide: Search Service Infrastructure

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

This guide details how to start up and validate the Search microservice infrastructure locally.

---

## 1. Prerequisites

- Docker and Docker Compose installed and daemon running.
- Host user with UID 1000 and GID 1000 (or `WWWUSER`/`WWWGROUP` exported in environment).
- Shared `broker` (RabbitMQ) container available on the `broker` network.

---

## 2. Infrastructure Setup & Startup

### Step 1: Start Search Service Containers
```bash
docker compose up -d search-back search-engine
```

Verify that both containers are running:
```bash
docker compose ps | grep search
```

---

## 3. End-to-End Validation Steps

### Test 1: Health Check & Observability (Port 8080)
Execute an HTTP GET request to the health endpoint:
```bash
docker compose exec search-back wget -qO- http://localhost:8080/api/health
```
or from the host / client network:
```bash
curl -s http://localhost:8080/api/health
```

**Expected Output**:
```json
{
  "status": "ok",
  "service": "search-service",
  "search_engine": "connected",
  "broker": "connected"
}
```

### Test 2: Elasticsearch Cluster State
Verify that Elasticsearch is accessible only from within the `search` network:
```bash
docker compose exec search-back wget -qO- http://search-engine:9200
```
**Expected Output**: HTTP 200 JSON payload with cluster name `docker-cluster` and tag line `"You Know, for Search"`.

### Test 3: Elasticsearch Data Persistence
Verify index creation and data retention across container restarts:
```bash
# 1. Create a test document
docker compose exec search-back wget --post-data='{"name":"Test Item"}' --header="Content-Type: application/json" -qO- http://search-engine:9200/test-index/_doc/1

# 2. Restart search-engine container
docker compose restart search-engine

# 3. Retrieve document after restart
docker compose exec search-back wget -qO- http://search-engine:9200/test-index/_doc/1
```
**Expected Output**: Document with `_id: 1` is returned successfully.

### Test 4: Host File Permissions & Editability
Verify that files created in `./services/search` from the host can be modified inside the container without permission errors:
```bash
echo "// test" > services/search/test_perm.go
docker compose exec search-back ls -la /app/test_perm.go
rm services/search/test_perm.go
```
**Expected Output**: Zero permission errors; file is owned by the local host user (UID 1000).

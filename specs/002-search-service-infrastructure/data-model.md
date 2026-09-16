# Data Model & Configuration Schema: Search Service Infrastructure

**Date**: 2026-09-15
**Feature**: [spec.md](./spec.md)

## 1. Environment & Configuration Schema

The Search service configuration is driven by standard environment variables in `.env` and injected into container runtimes.

| Variable Name | Default Value | Description | Sensitivity |
| :--- | :--- | :--- | :--- |
| `SEARCH_ES_HOST` | `http://search-engine:9200` | Elasticsearch endpoint URL | Low |
| `SEARCH_BROKER_URL` | `amqp://guest:guest@broker:5672/` | RabbitMQ connection URL | Medium |
| `SEARCH_PORT` | `8080` | Search service HTTP server port | Low |
| `WWWUSER` | `1000` | Host user ID for volume write permissions | Low |
| `WWWGROUP` | `1000` | Host group ID for volume write permissions | Low |

---

## 2. Container Service Models & Network Topology

```mermaid
classDiagram
    class SearchBack {
        +Image: golang:1.24-alpine
        +Port: 8080 (HTTP)
        +Networks: search, frontend, broker
        +Volume: ./services/search:/app
        +User: 1000:1000
    }
    class SearchEngine {
        +Image: elasticsearch:8.17.2
        +Port: 9200 (Internal)
        +Network: search
        +Volume: search-engine-data:/usr/share/elasticsearch/data
        +Heap: 256MB
    }

    SearchBack --> SearchEngine : HTTP (port 9200, plaintext)
    SearchBack --> Broker : AMQP (port 5672)
    Frontend --> SearchBack : HTTP (port 8080)
```

---

## 3. Elasticsearch Document Schemas (Baseline Index Model)

The Search service encapsulates Elasticsearch index management for searchable items (e.g. products, categories).

### `products` Index Mapping (Baseline):

```json
{
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "name": { "type": "text", "analyzer": "standard" },
      "description": { "type": "text" },
      "price": { "type": "double" },
      "category_id": { "type": "keyword" },
      "tags": { "type": "keyword" },
      "created_at": { "type": "date" },
      "updated_at": { "type": "date" }
    }
  }
}
```

---

## 4. Message Broker Event Contracts (Baseline)

Asynchronous domain event payloads consumed from RabbitMQ to maintain index synchronization.

### Event: `product.updated` / `product.created`:
- **Exchange**: `shop.events`
- **Routing Key**: `catalog.product.*`
- **Payload**:
  ```json
  {
    "event_id": "uuid",
    "event_type": "product.created",
    "timestamp": "2026-09-15T16:00:00Z",
    "payload": {
      "id": "prod_123",
      "name": "Wireless Headphones",
      "price": 99.99,
      "category_id": "cat_456"
    }
  }
  ```

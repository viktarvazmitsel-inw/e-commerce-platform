# Notification service

Go module `github.com/shop/services/notification`. Replace the placeholder module path `cmd/api/main.go` before adding packages.

This service owns these jobs:

1. Consume versioned notification commands from RabbitMQ, including the email commands Auth already publishes.
2. Persist one notification record per accepted command in this service's MongoDB.
3. Send the corresponding email through the `EmailSender` port.
4. Serve `GET` list and `GET` by id to callers whose access token role includes Analyst or Admin.

`cmd/api/main.go` is a broker probe: it dials `amqp://guest:guest@broker:5672/`, consumes queue `test`, and auto-acks. Do not copy it. Wire the process in `cmd/api` from config, with manual acks.

Follow the layering below. The service is not implemented yet; match this document when adding behavior.

## Dependency rule

Dependencies point inward. The core is `internal/domain` and `internal/usecase`.

| Layer | Path | May import |
|---|---|---|
| Domain | `internal/domain` | Standard library and other packages under `internal/domain` |
| Use cases | `internal/usecase` | Standard library and `internal/domain` |
| HTTP | `internal/interface/http` | Standard library, `internal/usecase`, `internal/domain`, HTTP router (`chi`) |
| Infrastructure | `internal/infrastructure` | Drivers (MongoDB, AMQP, JWT, mail), `internal/domain`, `internal/usecase` ports |
| Config | `internal/config` | Standard library |
| Composition | `cmd/api` | Every layer, to construct and inject dependencies |

Hard rules:

- Do not import third-party modules from `internal/domain` or `internal/usecase`.
- Do not import `internal/infrastructure` or `internal/config` from handlers, HTTP middleware, or use cases.
- HTTP handlers call a use case. They do not open MongoDB, RabbitMQ, or JWT code, and they do not send mail.
- The broker consumer is an infrastructure adapter. It decodes a delivery and calls a use case. It does not decide status transitions.
- Only `cmd/api` constructs infrastructure and passes it into use-case and middleware constructors.
- Do not call Auth, Catalog, Order, Search, or any other service over HTTP or RPC. Do not read their databases. Inter-service input arrives only on the broker.

## Layout

- `internal/domain` — `Notification`, role bits, sentinel errors for broken invariants.
- `internal/usecase` — one use case per file. Ports live in `interfaces.go`. Application sentinel errors live in `errors.go`.
- `internal/interface/http/handler` — one handler per query use case. Shared JSON helpers are `request.go` and `response.go`.
- `internal/interface/http/middleware` — access-token verification. Depends on a `TokenParser` port, not on a JWT library.
- `internal/infrastructure/db/mongo` — `notifications` collection. Map with document `ToDomain` / `FromDomain`. Do not leak the Mongo document type past this package.
- `internal/infrastructure/broker` — consume `email_exchange`, ack, and dead-letter. No mail library in this package.
- `internal/infrastructure/mail` — the only package that may import a mail library. Implements `EmailSender`. The concrete transport is not chosen yet; do not add provider SDK calls anywhere else, and do not name a provider in new use-case code.
- `internal/infrastructure/token` — RS256 access-token parse. Implements `TokenParser`.
- `internal/infrastructure/logger` — `slog` JSON adapter for the `Logger` port.
- `internal/config` — environment only. No credentials in source.

## Broker contract

Exchange `email_exchange` is direct and durable. Auth declares it. Consume it; do not redeclare it with a different type.

Queue `notification.email`: durable, quorum. Dead-letter queue `notification.email.dlq`. Bind routing keys `verification`, `password_reset`, and `email`.

Manual ack. Prefetch is bounded. Auto-ack is forbidden.

Two bodies are valid on this exchange:

Auth compatibility (routing key `verification` or `password_reset`, no `version` field):

```json
{"type":"verification","email":"user@example.com","token":"<one-time token>"}
```

`type` is `verification` or `password_reset` and matches the routing key. `email` and `token` are required.

Versioned command (routing key `email`). New publishers use this shape:

```json
{
  "version": 1,
  "id": "019a0b2c-7c3e-7a1e-8c3a-9b0e1f2a3b4c",
  "channel": "email",
  "type": "verification",
  "recipient": "user@example.com",
  "data": {"token": "<one-time token>"}
}
```

`version` must be `1`. `channel` must be `email`. `id` is required and is the idempotency key. `type` is a non-empty string. `recipient` is required. `data` is a string map; email types that need a one-time token require `data.token`.

Reject anything else as poison: invalid JSON, unknown version, missing recipient, empty type, or a compatibility body whose `type` disagrees with the routing key. Publish the original body to `notification.email.dlq` and ack the source delivery. Do not retry poison messages and do not insert a row for them.

Read `correlation_id` from AMQP headers when present. Generate one when it is absent. Log it and store it on the notification.

Idempotency key:

- Versioned command: `id`.
- Auth compatibility body: hex SHA-256 of `type`, `email`, and `token` joined with newlines. Store the hash only.

A second delivery with the same key does not insert another row and does not send another email when status is `sent` or `failed`. When status is `pending`, retry the send.

## Processing a command

Use case shape:

```go
type ProcessNotificationUseCase struct { /* unexported port fields */ }

func NewProcessNotificationUseCase(r NotificationRepository, s EmailSender, l Logger) *ProcessNotificationUseCase

func (uc *ProcessNotificationUseCase) Execute(ctx context.Context, cmd Command) error
```

`Execute` order:

1. Build the idempotency key. Load by that key.
2. When the row is `sent` or `failed`, return nil.
3. When no row exists, insert status `pending` without the raw token.
4. Call `EmailSender.Send` with recipient, type, and template data. The token exists only in that in-memory value.
5. On success, set status `sent` and return nil. The consumer acks.
6. On a permanent send failure, set status `failed`, record the reason, and return nil. The consumer acks.
7. On a transient send or database failure, return the wrapped error. The consumer nacks so the broker can redeliver. Do not ack a delivery whose outcome is still `pending`.

Known v1 types are `verification` and `password_reset`. An unknown type is a permanent failure: persist `failed` and ack. Add a new type in the domain allow-list and in the mail adapter together.

Do not store raw one-time tokens, passwords, or other secrets in MongoDB, logs, or HTTP responses. Log recipient, type, notification id, status, and correlation id.

## Domain

`Notification` fields: id, idempotency key, channel (`email`), type, recipient, status (`pending`, `sent`, `failed`), failure reason, correlation id, created at, updated at.

Transitions live on the entity. `MarkSent` is valid only from `pending`. `MarkFailed` is valid only from `pending`. A second transition returns a domain sentinel.

Roles use the same bit values as Auth's access-token `role` claim (decimal string of the bit field):

| Role | Value |
|---|---|
| Client | 1 |
| Analyst | 2 |
| Admin | 4 |

`CanReadNotifications` is true when the value includes Analyst or Admin. Client alone cannot read. Combined bits such as Admin+Analyst (`6`) can read. This check is local. Do not import Auth's permission map; Analyst has no Auth permissions and still may read notifications.

## Use cases

Name the file and type after the action (`process_notification.go`, `ProcessNotificationUseCase`, `list_notifications.go`, `get_notification.go`).

- Keep `Command`, `Input`, and result structs in that file.
- Add a port to `interfaces.go` when a use case needs persistence, mail, or logging. Depend on the interface, never on a concrete repository or client.
- `ListNotifications` and `GetNotification` call `CanReadNotifications` first and return `domain.ErrForbidden` for any other role.
- `GetNotification` returns `usecase.ErrNotificationNotFound` when the id is absent.
- List is newest first. `page` starts at 1. Default limit is 20. Maximum limit is 100.
- Return an existing sentinel from `usecase/errors.go` when callers must branch on it.
- Wrap unexpected infrastructure errors with `%w`. A database outage is not a new sentinel.

`EmailSender` accepts recipient, notification type, and template data. It does not accept a Mongo document. Rendering the message is the adapter's job.

## Handlers

A handler depends on a small local interface, not on the concrete use case:

```go
type GetNotificationUseCase interface {
    Execute(ctx context.Context, input usecase.GetNotificationInput) (domain.Notification, error)
}
```

Routes:

| Method and path | Behavior |
|---|---|
| `GET /notifications` | Page of every notification. Query `page`, `limit`. |
| `GET /notifications/{id}` | One notification by id. |
| `GET /health` | Process is up. |
| `GET /ready` | MongoDB and the broker connection are usable. |

`ServeHTTP` order:

1. Reject the wrong HTTP method with `405` and `METHOD_NOT_ALLOWED`.
2. Read the principal from the context set by middleware. A missing principal is `401`.
3. Read `page` and `limit` from the query on the list route. Reject non-integers and `limit` above 100 with `400`.
4. Call `Execute`.
5. Map errors with `errors.Is` in `handleUseCaseError`. `domain.ErrForbidden` is `403`. `usecase.ErrNotificationNotFound` is `404`. Unknown errors are `500` with a generic message.

Response envelope is `Response` (`success`, `data`, `errors`). `errors` is `APIError` (`code`, `message`, optional `details`). Write it only through `RespondJSON`. List `data` includes the items and the total count. A notification DTO exposes the domain fields and never a token.

Health and readiness are handlers wired in `cmd/api`. They are not use cases and they do not import the Mongo or AMQP drivers; `cmd/api` injects ping functions.

Middleware verifies the bearer access token through `TokenParser` (RS256, public key from config, `type` claim `access`). It stores user id and role on the context. Invalid or expired tokens are `401`. Middleware does not check Analyst or Admin; the use case does.

## Infrastructure

Implement the ports from `usecase/interfaces.go`. Translate a missing Mongo document into `usecase.ErrNotificationNotFound`. Return the driver error for anything else.

Unique index on the idempotency key. Index `created_at` descending for the list query.

Config comes from the environment. The compose file already provides `PORT`, `DB_HOST`, `DB_PORT`, `MONGO_INITDB_ROOT_USERNAME`, `MONGO_INITDB_ROOT_PASSWORD`, and `MONGO_INITDB_DATABASE` (`shop_note_db` on `notification-db:27017`). Also require `BROKER_URL` and the access-token public key. Do not default broker credentials to `guest`.

Logs are structured JSON and include `correlation_id` on consume, send, and HTTP request paths.

## Tests

Unit-test each use case with fake ports before wiring a driver. Cover:

- insert pending, send, mark sent
- duplicate delivery after `sent` does not send again
- transient send error leaves the row retryable
- permanent send error and unknown type mark `failed` and do not retry
- token is absent from the saved notification
- Client role is forbidden; Analyst and Admin can list and get
- poison payloads are rejected by the decoder and are not passed to the use case

Keep Mongo, AMQP, and mail libraries out of `internal/usecase` tests.

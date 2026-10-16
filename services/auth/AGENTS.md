# Auth service

Go module `authorization`. HTTP API for registration, login, sessions, email verification, profile updates, password recovery, and role changes.

Follow the layering below. Match existing use-case and handler style when adding behavior.

## Dependency rule

Dependencies point inward. The core is `internal/domain` and `internal/usecase`.

| Layer | Path | May import |
|---|---|---|
| Domain | `internal/domain`, `internal/domain/security` | Standard library and other packages under `internal/domain` |
| Use cases | `internal/usecase` | Standard library and `internal/domain` |
| HTTP | `internal/interface/http` | Standard library, `internal/usecase`, `internal/domain`, HTTP router (`chi`) |
| Infrastructure | `internal/infrastructure` | Drivers (pgx, redis, amqp, jwt, bcrypt), `internal/domain`, `internal/usecase` ports |
| Config | `internal/config` | Standard library |
| Composition | `cmd/api` | Every layer, to construct and inject dependencies |

Hard rules:

- Do not import third-party modules from `internal/domain` or `internal/usecase`.
- Do not import `internal/infrastructure` or `internal/config` from handlers or use cases.
- Handlers call a use case. They do not open Postgres, Redis, RabbitMQ, or JWT code, and they do not hash passwords.
- Only `cmd/api` constructs infrastructure and passes it into use-case constructors.

`cmd/api/main.go` is a standalone broker probe. Do not copy it. New HTTP behavior is a handler plus a use case, wired in `cmd/api`.

## Layout

- `internal/domain` — `User`, `Role`, `Permission`, `TokenPair`, and sentinel errors for broken invariants (`ErrInvalidEmail`, `ErrPermissionDenied`, `ErrUserDeactivated`).
- `internal/domain/security` — password-strength check, random token generation, SHA-256 token hashing, UUID generation. Pure functions, no I/O.
- `internal/usecase` — one use case per file. Ports live in `interfaces.go`. Application sentinel errors live in `errors.go`.
- `internal/interface/http/handler` — one handler per use case. Shared JSON helpers are `request.go` and `response.go`.
- `internal/infrastructure/db/postgre` — `users` table. Map with `UserModel.ToDomain` / `FromDomain`. Do not leak `UserModel` past this package.
- `internal/infrastructure/db/redis` — verification tokens, email/password-update tokens, refresh sessions.
- `internal/infrastructure/token` — RSA JWT issue and parse. Implements `TokenGenerator`.
- `internal/infrastructure/hasher` — bcrypt. Implements `PasswordHasher`.
- `internal/infrastructure/broker` — publishes verification and password-reset messages to RabbitMQ. Implements `EmailSender`.
- `internal/infrastructure/logger` — `slog` adapter for the `Logger` port.
- `internal/config` — TTL env vars (`ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL`, `VERIFICATION_TOKEN_TTL`, `RESET_TOKEN_TTL`).

## Use cases

Shape:

```go
type LoginUserUseCase struct { /* unexported port fields */ }

func NewLoginUserUseCase(r UserRepository, s SessionRepository, g TokenGenerator, h PasswordHasher) *LoginUserUseCase

func (uc *LoginUserUseCase) Execute(ctx context.Context, input LoginInput) (domain.TokenPair, error)
```

- Name the file and type after the action (`initiate_email_update.go`, `InitiateEmailUpdateUseCase`).
- Keep `Input` and `Result` structs in that file.
- Add a port to `interfaces.go` when a use case needs persistence, mail, tokens, hashing, or logging. Depend on the interface, never on a concrete repository or client.
- Orchestrate here: load the entity, call domain methods, then save and send side effects.
- Put invariant checks on `domain.User` (`EnsureActive`, `UpdateUserEmail`, `SetRole`, `RequirePermission`). Return those domain errors unchanged.
- Return an existing sentinel from `usecase/errors.go` for application failures (`ErrEmailAlreadyTaken`, `ErrInvalidCredentials`). Add a sentinel only when callers must branch on it.
- Wrap unexpected infrastructure errors with `%w`. Do not invent a new sentinel for a database outage.
- Hash one-time tokens with `security.HashToken` before saving them. Send the raw token in email.
- Registration may succeed when verification email fails. Log with `Logger.Warn` and return `VerificationSent: false`. Do not roll back the user.

## Handlers

A handler depends on a small local interface, not on the concrete use case:

```go
type LoginUserUseCase interface {
    Execute(ctx context.Context, input usecase.LoginInput) (domain.TokenPair, error)
}
```

`ServeHTTP` order:

1. Reject the wrong HTTP method with `405` and `METHOD_NOT_ALLOWED`.
2. Read the authenticated user id from `r.Context().Value("userID").(string)` when the route requires it. Missing or blank id is `401`.
3. Decode the body with `DecodeJSON`. It enforces a 1 MiB limit.
4. Validate transport shape (required fields, length, email format via `domain.IsEmailValid`). Field errors go in `APIError.Details`.
5. Trim strings, build the use-case input, call `Execute`.
6. Map errors with `errors.Is` in `handleUseCaseError`. Unknown errors are `500` with a generic message.

Response envelope is `Response` (`success`, `data`, `errors`). `errors` is `APIError` (`code`, `message`, optional `details`). Write it only through `RespondJSON`.

Do not put business rules in the handler. Password strength, role changes, and “email already taken” belong in the use case or domain.

## Domain rules already encoded

- New users are active, unverified, and `RoleClient`.
- Roles are bit flags: `RoleClient`, `RoleAnalyst`, `RoleAdmin`. `SetRole` rejects assigning `RoleAdmin` and a no-op change.
- Permissions (`role:change`, `user:list`, `user:fullProfile`, `user:softDelete`) are granted to `RoleAdmin` only. Check them with `RequirePermission`.
- Deactivated users fail `EnsureActive` with `domain.ErrUserDeactivated`.
- Store refresh tokens and email tokens as hashes. Compare sessions by hash.

## Infrastructure

Implement the port from `usecase/interfaces.go`. Translate driver errors into use-case sentinels at the repository boundary (unique violation `23505` → `usecase.ErrEmailAlreadyTaken`, no rows → `usecase.ErrUserIdNotFound` or `usecase.ErrUserEmailNotFound`). Return the driver error for anything else.

Keep SQL, Redis keys, JWT claims, and AMQP payloads inside infrastructure.

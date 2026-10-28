# Auth Platform

A Go/PostgreSQL authentication and user-management API foundation. It provides password sign-up/sign-in, opaque server-side sessions in HttpOnly cookies, email verification and password-reset tokens, profile and account lifecycle management, organizations, membership roles, session controls, PostgreSQL-backed throttling, and audit events.

This is not a drop-in Clerk replacement. OAuth/social sign-in, passkeys, MFA, invitations, tenant-specific domains, frontend components/SDKs, and billing are not implemented. Review authorization rules and configure monitoring, encrypted backups, and retention before launch.

## Code layout

- `app.go` wires configuration, routes, middleware, and authentication context.
- `authentication.go`, `users.go`, and `sessions.go` handle sign-in/sign-up, user CRUD, and session lifecycle respectively.
- `email.go`, `action_tokens.go`, and `mailer.go` separate verification/reset endpoints, token persistence, and SMTP transport.
- `organizations.go` contains organization and membership operations; `rate_limit.go` and `audit.go` own abuse controls and audit writes.
- `http.go` and `security.go` contain shared request/response and cryptographic/validation helpers.
- `test/unit/` contains black-box unit tests for configuration and exported validation policies.
- `test/integration/` contains PostgreSQL API integration suites split by authentication, users, sessions, email, organizations, and rate limiting.

## Run with Docker Compose

1. Copy `.env.example` to `.env`. Set unique random `TOKEN_PEPPER` and `POSTGRES_PASSWORD` values and real SMTP settings. Use hex or URL-safe database password characters. `APP_ORIGIN` must be the public frontend origin that consumes verification/reset links.
2. Run `docker compose up --build`.
3. The API listens on `127.0.0.1:8080`. Compose uses development mode for its private local database. Put an HTTPS reverse proxy in front before production use.

The migration service applies the numbered SQL files on startup. They are idempotent, but use a migration runner with recorded versions for future schema changes. Back up PostgreSQL regularly, test restoration, and store production secrets in a secrets manager.

## Configuration

| Variable | Required | Description |
|---|---:|---|
| `DATABASE_URL` | Yes | PostgreSQL URL. Production requires `sslmode=verify-full`. |
| `TOKEN_PEPPER` | Yes | Stable secret for hashing opaque session/action tokens; at least 32 characters. |
| `APP_ORIGIN` | Yes | Exact frontend origin for CORS and email links. Production requires HTTPS. |
| `COOKIE_SECURE` | Production | Must be `true` outside development. |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM` | Production | SMTP endpoint and sender. STARTTLS is mandatory. |
| `SMTP_USERNAME`, `SMTP_PASSWORD` | Optional | SMTP credentials when required. |
| `LISTEN_ADDR` | No | Bind address, defaults to `:8080`. |
| `APP_ENV` | No | Defaults to production checks. Use `development` only locally. |
| `TRUSTED_PROXY_CIDRS` | No | Comma-separated trusted proxy IP ranges. Forwarded IP is honored only for a trusted direct peer. Proxies must sanitize `X-Forwarded-For`. |

## API

All request and response bodies use JSON. The `session` cookie is HttpOnly, SameSite=Lax, and lasts 30 days. Browser clients should send credentials and use the configured origin.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/health/live` | Liveness probe |
| `GET` | `/health/ready` | PostgreSQL readiness probe |
| `POST` | `/v1/sign-up` | Create account (`email`, `password`, optional `name`) |
| `POST` | `/v1/sign-in` | Authenticate and set session cookie |
| `POST` | `/v1/sign-out` | Revoke current session |
| `GET` | `/v1/sessions` | List current user's active and recent sessions |
| `DELETE` | `/v1/sessions/{session}` | Revoke one of the current user's sessions |
| `GET`, `PATCH` | `/v1/me` | Read/update current profile |
| `DELETE` | `/v1/me` | Delete account after password confirmation; requires resolving owned organizations |
| `POST` | `/v1/email/verification` | Send verification email |
| `POST` | `/v1/email/verification/consume` | Consume `{ "token": "..." }` |
| `POST` | `/v1/password/reset` | Request reset email (`email`) |
| `POST` | `/v1/password/reset/consume` | Consume `{ "token": "...", "password": "..." }` |
| `GET`, `POST` | `/v1/organizations` | List/create organization (`name`, `slug`) |
| `DELETE` | `/v1/organizations/{org}` | Soft-delete organization (owner only) |
| `GET` | `/v1/organizations/{org}/members` | List members |
| `PUT`, `DELETE` | `/v1/organizations/{org}/members/{user}` | Manage membership (admin/owner; only owner grants admin) |

Passwords require at least 12 bytes and are capped at bcrypt's 72-byte limit. Action tokens expire after 30 minutes and are single-use. Password reset revokes all active sessions. Reset requests return the same response whether or not an account exists. Account deletion scrubs personal information, revokes sessions/action tokens, and disables the account.

## Verification and CI

- `go test ./...` runs unit tests from `test/unit/` and the domain suites from `test/integration/`. PostgreSQL integration coverage runs when `AUTH_TEST_DATABASE_URL` points to a disposable database; each suite creates and drops an isolated schema.
- CI runs `go vet`, race-enabled tests, and `govulncheck` with PostgreSQL. Dependabot checks Go modules and GitHub Actions weekly.
- PostgreSQL-backed limits currently allow 10 sign-ins and 5 sign-ups/reset/verification requests per client IP per minute. Other known API paths have a higher limit. Limits fail closed if the rate-limit store is unavailable. Old limit buckets are removed hourly.
- Audit events are stored without request payloads or credentials. Export them to a protected append-only system and configure an explicit retention period.

## Operations and security

- Put the API behind TLS termination and keep edge connection, request-size, and abuse limits enabled. Built-in distributed limits supplement edge controls.
- Set `TRUSTED_PROXY_CIDRS` only to known proxy networks; configure those proxies to sanitize forwarded headers.
- Configure authenticated SMTP with STARTTLS and monitor delivery failures. Tokens are stored hashed, never returned by the API, and never logged.
- Restrict database credentials to this service, require TLS for remote PostgreSQL, rotate credentials, and use encrypted backups.
- Organization owners cannot be demoted or removed. Only owners can grant admin. Account deletion is blocked until organization ownership has been transferred or the organization deleted.
- Production startup requires secure cookies, HTTPS `APP_ORIGIN`, SMTP configuration, and PostgreSQL `sslmode=verify-full`.

Dependency downloads require Go module proxy access. The workspace network restrictions prevented downloading dependencies here, so the local build could not be confirmed; CI performs compilation, tests, and vulnerability scanning after the changes are pushed.

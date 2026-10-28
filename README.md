# Auth Platform

A Go/PostgreSQL authentication and user-management API starter. It provides password sign-up/sign-in, opaque server-side sessions in HttpOnly cookies, email verification and password-reset tokens, profile management, organizations, and membership roles.

This is a deployable foundation, not a drop-in Clerk replacement. A production launch still needs an operational email domain/provider, HTTPS termination, backups, monitoring, abuse controls, and security review. OAuth/social login, passkeys, MFA, invitations, tenant-specific domains, frontend components/SDKs, and billing are not implemented.

## Run locally with Docker Compose

1. Copy `.env.example` to `.env`, then set unique random `TOKEN_PEPPER` and `POSTGRES_PASSWORD` values and real SMTP settings. `APP_ORIGIN` must be the public origin of the frontend that consumes verification/reset links.
2. Run `docker compose up --build`.
3. The API listens on `127.0.0.1:8080`; put an HTTPS reverse proxy in front for production. Set `COOKIE_SECURE=true` in production.

The migration service applies `migrations/001_init.sql` on startup. Back up PostgreSQL regularly and test restoration. Store secrets in a secrets manager in production rather than a checked-in environment file.

## Configuration

| Variable | Required | Description |
|---|---:|---|
| `DATABASE_URL` | Yes | PostgreSQL connection URL. Use TLS (`sslmode=verify-full`) outside local Compose. |
| `TOKEN_PEPPER` | Yes | Secret used to hash opaque session and action tokens; minimum 32 characters. Keep stable and secret. |
| `APP_ORIGIN` | Yes for email flows | Exact frontend origin used to construct verification/reset links and CORS allowlist. |
| `COOKIE_SECURE` | Production | Set to `true` when served over HTTPS. |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM` | Email flows | SMTP endpoint and sender. SMTP server must support STARTTLS. |
| `SMTP_USERNAME`, `SMTP_PASSWORD` | Optional | SMTP credentials when required. |
| `LISTEN_ADDR` | No | Bind address, defaults to `:8080`. |

## API

All request/response bodies use JSON. Session cookie is named `session`; it is HttpOnly, SameSite=Lax, and lasts 30 days. Browser clients should send credentials and use the exact configured origin.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/health/live` | Liveness probe |
| `GET` | `/health/ready` | Database readiness probe |
| `POST` | `/v1/sign-up` | Create account (`email`, `password`, optional `name`) |
| `POST` | `/v1/sign-in` | Authenticate and set session cookie |
| `POST` | `/v1/sign-out` | Revoke current session |
| `GET`, `PATCH` | `/v1/me` | Read/update current profile |
| `POST` | `/v1/email/verification` | Send verification email |
| `POST` | `/v1/email/verification/consume` | Consume `{ "token": "..." }` |
| `POST` | `/v1/password/reset` | Request reset email (`email`) |
| `POST` | `/v1/password/reset/consume` | Consume `{ "token": "...", "password": "..." }` |
| `GET`, `POST` | `/v1/organizations` | List/create organization (`name`, `slug`) |
| `GET` | `/v1/organizations/{org}/members` | List members |
| `PUT`, `DELETE` | `/v1/organizations/{org}/members/{user}` | Manage membership (admin/owner) |

Passwords require at least 12 bytes and are capped at bcrypt's 72-byte limit. Action tokens expire after 30 minutes and are single-use. Resetting a password revokes the account's active sessions. Reset requests return the same response whether or not an account exists.

## Security and operations

- Put the API behind a TLS reverse proxy with request size, connection, and per-IP rate limits. This service does not implement distributed rate limiting; add it at the edge before public exposure.
- Configure SMTP with authenticated STARTTLS and monitor delivery failures. Tokens are stored hashed, never returned by API, and never logged.
- Restrict database credentials to this service, require TLS for remote PostgreSQL, rotate credentials, and use encrypted backups.
- Add an audit/event pipeline and alerting appropriate to your compliance needs. Keep application logs free of credentials and personal data.
- Review organization authorization rules before use in a multi-tenant product. This starter has no invitation workflow or organization deletion API.

Dependency downloads and compilation require access to the Go module proxy; this workspace was created without network access, so the initial build has not been verified here.

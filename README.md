# Hono Event Starter, Go Edition

A Go/chi/templ version of the Bun/Hono event-sourced starter. It keeps the same architectural boundaries: immutable Orisun events are the source of truth, PostgreSQL stores read models and auth/session tables, projections are checkpointed, and Datastar SSE patches server-rendered fragments.

## Variants

This repository keeps database-backed variants on separate branches:

- `postgres`: PostgreSQL for app tables plus embedded Orisun Postgres. Use this branch when matching the original Hono starter's Postgres deployment shape.
- `sqlite`: embedded SQLite for app tables plus embedded Orisun SQLite. Use this branch for a local-first variant that does not require a Postgres service.

## Commands

```bash
task migrate
task seed
task sqlc
task dev
task test
task build
```

## Services

Reuse the root `docker-compose.yml` services:

```bash
docker compose up postgres orisun garage -d
```

Copy the root `.env.example` values into your shell or a local env loader. The Go app reads the same `POSTGRES_*`, `NATS_URL`, `ORISUN_*`, `BREVO_*`, `STORAGE_*`, `R2_*`, `APP_URL`, and `PORT` variables.

## Notes

- HTTP uses `github.com/go-chi/chi/v5`.
- The `postgres` branch generates pgx-backed database code with sqlc; SQL lives in `queries/` and generated code lives in `internal/dbsql/`.
- Views are authored as `.templ` components and generated with `go tool templ generate`.
- Static assets are served through `internal/resources` and referenced from templ with `resources.StaticPath`.
- Authentication is Go-native and stores password/session data in PostgreSQL tables compatible with the starter schema shape.
- Business changes are made through commands that append Orisun events; read models are updated by event handlers.

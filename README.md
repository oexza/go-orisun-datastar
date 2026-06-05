# Go Event Starter

A Go/chi/templ starter for event-sourced web applications. Immutable Orisun events are the source of truth, PostgreSQL stores read models and auth/session tables, projections are checkpointed, and Datastar SSE patches server-rendered fragments.

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

Start PostgreSQL for the Postgres-backed variant:

```bash
docker compose up postgres -d
```

The starter runs without email or object-storage credentials. Outbound emails are logged, and uploaded profile images are written to `static/uploads`.

Copy the root `.env.example` values into your shell or a local env loader if you want to override defaults. The Go app reads `POSTGRES_*`, `NATS_URL`, `ORISUN_*`, `UPLOAD_DIR`, `UPLOAD_BASE_URL`, `APP_URL`, and `PORT` variables.

## Notes

- HTTP uses `github.com/go-chi/chi/v5`.
- The `postgres` branch generates pgx-backed database code with sqlc; SQL lives in `queries/` and generated code lives in `internal/dbsql/`.
- Views are authored as `.templ` components and generated with `go tool templ generate`.
- Static assets are served through `internal/resources` and referenced from templ with `resources.StaticPath`.
- Authentication is Go-native and stores password/session data in PostgreSQL tables compatible with the starter schema shape.
- Business changes are made through commands that append Orisun events; read models are updated by event handlers.

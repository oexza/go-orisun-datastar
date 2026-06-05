# Go Event Starter

A Go/chi/templ starter for event-sourced web applications. Immutable Orisun events are the source of truth, SQLite stores read models and auth/session tables, projections are checkpointed, and Datastar SSE patches server-rendered fragments.

## Variants

This repository keeps database-backed variants on separate branches:

- `sqlite`: embedded SQLite for app tables plus embedded Orisun SQLite. This branch is the local-first variant and does not require a Postgres service.
- `postgres`: PostgreSQL for app tables plus embedded Orisun Postgres. Use this branch when matching the original Hono starter's Postgres deployment shape.

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

The starter runs without email or object-storage credentials. Outbound emails are logged, and uploaded profile images are written to `static/uploads`.

Copy the root `.env.example` values into your shell or a local env loader if you want to override defaults. The Go app reads `SQLITE_PATH`, `ORISUN_SQLITE_DIR`, `NATS_URL`, `ORISUN_GENERAL_BOUNDARY`, `UPLOAD_DIR`, `UPLOAD_BASE_URL`, `APP_URL`, and `PORT` variables. SQLite defaults to `data/app.sqlite`; embedded Orisun defaults to `data/orisun`.

## Notes

- HTTP uses `github.com/go-chi/chi/v5`.
- The `sqlite` branch generates zombiezen SQLite statements with sqlc and `github.com/delaneyj/toolbelt/sqlc-gen-zombiezen`; SQL lives in `queries/` and generated code lives in `internal/dbsql/`.
- Views are authored as `.templ` components and generated with `go tool templ generate`.
- Static assets are served through `internal/resources` and referenced from templ with `resources.StaticPath`.
- Todo SSE uses a Frases-style ViewStore path: NATS notifications update per-session view state, and the SSE route patches from the ViewStore watcher.
- Authentication is Go-native and stores password/session data in SQLite tables compatible with the starter schema shape.
- Business changes are made through commands that append Orisun events; read models are updated by event handlers.

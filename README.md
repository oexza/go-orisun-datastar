# Hono Event Starter, Go Edition

A Go/chi/templ version of the Bun/Hono event-sourced starter. It keeps the same architectural boundaries: immutable Orisun events are the source of truth, PostgreSQL stores read models and auth/session tables, projections are checkpointed, and Datastar SSE patches server-rendered fragments.

## Commands

```bash
task migrate
task seed
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
- Views are authored as `.templ` components and generated with `go tool templ generate`.
- Static assets are served through `internal/resources` and referenced from templ with `resources.StaticPath`.
- Authentication is Go-native and stores password/session data in PostgreSQL tables compatible with the starter schema shape.
- Business changes are made through commands that append Orisun events; read models are updated by event handlers.

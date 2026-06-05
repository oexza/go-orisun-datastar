# Go-Orisun-Datastar

Go-Orisun-Datastar is a Go/chi/templ starter for event-sourced web applications. [Orisun](https://github.com/oexza/Orisun) stores immutable domain events, PostgreSQL stores read models and auth/session tables, Datastar streams server-rendered UI updates over SSE, and Tailwind/DaisyUI provide the frontend system.

The default setup is intentionally simple: email messages are logged, profile uploads are written to `static/uploads`, and no third-party email or object-storage credentials are required.

## Branches

This repository keeps database-backed variants on separate branches:

- `postgres`: PostgreSQL for app tables plus embedded Orisun Postgres. This branch is the Postgres-backed variant.
- `sqlite`: embedded SQLite for app tables plus embedded Orisun SQLite. Use this branch when you want the easiest local development path without a Postgres service.

## Features

- Go HTTP server with `github.com/go-chi/chi/v5`
- Server-rendered `.templ` views
- Datastar SSE for form submissions, redirects, indicators, fragments, and long-lived read streams
- [Orisun](https://github.com/oexza/Orisun) event sourcing with typed domain events
- CQRS-style read models and replayable event handlers
- pgx-backed sqlc database access
- Authentication, sessions, email verification, password reset, profile editing, and todos
- Local logged email sender for development and demos
- Local filesystem upload storage served from `/static/uploads`
- Tailwind v4 and DaisyUI compiled by `go tool gotailwind`
- Task-based development workflow

## Requirements

- Go matching the version in `go.mod`
- Task
- PostgreSQL

All Go tools used by the project are pinned in `go.mod` and invoked with `go tool` through `Taskfile.yml`.

## Quick Start

Start PostgreSQL, then run:

```bash
task migrate
task dev
```

Open `http://localhost:3000` unless you override `PORT`.

For a production-style build:

```bash
task build
./server
```

## Commands

```bash
task dev          # run templ, CSS, and server watchers
task server-watch # run the Go server through air
task css          # compile src/input.css to static/style.css
task css-watch    # watch and compile CSS
task templ        # generate *_templ.go files
task sqlc         # generate database code
task migrate      # run database migrations and exit
task seed         # run seed tasks and exit
task test         # run Go tests
task vet          # run go vet
task build        # build ./cmd/server
```

## Configuration

The app has useful defaults. Override these environment variables when needed:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `3000` | HTTP server port |
| `APP_URL` | `http://localhost:$PORT` | Base URL used in generated links |
| `DATABASE_URL` | derived from `POSTGRES_*` | Full Postgres connection string |
| `POSTGRES_HOST` | `localhost` | Postgres host |
| `POSTGRES_PORT` | `5432` | Postgres port |
| `POSTGRES_USER` | `postgres` | Postgres user |
| `POSTGRES_PASSWORD` | `postgres` | Postgres password |
| `POSTGRES_DB` | `go_orisun_datastar` | Postgres database |
| `POSTGRES_USE_SSL` | `false` | Enables required SSL mode when true |
| `NATS_URL` | `nats://localhost:4224` | NATS URL used by app-level notifications |
| `ORISUN_GENERAL_BOUNDARY` | `go_orisun_datastar` | Orisun boundary name |
| `UPLOAD_DIR` | `static/uploads` | Local upload destination |
| `UPLOAD_BASE_URL` | `/static/uploads` | Public URL prefix for uploaded files |
| `BETTER_AUTH_SECRET` | development secret | Session/auth secret |
| `NODE_ENV` | `development` | Controls development cookie behavior and hot reload |

The `sqlite` branch replaces the Postgres-specific variables with `SQLITE_PATH` and `ORISUN_SQLITE_DIR`.

## Architecture

All business changes are written as Orisun events. Feature services append events, event handlers project those events into read models, and HTTP/SSE handlers render UI from the read models.

Typical write flow:

1. A chi handler parses input and builds command metadata.
2. A feature service validates the command and appends typed Orisun events.
3. Event handlers consume events, update read models, and checkpoint progress.
4. NATS notifications invalidate or refresh view state.
5. Datastar SSE streams patch server-rendered templ fragments.

Keep direct SQL writes limited to read models, auth/session tables, and event-handler checkpoints. Domain state should flow through events.

## Styling

Edit `src/input.css`; do not edit `static/style.css` directly. DaisyUI is loaded through local Tailwind plugin bundles under `src/daisyui`, and `task css` compiles the generated stylesheet.

## Generated Files

The repository includes generated files so the app is easy to inspect and build:

- `internal/views/*_templ.go` from templ
- `internal/dbsql/*.go` from sqlc
- `static/style.css` from Tailwind/DaisyUI

Regenerate them with:

```bash
task sqlc
task templ
task css
```

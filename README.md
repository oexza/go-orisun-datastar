# Go-Orisun-Datastar

Go-Orisun-Datastar is a Go/chi/templ starter for event-sourced web applications. [Orisun](https://github.com/oexza/Orisun) stores immutable domain events, SQLite stores read models and auth/session tables, Datastar streams server-rendered UI updates over SSE, and Tailwind/DaisyUI provide the frontend system.

The default setup is intentionally local-first: email messages are logged, profile uploads are written to `static/uploads`, and no third-party email or object-storage credentials are required.

## Branches

This repository keeps database-backed variants on separate branches:

- `sqlite`: embedded SQLite for app tables plus embedded Orisun SQLite. This branch is the easiest local development path.
- `postgres`: PostgreSQL for app tables plus embedded Orisun Postgres. Use this branch when you want the database and event store backed by Postgres.

## Features

- Go HTTP server with `github.com/go-chi/chi/v5`
- Server-rendered `.templ` views
- Datastar SSE for form submissions, redirects, indicators, fragments, and long-lived read streams
- [Orisun](https://github.com/oexza/Orisun) event sourcing with typed domain events
- CQRS-style read models and replayable event handlers
- sqlc-generated database access
- Authentication, sessions, email verification, password reset, profile editing, and todos
- Local logged email sender for development and demos
- Local filesystem upload storage served from `/static/uploads`
- Tailwind v4 and DaisyUI compiled by `go tool gotailwind`
- Task-based development workflow

## Requirements

- Go matching the version in `go.mod`
- Task

All Go tools used by the project are pinned in `go.mod` and invoked with `go tool` through `Taskfile.yml`.

## Quick Start

```bash
task migrate
task dev
```

Open `http://localhost:3001` unless you override `PORT`.

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

The app has useful defaults and can run without a `.env` file. Override these environment variables when needed:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `3001` | HTTP server port |
| `APP_URL` | `http://localhost:$PORT` | Base URL used in generated links |
| `SQLITE_PATH` | `data/app.sqlite` | SQLite read-model/auth database path |
| `ORISUN_SQLITE_DIR` | `data/orisun` | Embedded Orisun SQLite directory |
| `NATS_URL` | `nats://localhost:4224` | NATS URL used by app-level notifications |
| `ORISUN_GENERAL_BOUNDARY` | `go_orisun_datastar` | Orisun boundary name |
| `UPLOAD_DIR` | `static/uploads` | Local upload destination |
| `UPLOAD_BASE_URL` | `/static/uploads` | Public URL prefix for uploaded files |
| `BETTER_AUTH_SECRET` | development secret | Session/auth secret |
| `NODE_ENV` | `development` | Controls development cookie behavior and hot reload |

The `postgres` branch replaces the SQLite-specific variables with `DATABASE_URL` or `POSTGRES_*` variables.

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

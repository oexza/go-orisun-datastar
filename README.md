# Go-Orisun-Datastar

Go-Orisun-Datastar is a Go starter for event-sourced, server-rendered web applications. It combines [Orisun](https://github.com/oexza/Orisun), chi, templ, Datastar, sqlc, Tailwind, and DaisyUI into a small application that is useful as a reference and as a starting point.

The `sqlite` branch is local-first. SQLite stores auth/session tables and read models, embedded Orisun stores immutable domain events in SQLite, and the app borrows Orisun's embedded NATS/JetStream connection for event notifications and view-state updates. Email messages are logged, uploads are written to `static/uploads`, and no external email, object storage, or NATS service is required.

## Branches

This repository keeps database-backed variants on separate branches:

- `sqlite`: SQLite for app tables plus embedded Orisun SQLite. This is the easiest local development path.
- `postgres`: PostgreSQL for app tables plus embedded Orisun Postgres. Use this when you want Postgres-backed app data and event storage.

If a branch is already checked out in a separate worktree, use that worktree path or remove it before switching branches in this checkout.

## Features

- Go HTTP server with `github.com/go-chi/chi/v5`
- Server-rendered `.templ` views
- Datastar SSE for form submissions, redirects, indicators, fragment patches, sparse signals, and long-running streams
- Datastar SDK SSE compression with `datastar.WithCompression()`
- HTTP response compression for normal HTML, CSS, JavaScript, JSON, text, and SVG responses
- [Orisun](https://github.com/oexza/Orisun) event sourcing with typed domain events
- Embedded Orisun NATS/JetStream reused directly by the app; no NATS URL is needed
- CQRS-style read models and replayable event-handler slices
- Command-handler slices for business writes
- sqlc-generated SQLite access through the toolbelt zombiezen/sqlite generator
- UUIDv7 identifiers for generated IDs
- Authentication, sessions, email verification, password reset, profile editing, and todos
- Local logged email sender for development and demos
- Local filesystem upload storage served from `/static/uploads`
- Tailwind v4 and DaisyUI compiled by `go tool gotailwind`
- Task-based development workflow

## Requirements

- Go matching the version in `go.mod`
- Task

All project tools are pinned in `go.mod` and invoked with `go tool` through `Taskfile.yml`.

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
| `ORISUN_GENERAL_BOUNDARY` | `go_orisun_datastar` | Orisun boundary name |
| `UPLOAD_DIR` | `static/uploads` | Local upload destination |
| `UPLOAD_BASE_URL` | `/static/uploads` | Public URL prefix for uploaded files |
| `BETTER_AUTH_SECRET` | development secret | Session/auth secret |
| `NODE_ENV` | `development` | Controls development cookie behavior and hot reload |

The `postgres` branch replaces the SQLite-specific variables with `DATABASE_URL` or `POSTGRES_*` variables.

## Architecture

All business changes are written as Orisun events. HTTP handlers parse input and call command handlers. Command handlers validate intent, load the minimal event context they need, and append typed events. Event handlers project those events into read models and checkpoint progress.

The UI flow follows [the Tao of Datastar](https://data-star.dev/guide/the_tao_of_datastar): keep state on the backend, render templ fragments on the server, patch elements or sparse signals over SSE, and use redirects or normal anchors for navigation.

Typical write flow:

1. A chi handler parses input and builds command metadata.
2. A command-handler slice validates the command and appends typed Orisun events.
3. Event-handler slices consume events, update read models, and checkpoint progress.
4. Embedded NATS notifications invalidate or refresh server-side view state.
5. Datastar streams compressed SSE patches for templ fragments and sparse signals.

Keep direct SQL writes limited to read models, auth/session tables, and event-handler checkpoints. Domain state should flow through events.

## Slice Layout

Feature folders are organized around behavior:

- command handlers live in their own files, such as `create_todo.go`, `rename_todo.go`, and `upload_profile_image.go`
- each command slice owns its load context instead of depending on a shared aggregate loader
- event handlers live in their own files, such as `todo_read_model_event_handler.go`
- read models own query/projection persistence
- `ports.go` files hold consumer-driven interfaces only when a slice needs an external capability

Generated identifiers should go through `internal/uuidv7.NewString()`.

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

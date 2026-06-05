# Agent Guide

This repository is Go-Orisun-Datastar, a Go/chi/templ starter for event-sourced Datastar applications. Treat the sibling `../frases-backend` project as the richer reference implementation for architecture, UX patterns, feature boundaries, Datastar flows, CQRS-style SSE, NATS KV view-state, and visual direction.

The app uses server-rendered templ components, Datastar SSE, SQLite read models, embedded Orisun SQLite events, NATS notifications, logged email, and local filesystem upload storage.

## Commands

Use Task, not Make.

```bash
task dev
task server-watch
task css
task css-watch
task build
task test
task vet
task migrate
task seed
task templ
```

`task dev` runs the Go server watcher, templ watcher, and CSS watcher together. Air, templ, and gotailwind are pinned as Go tools in `go.mod`; use them through Task.

`task templ` runs `go tool templ generate`; `task dev` also starts `templ-watch`.

`static/style.css` is generated from `src/input.css`. Never edit `static/style.css` directly.

## Reference Projects

The sibling `../frases-backend` is the canonical implementation for expansive patterns. Before adding a feature or pattern here, inspect the matching feature there and port the concept deliberately.

Translate, do not blindly copy:

- HTTP route ideas become chi handlers in `internal/httpui`; keep route registration split by feature file rather than growing `router.go`.
- JSX render helpers become real `.templ` components in `internal/views`; shared layout/navigation/indicator components belong in `components.templ`, and generated `*_templ.go` files are build artifacts from `task templ`.
- TypeScript command classes and handlers become Go command structs plus handler functions with narrow ports.
- Frases command metadata becomes `eventstore.CommandMetadata`; HTTP mutation routes should attach `eventstore.HTTPCommandMetadata` when invoking command handlers/services.
- Drizzle schema/index changes become SQL migrations under `migrations`.
- Datastar usage should call the official Datastar Go SDK directly. Keep app-specific helpers close to the route/feature using them.
- `ViewStore` / `NatsViewStore` concepts live in `internal/viewstore`; use that abstraction for per-session view-state instead of ad hoc map globals.

## Architecture

All business mutations are recorded as immutable Orisun events. SQLite stores read models and auth/session tables. Event handlers update read models after events are published and checkpointed.

Typical write flow:

1. HTTP handler parses and validates input.
2. Feature command/service loads Orisun events and reconstructs state.
3. Command appends new domain event(s).
4. Event handler updates SQLite read models and publishes NATS notifications.
5. Datastar SSE watchers patch server-rendered fragments.

Keep these boundaries intact:

- Do not mutate business state directly in SQLite.
- Do not bypass event saving for domain changes.
- Use event-handler checkpoints for replayable event handlers.
- Keep route, command/service, event handler, read model, and UI code within the owning feature slice.
- Every Orisun query criterion and append subset query should include `eventType`; subscriptions are the exception.
- When changing Orisun query or append shapes, update the matching event-store indexes/documentation if present.
- When changing SQLite read-model queries, add/update SQL indexes in migrations for predicates, joins, ordering, and fan-out paths.

## Feature Ownership

Features own their read models. A feature must not query another feature's projection tables directly. If it needs data from another feature, project the required facts into its own read model through its own event handler.

Command handlers and event handlers should use narrow injected ports for external side effects rather than importing broad infrastructure into domain logic.

Follow the `frases-backend` command-handler pattern when porting mutations:

- Define a command struct for the user's intent, including only inputs and optional command metadata.
- Validate input either when constructing the command or at the start of the handler.
- Load only the events needed to build a small context model.
- Let the context model enforce existence, ownership, idempotency, and latest event position.
- Save events through injected `eventstore.Saver` and load context through injected `eventstore.Retriever`.
- Keep HTTP handlers and services as orchestration layers; they should not own domain decision logic.

## CQRS-Style SSE

Keep command responses and read-model updates separate.

- Mutation handlers should append events and return a small Datastar SSE response: empty acknowledgement, redirect, server-rendered fragment patch, or a focused signal patch.
- Mutation handlers should not render large fresh read-model state directly as the primary success path.
- Long-lived read streams subscribe to NATS/read-model notifications, reload the read model, render a server fragment, and patch it with Datastar.
- Long-lived SSE handlers should follow the Northstar loop shape: create one Datastar SSE stream, bridge watcher/subscription callbacks into a buffered channel, and perform all SSE writes from the handler's own `select` loop. Do not write to the same SSE response from subscription callback goroutines.
- Long-lived `data-init` streams should disable Datastar request cancellation when they are meant to stay open alongside other interactions.
- Dev hot reload follows Northstar's `/reload` SSE and `/hotreload` ping pattern; keep it gated to development.
- This preserves the command/read split: commands change facts; read streams publish the projected UI.

For example, todo create clears the input over SSE, while `/todos/stream` patches `#todo-list` after the event handler updates the read model.

## NATS KV View-State Pattern

`frases-backend` uses two related but different NATS patterns:

- NATS pub/sub for durable projection invalidation or small event notifications.
- NATS KV for per-session view-state watched by Datastar SSE routes.

The full KV view-state flow is:

1. Page render initializes a per-session view-state key such as `create-flashcard//{sessionID}//{deckID}`.
2. A long-lived SSE route watches that key with `ignoreDeletes`.
3. Action routes save domain events or write small transient UI state to KV, then return an empty/small Datastar SSE response.
4. The KV watcher patches the smallest necessary UI surface or patches Datastar signals.
5. Projection NATS notifications may be consumed by the SSE route and written into KV, so the KV watcher remains the single view-state patch path.

Keep KV payloads small. Do not store generated images, uploaded file bytes, or other large blobs in NATS KV. Durable media belongs in object storage/domain events; large transient media should be streamed directly to the requesting action response as a targeted signal patch.

Current implementation status: todo SSE now uses `internal/viewstore` for the Frases-style view-state path. Projection notifications trigger a read-model reload into ViewStore, and the SSE route patches only from the ViewStore watcher. Other features should follow this pattern as they adopt view-state streaming.

## Datastar and UI

The backend owns UI state. Prefer server-rendered HTML fragments sent over SSE.

Priority order:

1. Server-rendered morphs via the SDK's `PatchElementTempl` / `PatchElements` helpers.
2. Backend redirects via the SDK's `Redirect` helper for page navigation after commands.
3. Datastar signals for local interaction state, form-bound data, and small server-patched UI messages.
4. Focused scripts via the SDK's `ExecuteScript` only when morphs, redirects, and signals are insufficient.
5. Vanilla JavaScript only for isolated behavior that cannot be expressed through Datastar attributes or SSE events.

Guidelines:

- Use anchor tags and browser navigation for page changes.
- Use `data-indicator` for loading states.
- Do not use optimistic UI unless the server confirms the state.
- Morph targets need stable IDs.
- Datastar-triggered routes should return `text/event-stream` responses through the Go SDK, including validation and error paths.
- Keep NATS payloads small; use payloads as invalidation/notification, then reload read models server-side.
- Use the official Datastar Go SDK for SSE event formatting; do not hand-write SSE protocol lines in handlers.

## Visual System

The sibling projects' DaisyUI/Tailwind UI is the visual reference. Follow `../northstar` for CSS tooling: Tailwind is compiled through `go tool gotailwind`, and DaisyUI is loaded from local plugin files under `src/daisyui`. Esbuild belongs to JS/TypeScript bundling if this Go port later needs a browser bundle; do not use raw esbuild as the Tailwind CSS processor.

When porting DaisyUI-style screens:

- Put source styling in `src/input.css`.
- Keep generated output in `static/style.css`.
- Reference static assets through `internal/resources.StaticPath` from templ layouts instead of hardcoding `/static/...`.
- Do not inline broad styling in Go handlers.
- Prefer reusable templ/render helpers over repeated HTML strings.
- Keep DaisyUI/Tailwind directives in `src/input.css` and verify them with `task css`.

## Important Paths

| Path | Purpose |
| --- | --- |
| `cmd/server/main.go` | App entry point, infrastructure wiring, event-handler startup |
| `internal/httpui/router.go` | Top-level HTTP router and middleware |
| `internal/httpui/*_handlers.go` | Feature route registration and handlers |
| `internal/httpui/reload.go` | Development hot-reload SSE endpoints |
| `internal/views` | Server-rendered `.templ` components and generated Go |
| `internal/resources` | Static asset handler and path helpers |
| `internal/features/todo` | Example Todo feature, commands, read model, event handler |
| `internal/features/profile` | Profile commands/storage integration |
| `internal/auth` | Auth, sessions, onboarding, password flows |
| `internal/eventstore` | Orisun adapter, event types, checkpoints, global event handler |
| `internal/natsbus` | NATS notification bus |
| `internal/viewstore` | Frases-style per-session view-state store with NATS KV and memory fallback |
| `internal/appdb` | SQLite database wrapper backed by `github.com/delaneyj/toolbelt/db` |
| `internal/storage` | Local upload storage |
| `migrations` | SQLite schema and index migrations |
| `src/input.css` | Editable CSS source |
| `static/style.css` | Generated stylesheet served by the app |
| `Taskfile.yml` | Project task definitions |

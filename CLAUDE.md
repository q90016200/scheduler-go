# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

總是用繁體中文回覆。

## Commands

- Run server: `go run main.go` (reads config from `.env` in repo root)
- Run DB migrations: `go run cmd/migrate/migrate.go` (auto-migrates `User`, `Task`, `TaskGroup`, `TaskExecutionLog`)
- Build: `go build ./...`
- Vet: `go vet ./...`
- Tidy deps: `go mod tidy`
- No test suite exists in this repo currently (no `*_test.go` files).
- Frontend (`/web`, Vue3 + TypeScript + Vite): `cd web && npm install`, `npm run dev` (dev server, proxies `/api` to the Go backend per `web/vite.config.ts`), `npm run build` (outputs `web/dist`, served by the Go binary in production — see `internal/router/router.go`'s `NoRoute` SPA fallback).

## Architecture

Gin HTTP API + cron-based task scheduler, backed by PostgreSQL via GORM.

**Layering**: `controllers` (gin handlers, request binding/validation) → `services` (business logic, direct GORM CRUD) → `models` (GORM structs + request DTOs). There is **no repository layer** — `services` hold a `*gorm.DB` directly. This was a deliberate simplification (repositories were a thin pass-through with no independent logic).

**Wiring** (`main.go`): `config.LoadConfig` → `logger.Init` → `database.InitializeDB` → `scheduler.NewCronScheduler(db)` → `app.NewAppServices(db, cronScheduler)` → `router.SetupRouter`. `app.AppServices` is the single dependency-injection struct passed to controllers.

**Import-cycle constraint**: `internal/scheduler` is imported by `internal/services` (for `AddJob`/`RemoveJob` when tasks are created/updated/deleted). Because of this, `scheduler` cannot import `services` back — `CronScheduler` therefore keeps its own direct `*gorm.DB` reference and re-implements its own minimal task query in `cron.go` instead of depending on `TaskService.ListTasks`. Keep this direction in mind before adding new cross-package calls between the two.

**Scheduler** (`internal/scheduler/`):
- `cron.go`: `CronScheduler` wraps `robfig/cron/v3`. A fixed every-minute cron job polls `tasks` where `status = active`, and diffs against the in-memory `jobIDs` map to `AddJob`/`RemoveJob` as tasks are created/updated/deleted or flip status. A second daily cron job (`0 3 * * *`) deletes `TaskExecutionLog` rows older than `config.AppConfig.ExecutionLogRetentionDays` (env `EXECUTION_LOG_RETENTION_DAYS`, default 120).
- `jobs.go`: per-`Task.Type` execution. `"http"` uses `resty` for GET/POST/PUT/DELETE against `task.Endpoint`. `"rpc"` calls a gRPC service **without precompiled protobuf stubs** — it dials `task.Endpoint`, uses server reflection (`jhump/protoreflect/grpcreflect`) to resolve the service/method described by `task.Method` (format: `"package.Service/MethodName"`), builds the request from `task.Payload` (a JSON string) via `dynamic.NewMessage(...).UnmarshalJSON`, and invokes it with `grpcdynamic.NewStub`. This generic/dynamic design mirrors the schema-less `"http"` job type since job targets are arbitrary and user-configured, not known at compile time. Every run (success or failure) is persisted to `TaskExecutionLog` via `logExecution` — the `Output` column is typed `jsonb` in Postgres (chosen over a plain text column specifically so `output` can later be filtered/aggregated on via Postgres's `jsonb` operators/GIN indexes), so `toJSONColumn` wraps non-JSON output (e.g. plain-text/HTML HTTP bodies) as `{"raw": "..."}` before insert, since Postgres rejects invalid JSON in a `jsonb` column.

**Validation**: `Task.Method`'s required format depends on `Task.Type` (e.g. `GET/POST/...` for `http`, `"pkg.Service/Method"` for `rpc`). This is enforced via a struct-level validator (`pkg/utils/validator.go`, `RegisterStructValidation(validateTaskMethod, ...)`) rather than per-field tags, since the constraint spans two fields.

**Auth**: JWT stored in an `auth_token` cookie (not an `Authorization` header). `internal/auth/jwt.go` issues/parses tokens; `internal/middleware/auth_middleware.go` is the gin middleware that *would* guard write routes. **Currently disabled**: `JWTAuthMiddleware()` is not mounted anywhere in `internal/router/router.go`, and the frontend's login-redirect guard (`web/src/router/index.ts`) and `App.vue`'s logged-in-only shell were removed — all API routes and pages are open, temporarily, until login is re-enabled. The auth controller/service/store/views are untouched, so re-enabling is just re-adding the middleware calls and the router guard.

**Response envelope**: all API responses go through `pkg/response.Success`/`Fail`, which wrap payloads in a `{code, message, data, error, trace}` shape and pull `trace_id` from gin context (set by `OpenTracingMiddleware`).

**CORS & frontend serving**: `internal/middleware/cors_middleware.go` is only mounted when `config.AppConfig.AppEnv == "development"` (see `router.go`), since the Vue frontend (`/web`) is served same-origin in production via `r.Static`/`r.NoRoute` in `router.go` (serving `web/dist`, with an SPA fallback to `web/dist/index.html` for non-`/api` paths) — no CORS needed once same-origin.

**Go version**: `go.mod` pins `go 1.24.4`. When bumping `google.golang.org/grpc` or `github.com/jhump/protoreflect`, check each candidate version's own `go.mod` `go` directive first — `go get`/`go mod tidy` will otherwise silently bump this project's `go` directive to satisfy a newer dependency.

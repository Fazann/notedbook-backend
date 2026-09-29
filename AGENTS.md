# AGENTS.md — Backend (Go API)

Guide for AI coding agents working on this repo. Read it fully before making changes.
For setup, env variables and the full folder tree, see `README.md`.

## Project overview

REST API for a personal life-management app: **expenses**, **planning** (goals, milestones) and
**boards** (kanban). The web client is the separate `notebook-frontend` repo (Next.js), which talks to
this API under `/api`. The API is the source of truth: all validation, permissions and totals are
enforced here, never trusted from the client.

## Tech stack

| Concern | Choice |
|---|---|
| Language | Go (see `go.mod`) |
| HTTP | Gin |
| Database | PostgreSQL via GORM (`gorm.io/driver/postgres`, pgx) |
| Cache | Redis (`github.com/redis/go-redis/v9`) |
| File storage | AWS S3 (`aws-sdk-go-v2`, transfer manager) |
| Config | `.env` loaded by `godotenv` in `config/config.go` |
| Logging | `log/slog` wrapped by `pkg/logger` |
| i18n | YAML files in `pkg/i18n/locales` → generated key constants |
| API docs | swaggo/swag annotations + gin-swagger |
| Hot reload | air (`.air.toml`) |

Do not add a dependency when the standard library or an existing one covers it (no second router,
ORM, logger, config or YAML library). Ask before adding anything new.

## Commands

```bash
make run        # API with hot reload (air)
make migrate    # GORM AutoMigrate (cmd/migration)
make swagger    # regenerate docs/ from handler annotations
make locale     # validate locale files + regenerate pkg/i18n/keys_gen.go
go build ./...  # must pass
go vet ./...    # must pass
gofmt -l .      # must print nothing
go test ./...   # must pass
```

`make build` and `make test-util` reference files that do not exist yet (`Dockerfile`, `tests/util`).
Do not rely on them; say so if a task needs them.

`make seed` runs only `cmd/seed/main.go` (`go run` on one file), so keep seed logic in the module
(e.g. `internal/exspense/service/category_seed.go`) and call it from `main.go`. Seeds must be safe to run again.

`docs/` is git-ignored and `routes` imports it: run `make swagger` before building on a fresh clone.

## Architecture

```
routes/            → registers URLs, one group per module
internal/<module>/
  handler/         → Gin: parse and validate input, call the service, write JSON
  service/         → business logic; returns response.SvcRes
  repo/            → database access, embeds shared/baserepo.BaseRepo[T]
  dto/             → request / response structs (JSON + Swagger shape)
shared/models/     → GORM models (tables)
shared/baserepo/   → generic CRUD (BaseRepo) and filter/sort QueryBuilder
pkg/               → reusable, feature-free packages (db, redis, i18n, response, logger, enums, utils...)
cmd/               → entry points: app, migration, locale
```

Rules:
- A request flows **route → handler → service → repo → database**. Handlers never call repos or `db.DB`
  directly; services never touch `*gin.Context`.
- One module never imports another module's `internal/` packages. Move shared code to `pkg/` or `shared/`.
- `pkg/` packages must not import `internal/`.
- Never return GORM models from handlers — map them to a DTO in `dto/`. Models can hold internal fields
  (passwords, S3 paths, `deleted_by`).
- Constructors follow the existing pattern: `NewXxxHandler()`, `NewXxxSvc()`, `NewXxxRepo()`, with the
  interface named `XxxServiceImpl` / `XxxRepoImpl`.

## Handlers

- Parse path/query/body, validate, then call the service with `c.Request.Context()`.
- Write every response with `pkg/response`:
  - `response.JsonSuccessRes(c, res.Data, res.StatusCode)`
  - `response.JsonSuccessResWithPage(c, res)` for paginated lists
  - `response.JsonErrRes(c, err, status)` — pass the `error`, never `err.Error()` or a string
- Never call `c.JSON` directly, so every response has the same shape.
- Every handler has a Swagger annotation block (copy one from `internal/attachments/handler`), including
  the `Accept-Language` header param and one `@Failure` line per error code it can return.
  Run `make swagger` after changing annotations or DTOs.

## Services and errors

- Return `response.SvcRespSuccess(data, status)` or `response.SvcRespErr(err, status)`.
- Errors the client should see are **translation keys** from `pkg/i18n`, never hand-written strings:

  ```go
  return response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound)
  return response.SvcRespErr(i18n.Err(i18n.FileTooLarge, i18n.Params{"max": "100 MB"}), http.StatusRequestEntityTooLarge)
  ```

- Unexpected errors (DB, S3, Redis) are passed through unchanged with a 5xx status. `JsonErrRes` logs
  them and sends a generic translated message, so internal details never reach the client. Wrap with
  context when useful: `fmt.Errorf("upload to s3: %w", err)`.
- Check `gorm.ErrRecordNotFound` with `errors.Is` and turn it into `i18n.RecordNotFound` / 404.
- Use `errors.Is` / `errors.As`, not `==` or string comparison, for errors that may be wrapped.

## Error response contract (shared with the frontend)

```json
{ "code": "RECORD_NOT_FOUND", "message": "<translated>", "status_code": 404 }
```

- `code` is the i18n key. The frontend branches on it (e.g. `INVALID_CREDENTIALS`, `USERNAME_TAKEN`),
  so **never rename or reuse an existing key** — add a new one instead.
- Success: `{ "message": "SUCCESS", "data": ..., "status_code": 200 }`.
- Changing a response shape or an endpoint path breaks the frontend. Call it out explicitly in your
  summary when you do.

## Translations (i18n)

- Languages: `en` (default and source of truth), `km` (Khmer), `ms` (Malay). The request language is
  `?lang=`, then `Accept-Language`.
- Add a key to **all three** `pkg/i18n/locales/*.yml` files, then run `make locale`. It fails if a file is
  missing a key, has an extra one, or uses different `{placeholders}`.
- Keys are `UPPER_SNAKE_CASE`, named by meaning (`FILE_TOO_LARGE`), not by the English sentence.
- Placeholders are `{name}` and filled with `i18n.Params`. Never build messages by concatenating strings.
- If unsure about a Khmer or Malay translation, still add it and mention it in your summary for review.
- Never edit `pkg/i18n/keys_gen.go` by hand.

## Database

- Use the shared pool `db.DB` (opened once by `db.Connect()`); never open another connection.
- Always pass the request context: `db.WithContext(ctx)`. `baserepo` already does this.
- `SkipDefaultTransaction` is on: wrap multi-step writes in `db.Transaction(func(tx *gorm.DB) error {...})`.
- Use GORM placeholders (`"username = ?", v`) — never format user input into SQL. For dynamic list
  filters use `baserepo.QueryBuilder` with `filter` / `sort` struct tags.
- Models embed `BaseModelID` and `BaseModelGorm` (timestamps, soft delete, created/updated/deleted by).
- Give every column an explicit `gorm:"type:..."`. Enum-like fields are `varchar` with a Go string type
  in `pkg/enums` (see `AttachmentTypeEnum`); validate the value in the service.
- New model → add it to the `AutoMigrate` list in `cmd/migration/main.go`. Changes that `AutoMigrate`
  cannot do safely (renames, type changes, dropping columns, data backfills) need a plan: ask first.
- Add an index for columns used in lookups or filters; add a unique index where the value must be unique
  (e.g. `username`).

## Config

- All env variables are read in `config/config.go` only. Other packages use `config.Xxx`.
- New variable → read it in `config.go` (with `getEnvDefault` / `getEnvInt` when it has a default),
  add it to `.env.example`, and document it in the README's "Environment variables" table.
- Never commit `.env` or real secrets. Never log secrets, passwords, tokens or full DSNs.

## Logging

- Use `pkg/logger`, never `fmt.Println` or the `log` package in app code.
- Structured key/value pairs, lowercase message:
  `logger.Error("upload to s3 failed", "err", err, "bucket", config.S3Bucket)`.
  Not printf style — `logger.Info("port: %s", port)` prints `!BADKEY`.
- `logger.Fatal` only during startup (config, DB, Redis). Never in request handling.

## Security

- Passwords: hash with bcrypt; never store, return or log them.
- Validate and limit every input (length, format, allowed values, upload size with `http.MaxBytesReader`).
- Check that the current user owns a record before reading or changing it.
- Do not expose internal IDs of other users, S3 keys or stack traces in responses.

## Code style

- `gofmt` formatting; imports grouped: standard library, then third-party and module packages.
- Accept `context.Context` as the first argument in services and repos.
- Keep comments short and only where the code is not obvious. Match the density of nearby code.
- Generics are fine where they remove real duplication (see `baserepo`); don't add abstractions for one caller.

## Tests

- Unit test pure logic (services with fake repos, helpers, i18n). Use table-driven tests.
- Tests must not need a real database, Redis or S3, and must not depend on translated wording — compare
  against the catalog or check behavior (see `pkg/i18n/i18n_test.go`).
- Packages that import `config` load `.env` from the working directory if it exists (tests run from the
  package folder, so they see only real env vars). Prefer injecting values over reading `config` in logic under test.

## Generated files — do not edit

| File | Regenerate with |
|---|---|
| `pkg/i18n/keys_gen.go` | `make locale` |
| `docs/` | `make swagger` |

## Definition of done

- [ ] `go build ./...`, `go vet ./...`, `go test ./...` pass and `gofmt -l .` prints nothing
- [ ] Follows route → handler → service → repo; DTOs returned, not models
- [ ] Client-facing errors are i18n keys, added to en/km/ms, `make locale` run
- [ ] Swagger annotations added or updated, `make swagger` run
- [ ] New env variables in `config.go`, `.env.example` and README
- [ ] New models in `cmd/migration/main.go`
- [ ] Tests added or updated for changed logic
- [ ] Any change to endpoints or response shapes called out for the frontend

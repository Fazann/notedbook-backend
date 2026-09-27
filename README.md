# Notebook Backend

REST API for the Notebook app (expenses, planning, boards), written in Go with Gin, GORM (PostgreSQL),
Redis and S3 file storage. The web client is the separate `notebook-frontend` repo.

## Requirements

- Go 1.26+
- PostgreSQL
- Redis
- An S3-compatible bucket (for file uploads)
- [air](https://github.com/air-verse/air) for hot reload: `go install github.com/air-verse/air@latest`

## Getting started

```bash
# 1. Create .env in the project root (see "Environment variables" below)

# 2. Generate the Swagger docs — docs/ is git-ignored, and the app does not compile without it
make swagger

# 3. Create the database tables
make migrate

# 4. Start the API with hot reload
make run
```

The API is served at `http://localhost:<PORT>/api`, and Swagger UI at
`http://localhost:<PORT>/swagger/index.html` (not served when `ENVIRONMENT=production`).

## Environment variables

Copy `.env.example` to `.env`. All variables are read in `config/config.go`.

| Variable | Default | Description |
|---|---|---|
| `ENVIRONMENT` | | `production` turns off Swagger UI and switches logs to JSON. |
| `PORT` | | HTTP port. |
| `JWT_SECRET` | | Secret for signing access and refresh tokens. |
| `ACC_TK_EXPIRED` | `1` | Access token lifetime in hours. |
| `REF_TK_EXPIRED` | `1` | Refresh token lifetime in days. |
| `DB_USERNAME`, `DB_PASSWORD`, `DB_NAME`, `DB_HOST`, `DB_PORT` | | PostgreSQL connection. |
| `DB_SSLMODE` | `disable` | PostgreSQL `sslmode`. |
| `DB_TIMEZONE` | `UTC` | PostgreSQL session time zone. |
| `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS` | `100`, `25` | Connection pool size. |
| `DB_CONN_MAX_LIFETIME`, `DB_CONN_MAX_IDLE_TIME` | `30`, `5` | Connection lifetimes in minutes. |
| `REDIS_ADDR`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB` | `REDIS_DB=0` | Redis connection. The port is added only when `REDIS_ADDR` has none. |
| `S3_BUCKET`, `S3_ID`, `S3_SECRET`, `S3_REGION`, `S3_PREFIX_URL`, `S3_PROJECT_NAME` | | Attachment storage on S3. |
| `SMTP_HOST` | | SMTP server for emails (password reset codes). Emails are not sent when empty. |
| `SMTP_PORT` | `587` | `587` uses STARTTLS, `465` uses implicit TLS. |
| `SMTP_USERNAME`, `SMTP_PASSWORD` | | SMTP login. Leave empty for a server without authentication. |
| `SMTP_FROM` | | Sender address. Emails are not sent when empty. |
| `SMTP_FROM_NAME` | `Notebook` | Sender name, also used as the app name in email templates. |
| `OTP_SECRET` | | Secret for the AES-GCM key returned by `/auth/mail-otp/verify`. Any long random string; password reset fails when empty. |

## Makefile commands

| Command | What it does |
|---|---|
| `make run` | Starts the API with hot reload (`air`, config in `.air.toml`). Rebuilds `./cmd/app` into `tmp/main` on change. |
| `make migrate` | Runs `cmd/migration`: connects to PostgreSQL and creates / updates tables with GORM `AutoMigrate`. |
| `make seed` | Runs `cmd/seed` to insert sample data. `cmd/seed/main.go` does not exist yet. |
| `make swagger` | Formats the Swagger annotations and generates `docs/` (`docs.go`, `swagger.json`, `swagger.yaml`) with swag v1.16.6. Run it after changing a handler annotation or a response DTO. |
| `make locale` | Checks `pkg/i18n/locales/*.yml` (same keys and `{placeholders}` in every language) and generates the key constants in `pkg/i18n/keys_gen.go`. Run it after adding or renaming a translation key. |
| `make build` | Builds the Docker image `clinic-api:latest` for `linux/amd64`. Needs a `Dockerfile`, which does not exist yet, and the image name is still the one from the clinic project. |
| `make test-util` | Copies `.env.example` to `.env` if needed, then runs the tests in `./tests/util`. That folder does not exist yet; run all tests with `go test ./...`. |

## Folder structure

```
.
├── cmd/                         # Entry points (one main package each)
│   ├── app/                     # The API server; also holds the Swagger general info (@title, @BasePath)
│   ├── migration/               # Runs GORM AutoMigrate          → make migrate
│   └── locale/                  # Translation key generator       → make locale
├── config/                      # Loads .env into package variables (config.Port, config.DbHost, ...)
├── routes/
│   ├── route.go                 # InitRouter: API routes + Swagger UI
│   └── api_route.go             # /api/* route groups, one per module
├── internal/                    # Feature modules
│   └── attachments/
│       ├── dto/                 # Request / response structs
│       ├── handler/             # Gin handlers: parse input, call the service, write JSON (+ Swagger annotations)
│       ├── service/             # Business logic; returns response.SvcRes with an i18n key on error
│       └── repo/                # Database access, built on shared/baserepo
├── shared/
│   ├── models/                  # GORM models (tables)
│   └── baserepo/                # Generic CRUD repository used by every module's repo
├── pkg/                         # Reusable packages with no feature logic
│   ├── db/                      # PostgreSQL connection pool (db.Connect, db.DB, db.Close)
│   ├── redis/                   # Redis client (redis.InitRedis, redis.RedisClient)
│   ├── i18n/                    # Translations (en, km, ms)
│   │   ├── locales/             # en.yml (source of truth), km.yml, ms.yml
│   │   └── keys_gen.go          # Generated key constants — do not edit
│   ├── response/                # JSON success / error / paginated responses
│   ├── logger/                  # Structured logging (log/slog)
│   ├── enums/                   # Shared enum types (e.g. AttachmentTypeEnum)
│   ├── constants/               # Shared constants and errors
│   └── utils/                   # Helpers (S3 upload, ...)
├── docs/                        # Generated Swagger docs (git-ignored)  → make swagger
├── tmp/                         # air build output (git-ignored)
├── .air.toml                    # Hot reload config
└── Makefile
```

A request flows **route → handler → service → repo → database**. Handlers never call repos directly.

### Adding a new module

1. Create `internal/<module>/` with `dto/`, `handler/`, `service/` and `repo/`, following `internal/attachments`.
2. Add the model to `shared/models/` and to the `AutoMigrate` list in `cmd/migration/main.go`, then run `make migrate`.
3. Register the routes in `routes/api_route.go`.
4. Add Swagger annotations on the handlers and run `make swagger`.
5. Add any new error messages to all three locale files and run `make locale`.

## API conventions

**Success**

```json
{ "message": "SUCCESS", "data": { }, "status_code": 200 }
```

**Error** — `code` is a stable translation key for clients to check; `message` is translated.

```json
{ "code": "RECORD_NOT_FOUND", "message": "The record was not found.", "status_code": 404 }
```

The language comes from the `?lang=` query, then the `Accept-Language` header: `en` (default), `km`, `ms`.
Errors without a translation key are logged, and the client gets a generic message for the HTTP status.

## Translations (i18n)

Messages live in `pkg/i18n/locales/{en,km,ms}.yml`; `en.yml` decides which keys exist.

```yaml
# en.yml
FILE_TOO_LARGE: "The file is too large. The maximum size is {max}."
```

Run `make locale` after editing, then use the generated constant. Services return only the key:

```go
return response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound)
return response.SvcRespErr(i18n.Err(i18n.FileTooLarge, i18n.Params{"max": "100 MB"}), http.StatusRequestEntityTooLarge)
```

Handlers pass it to `response.JsonErrRes(c, res.Error, res.StatusCode)`, which translates it.

## Tests

```bash
go test ./...
```

# VMS backend

The VMS backend follows the sihp-be layer-first structure: handler, usecase, repository, serializer, entity and DTO. Module interfaces and implementation subdirectories use explicit constructors and dependency injection. The approved VMS contract uses Fiber 3 and OpenAPI 3.1.1.

Current source registers the public catalog and seven staff session operations, plus health checks. Three SQL migrations define 14 of the 20 planned business entities and six justified support tables. Registration is not lifecycle or frontend conformance: bookings, administration, reporting, documents and the notification worker remain unimplemented. See the [audited API contract](../docs/contracts/API-CONTRACT.md), [known gaps](../docs/contracts/GAPS.md) and [validation results](../docs/contracts/validation-results.json). No live database state is inferred from these source counts.

The existing Swagger command publishes only source-registered operations from the reviewed manifest. Run `python ../docs/contracts/check.py --runtime --consumers` from this directory to reject unreviewed documentation/interface/DTO/consumer drift. Add `--strict` to require complete target conformance; it deliberately fails while the recorded missing operations and gaps remain.

## Manual setup

Use Go 1.27.4 (or automatic Go toolchain download), PowerShell or GNU Make, and a prepared PostgreSQL 18 database. Run commands from the backend directory. Docker is optional for manual API startup; it is used by the separate isolated rehearsal setup.

1. Create the local configuration:

   ```powershell
   make env-setup
   ```

   This copies `.env.example` to `.env` only when `.env` does not exist. Existing configuration is never overwritten. Reuse the original `APP_ENVIRONMENT`, `APP_HOST`, `APP_PORT`, `DB_*` and `SWAGGER_*` names from the example. Edit `.env` before startup; `APP_HOST` is a hostname/IP only, with the port in `APP_PORT`.

2. Configure the existing PostgreSQL runtime account and rate key using the example's original field names:

   ```dotenv
   APP_ENVIRONMENT=development
   APP_HOST=127.0.0.1
   APP_PORT=8080
   DB_HOST=127.0.0.1
   DB_PORT=55450
   DB_USER=vms_api
   DB_PASSWORD=YOUR_EXISTING_DATABASE_PASSWORD
   DB_NAME=vms_stage1
   DB_SSL_MODE=disable
   RATE_LIMIT_KEY=YOUR_EXISTING_BASE64_KEY
   ```

   Enter the database password as its raw value; the loader handles connection-URI encoding. Keep values private. Do not regenerate existing credentials or the rate key. For Docker secrets, `DB_PASSWORD_FILE` and `RATE_LIMIT_KEY_FILE` can replace their direct values; leave the direct settings empty. The prepared password file is `.runtime/stage1/db_api_password`, and its existing rate key is `.runtime/stage1/rate_key`.

   For a new setup only, generate a key with:

   ```powershell
   [Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
   ```

   Process variables override `.env`. Production requires `DB_SSL_MODE=verify-full`. Swagger is optional. JWT/SMTP and other unused template settings are not required for this slice. No duplicate VMS environment, listen or database-URL settings are supported; use the names above. `RATE_LIMIT_KEY` is needed for private IP digests and cannot be replaced by the old request-count setting.

3. Prepare the schema explicitly before API startup. The API never migrates automatically. See [isolated database and migration guide](docs/LOCAL-DEVELOPMENT.md). For a reviewed, provisioned target, run the migration tool with `DB_USER=vms_migrator` and its password configured in that process and the exact expected database:

   ```powershell
   go run ./cmd/vms-migrate -mode plan
   # Database operation: use vms_migrator credentials in this process only.
   go run ./cmd/vms-migrate -mode up -expect-database vms_stage1
   ```

   Restore/use runtime `vms_api` credentials for the API. Never store migration credentials as the API default. The existing local rehearsal is already migrated; do not recreate or reset it.

4. Start VMS:

   ```powershell
   make run
   # or build and run bin/vms-api (.exe on Windows):
   make start
   ```

   `make start` checks/creates `.env`, builds and starts VMS. An unconfigured example fails validation. Neither command starts containers or applies migrations. Default manual API address: `http://127.0.0.1:8080`. Check `/health/ready` and `/v1/public/catalogs`. Ctrl+C stops the server gracefully.

Without Make, use `go run ./cmd/vms-dev -mode env-setup`, `-mode run`, `-mode start`, `-mode build` or `-mode swagger`. These are the exact commands behind the Make targets.

## Windows PowerShell commands

Use `vms.ps1` without installing Make. It supports Windows PowerShell 5.1 and PowerShell 7 and invokes the same Go commands as the Makefile:

| Make | Windows PowerShell |
| --- | --- |
| `make env-setup` | `.\vms.ps1 env-setup` |
| `make build` | `.\vms.ps1 build` |
| `make run` | `.\vms.ps1 run` |
| `make start` | `.\vms.ps1 start` |
| `make swagger` | `.\vms.ps1 swagger` |
| `make test` | `.\vms.ps1 test` |
| `make lint` | `.\vms.ps1 lint` |
| `make migration-plan` | `.\vms.ps1 migration-plan` |

Run `.\vms.ps1` for help. You run `env-setup` yourself, then edit your local configuration before `run` or `start`. API startup reads your configuration normally. The assistant must not open actual environment or credential files; configuration examples are in `.env.example` only.

If your Windows execution policy blocks scripts, a command-scoped invocation is available without changing the saved policy:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\vms.ps1 run
# PowerShell 7 alternative:
pwsh -NoProfile -ExecutionPolicy Bypass -File .\vms.ps1 run
```

The script resolves the backend directory itself, restores your original directory and returns the Go command's exit code. It never starts containers or applies database migrations.

## Swagger

Set these in `.env` before startup:

```dotenv
SWAGGER_USERNAME=YOUR_DOCS_USERNAME
SWAGGER_PASSWORD=YOUR_PRIVATE_DOCS_PASSWORD
```

A password file is also supported through `SWAGGER_PASSWORD_FILE` instead of the direct password. The original example names are used without a VMS prefix or an additional enable flag. Configure both credentials to enable protected Swagger; leave both absent to disable it. Use loopback locally and HTTPS at any external gateway.

```powershell
make swagger
make run
```

Open `http://127.0.0.1:8080/swagger/index.html`; JSON is at `/swagger/openapi.json`. Assets are bundled locally. `make swagger` generates `docs/openapi.json` from the approved planning contract, selecting only implemented operations. It requires the parent workspace `../docs` planning file; the generated file is embedded into the binary, so regenerate before building when operation coverage changes. It does not use the old `swag init` target, which produces Swagger 2. The full 38-operation planning contract remains separate and unchanged.

## Structure and checks

- `cmd`: API process; `cmd/vms-migrate`: explicit migrations; `cmd/vms-dev`: portable development commands.
- `internal/handler`, `internal/usecase/<module>/implementation`, `internal/repository/<module>/implementation`, `internal/serializer/<module>/implementation`: sihp-be-style layers.
- `internal/server/rest/router`: feature route registration; `config`: VMS configuration.
- `pkg/dto`, `internal/entity`: wire/domain types; `migration/files`: only VMS migrations.
- `make test`, `make lint`, `make build`, `make migration-plan`: tests, vet, binary build and offline SQL-plan inspection.

The running isolated container uses port 58050, distinct from manual port 8080. Manual code changes do not update that image automatically. See [current local rehearsal details](docs/LOCAL-DEVELOPMENT.md) and [implementation checkpoint](../docs/BACKEND-IMPLEMENTATION-CHECKPOINT-2026-11-12.md).

The existing template is adapted in place for VMS. Unused template user CRUD, UUID entity hooks, pagination/envelope types and generic helpers are removed; no parallel Monogo implementation is maintained. Only add shared types and helpers when an implemented VMS operation needs them.

### Reference mapping

| SIHP-BE pattern | VMS adaptation |
| --- | --- |
| `cmd/main.go`, `config/config.go` | Existing API entrypoint and configuration adapted in place. |
| `internal/server/rest/router/*_router.go` | `PublicRouter` constructs repositories, serializer, usecases and handler; registers `/v1/public/catalogs`. |
| `internal/server/rest/middleware` | `restmiddleware` owns HTTP request context, safe errors and public admission. Routers apply middleware; admission usecase/repository retain the limit logic and persistence. |
| `internal/handler`, `internal/usecase/<module>/implementation` | Catalog HTTP binding and usecase; admission usecase. Constructors inject module interfaces. |
| `internal/repository/<module>/implementation`, `pkg/helper/database` | GORM queries and the PostgreSQL connection pool. No MongoDB implementation or second database path. |
| `internal/serializer/<module>/implementation`, `internal/entity`, `pkg/dto` | Catalog projection, domain/query types and the approved public response. |
| `migration/files`, sql-migrate | VMS-only SQL, embedded by the explicit migration command; API startup never migrates. |

Required differences from the reference are Fiber 3, integer business IDs serialized as strings, direct OpenAPI 3.1.1 responses, safe errors, published-notice handling and shared PostgreSQL admission. These follow the approved VMS contract; SIHP response envelopes, UUID business IDs, JWT middleware and domain modules are not copied. `cmd/vms-dev`, `cmd/vms-migrate` and `ops/stage1` support the existing manual workflow and isolated database setup; they are tooling, not additional application layers.

## Docker and documentation layout

There is one `Dockerfile` and one `docker-compose.yml`, matching sihp-be's root layout. The Dockerfile has `builder`, `dev`, `prod` and optional `local` targets. `prod` is the default source build; the isolated Compose setup selects `local` with prebuilt Linux binaries to conserve Docker disk space. Existing VMS project, database, volume, port and service names are preserved.

Supporting backend documentation lives under `docs/`, including [local setup and container operations](docs/LOCAL-DEVELOPMENT.md) and the embedded Swagger document under `docs/`. This root README is the entry point. Workspace-wide planning and evidence remain in the parent project's existing `../docs/` folder.

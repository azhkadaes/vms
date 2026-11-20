# VMS backend

The VMS API manages the life of a visit, from registration and staff decisions to QR admission, reception attendance and reporting. It uses Go and Fiber with PostgreSQL. The same service manages staff sessions, employees, role permissions and access to private attachments. Business rules run on the server, including schedule conflicts, booking ownership and attendance eligibility.

The backend serves the 40 operations in the [contract manifest](../docs/contracts/manifest.json), plus health checks. The [physical schema](../docs/contracts/PHYSICAL-SCHEMA-2026-11-18.md) describes the migrations. The [UI contract review](../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md#6-frontend-work-not-backend) tracks the client changes for this contract.

The Swagger command publishes only the implemented operations, from the contract manifest and source baseline in `../docs/contracts`. Run `make contract-check` (or `.\vms.ps1 contract-check`, or `python ../docs/contracts/check.py --backend . --runtime --consumers`) from this directory or from a worktree. It rejects drift between registered routes, the documented contract, the served OpenAPI, Go interfaces and DTOs, the error-code catalogue, migrations and frontend consumers. Add `--strict` to require full conformance; it fails while operations are missing or known gaps remain. The per-slice procedure is in [the Go conventions](../docs/BACKEND-GO-CONVENTIONS.md), section 8, step 4.

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

   Uploads (ops 02 and 19) stay disabled until both of these are set. Until then both routes answer 503, and the public catalog reports `attachmentsEnabled: false`:

   ```text
   UPLOAD_DIR=D:/vms-private/uploads
   UPLOAD_TOKEN_KEY=NEW_BASE64_KEY_OF_32_RANDOM_BYTES
   ```

   `UPLOAD_DIR` must be an absolute private directory that nothing serves. The API creates it if needed and checks at startup that it can write there. `UPLOAD_TOKEN_KEY` (or `UPLOAD_TOKEN_KEY_FILE`) is a dedicated key: generate it like the rate key, and keep it different from every other key. `UPLOAD_TOKEN_KEY_VERSION` (default 1) is stored with each upload. When you replace the key, raise the version. A replay of an upload issued under the old key then answers 409, rather than a token derived from the wrong key.

   Static admission QR issuance (op 07) requires `VMS_ADMISSION_QR_KEY` or `VMS_ADMISSION_QR_KEY_FILE`. Generate a separate key of at least 32 random bytes, encoded in base64. Use the same key across API instances and restarts. Changing it while issued credentials are active makes their payloads impossible to regenerate. Without the key, issuance returns 503.

   Process variables override `.env`. Production requires `DB_SSL_MODE=verify-full`. Swagger is optional. Keep `RATE_LIMIT_KEY` stable across instances and restarts. The backend uses it for private rate-limit digests and for matching existing public booking submissions and recovery.

3. Prepare the schema explicitly before API startup. The API never migrates automatically. See [isolated database and migration guide](docs/LOCAL-DEVELOPMENT.md). For a reviewed, provisioned target, run the migration tool with `DB_USER=vms_migrator` and its password configured in that process and the exact expected database:

   ```powershell
   go run ./cmd/vms-migrate -mode plan
   # Database operation: use vms_migrator credentials in this process only.
   go run ./cmd/vms-migrate -mode up -expect-database vms_stage1
   ```

   Restore/use runtime `vms_api` credentials for the API. Never store migration credentials as the API default. A stage-1 database migrated before the per-table rebuild must be reset once first; the user runs [the reset procedure](docs/LOCAL-DEVELOPMENT.md#schema-rebuild-and-stage-1-reset-18-november-2026).

4. Start VMS:

   ```powershell
   make run
   # or build and run bin/vms-api (.exe on Windows):
   make start
   ```

   `make start` checks/creates `.env`, builds and starts VMS. An unconfigured example fails validation. Neither command starts containers or applies migrations. Default manual API address: `http://127.0.0.1:8080`. Check `/health/ready` and `/v1/public/catalogs`. Ctrl+C stops the server gracefully.

Without Make, use `go run ./cmd/vms-dev -mode env-setup`, `-mode run`, `-mode start`, `-mode build` or `-mode swagger`. These are the exact commands behind the Make targets.

## Windows PowerShell commands

Use `vms.ps1` without installing Make. It supports Windows PowerShell 5.1 and PowerShell 7 and invokes the same commands as the Makefile:

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
| `make contract-check` | `.\vms.ps1 contract-check` |

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

Open `http://127.0.0.1:8080/swagger/index.html`; JSON is at `/swagger/openapi.json`. Assets are bundled locally. `make swagger` generates `docs/openapi.json` from `../docs/contracts/manifest.json` and `../docs/contracts/source-baseline.json`, selecting only implemented operations; each carries its contract number as `x-operation-number`. It fails if the two files disagree. The generated file is embedded into the binary, so regenerate before building when operation coverage changes, and update the served-operation list in `internal/server/rest/server_test.go`. It does not use the old `swag init` target, which produces Swagger 2.

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
| `internal/server/rest/router/*_router.go` | `PublicRouter` constructs repositories, serializer, usecases and handler; registers `/v1/public/catalogs`. `StaffRouter` registers `/v1/staff/catalogs` and `RoleRouter` registers `/v1/staff/roles`, each as its own group behind `deps.StaffAuth` and `deps.Workspace`. |
| `internal/server/rest/middleware` | `restmiddleware` owns HTTP request context, safe errors and public admission. Routers apply middleware; admission usecase/repository retain the limit logic and persistence. |
| `internal/handler`, `internal/usecase/<module>/implementation` | Catalog HTTP binding and usecase; admission usecase. Constructors inject module interfaces. |
| `internal/repository/<module>/implementation`, `pkg/helper/database` | GORM queries and the PostgreSQL connection pool. No MongoDB implementation or second database path. |
| `internal/serializer/<module>/implementation`, `internal/entity`, `pkg/dto` | Catalog projection, domain/query types and the approved public response. |
| `migration/files`, sql-migrate | VMS-only SQL, embedded by the explicit migration command; API startup never migrates. |

Required differences from the reference are Fiber 3, integer business IDs serialized as strings, direct OpenAPI 3.1.1 responses, safe errors, published-notice handling and shared PostgreSQL admission. These follow the approved VMS contract; SIHP response envelopes, UUID business IDs, JWT middleware and domain modules are not copied. `cmd/vms-dev`, `cmd/vms-migrate` and `ops/stage1` support the existing manual workflow and isolated database setup; they are tooling, not additional application layers.

## Docker and documentation layout

The [Docker setup](docs/LOCAL-DEVELOPMENT.md#start-a-fresh-local-docker-project) builds the API and migration tool, starts PostgreSQL, and checks that the API is ready. Compose uses the source build by default. The guide also shows how to use prebuilt Linux binaries when the Docker disk is tight.

Supporting backend documentation lives under `docs/`, including [local setup and container operations](docs/LOCAL-DEVELOPMENT.md) and the embedded Swagger document under `docs/`. This root README is the entry point. Workspace-wide planning and evidence remain in the parent project's existing `../docs/` folder.

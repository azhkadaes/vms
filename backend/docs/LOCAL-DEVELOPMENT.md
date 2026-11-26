# VMS first implementation slice

> Current contract: [the 40-operation reference](../../docs/contracts/FRONTEND-DRIVEN-REST-API-2026-11-16.md), with each operation's status in [the manifest](../../docs/contracts/manifest.json). The public and staff catalogs, role permissions (ops 30, 31) and seven identity routes now exist; the first-slice rehearsal details below describe their earlier checkpoint. This audit changed documentation/checks only and ran no database or container operation.

## Schema rebuild and stage-1 reset (18 November 2026)

The four multi-table migrations were replaced by one schemas prerequisite plus one migration per table (`migration/files/202611180001` to `202611180023`), built from [the physical schema](../../docs/contracts/PHYSICAL-SCHEMA-2026-11-18.md). The existing stage-1 database still holds the old tables and migration ledger, so it must be reset once. **The user runs this; no agent does.** It drops only the `vms` and `support` schemas (all synthetic rows and the ledger). Roles, passwords, database grants, the `.runtime/stage1` credential files and the `pg18_data` volume are kept.

Run from `D:/oikn/VMS/backend` in PowerShell 7. Passwords are read inside the database container from its mounted secret and are never printed.

```powershell
# 0. Stop the API so no request runs during the reset (the database keeps running).
docker compose -f docker-compose.yml --profile api stop api

# 1. Drop both schemas as vms_migrator. The script refuses any other database or role.
Get-Content -Raw ops/stage1/reset-schema.sql | docker compose -f docker-compose.yml exec -T db sh -c 'PGPASSWORD="$(cat /run/secrets/db_migrator_password)" psql -h 127.0.0.1 -U vms_migrator -d vms_stage1 -X -v ON_ERROR_STOP=1 -f -'

# 2. Rebuild the binaries and image so vms-migrate embeds the new migrations, then migrate up.
pwsh -File ops/stage1/build-linux.ps1
docker compose -f docker-compose.yml build api
docker compose -f docker-compose.yml --profile tools run --rm migrate -mode plan
docker compose -f docker-compose.yml --profile tools run --rm migrate -mode up -expect-database vms_stage1
# Expect "Applied 23 VMS migrations". The migrator recreates both schemas itself.

# 3. Re-seed the synthetic catalog (updated: vms.ruangan no longer has a ruangan column).
Get-Content -Raw ops/stage1/seed-test-catalog.sql | docker compose -f docker-compose.yml exec -T db sh -c 'PGPASSWORD="$(cat /run/secrets/db_migrator_password)" psql -h 127.0.0.1 -U vms_migrator -d vms_stage1 -X -v ON_ERROR_STOP=1 -f -'

# 4. Start the API and verify readiness, the catalog and the repository integration suites.
docker compose -f docker-compose.yml --profile api up -d api
pwsh -File ops/stage1/verify.ps1
```

If step 1 fails, nothing was dropped (the script runs in one transaction). If step 2 fails part-way, inspect `-mode status` and repeat step 1 before retrying. The reference seeds (7 deputy groups, 21 directorates, 4 roles, 7 features, 10 actions, 14 permissions, one default policy per role with the frontend's default permissions, all `created_by = system`) come from the migrations themselves; no employees are seeded.

Integration suites after the rebuild:

- `VMS_INTEGRATION_DATABASE_URL` (vms_api on `vms_stage1`): `go test -tags integration ./internal/repository/... ./pkg/helper/database`. These roll back their fixtures. The staff catalog test needs an inactive official that no employee is actively assigned to (the synthetic seed in step 3 has one) and skips without it, because `vms_api` cannot insert officials. The role concurrency tests need several connections, so they commit their employees and sessions and leave them deactivated and revoked (`vms_api` cannot delete employees); every role save they make is rolled back, so the seeded grants never change.
- `VMS_INTEGRATION_MIGRATION_DATABASE_URL` (vms_migrator on a separate, disposable `vms_scratch` database): `go test -tags integration ./migration`. It runs up → down → up and compares the applied catalog with the physical schema. It refuses `vms_stage1`.

12 November 2026. The existing entry point, configuration, layers and tooling are adapted in place for VMS; no separate Monogo implementation is retained. The main README and default Makefile now describe VMS manual startup, optional .env configuration and protected Swagger. Use this guide for the isolated container rehearsal. Existing .cursor rule renames are user changes and remain untouched.

## Implemented scope

Follow sihp-be's layer-first layout for new code: handlers delegate to module usecase interfaces, usecases coordinate repositories and serializers, repositories own GORM access, and serializers map entities to DTOs. Interfaces live in each module package; concrete implementations live in its implementation subdirectory. Use explicit names such as CatalogRepository, CatalogUsecase, CatalogSerializer and NewCatalogUsecase, private repo/usecase/serializer structs, named interface parameters and explicit constructor injection. Register feature routes in router files. Keep comments focused on constraints and non-obvious behavior.

Reference sources: sihp-be/internal/usecase/auth, internal/repository/auth, internal/serializer/sihp, internal/handler/auth_handler.go and internal/server/rest/router/auth_router.go. Approved VMS differences remain: Fiber 3, the frozen response contract, safe centralized errors, PostgreSQL-backed admission and VMS session/authorization rules. Do not copy reference response envelopes, raw error exposure or Swagger 2 output over those decisions.

- cmd/main.go: Fiber 3 public GET /v1/public/catalogs plus /health/live and /health/ready. No template user CRUD, staff endpoints or booking mutations are registered.
- sihp-be-style repository/usecase/serializer interfaces and implementation packages. Active officials join Jabatan for VIP; active rooms use stable integer IDs serialized as strings. Both catalog queries share a read-only repeatable-read snapshot.
- PostgreSQL-backed120/minute public request admission with keyed IP digests, serialized per-bucket writes, trusted-proxy allowlist and429/Retry-After. Limiter/database failure never returns a fabricated empty success. Health probes do not depend on the limiter.
- Required explicit database credentials, TLS validation policy, request IDs, redacted safe errors, no-store/nosniff, explicit15-second usecase contexts and fresh shutdown-drain context. A published notice is returned only when explicitly configured; new uploads are available only when `UPLOAD_DIR` and `UPLOAD_TOKEN_KEY` are configured (see the root README).
- Version/ID/Unicode/password-bound primitives, request/usecase OpenTelemetry spans. No exporter/collector acceptance is claimed; remaining instrumentation/export configuration belongs to subsequent development.
- cmd/vms-migrate uses the same sql-migrate tool family as sihp-be. Its embedded source contains only the VMS SQL in migration/files. The first SQL migration creates 3 business tables (Jabatan/Pejabat/Ruangan) and 2 support rate-limit tables; other approved entities follow in their owning slices.

Identity/session foundation now includes an unapplied second migration, current-authority repository, session validation usecase and explicit Session serializer. The migration adds the ten identity business entities (employee, roles/policies/permission graph and assignment links) plus private credential/session support tables. No accounts, role grants or policies are seeded. Runtime access is read-only and excludes credential hashes and refresh hashes. The repository reads one repeatable-read snapshot without renewing activity; the usecase rejects inactive/revoked/expired sessions, changed credential generations and stale role revisions. Role-selection/password-change gates grant no workspace access. Resource mutations must additionally recheck authority inside their own transaction.

This foundation is deliberately not registered as an HTTP route: SessionProof must come from a future cryptographically verified access token, never request JSON. Login/password verification, JWT signing/verification, shared login admission, CSRF/document initialization, refresh rotation/recovery, renewal, role switching, logout and password transitions remain pending. The prepared migration has only been parsed offline; its PostgreSQL DDL, grants and queries are not yet runtime-verified. Existing containers/databases still describe migration 001. No booking/notification worker, live email, genuine visitor registration, production uploads or production deployment is enabled. Existing frontend adapters still need the selected room-object and other coordinated contract changes; no real frontend integration pass is claimed.

## Configuration

Use the original example names: `APP_ENVIRONMENT`, `APP_HOST` (host/IP only), `APP_PORT`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE` and `SWAGGER_USERNAME`/`SWAGGER_PASSWORD`. The loader constructs the database URI; no separate environment/listen/URL variables are needed. `RATE_LIMIT_KEY` is the dedicated digest key. Optional `_FILE` sources are for secret mounts, not additional required setup. Compose uses the existing database password and rate-key files directly. See the root README for manual values. Unused template options remain omitted until their feature needs them.

## Prepared isolated environment

docker-compose.yml defines project vms-backend-stage1, a new pg18_data volume, PostgreSQL18.6, local-only default ports55450(DB)/58050(API), separate owner/migration/runtime accounts and file-based private credentials. PostgreSQL18 uses /var/lib/postgresql as its volume mount. The original isolated VMS project and volume identifiers are preserved; no unrelated database directory or volume is referenced.

Compose builds the API and migrator into the same non-root image. Its default `prod` build compiles the Go source in Docker. The prebuilt Linux option below uses the same pinned toolchain and saves Docker disk space. The default ports bind to loopback. The API and migration tool use separate profiles. This Compose file is for local development.

## Persistent Docker on Windows

1. Open Docker Desktop and wait for its engine. Double-click `Start VMS.cmd` in the VMS checkout. The first run builds the frontend and API images, starts PostgreSQL, and applies the migrations before the API starts. Later runs reuse the images and database.
2. Open the [visitor app](http://localhost:58101/) or the [staff app](http://localhost:58101/admin/). Both apps make requests to the real API through the same address. Use `localhost` as shown for staff sign-in.
3. For API testing, set `SWAGGER_USERNAME` and `SWAGGER_PASSWORD` in `backend/.env`, then run `Start VMS.cmd` again. The launcher prints the Swagger and readiness URLs for the configured port. With the default port, open [Swagger](http://localhost:8080/swagger/index.html) or [API readiness](http://localhost:8080/health/ready). Leave both Swagger credentials blank to disable Swagger.
4. Double-click `Stop VMS.cmd` to stop the stack. Start it again with the same file. The database and uploads stay in Docker volumes `vms-postgres` and `vms-uploads`.

The Compose project is named `vms`. Port `58101` serves both apps. Set `APP_PORT` in `backend/.env` to change both the backend listener inside Docker and its direct port on the local machine. It defaults to `8080`. Ports `8180` and `58101` are reserved for the frontend. Both published ports listen only on the local machine, and PostgreSQL uses the private Compose network.

Docker Desktop lists `58101:8180` under the frontend container and the direct API port under `vms-backend`. The frontend uses its own network connection and contacts `api` through a dedicated gateway network, so restarting Docker no longer leaves it attached to an old backend network namespace. Only the frontend address `172.30.81.2` is allowed to initialize staff forms, and the gateway secret is still required. The frontend health check also verifies backend readiness.

For a new empty catalog, run `".\Start VMS.cmd" -SeedTestCatalog` from Command Prompt, or `& '.\Start VMS.cmd' -SeedTestCatalog` from PowerShell. This inserts fictional officials and rooms once and turns on a clearly labeled testing notice if no operator notice is configured. The test data stays in the local database on restart. In Swagger, try `GET /v1/public/catalogs`, `POST /v1/public/attachments`, and `POST /v1/public/bookings`.

The bundled gateway is for local testing. Production staff sign-in requires employee accounts and a separate HTTPS gateway. Do not enter real visitor details under the test notice. Use `-Rebuild` after pulling changes to either app or the backend.

If local test staff accounts have been provisioned, keep their login details in `backend/.runtime/local/staff_test_accounts.env`. The file is ignored by Git and excluded from Docker builds. The launcher does not load it as application configuration.

Verified on Windows on 20 November 2026. The launcher applied all 30 migrations on the first start and zero on the second. Swagger authentication, a visitor photo upload and a test booking passed. The same booking was found after using the stop and start launchers.

The production frontend builds were tested in Chrome against the Docker API. The visitor form loaded the live catalog. A local test operator could sign in, open the staff dashboard, and keep their session after reloading.

## Start a fresh local Docker project

First inventory current Docker projects/containers/volumes and listening ports. Confirm the named project/volume are new and the two loopback ports are available. If they already exist, inspect them and preserve their contents rather than reuse/delete silently. Use only the VMS Compose file below; legacy make migrate-up does not apply.

Run from the backend directory. Set `VMS_NOTICE_VERSION`, `VMS_NOTICE_ID` and `VMS_NOTICE_EN` to the approved privacy notice before enabling public registration. Leaving them unset keeps registration unavailable. Set `VMS_STAGE1_STAFF_ORIGIN` to the exact staff site origin if you serve it through a separate gateway.

```powershell
pwsh -File ops/stage1/prepare.ps1
docker compose --profile api --profile tools config --quiet
docker compose build api
docker compose up -d --wait db
docker compose --profile tools run --rm migrate -mode up -expect-database vms_stage1
docker compose --profile api up -d api
Invoke-RestMethod http://127.0.0.1:58050/health/ready
```

The migration command applies the current 30 migrations on a fresh database. It can be run again on the same database. If the database is from the original four-migration stage-1 setup, follow the separate [legacy reset procedure](#schema-rebuild-and-stage-1-reset-18-november-2026) only for that disposable database.

`prepare.ps1` generates private files under the ignored `.runtime/stage1` directory. For an existing directory, run `pwsh -File ops/stage1/prepare.ps1 -AddMissingFeatureKeys` to add the staff, upload and QR keys while preserving existing credentials. The API writes uploads to a separate named volume. Only the migrator gets the migration password and has permission to change the schema.

Staff sign-in requires existing employee accounts and a same-origin staff gateway. The browser lifecycle uses secure cookies. If you add a gateway, set `VMS_STAGE1_STAFF_ORIGIN` to its HTTPS origin and `VMS_STAGE1_GATEWAY_NETWORKS` to the exact private gateway peer CIDRs. Keep the gateway initialization key private and the initialization route behind the gateway. The fresh database seeds roles and permissions only.

If Docker cannot download the build dependencies or disk space is tight, build the two Linux/amd64 binaries using the pinned Go toolchain and select the smaller local image before running the same Compose commands:

```powershell
pwsh -File ops/stage1/build-linux.ps1
$env:VMS_STAGE1_BUILD_TARGET = 'local'
$env:VMS_STAGE1_BINARY_CONTEXT = './.runtime/linux'
docker compose build api
```

Verify actual DDL/roles, process startup/readiness, direct catalog responses, real rate-limit concurrency/restart behavior and denied runtime DDL. Synthetic catalog seeds may be used in this disposable database; no real-person data is required. The opt-in Go integration test requires -tags integration and an explicit VMS_INTEGRATION_DATABASE_URL for vms_api on vms_stage1. Ordinary go test ./... never runs it.

For a disposable database populated with `ops/stage1/seed-test-catalog.sql`, `pwsh -File ops/stage1/verify.ps1` checks the catalog, upload capability, staff session protection, and repository behavior. Pass `-NoticeVersion` if you started the API with a published notice. The integration suites insert test records.

On 20 November 2026, the source image ran against a fresh PostgreSQL 18.6 container. The first migration run applied 30 migrations and a second applied zero. A synthetic visit completed photo upload, registration and recovery, staff login and approval, private photo download, QR issuance, check-in and checkout. The prebuilt image also built successfully and listed all 30 migrations.

Stop only this reviewed project when its run is finished; retain its named volume for inspection. Removing volumes or resetting data needs a separately identified target/review. No down migration is exposed by the VMS command. Existing-data migration and production deployment remain separate later scopes.

## Offline verification

C: ran out of build temporary space during the first compilation. Use task-owned D: scratch directories. With explicit approval, the verified original Go build cache was cleared; source, module caches and database data were preserved:

```powershell
$env:GOCACHE='D:/oikn/VMS/backend/.runtime/go-cache'
$env:GOTMPDIR='D:/oikn/VMS/backend/.runtime/go-tmp'
go test ./...
go vet ./...
go run ./cmd/vms-migrate -mode plan
```

The last command only parses embedded migration statements and requires no database configuration. Unit/HTTP test doubles do not prove PostgreSQL locking, actual migration syntax, Docker secret-file permissions or container runtime compatibility. Evidence and current authorization are tracked in ../../docs/BACKEND-IMPLEMENTATION-CHECKPOINT-2026-11-12.md relative to this guide.

## Completed local rehearsal

The isolated environment is now running on the ports above. One migration applied successfully; repeating it applied zero changes. Three business tables and the support tables/ledger exist. Runtime DDL is denied. Five synthetic catalog rows remain for inspection; do not rerun preparation or the guarded seed against this existing environment.

The initial migration attempt exposed sql-migrate's requirement to issue CREATE SCHEMA. After confirming no application tables/ledger had been created, only the migrator received database-local CREATE; init-db.sh now includes that grant for future fresh environments. The API has no database/schema CREATE. No database reset was used.

The real catalog check caught and corrected a GORM VIP alias-mapping error. The rebuilt API passes active-only catalog projection, position-derived VIP, string IDs, private metadata exclusion, disabled uploads, unknown-query rejection and legacy-route exclusion. The opt-in PostgreSQL test passes concurrent admission (120 accepted/10 rejected), expiry recovery and runtime privilege checks. An actual API restart preserves an admitted-request record and restores readiness. Ordinary tests contain 29 passing cases; go vet passes.

For this existing fixture environment, `pwsh -File ops/stage1/verify.ps1` checks API behavior and runs the opt-in PostgreSQL test. It reads private credentials internally and cleans only its unique synthetic rate-limit bucket. Results live in ignored .runtime/stage1; the public evidence summary is docs/implementation-evidence/2026-11-12/stage1-local-rehearsal.json at the workspace root. The source-build prod target, race detector, frontend integration and production topology remain unverified. Only 1 of 38 business operations and 3 of 20 business entities are implemented.

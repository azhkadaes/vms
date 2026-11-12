# VMS backend implementation checkpoint

Updated 12 November 2026, WITA. Owner/source chat: `01a1ca52-c89c-7c22-9a07-88b31f9d5c86`; retrieve history through that Codex chat. Workspace `D:/oikn/VMS`; nested backend base `e4ca52f`. No worktree or commit created. Preserve the user's existing `.cursor` rule renames. Reference `C:/Users/ASUS/go/src/github.com/thdoikn/sihp-be` remains unchanged.

**Current user restriction:** the assistant may access `.env.example` only for configuration work. Do not read or write actual environment files, private credential files or their values, including indirectly by starting the API or running setup against the user's environment. Earlier credential/configuration actions below are historical and do not authorize repeating them. The user runs configuration/startup commands themselves.

Latest Swagger naming correction: the user requires original example names. Loader and README now use SWAGGER_USERNAME/SWAGGER_PASSWORD, without VMS_ or a separate enable flag. Both credentials enable protected docs; both absent disable them; a partial pair fails validation. SWAGGER_PASSWORD_FILE remains optional. The current .env.example was inspected and left unchanged. Only map-based Swagger configuration and HTTP route tests ran for this correction; no environment/credential files or live startup were accessed. Earlier VMS_SWAGGER naming below is historical and superseded.

The user approved planning, then instructed “okay lets start proceeding the plan,” authorizing implementation and local rehearsal. Existing-data cutover, production deployment, live email, genuine visitor registration and production document storage remain outside this run. Frozen planning artifacts remain unchanged; their historical planning-only approval does not cancel the later implementation instruction.

## Implemented and verified

The first vertical slice implements **1 of 38 business operations**, `GET /v1/public/catalogs`, plus liveness/readiness. It uses Go 1.27.4, Fiber 3, GORM, explicit repository/usecase/serializer interfaces and implementations following compatible sihp-be patterns. Separate `cmd/vms-api` and `cmd/vms-migrate` entrypoints exclude template routes and legacy migrations.

The catalog returns active officials/rooms, position-derived VIP, canonical string IDs, empty arrays rather than null, and no private room metadata. Uploads remain disabled. PostgreSQL provides shared 120/minute public admission using keyed IP digests, row serialization and database time. Trusted proxies must be configured explicitly. Safe errors, request IDs, no-store, bounded contexts and graceful shutdown are implemented. OTel request/usecase spans exist; exporter acceptance remains pending.

- 29 unit/HTTP/real-socket test cases pass after the catalog mapping fix; `go vet ./...` passes. Both entrypoints build for Windows and Linux/amd64.
- Earlier govulncheck 1.1.4 on both entrypoints reported no vulnerabilities after patching pgx to 5.9.2, retained Fiber 2 to 2.52.13 and x/crypto to 0.59.0. Native race-detector coverage was not run: no host GCC.
- The real PostgreSQL integration test passes: 130 concurrent requests across two repository instances yield exactly 120 admitted/10 denied, rejection does not extend the window, expired synthetic events recover, and the runtime account cannot create schema objects.
- Container API checks pass: readiness, active-only/VIP projection, string IDs, omitted private metadata, disabled uploads, rejected unknown query and absent legacy user route.
- Restarting only the API recovers readiness and retains a prior admission record. Repeating the migration applies zero changes.

Evidence: [local rehearsal](implementation-evidence/2026-11-12/stage1-local-rehearsal.json). The earlier `stage1-offline.json` is a historical pre-container snapshot, not current deployment status.

## Running local environment

Compose project `vms-backend-stage1`; new volume `vms-backend-stage1_pg18_data`; database `vms_stage1`. PostgreSQL 18.6 is healthy on `127.0.0.1:55450`; the nonroot API is ready on `127.0.0.1:58050`. Only these new services were changed; unrelated containers/volumes were preserved. Services and synthetic fixtures remain running for inspection.

One migration is applied: **3 of 20 business entities** (`vms.jabatan`, `vms.pejabat`, `vms.ruangan`), two limiter tables and the support migration ledger. Five labelled synthetic catalog rows remain: one position, two officials, two rooms. No visitor or employee data was registered.

The first migration attempt failed before tables/ledger were created because sql-migrate issues `CREATE SCHEMA IF NOT EXISTS`. After inspecting the empty state, database-local CREATE was granted only to `vms_migrator`, and bootstrap SQL was corrected for future fresh setups. The retry applied one migration. `vms_api` has neither database CREATE nor business-schema CREATE. No volume reset or down migration occurred.

A real DB run caught a GORM `VIP` alias-mapping error; an explicit private persistence-row column tag and mapping fixed it. The rebuilt API passed the fixture check. Do not claim unit tests alone established this behavior.

Private credentials stay in ignored `backend/.runtime/stage1`, accessible only to the verified task/host accounts. Do not print values, expanded connection URLs or raw database logs. Preparation refuses overwrite: **do not regenerate credentials or reseed the existing environment**. Retain the volume; do not use `down -v` or legacy Compose/migration commands.

## Resource notes and next work

Latest structure consolidation: backend now has one Dockerfile (builder/dev/prod/local targets) and one docker-compose.yml. These supersede Dockerfile.vms, Dockerfile.vms-local and compose.vms-stage1.yml; the old template Dockerfile/Compose were replaced and duplicate task-created files removed, including Makefile.monogo. Project vms-backend-stage1, services, named volume, DB port and secret-file references are preserved. Supporting backend guide moved from VMS-STAGE1.md to backend/docs/LOCAL-DEVELOPMENT.md; root README links to it. Parent workspace planning/evidence stay in their existing docs folder. Static YAML checks, build-target/reference checks and active documentation links pass. No Docker operations, environment/credential access or new image build occurred. Historical evidence paths/image IDs below describe previous builds; use current Dockerfile/docker-compose.yml for future work.

Windows command follow-up: backend/vms.ps1 now mirrors all eight Make targets (env-setup/build/run/start/swagger/test/lint/migration-plan), defaults to help, resolves its own backend directory and returns child exit codes. README includes the command mapping. Verified syntax, all eight mappings using a mocked Go command, directory restoration and failure-code propagation. No actual startup/setup or configuration/credential access occurred for this change.

Manual workflow follow-up: the default backend README, Makefile and .env.example now target VMS. Retained template copies are docs/MONOGO-TEMPLATE-README.md, Makefile.monogo and .env.monogo.example. `make env-setup/build/run/start/swagger` invoke the portable cmd/vms-dev helper. Config loads optional .env without overriding process variables; the migration command loads it only for explicit database modes. Swagger is opt-in and Basic-auth protected, with bundled UI assets and embedded OpenAPI 3.1.1 showing only the implemented catalog operation. `make swagger` derives that runtime document from the unchanged parent planning contract; it does not run Swagger 2 generation.

33 test cases, go vet and native build pass. A native smoke run on temporary loopback port 58051 passed .env loading, process override, DB readiness, catalog projection and authenticated Swagger UI/spec; that smoke process was stopped. GNU Make is unavailable here; helper setup/generation/build commands and native startup were verified, rather than claiming make itself ran. The existing container image was not rebuilt. Local .env now points to private files for a host DB URL, existing rate key and new Swagger password, with manual listen default 8080 and Swagger username vms-docs. Values were not printed. No schema, visitor or production data change was made. See manual-workflow.json for source evidence; no browser visual check is claimed.

Subsequent structure alignment (same date): the user explicitly required sihp-be's code structure and writing approach. Compared owning auth repository/usecase/handler/router and SIHP serializer sources. New modules now use explicit CatalogRepository/CatalogUsecase/CatalogSerializer and AdmissionRepository/AdmissionUsecase names, NewCatalogRepository-style constructors, private repo/usecase/serializer implementations, named interface parameters, interface assertions, explicit wiring variables and a separate catalog router file. The approved wire response and VMS behavior are unchanged. VMS-STAGE1.md records these conventions for later slices.

Post-refactor checks: all 29 ordinary tests and go vet pass; the integration suite compiles. An intermittent Windows socket-test failure was isolated to uploading a body while the server rejects the declared oversized Content-Length. The test now sends the oversized headers directly over TCP and verifies the actual early 413/correlation response; 20 consecutive runs pass. No new container build, DB migration or real PostgreSQL integration run was performed for this naming/router refactor. Running images and stage1-local-rehearsal.json still identify the preceding tested build; source verification is recorded separately in structure-alignment.json.

Go scratch uses `.runtime/go-cache` and `.runtime/go-tmp` on D: after C: exhausted temporary space. With explicit approval, only the verified original Go build cache was cleared; no source, module cache or database data was deleted. Local image builds use `ops/stage1/build-linux.ps1`, `Dockerfile.vms-local` and the binaries-only `.runtime/linux` named context. The full source-build Dockerfile was not exercised. See [run guide](../backend/VMS-STAGE1.md).

Next capability: identity/session foundation. Inspect sihp-be owning auth sources and the frozen VMS auth contract first. Reuse compatible patterns, retain approved VMS session/CSRF/authority exceptions, and defer nonurgent absent-reference mechanics rather than restart the 100-question planning sequence. Identity/session, the other 37 operations/17 business entities, booking writes, worker/outbox, documents, administration/reports, frontend integration and production release remain pending. This checkpoint does not claim the full backend or identity slice is complete.

Monitoring: hooks remain unobserved; bounded snapshot after automatic compaction observed about 7.3% input/window. Economics unknown. Check this trusted chat at meaningful milestones; offer context choices at 80%, respecting recorded preferences.

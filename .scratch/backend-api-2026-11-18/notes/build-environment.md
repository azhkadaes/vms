# Build environment and follow-ups for implementer agents

Shared notes for every agent working on the [backend spec](../spec.md). Read this before running Go or the contract tooling.

## Go toolchain

- The `go` on PATH (`C:\Program Files\Go\bin`) is **1.26.1**. The module requires **go 1.27.4**.
- **Prepend the cached toolchain** to PATH before any `go` command: `C:\Users\ASUS\go\pkg\mod\golang.org\toolchain@v0.0.1-go1.27.4.windows-amd64\bin`.
- **Always set:**
  - `GOTOOLCHAIN=local`
  - `GOPROXY=off`
  - `GOCACHE=D:\oikn\VMS\backend\.runtime\go-cache`
  - `GOTMPDIR=D:\oikn\VMS\backend\.runtime\go-tmp`

  The C: drive is low on space.
- No new third-party dependencies unless the ticket says so. Module downloads are disabled.

- **The C: drive has almost no free space.** Also set `TMP` and `TEMP` to `D:\oikn\VMS\backend\.runtime\go-tmp`.

## Git layout (implement-spec)

- The backend is a nested git repo at `D:\oikn\VMS\backend`. Its main checkout stays on the integration branch **`vms/backend-api`**, and only merger agents commit there.
- **Implementers:** create a worktree with `git -C D:/oikn/VMS/backend worktree add D:/oikn/VMS/backend-wt-<NN> -b ticket/<NN>-<slug> vms/backend-api`. A worktree at that path keeps `../docs/contracts` resolvable.
- The outer repo `D:\oikn\VMS` (docs, `.scratch`) is not committed by agents.
- Commit trailer: `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Integration tests (from ticket 00)

- **`VMS_INTEGRATION_DATABASE_URL`** points at `vms_stage1` as `vms_api` on loopback. It is used by the repository and readiness suites. Fixtures run in a rolled-back transaction via `internal/repository/integrationtest`.
- **`VMS_INTEGRATION_MIGRATION_DATABASE_URL`** points at a separate loopback `vms_scratch` database as `vms_migrator`. It is used by the destructive migration cycle test, which refuses `vms_stage1`.
- In your throwaway cluster, create `vms_stage1` and `vms_scratch` with no schemas; the prerequisite migration creates them. Apply migrations with the built `vms-migrate` from a directory with no `.env`.
- **Migrations, one per table:** `migration/files/YYYYMMDDNNNN_create_<schema>_<table>.sql`. `migration/source_test.go` asserts the exact ordered ID list and that each file creates and drops only its own table, so update it when you add tables. Readiness (`pkg/helper/database`) must list every migrated table; a unit test enforces that.

## Forbidden

- Never read or write `backend/.env`, `backend/.runtime/stage1/**`, or any credential or secret file. `.env.example` may be read.
- Do not touch Docker containers, the stage-1 database (port 55450) or its volume, and do not run migrations against stage-1. The user resets stage-1 themselves.
- Never push. Never rewrite another ticket's commits.

## Throwaway PostgreSQL for integration tests

PostgreSQL 18 binaries are at `C:\Program Files\PostgreSQL\18\bin`. Use a private cluster per agent, with trust auth on `127.0.0.1`:
- **Data directory:** `D:\oikn\VMS\backend\.runtime\pg-scratch-<ticket>`.
- **Port:** `56000 + <ticket number>`.
- **Roles and database:** create the ones the migration command and the integration-test guard expect (`vms_migrator`, `vms_api`, database `vms_stage1`; see `ops/stage1/init-db.sh` and the guard in the integration tests). See ticket 00's committed procedure if one exists.
- **Cleanup:** stop the cluster and delete its directory when done.

## Error and transport helpers (from ticket 01)

| Need | Use |
| --- | --- |
| Field errors | `failure.Validation().Add(key, msg)…Err()`. `Err()` is nil when nothing was added. |
| Status error | `failure.New(status, code)`, using codes from the catalogue in `pkg/helper/failure/catalogue.go` |
| Cause for logging | `WithCause` attaches a cause that is logged but never returned |
| Not-found | In each repository interface package, declare `var ErrNotFound = fmt.Errorf("<module>: %w", failure.ErrNotFound)`. Return it wrapped with operation context. **Never return `gorm.ErrRecordNotFound`**; it stays a 500. |
| Query parsing | `handler.Query(c, allowedKeys...)` |
| Uploads | `handler.Multipart(c, maxFileBytes)` → `UploadForm{File, Filename, Reference, Purpose}`. The server's `BodyLimit` is still 256 KiB, so ticket 05 must raise it for upload routes while JSON keeps its own cap. |
| Tri-state field | `dto.Optional[T]` (omitted / null / value), compatible with `DecodeJSON` |
| Timestamps | Serializers must call `.UTC()` on every `time.Time`. The contract capture test asserts `Z`. |

## Staff routes and authority (from ticket 02)

- The old `CurrentAuthority` helper is gone. In a write transaction, use `authrepository.AuthorityLock` (implementation: `authrepositoryimplementation.NewAuthorityLock(tx)`). Call `LockAuthority` first, then `authusecase.SessionValidator.Validate`, then the policy checks.
- Use the `policy.Role*` and `policy.Perm*` constants, never string literals.
- Mount every staff business group as `deps.App.Group("/v1/staff/<area>", deps.StaffAuth, deps.Workspace)` with a literal prefix. Never mount on `/v1/staff` itself: that would put login and refresh behind Bearer.
- A-18: when both role selection and password change are pending, the session is in the role-selection stage until a role is chosen.

## Contract pipeline (from ticket 03)

- The manifest, source baseline and checker track the 40-operation contract by number. The per-slice procedure is in `docs/BACKEND-GO-CONVENTIONS.md` section 8, step 4.
- Run the checker against your own checkout: `python ../docs/contracts/check.py --backend . --runtime --consumers` from the worktree (`make contract-check`; without make, `.\vms.ps1 contract-check`). It needs the Go toolchain settings above on PATH.
- Implementers do not edit `docs/contracts`. To validate contract updates, copy `manifest.json` and `source-baseline.json` to a scratch directory, edit the copy, then run `go run ./cmd/vms-dev -mode swagger -contracts <dir>` and `check.py --backend . --contracts <dir> --runtime --consumers`. Return the edits as `contractUpdates`.
- `check.py --backend . --write-inventory` refreshes the manifest's interface, DTO and migration inventory; review its diff.

## Staff business router (from ticket 04)

- `router.RegisterStaffRoutes(deps, catalog)` in `internal/server/rest/router/staff_router.go` is the shared staff business router; `RestServer.RegisterRoutes` calls it after the verifier-failure fallback. Extend its parameters for new staff areas instead of adding a parallel router.
- Expect merge conflicts there and in `internal/handler/contract_test.go` and `internal/server/rest/server_test.go` (`servedOperations`/`servedPaths`) when several tickets add routes.
- Handlers take the live authority from `restmiddleware.Session(c)`, never the client-selected role.
- The staff catalog integration test needs an unassigned inactive official in `vms_stage1`. Apply `ops/stage1/seed-test-catalog.sql` as `vms_migrator` after migrating your throwaway cluster; without it the test skips.

## Contract tooling follow-ups (owned by ticket 03)

**Applied by ticket 03 on 19 November 2026.** The list is kept for reference. Before that, `python ../docs/contracts/check.py --runtime` stopped early with "New/removed frontend source needs classification and traceability review", because of pre-existing frontend drift. Ticket 03 had to handle these:

1. In `source-baseline.json` `runtimeKnownGaps`, remove `field-validation`; the error schema gap is fixed.
2. Change the `runtimeExpectations` for `unknown-query` / `unknown-json` to `invalid_query_parameter` / `invalid_request` with no `fieldErrors` on 400, and drop the matching BE-ERRORS gaps.
3. In `manifest.json` `sourceDTOs`, add `"pkg/dto/optional.go:Optional": []`. DTO times stay `time.Time`.
4. Add `internal_error` (500) and `method_not_allowed` (405) to the documented error-code catalogue, and use the catalogue messages in the error examples.
5. Op 36: a wrong current password is now 422 `validation_failed` with `fieldErrors.currentPassword`. Add the `wrong-current-password` runtime expectation.
6. Make the checker able to target a given backend checkout directory (default `../backend`), so agents working in worktrees such as `D:\oikn\VMS\backend-wt-NN` can run it against their own tree.
7. Replace `manifest.json` `migrationTables` with the 22 per-table entries from ticket 00. Generate them with `invariants.inventory_tables`; they run `vms.kedeputian` (`202611180002_create_vms_kedeputian.sql`) through `support.refresh_recovery` (`202611180023_create_support_refresh_recovery.sql`).
8. Old migration file names still appear in `SCHEMA-IMPLICATIONS.md`, `GAPS.md`, `DATABASE-MAPPING-2026-11-16.md`, `EMPLOYEE-ORGANIZATION.md` and the manifest's `sourceCorrections` text. Update them.
9. `backend/README.md` line 54 ("already migrated; do not recreate or reset it") is stale since the per-table rebuild. Point it to the reset procedure in `backend/docs/LOCAL-DEVELOPMENT.md`.

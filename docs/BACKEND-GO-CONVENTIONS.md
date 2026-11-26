# VMS backend: Go conventions for new modules

Source read 18 November 2026, backend base `e4ca52f` plus working tree. This is the code template the [backend implementation spec](specs/backend-api-2026-11-18/spec.md) tickets follow. The backend's sihp-be-derived layering is house style and must be followed. Section 7 lists existing code that must **not** be copied.

Paths are relative to `backend/`.

## 1. Stack (fixed)

| Concern | Choice |
| --- | --- |
| Language and HTTP | Go 1.27.4, Fiber v3 (`gofiber/fiber/v3`) |
| Database | PostgreSQL 18. GORM v1.31 with the pgx driver. **Raw SQL** via `Raw`/`Exec`; the builder only for simple reads. No `AutoMigrate`, no GORM models |
| Migrations | rubenv/sql-migrate, embedded `migration/files/*.sql`. Applied only by `cmd/vms-migrate` as `vms_migrator` |
| Validation | No library: strict `encoding/json` decoding plus usecase checks plus DB `CHECK`/FK backstops |
| Observability | `log/slog`; OTel API spans (`<module>.<operation>`) |
| Tests | stdlib `testing` only: hand-written fakes, table-driven `t.Run`, Fiber `app.Test`, real PostgreSQL behind `//go:build integration` |
| OpenAPI | Generated from `docs/contracts/manifest.json` and `source-baseline.json` by `go run ./cmd/vms-dev -mode swagger` (`make swagger`); only implemented operations, each with its contract number as `x-operation-number`. Not swag; do not add swag comments |

## 2. Layering and dependency direction

```
router (wiring only) → handler → usecase (iface) → repository (iface) → GORM
                                  usecase → serializer (iface): entity → dto
                                  usecase → policy (pure functions)
middleware → usecase (iface)
```

| Layer | Owns | Must not |
| --- | --- | --- |
| `internal/entity` | Domain/query values. No tags, no imports beyond stdlib | Carry JSON tags or GORM tags |
| `pkg/dto` | Wire shapes: lowerCamel JSON tags, string IDs, `omitempty` only on optional fields, `[]` never `null` | Import `internal/*` |
| `internal/policy` | **Pure** rules: role/permission constants, ownership, scope, projection choice, lifecycle transitions, WITA timing, schedule windows, overlap math | Do IO or read the clock (`now` is a parameter) |
| repository | SQL, locks, transactions, DB clock (`clock_timestamp()`), not-found translation | Contain business rules or return `gorm.ErrRecordNotFound` |
| serializer | Projection entity → dto per audience/view. Copies slices, emits only allowed keys | Decide authorization |
| usecase | Validation (422), authorization (403/404), lifecycle (409), orchestration inside a repository transaction, span | Import Fiber, write HTTP statuses directly (use `failure`) |
| handler | Content type, strict JSON/query/multipart parsing, path IDs, cookies (auth only), `c.JSON`/`SendStatus`. Returns the error unchanged | Contain rules |
| router | Construct repo → serializer → usecase → handler; register literal paths | Use globals or dynamic paths (the AST contract check rejects them) |

**Transactions** start inside repository implementations. Writes use the unit-of-work closure already used by sessions:

```go
// repository/booking/booking_repository.go  (interface package: bookingrepository)
type BookingTransaction interface {
	Now() time.Time                                     // clock_timestamp() read once at tx start
	LockAuthority(proof entity.SessionProof) (entity.SessionSnapshot, error) // locks vms.pegawai row first
	LockVisit(id int32) (entity.Visit, error)          // FOR UPDATE; ErrNotFound when absent
	LockResources(officerIDs, roomIDs []int32) error   // ascending ID order, officers then rooms
	Conflicts(window entity.ScheduleWindow, excludeVisitID int32) ([]int32, error)
	SaveVisit(v entity.Visit) error                    // version guard: WHERE id=? AND version=?
	RevokeAdmissions(visitID int32) error
	Audit(event entity.AuditEvent) error
}
type BookingRepository interface {
	Transaction(ctx context.Context, fn func(tx BookingTransaction) error) error
	// read snapshots (REPEATABLE READ, READ ONLY) return DB time with the rows
}
```

**Lock order**, used by every write: employee (authority) → visit → officers → rooms (ascending ID) → admission credentials. Resource locks are taken before the overlap query (Read Committed). Never mix `db` and `tx`, and never pass a transaction through `context`.

**Authority recheck** inside writes is split by layer. The repository's `LockAuthority` locks the employee and reads the snapshot on the same tx. The usecase validates it through `authusecase.SessionValidator`:

```go
Validate(snapshot entity.SessionSnapshot, proof entity.SessionProof) (dto.Session, error)
```

That validation is extracted from `GetSession`. This replaces `pkg/helper/database/authority.go`, which must be deleted because `pkg` must not import `internal`.

## 3. Naming and files

| Item | Convention |
| --- | --- |
| Interface package / dir | `internal/<layer>/<module>/<module>_<layer>.go`, package `<module><layer>` (`bookingrepository`) |
| Implementation | `internal/<layer>/<module>/implementation/<module>_<layer>.go`, package `<module><layer>implementation`. Private struct (`repo`, `usecase`, `serializer`, or descriptive when one package has several), `var _ Iface = (*impl)(nil)` |
| Constructors | `New<Module><Layer>(named interface params...) <Iface>`. Handlers: `New<Module>Handler(...) *<module>Handler` |
| Several interfaces per module | Allowed and preferred when callers differ (`PublicBookingUsecase`, `StaffBookingUsecase`, `ReceptionUsecase` in `bookingusecase`) |
| Receivers | `r` repo, `u` usecase, `s` serializer, `h` handler, `t` tx |
| Methods | `Get<Thing>` reads, verbs for commands (`Approve`, `Reschedule`, `Attend`), `To<Dto>` serializers. Handler method = operation name |
| Router aliases | Implementations aliased `<module><layer>`, interfaces `<module>contract` |
| Sentinels | `var ErrX = errors.New("snake_code")` in the interface package |
| JSON | lowerCamel; `ID`→`Id` (`assignedOfficerIds`). Times: `time.Time` UTC RFC 3339 `Z`. `serverTime`/`timing.*`: int64 epoch ms |
| Imports | Three groups: stdlib, third-party, `github.com/azhkadaes/vms/backend/...` |
| Comments | One-line doc on exported items stating the constraint or reason |

### New module map

| Module | Repository | Usecase interfaces | Serializer | Handler file | Router |
| --- | --- | --- | --- | --- | --- |
| catalog (extend) | `CatalogRepository.GetStaffCatalogs` | `CatalogUsecase.GetStaffCatalogs` | `ToStaffCatalog` | `catalog_handler.go` | `staff_router.go` |
| upload (tables `form_documents`, `support.form_document_uploads`) | `UploadRepository` (+ tx) | `UploadUsecase` | `UploadSerializer` | `upload_handler.go` | public + staff |
| booking | `BookingRepository` / `BookingTransaction`, `AdmissionRepository` (or methods on the tx) | `PublicBookingUsecase`, `StaffBookingUsecase`, `ReceptionUsecase` | `BookingSerializer` (public / staff views / admission) | `public_booking_handler.go`, `staff_booking_handler.go`, `reception_handler.go` | public + staff |
| report | `ReportRepository` | `ReportUsecase` | `ReportSerializer` | `report_handler.go` | `staff_router.go` |
| employee | `EmployeeRepository` (+ tx) | `EmployeeUsecase` | `EmployeeSerializer` | `employee_handler.go` | `admin_router.go` |
| role | `RoleRepository` (+ tx) | `RoleUsecase` | `RoleSerializer` | `role_handler.go` | `admin_router.go` |
| storage adapter | n/a | n/a | n/a | n/a | wired in router |

The storage adapter sits at `internal/infrastructure/storage` with interface `Storage{Put, Open, Delete}`. The `implementation/` subpackage holds the private local filesystem implementation with UUID object keys.

## 4. Transport helpers (shared, added once)

| Helper | Behaviour |
| --- | --- |
| `handler.DecodeJSON` (existing) | `application/json` only (415), unknown fields and trailing values rejected (400) |
| `handler.NoQuery` (existing) | Any query string → 400 `invalid_query_parameter` |
| `handler.Query` (new) | Typed, allow-listed query parser. Unknown key → 400. Repeated keys keep order. Empty value = unset. Integers bounded |
| `handler.Multipart` (new) | Bounded multipart reader for `file`, `reference`, `purpose`. Size cap before buffering (413) |
| `dto.Optional[T]` (new) | Tri-state JSON field: `Set`, `Null`, `Value`. Distinguishes omitted, `null` and value (decision 15 room rule; note clearing) |
| `validate` (existing) | `ID`, `Reference`, `Text`, `Digits`, WITA `Start`; reuse before writing new helpers |
| `failure` (extend) | `Fields map[string][]string`; `failure.Validation()` builder (`.Add(key, msg)`, `.Err()`); `failure.New(status, code)`; Indonesian message catalogue per code |
| `restmiddleware.Failure` (extend) | Writes `{code, message, requestId, fieldErrors?}` with per-code messages. Logs status, code and a **redacted cause class**. Maps a repository `ErrNotFound` to 404 |

**Staff route protection** is hoisted into `router.Dependencies`. `StaffAuth` is Bearer verification plus live authority plus limiter. `Workspace` blocks the role-selection and password-change states. Every business router mounts `deps.App.Group("/v1/staff/...", deps.StaffAuth, deps.Workspace)`, so protection no longer depends on registration order.

## 5. Errors

| Status | Code(s) | When |
| --- | --- | --- |
| 400 | `invalid_request`, `invalid_query_parameter` | Malformed JSON/query/ID |
| 401 | `unauthenticated`, `invalid_credentials` (login only) | Missing or expired session, bad login |
| 403 | `forbidden`, `csrf_expired` | In scope but role/permission/owner denied |
| 404 | `not_found` | Unknown or out-of-scope record (non-disclosing), proof mismatch |
| 409 | `version_conflict`, `invalid_state`, `schedule_conflict`, `reschedule_window_closed`, `idempotency_conflict`, `notice_outdated`, `credential_expired`, `revision_conflict`, `role_changed` | Concurrency, lifecycle or freshness |
| 413 / 415 | `payload_too_large`, `unsupported_media_type` | Upload size or type |
| 405 | `method_not_allowed` | Route exists, method does not |
| 422 | `validation_failed` (+ `fieldErrors`), `export_too_large` | Field rules, wrong-purpose tokens, wrong current password (`currentPassword`) |
| 429 | `rate_limited` + `Retry-After` | D29–D33 budgets |
| 500 | `internal_error` | Anything unmapped. Logged with a redacted cause class |
| 503 | `service_unavailable` | Database, limiter or auth unavailable (fail closed) |

Messages per code live in `pkg/helper/failure/catalogue.go`. **Not-found:** each repository interface package declares `var ErrNotFound = fmt.Errorf("<module>: %w", failure.ErrNotFound)` and returns it wrapped; the handler maps it to 404. `gorm.ErrRecordNotFound` must never leave a repository.

## 6. Tests

| Seam | What it proves | Prior art |
| --- | --- | --- |
| `policy` unit tables | Transitions, ownership, projection key sets, timing boundaries (`opensAt` inclusive, missed at `endsAt + 1 ms`, reschedule `end + 7 d` inclusive), overlap math | `internal/policy/visit_test.go` |
| Usecase with fake repository/tx | Validation, authorization, version checks, audit emitted, serializer output | `usecase/auth/implementation/auth_usecase_test.go` |
| HTTP via exported `Register<Area>Routes` + fake usecases | Strict decoding, status/code mapping, contract captures for `check.py --runtime` | `internal/handler/contract_test.go`, `server/rest/server_test.go` |
| Real PostgreSQL (`-tags integration`) | Locks, overlap under concurrency, idempotent create, single attendance transition, version races | `repository/admission/implementation/admission_repository_integration_test.go` |

- Test names are behaviour sentences: `TestApproveRejectsStaleVersionWithoutWriting`.
- Fakes record calls so a test can assert boundaries ("no write after authorization failure").
- No `t.Parallel`, no assertion libraries.

## 7. Existing code not to copy (and the fix)

1. `pkg/helper/database/authority.go` imports `internal`. Replace it with `LockAuthority` plus `SessionValidator` (section 2).
2. Usecases check `gorm.ErrRecordNotFound`, and unmapped not-found errors become 500. New repositories translate it to `<module>repository.ErrNotFound`.
3. `session_repository.go` returns unwrapped errors. Always `fmt.Errorf("<operation>: %w", err)`.
4. Bearer middleware is built from local variables inside `AuthRouter` and depends on `group.Use` order. Hoist it as in section 4.
5. Policy compares role/permission string literals. Add `policy.RolePA`… and `policy.PermApprove`… constants and use them everywhere new.
6. Hard-coded limits sit in repositories. Put limits in usecase constants or config.
7. `catalog_handler.go` re-implements `NoQuery` and carries unused swag comments. Use the helpers instead.
8. Entities are scanned with `SELECT *` and GORM snake naming. Use explicit `AS` aliases or private row structs for Indonesian columns.
9. `failure.Field` supports only one field and one message. Use the new validation builder.
10. `fiber.NewError(409|422)` maps to 500. Always use `failure.New` for these statuses.
11. Auth usecases have no spans. Every new usecase method starts `<module>.<operation>`.
12. Run `go mod tidy` (`x/crypto` is marked indirect but imported directly).

## 8. Checklist: one module slice

1. **Migration:** one file per table, `migration/files/YYYYMMDDNNNN_create_<schema>_<table>.sql`, built exactly as in [the physical schema](contracts/PHYSICAL-SCHEMA-2026-11-18.md). That schema follows the user's DB diagram and cites every agreed departure. Never add a column to a diagram table without an agreement; derive the value or put operational state in `support`.
   - Up and Down in the same file. Down drops only that table, its grants and any ENUM type it introduced.
   - Business tables go in `vms` with the diagram's (Indonesian) column names; operational tables go in `support` with English names.
   - **Types:**
     - diagram `string` → `VARCHAR(255)`
     - diagram `ENUM` → a named PostgreSQL ENUM; other value sets → `VARCHAR … CHECK IN (...)`
     - events → `TIMESTAMPTZ`
     - `created_by`/`updated_by` → `VARCHAR(255)` holding the employee ID as text, or `system`
   - **Rules:** text rules as `char_length(btrim(x)) BETWEEN a AND b`. Reference seeds live in their table's migration.
   - **Grants:** explicit least-privilege grants to `vms_api` (including `USAGE ON SEQUENCE`).
   - **Bookkeeping:** update `migration/source_test.go`; manifest `migrationTables` follows in step 4.
2. **Code:** entity → dto → repository (interface + implementation) → policy additions → serializer → usecase (span, `failure`) → handler → router.
3. **Tests:** at the seams in section 6. Integration tests for every invariant that depends on locking.
4. **Contract pipeline.** `docs/contracts` lives in the outer repository and is shared by every checkout, so it is not committed with the slice.
   - **Who edits it.** An implementer in a worktree does not edit `docs/contracts`. Copy `manifest.json` and `source-baseline.json` to a scratch directory, make the edits below there, check them with `-contracts`/`--contracts` pointing at that copy, and hand the edits over as `contractUpdates`. The merger applies them to `docs/contracts` on the integration branch and runs the check. Until they are applied, the shared check fails on the slice's new routes; that is expected.
   - **`manifest.json` → `operations`.** Find the operation by its number in [the operation reference](contracts/FRONTEND-DRIVEN-REST-API-2026-11-16.md); number, method and path are checked against those headings. Set `status` to `implemented` and add:
     - `handler`: `internal/handler/<file>.go:<receiver>.<Method>`
     - `calls`: the delegation the handler must keep, such as `h.usecase.Approve`
     - `usecase`: `<Interface>.<Method>`
     - `requestDto` / `responseDto`: `pkg/dto` type names, or `null`
   - **`source-baseline.json` → `spec.paths`.** Add the operation with:
     - `operationId` equal to the manifest `id`, `x-operation-number`, `x-reject-unknown-query: true`
     - one `x-error-conditions` entry for every error status, each response a `$ref` to `#/components/responses/Error<status>`
     - request and response schemas under `components.schemas`, with examples that match what the server returns

     The checker compares every `pkg/dto` field, including nested DTOs, with these schemas: required fields, `omitempty` and primitive types.
   - **New error codes.** Add the code to `pkg/helper/failure/catalogue.go`, `spec.x-error-catalogue`, the `ApiError.code` enum and the examples of the matching `Error<status>` response.
   - **Captures.** Add cases to `TestContractHTTP` in `internal/handler/contract_test.go`, or add a test to manifest `runtimeCaptures`. Each implemented operation needs at least one successful capture. Add one `runtimeExpectations` entry per capture: `operation`, `status`, and optionally `code`, `fieldPaths` or `fieldErrors: "absent"`.
   - **Inventory.** From the backend checkout, run `python ../docs/contracts/check.py --backend . --write-inventory` (add `--contracts <dir>` for a draft). It rewrites manifest `sourceInterfaces`, `sourceDTOs` and `migrationTables` from the Go and SQL source. Review that diff: it is exactly the unreviewed drift.
   - **New frontend request calls.** Map them in manifest `frontendRequestCalls` with their operation numbers.
   - **Served OpenAPI.** Run `make swagger` (`.\vms.ps1 swagger`), or for a draft `go run ./cmd/vms-dev -mode swagger -contracts <dir>`. Update `servedOperations` and `servedPaths` in `internal/server/rest/server_test.go`, and commit `docs/openapi.json` with the slice.
   - **Readiness.** Extend `Readiness` when new tables are required.
   - **Check.** Run `make contract-check` (`.\vms.ps1 contract-check`), which is `python ../docs/contracts/check.py --backend . --runtime --consumers`.
5. **Gate:** `go test ./...`, `go vet ./...`, `make contract-check`, `go test -tags integration ./...` against `vms_stage1`.

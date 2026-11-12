# VMS decisions for backend development

Updated 12 November 2026 WITA. User steering: inspect how sihp-be implements each topic before proposing another choice. If the reference provides no answer, defer it to the relevant backend development phase unless it is urgent for VMS. This is a planning backlog, not implementation approval.

## Working rule

1. Read the relevant owning sihp-be code/configuration, not just its README or dependency list. Record the path and observed behavior.
2. Reuse a suitable reference pattern when compatible with already selected VMS rules. Absence of code is not evidence that an approach was deliberately rejected.
3. Preserve earlier VMS decisions and explicit exceptions. Do not replace integer IDs with reference UUIDs, direct JSON with reference envelopes, or the selected framework major merely because the reference differs. Surface a real conflict only when it blocks the work at hand.
4. If no applicable reference exists, add the issue to its development phase below. Do not ask the user to settle it now or silently present an unselected proposal as approved.
5. Ask now only for a concrete blocker to required VMS behavior, authorization/privacy, data integrity/recovery, or a contract boundary that cannot safely remain open until that phase. Explain the blocking workflow and why deferral is unsafe. Ask one choice at a time.

After the reference-first steering, the user explicitly approved freezing the planning baseline, including its labelled technical proposals. Existing D01–D100 selections and the approved OpenAPI are retained. Future nonurgent decisions remain deferred under the working rule above; the final documentation generator/workflow remains a later development decision. No new implementation, database/container operation, migration or deployment is authorized.

## Reference evidence checked

| Topic | Observed sihp-be implementation | VMS treatment |
| --- | --- | --- |
| HTTP framework | go.mod: github.com/gofiber/fiber/v2 v2.52.9; internal/server/rest/server.go constructs fiber.New | Go/Fiber reference. Existing VMS Fiber v3 choice remains; adapt framework-specific wiring during development. No FastAPI implementation identified in inspected routes/dependencies. |
| API documentation | go.mod includes swaggo/swag and gofiber/swagger; Makefile swagger target runs swag init -g cmd/main.go --output docs --parseDependency; handler comments carry @Param/@Success/@Router; docs/swagger.json declares swagger:2.0 | Reference uses code-annotation-generated Swagger2 documentation and a basic-auth-protected /swagger/* UI. Keep the approved VMS OpenAPI3.1 planning contract; defer final generator/tool/version/synchronization workflow to HTTP integration. Do not describe the reference as OpenAPI3.1 or treat FastAPI as a documentation format. |
| Layers/interfaces | internal/handler/auth_handler.go depends on auth usecase interface; internal/repository/auth/auth_repository.go and internal/serializer/sihp/sihp_serializer.go define explicit interfaces | Reuse selected structural approach. |
| Validation | pkg/dto/sihp.go and pkg/dto/user.go use go-playground validator tags, including required/email/oneof/max/min | Reuse typed DTO validation structure; it does not answer every VMS-specific field/byte/code-point bound. Selected bounds remain; defer remaining parser/storage details to the owning slice. |
| Entity identity/timestamps | internal/entity/base/base.go uses UUID PK, timestamptz created/updated values and GORM soft-delete hooks | Structural evidence, not a reason to undo D95 integer business IDs or mandate soft-delete everywhere. VMS defaults/deletion constraints are recorded in the approved planning baseline; actual DDL remains unimplemented/unverified. |
| Authentication | internal/server/rest/middleware/jwt.go parses Bearer and stores admin identity in request locals; auth usecase verifies bcrypt | Reuse appropriate handler/usecase/JWT helper separation. Inspected code does not establish VMS session authority/rotating-cookie/CSRF/role-switch protocols. Existing selected VMS requirements remain; defer helper formats and key implementation to identity slice. A refresh-expiry config field alone is not evidence of implemented refresh rotation. |
| Transactions | internal/repository/transaction-data/implementation/transaction_data_repository.go uses a GORM transaction | Reuse transactional repository pattern where compatible. This is not evidence of VMS schedule row locks, conflict/version checks or idempotency protocols. Implement/review those in their owning slices. |
| SMTP, upload replay and attendance QR | No matching implementation found in the inspected internal/pkg/config Go sources | Do not invent reference precedent or ask extra implementation-detail questions now. D96/D98/D99 remain selected VMS-specific requirements; unresolved mechanics wait for the relevant slice. |

## Deferred phase backlog

| Development phase | Deferred work | Reopen only when |
| --- | --- | --- |
| Foundation/configuration | Exact Go/Fiber/GORM/driver/image patch pins, build and vulnerability checks; concrete shared error helpers, bounded logging and constructor wiring | Preparing an authorized build or an actual incompatibility blocks the selected stack. |
| Persistence/migrations | Translate schema register into actual DDL; choose constraint names, timestamp defaults, FK update/delete details and retained restricted-snapshot clearing mechanics; verify cardinality/indexes against selected ownership | Designing that slice's migration. Ask only if a newly found choice changes a selected business invariant, privacy obligation or required recovery behavior. |
| HTTP/DTO integration | Follow reference annotations/Swagger UI approach where compatible; choose generator and Swagger/OpenAPI version handling; settle concrete response/cookie/header names and documentation synchronization | Implementing the HTTP boundary. Preserve selected payload/security semantics and frontend contract; no generator migration is authorized now. |
| Identity/access | Credential prefix/encoding, JWT signing configuration, hash representation, key provisioning/rotation, exact CSRF transport binding and authority-revision storage | Building the identity slice. Do not defer required authorization/CSRF checks, but do defer their nonblocking implementation mechanics. |
| Booking/recovery | Lock order implementation, transaction retries, reference digest encoding, error mapping and actual DDL backstops | Building real transactional booking operations and meaningful concurrency verification. |
| Documents | D99 canonical HMAC derivation encoding/key-version representation, accepted upload replay lookup, staging/cleanup races, streamed receive deadlines and preview policy | Implementing isolated development document storage. Production storage/scanning remains separately deferred. |
| Attendance | D96 consumed-versus-revoked QR persistence/lookup, checkout-only recovery and race tests | Building reception/admission slice; preserve selected late checkout and single-use admission. |
| Notifications | SMTP field-parser specifics, approved host/port settings, encryption/key lifecycle, bounded delivery concurrency/provider timeout and fake-sink verification | Building worker/delivery adapter. No live send or credentials are authorized by this backlog. |
| Frontend integration | Apply already selected payload changes, D100 Pending reschedule availability, code-point/UTF-8-byte guidance and accessible validation/focus/recovery behavior | Connecting that frontend flow to the authorized backend slice. |
| Release/operations | Actual domains/TLS, approved outbound destinations, secrets, privacy notice, retention/purge/rights handling, backup/restore and production document enablement | Separate release planning/approval. Existing-data migration remains its own later scope. |

This inventory is source inspection only. No reference/VMS build, runtime, deployment or compatibility result is claimed. Future questions must cite the relevant reference check or explicitly state that no equivalent was found.

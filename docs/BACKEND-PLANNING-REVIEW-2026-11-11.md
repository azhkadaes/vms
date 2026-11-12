# VMS backend planning review

Approved planning baseline, 12 November 2026 WITA. The user explicitly approved freezing the reviewed plan, schema register and 38-operation OpenAPI, including the labelled schema/credential/SMTP/transport proposals. This approves planning only: no application edits, database/container operations, migrations or deployment. Reference-first handling and deferral of nonurgent future decisions remain in force. Documentation generator/tool selection and runtime/release gates remain deferred as recorded.

Consolidated planning baseline approved and frozen, 12 November 2026 WITA. No application implementation, database/container operations, migration or deployment is authorized.

Reference-first working rule, retained after the subsequent explicit planning approval: check sihp-be's actual implementation before proposing each further decision; defer missing/nonurgent choices to backend development. The [reference comparison and deferred backlog](BACKEND-DEVELOPMENT-DECISIONS-LATER.md) records the evidence and owning phases. D01–D100 selections remain intact. The user subsequently approved the labelled technical proposals as part of this planning baseline. They are planning specifications, not verified implementation; remaining mechanics are handled in their owning phases. The final API-documentation generator/version workflow is likewise deferred; sihp-be currently generates Swagger2 from Go annotations.

Read the [architecture and decision register](BACKEND-PLAN-2026-11-11.md), [API contract](BACKEND-API-CONTRACT-2026-11-11.md) and [recovered business/database decisions](BACKEND-ALIGNMENT-DECISIONS-RECOVERED-2026-11-11.md).

## Changes completed for this review

The register now contains100 selected decisions. All20 original diagram entity field lists have a reconciliation entry covering selected exceptions and proposed PK/FK/cardinality, null/default/unique/link/deletion rules. [OpenAPI](BACKEND-OPENAPI-2026-11-12.json) now covers the same38 method/path operations as the API inventory. No database schema has been applied.

| Selection | Result |
| --- | --- |
| D94 | Employee assignedOfficerIds max100 unique IDs; retain deliberate inactive references, no partial update/truncation. |
| D95/D97 | Business IDs and version counters use positive PostgreSQL INTEGER, max2,147,483,647. Wire IDs stay strings; versions stay numbers; no overflow/reset. |
| D96 | A QR consumed for successful check-in may resolve/check out that checked-in visit after admission expiry; revocation still blocks it. Admission consumption and revocation are distinct stored states. |
| D98 | Initial delivery adapter is authenticated SMTP over verified TLS; strict host/port/tlsMode/username/password/sender schema. Storage provider remains deferred. |
| D99 | Same-reference upload retry regenerates the same token from a dedicated key/immutable metadata; retain matching key through fixed one-hour lifetime, no recoverable token copy. |
| D100 | Reschedule Pending or Approved, excluding Missed. Add Pending deliberately to frontend availability; preserve nonreserving Pending state. |

Source checks also corrected two planning omissions: public privacyVersion is optional under D90/D91, and restricted Edit detail carries its required visitKind/date/time. Employee new-password bounds now consistently reflect D80/D81 rather than the current frontend max256. Existing application files have not been edited.

## Technical proposals included in baseline review

The following proposals were explicitly included in the subsequent consolidated planning approval; they are not attributed to the earlier individual selections: positive generated INTEGER PKs and matching FKs; unique visit-owned snapshot/detail and single-file links; no directory cascade deletion; explicit inactive handling; required/nullable/default/check constraints described in the schema register; ordered roster/unique attendance extensions; bounded canonical permission/action keys; purpose/version-prefixed32-byte credentials and SHA-256 private lookup digests; HMAC-SHA-256 upload derivation under D99; and retaining an already-created restricted snapshot when its current optional fields are cleared, with audit and no invented placeholder at initial creation.

The OpenAPI fixes concrete transport naming for review: vms_refresh, vms_csrf, vms_login_binding cookies; X-VMS-CSRF header; token response {accessToken,tokenType:"Bearer",expiresAt,session}. These are interface naming proposals, not new endpoints. Cookie/header presence alone never replaces signed binding/origin/current-authority validation. SMTP uses canonical decimal-string port1..65535 and explicit implicit_tls/starttls mode; effective configuration must be complete before enablement. Actual hosts/credentials are not invented or configured.

## Document verification

The reproducible [validation reader](planning-evidence/validate-planned-openapi.py) checks the draft against the [official OpenAPI3.1 schema](https://spec.openapis.org/oas/3.1/schema/2025-09-15), validates70 component schema definitions, resolves internal references, checks exact38-operation method/path coverage and unique operation IDs, and exercises65 positive/negative schema cases. Cases include ID/version overflow, unknown/mass-assigned fields, optional privacy version/mandatory acknowledgment, Unicode password byte limits, assignment limit/duplicates, restricted projection leaks, required restricted-update schedule and SMTP field/port/TLS/secret constraints. Results: [planning validation evidence](planning-evidence/openapi-validation-2026-11-12.json).

These are planning-document checks. They do not verify application behavior, database constraints, concurrency, SMTP delivery, runtime compatibility or accessibility of an integrated UI. Byte-length extensions are explicitly checked by this reader; ordinary OpenAPI tooling may not enforce them without custom validation.

## Current direction

- Go 1.27, Fiber v3, GORM and PostgreSQL 18; reviewed exact patch/image pins and compatibility checks precede implementation.
- One modular monolith codebase, layer-first handler/usecase/repository/serializer/entity folders and explicit interface/implementation pairs, following sihp-be structurally. Business rules are independent of HTTP and persistence implementations.
- Separate API and worker processes; transactional PostgreSQL outbox, bounded retry policy and cleanup. Lock scheduling resources in a fixed order inside transactions; retain version checks, audit and notification intent.
- 38 frontend HTTP operations, with separate visit actions and shared bounded bookings views for list/reception/dashboard. Direct JSON responses and consistent safe errors. Stable room IDs replace name-based writes.
- Local password login; future SSO remains deferred, with identity verification separated from VMS session/authorization handling.
- Bearer access JWT held in memory for 5 minutes; persistent opaque HttpOnly refresh cookie, rotation on each refresh and 10-second same-request recovery. PostgreSQL remains authoritative for expiry, current roles/permissions and revocation.
- Sign out after 8 hours inactive or 24 hours from login, whichever comes first. Background polling/refresh never extends inactivity. Actual activity renews at most once per minute; warning appears 2 minutes before idle expiry.
- Concurrent browser/device sessions; role switch affects only the current session/shared tabs. Password change replaces current credentials and revokes other sessions without resetting the absolute deadline.
- Signed session-bound double-submit CSRF with strict origin checks and SameSite=Lax cookies. Nginx serves staff HTML and obtains cookie initialization through a private Go call authenticated by a dedicated secret over verified TLS. No public bootstrap endpoint.
- One host with Docker Compose services for Nginx/API/worker/PostgreSQL, separate visitor/staff hostnames and private internal connections. This is not a high-availability deployment claim.
- Draft attachments use replaceable private local storage in development only. Preview is a dialog with Download and Close/X; outside click does not dismiss, Escape does, with accessible focus behavior. Upload token lasts 1 hour; unlinked content is cleanup-eligible after 24 hours.
- Strict write JSON and query allowlists. Reasons max 1,000 characters; organization max 255. Ordinary JSON body max 256 KiB; upload body max 6 MiB with one file max 5 MiB. JSON execution and post-receive upload processing each have a 15-second budget; upload receive budget is 60 seconds total, enforced through streamed forwarding and verified Go deadlines.

## Rate limits selected

| Scope | Initial allowance |
| --- | --- |
| Login failures per normalized account | 5 per rolling 15 minutes |
| Login requests per trusted IP | 100 per rolling 15 minutes |
| Public lookup/cancel/admission per IP | 30 per rolling minute, shared |
| Public booking creation/recovery requests per IP | 30 per rolling minute, shared |
| New committed public bookings per IP | 30 per rolling hour; replay is not a new booking |
| New accepted uploads per IP | 60 per rolling hour, shared public/staff |
| All public API requests per IP | 120 per rolling minute |
| Authenticated staff requests per employee | 120 per rolling minute across sessions |
| Password-change attempts per employee | 5 per rolling 15 minutes |
| Refresh requests per identified session | 10 per rolling minute, including recovery |
| All refresh requests per trusted IP | 100 per rolling minute, including invalid cookies |

Shared PostgreSQL counters; applicable limits combine. Excess returns 429/Retry-After. Counter failure returns 503 for affected operations; logout/basic health exemptions do not bypass required authentication/revocation. No throughput capacity is claimed from these selected initial values.

## Remaining evidence and freeze gates

The original33 database decisions, all20 entity field lists and the new physical/schema/lifecycle proposals are consolidated in the [schema evidence register](BACKEND-SCHEMA-EVIDENCE-2026-11-11.md). The machine-readable API draft exists and passes the documented checks. The user has completed the final baseline approval and the planning files are frozen. An approved baseline still needs actual implementation/DDL verification afterward and does not establish runtime correctness.

Selected framework/database/toolchain major versions still need exact patch/image pins, compatibility and vulnerability review. Verify Fiber v3 context/streaming behavior, total upload receive enforcement, independent Set-Cookie propagation and refresh race/recovery behavior in isolated implementation tests. No application builds, integration tests or runtime acceptance have been performed in this planning review.

Production requires actual domains/TLS/secret provisioning, privacy notice and organizational retention/deletion policy, backup/restore verification and acceptance. Production document storage/scanning enablement remains deferred. Existing-data migration has a separate later scope and is not required for the fresh isolated baseline.

The consolidated planning baseline is approved. Approval is limited to the planning documents and does not authorize application edits, database/container operations, migrations or deployment.

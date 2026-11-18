# Implementation gaps

Review date: 2026-11-13 WITA. Phase 1 has 29 target operations: one matching catalog route and 28 missing target routes. Seven old identity routes remain in source and must be replaced/removed for Phase 1. Session lifecycle and account/role writes are deferred, not target gaps. A baseline check can pass with known gaps; strict target conformance fails until target-blocking gaps are resolved. DB-RELEASE is a production-only gate and does not block a disabled-capability contract. See [validation results](validation-results.json) for actual executed evidence and limitations.

| Gap | Status / gate | Actual finding | Required change |
| --- | --- | --- | --- |
| BE-VISITS | missing; target | Visits, scheduling, public lookup/recovery and commands have no handlers/interfaces/repositories/DTOs yet. | Implement plan slices 4-5 with atomic audit/outbox/resource locks; no generic CRUD or parallel service. |
| BE-READS | missing; target | Staff catalogs, typed booking views and detail have consumer adapters but no backend routes. | Add authorized catalog/read interfaces; scope before search/count/sample; preserve all three typed views. |
| BE-ATTENDANCE | missing; target | QR issuance/resolve/checkin/checkout do not exist in the live backend. AdmissionUsecase currently means HTTP request admission/rate limiting, not visit QR. | Use separate reception module within existing layers; do not overload request-limiter interface. |
| BE-DOCUMENTS | missing; target | Upload/link/private download operations absent; development attachment capability is false. | Keep disabled until token ownership, private bytes, content verification and cleanup implemented. |
| BE-ADMIN | missing; target | Read-only employee page/detail routes are absent. Employee/account/role writes are deferred. | Implement only authorized directory reads for Super Admin; provision accounts/roles/PA assignments via controlled database setup. |
| BE-INTEGRATIONS | missing; target | Safe integration metadata/config endpoints and SMTP worker absent. | Implement SMTP-only keys and write-only encrypted secret; storage remains disabled metadata. |
| BE-REPORTS | missing; target | Summary/export routes absent. | Aggregate ordinary visits on demand; no report/dashboard tables. |
| FE-CATALOG | incompatible; target | wire.ts and Catalogs.rooms expect string[]; backend returns room objects. attachmentsEnabled is stripped by the current Zod object. | Align Room objects and capability schema/forms together; retain room label in read records. |
| FE-AUTH | incompatible; target | Frontend still calls session routes, sends no Bearer and expects legacy Session/TokenResult/gates. New Phase 1 login/me consumers do not exist. | Use /staff/auth/login and /staff/auth/me, AuthResult.user/CurrentUser, memory-only token and fixed expiry. Local logout; remove role/password/lifecycle UI from Phase 1 routes. No CSRF/cookie/refresh implementation required. |
| FE-ROOM | incompatible; target | BookingDetail sends room labels as room, not selected roomId. | Bind options to stable IDs; omit unassigned; no silent name-to-ID coercion. |
| FE-PENDING | incompatible; target | domain.can reschedule gate allows Approved only, while selected D100 includes Pending. | Enable Pending deliberately while preserving Missed exclusion and server state validation. |
| FE-BOUNDS | incompatible; target | Some frontend organization/email/reason/text checks are looser or count UTF-16; identifiers accept any nonempty string. | Align canonical IDs, ASCII email max254, code-point text, reason max1000; fixtures remain explicit preview only. |
| FE-UPLOAD | incompatible; target | Forms expose/call upload without consuming attachmentsEnabled=false; backend capability is false. | Disable/hide optional new uploads with explanation; continue booking without files. |
| FE-FILEVIEW | incompatible; target | AttachmentAccess only downloads; default request has no disposition while target defaults inline. | Offer View plus Download via same authorized bytes route and explicit disposition; release object URLs. |
| FE-CORRELATION | incompatible; target | Detail and page metadata are not correlated with the request. Restricted validators strip unknown raw fields before checking their projection, so an extra private field can escape detection. | Verify requested ID/page/pageSize and inspect restricted raw projection allowlists before normalization. Ordinary consumers continue tolerating compatible additive fields under D57; server serializers still emit only documented authorized fields. |
| BE-AUTH-TESTS | unverified; target | Existing HTTP fixtures cover superseded session handlers, not the Phase 1 login/me and database role checks. | Verify bcrypt login, token claims/expiry, inactive/missing accounts, exactly-one-role checks, manual role/assignment changes, fixed permissions and protected resource denials on the owning implementation slice. |
| DB-RELEASE | unresolved; production only | Production attachment content/scanner/provider, organizational retention/purge and deployment domains/secrets remain deferred. | Keep capability disabled; these block production enablement, not the disabled v1 contract or this audit. |
| BE-ERRORS | incompatible; target | ApiError and frontend require fieldErrors arrays; current failure.Field/HTTP middleware emit string values. Unknown JSON/query keys use legacy codes without selected D57/D58 rejected paths. Source runtime fixtures reproduce these errors; their password/logout samples are deferred source evidence. | Align transport decoding and safe failure mapping with D57/D58 and Record<string,string[]>; retain semantic 422 validation. Add precise code/field/schema assertions without changing the frontend to discard errors. |
| BE-PHASE1-AUTH | missing; target | Existing auth router/middleware/config require persisted sessions, refresh/CSRF/gateway keys, role gates and credential revisions. Neither new Phase 1 auth route is implemented. | Replace the auth slice through the same interfaces/layers with login/me, AccessProof, current DB role/assignments and fixed permission mapping. Stop exposing legacy session/initialization routes in the Phase 1 build. Do not merely disable middleware. |
| FE-DIRECTORY | incompatible; target | Existing employee editor and role management expose writes; directory parser/form policy expects password metadata no longer in Phase 1 EmployeePage. | Keep directory/detail read-only; hide/disable account/role/assignment/password editors and remove their requests. Adapt EmployeePage parser without password policy metadata. |

## Operations requiring backend implementation

| Operation | Target route | Missing layer owner |
| --- | --- | --- |
| staffLogin | POST /v1/staff/auth/login | auth; handler/use-case/repository/serializer/DTO |
| staffMe | GET /v1/staff/auth/me | auth; handler/use-case/repository/serializer/DTO |
| staffCatalogs | GET /v1/staff/catalogs | catalog; handler/use-case/repository/serializer/DTO |
| publicCreateBooking | POST /v1/public/bookings | visit; handler/use-case/repository/serializer/DTO |
| resolveSubmission | POST /v1/public/submissions/resolve | visit; handler/use-case/repository/serializer/DTO |
| publicLookup | POST /v1/public/lookup | visit; handler/use-case/repository/serializer/DTO |
| publicCancel | POST /v1/public/bookings/{id}/cancel | visit; handler/use-case/repository/serializer/DTO |
| publicAdmission | POST /v1/public/bookings/{id}/admission | reception; handler/use-case/repository/serializer/DTO |
| publicUpload | POST /v1/public/attachments | document; handler/use-case/repository/serializer/DTO |
| staffUpload | POST /v1/staff/attachments | document; handler/use-case/repository/serializer/DTO |
| staffAttachment | GET /v1/staff/bookings/{id}/attachment | document; handler/use-case/repository/serializer/DTO |
| staffBookings | GET /v1/staff/bookings | visit; handler/use-case/repository/serializer/DTO |
| staffBooking | GET /v1/staff/bookings/{id} | visit; handler/use-case/repository/serializer/DTO |
| staffCreateBooking | POST /v1/staff/bookings | visit; handler/use-case/repository/serializer/DTO |
| staffApprove | POST /v1/staff/bookings/{id}/approve | visit; handler/use-case/repository/serializer/DTO |
| staffReject | POST /v1/staff/bookings/{id}/reject | visit; handler/use-case/repository/serializer/DTO |
| staffReschedule | POST /v1/staff/bookings/{id}/reschedule | visit; handler/use-case/repository/serializer/DTO |
| staffRestrictedUpdate | POST /v1/staff/bookings/{id}/update | visit; handler/use-case/repository/serializer/DTO |
| staffCancel | POST /v1/staff/bookings/{id}/cancel | visit; handler/use-case/repository/serializer/DTO |
| staffCheckin | POST /v1/staff/bookings/{id}/checkin | reception; handler/use-case/repository/serializer/DTO |
| staffCheckout | POST /v1/staff/bookings/{id}/checkout | reception; handler/use-case/repository/serializer/DTO |
| staffResolve | POST /v1/staff/resolve | reception; handler/use-case/repository/serializer/DTO |
| staffEmployees | GET /v1/staff/employees | management; handler/use-case/repository/serializer/DTO |
| staffEmployee | GET /v1/staff/employees/{id} | management; handler/use-case/repository/serializer/DTO |
| staffIntegrations | GET /v1/staff/integrations | management; handler/use-case/repository/serializer/DTO |
| patchIntegration | PATCH /v1/staff/integrations/{id} | management; handler/use-case/repository/serializer/DTO |
| reportSummary | GET /v1/staff/reports/summary | report; handler/use-case/repository/serializer/DTO |
| reportExport | GET /v1/staff/reports/export | report; handler/use-case/repository/serializer/DTO |

## Deferred findings (outside the Phase 1 gate)

| Previous finding | Phase 1 disposition |
| --- | --- |
| FE-SESSION | deferred with its owning feature; not a Phase 1 conformance requirement |
| FE-PASSWORD | deferred with its owning feature; not a Phase 1 conformance requirement |
| FE-ASSIGNMENTS | deferred with its owning feature; not a Phase 1 conformance requirement |
| BE-LOGOUT | deferred with its owning feature; not a Phase 1 conformance requirement |
| BE-LIMITS | deferred with its owning feature; not a Phase 1 conformance requirement |
| BE-REVISION | deferred with its owning feature; not a Phase 1 conformance requirement |
| DB-PROFILE | Superseded by inspected SQL source; see sourceCorrections. Not a missing migration. |

Source correction **DB-PROFILE**: 202611120002_identity_session_foundation.sql already defines jabatan/kedeputian/direktorat/nomor_hp/nip. Prior missing-column finding is superseded; no profile migration is required merely for the Phase 1 directory.

Source correction **DB-EMPLOYEE-ORGANIZATION**: User requirement 2026-11-15: one current official per employee, with automatic organizational affiliation. The additive migration normalizes Kedeputian and Direktorat, links existing Jabatan, and adds one-active-assignment uniqueness. It seeds verified unit names only. See EMPLOYEE-ORGANIZATION.md; no employee-write endpoints are added to Phase 1.

## What already matches

- Existing catalog and identity code use interfaces in module packages, concrete implementation subpackages, constructor injection and DTO serialization.
- Catalog reads active officials joined to Jabatan for VIP and stable integer room IDs in one repeatable-read snapshot; notice comes from configuration and attachmentsEnabled is false.
- Current source uses a stored-session authority reader; this is explicitly superseded by the Phase 1 token/account/single-role reader and is not target conformance.
- Existing SQL uses sql-migrate Up/Down files and retained directory/assignment rows. Session/refresh/policy structures remain source history; Phase 1 does not require them or authorize deleting deployed tables.
- Frontend production adapters already use direct camelCase records, typed paginated views, versioned commands, public credential POST bodies and explicit preview isolation.

Source matches are not database deployment evidence. No migrations or live database actions were performed for this audit.


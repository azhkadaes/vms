# Frontend/backend compatibility recheck

Reviewed 4 November 2026 (WITA).

Frontend: `D:/oikn/VMS/frontendx`. Backend: current working tree at `C:/Users/ASUS/go/src/github.com/thdoikn/VMS-BE`, including uncommitted changes. `D:/oikn/VMS/backend` currently contains no implementation files.

**Result: the applications are not directly compatible.** Backend core flows now exist, but their routes, payloads, projections, permissions, and visitor verification workflow differ from the frontend contract. Changing `VITE_API_BASE_PATH` cannot resolve this.

## Blocking differences

| Area | Frontend currently expects | Backend currently supplies | Effect |
| --- | --- | --- | --- |
| Staff session | Top-level `id`, `name`, one canonical `role`, `active`, permissions and assignments | `active`, nested `employee`, `roles[]`, permissions and assignments | Sign-in route and cookie transport match, but subsequent session parsing rejects the response with 502. A canonical-role policy must be agreed; selecting the first role would be unsafe. |
| Permissions | `bookings`, `approve`, `reject`, `reschedule`, `checkin`, `checkout`, `people`, `roles`, `reports`, etc. | `booking.view`, `booking.approve`, `checkin.perform`, `employee.manage`, `role.manage`, etc. | Even after session normalization, literal frontend permission checks hide navigation/actions unless permission semantics are mapped. |
| Visitor booking | `POST /v1/public/bookings`; JSON idempotency key; `time`, `name`, named visitors, `attachmentToken`; full booking response | `POST /v1/visitor/bookings`; `Idempotency-Key` header; `startTime`, `endTime`, `representativeName`, `identityNumber`, `expectedVisitorCount`, `identityUploadToken`; compact creation receipt | Route, required form data, submission recovery, and success parser differ. Backend identity-number requirements also require a frontend form decision. |
| Visitor access | `POST /public/lookup` with code/email, then booking cancellation/admission using supplied booking details | Email challenge and verification under `/visitor/booking-access`, followed by a visitor cookie scoped to `/v1/visitor` | Requires a verification UI and session-based visitor requests. Prefix substitution cannot implement this workflow. |
| Booking lists/details | `page`, `pageSize`, view/search/sort/date filters; `totalScoped`, optional dashboard metrics; `id`, `date`, `time`, `duration`, `room`, `timing` | `limit`, `offset`, status filtering; `{items,total,limit,offset}`; `bookingId`, `startsAt`, `endsAt`, `roomName`, `displayStatus` | List/detail schemas reject the response. Several filters and dashboard aggregates need backend support; converting pagination alone is insufficient. |
| Staff decisions | Approve with room name/duration; mutation returns full booking; direct reschedule action | Approve takes `roomId`, `reason`, `version`; mutations return booking ID/status/version; rescheduling creates a proposal requiring visitor acceptance | Approval requires room IDs/catalog changes and a follow-up read. Rescheduling requires an agreed workflow and UI changes. |
| Reception/QR | `/staff/resolve` with `credential`; generic `checkin`/`checkout` action paths; public `/admission` response includes `serverTime` | `/staff/admissions/resolve` with `bookingCode` or `qrToken`; `/check-in` and `/check-out`; visitor `/qr` and `/qr/replace` omit `serverTime` | Route, credential payload, mutation response, and QR parsing differ. |
| Documents | Public/staff `/attachments`; reusable reference; `{token,expiresAt,serverTime}`; PDF/JPEG/PNG up to 5 MB; download `/attachment` | Visitor `/identity-files`; `{uploadToken,expiresAt}`; JPEG/PNG identity image up to 2 MB; staff download `/identity-file` | Supporting documents and required identity images have different purposes, rules, routes and response shapes. |
| Catalogs/admin/remaining features | Public/staff `/catalogs`; `/staff/employees`, `/staff/roles`, `/staff/integrations`, reports; staff booking creation and cancellation | Officers/rooms have separate routes; employees/roles under `/staff/admin` with different bodies/projections; no corresponding registered routes for several requested features | Current catalog loading blocks booking entry. Employee/role management needs adaptation. Reports, integration settings, staff booking creation/cancellation and restricted-visit projections remain unsupported by the inspected registered routes. |
| Errors | Top-level `{message,code,fieldErrors}` | `{error:{code,message,field}}` | HTTP status survives, but meaningful backend messages, codes and field errors are lost. |

## Evidence locations

Frontend source:

- `packages/data/src/session-contract.ts`, `session.tsx`, `domain.ts`: session parsing, authentication and permission checks.
- `packages/data/src/repositories.ts`, `read-model.ts`, `wire.ts`: paths, request bodies and successful-response validation.
- `packages/data/src/http.ts`: cookies and error normalization.
- `packages/data/src/attachments.ts`, `management.ts`: upload/download and administration contracts.
- `apps/visitor/src/pages/Booking.tsx`: actual visitor submission fields.
- `packages/data/src/config.ts`: loopback development automatically uses preview behavior; a normal local preview is not a live integration test.

Backend source:

- `internal/server/rest/server.go`: actual registered routers and storage/mail configuration gates.
- `internal/server/rest/router/{staff_auth_router,staff_booking_router,visitor_access_router,booking_router,staff_admin_router,employee_router,officer_router}.go`.
- `pkg/dto/{backoffice,visitor_booking}.go`: session, administration, upload and submission DTOs.
- `internal/handler/{staff_booking_handler,visitor_booking_handler}.go`: actual mutation payloads and responses.
- `internal/usecase/{staff_booking/staff_booking_usecase,visitor_access/service}.go`: page/detail, visitor and QR projections.
- `internal/authz/catalog.go`, `internal/server/rest/error_handler.go`: permissions and error envelope.

## Verification

- Frontend typecheck passed.
- All 96 frontend unit tests passed.
- All 34 isolated preview integration tests passed; these exercise the simulated frontend service, not the Go backend.
- Both production frontend builds passed; Rollup emitted non-blocking Zod comment-annotation warnings.
- A direct, in-memory probe passed backend-shaped examples from the inspected DTOs through frontend validators: session, staff list, staff detail, visitor creation receipt and QR were all rejected. This confirms response-shape incompatibility; it is not a live server test.
- An HTTP-client probe using the backend error envelope retained status 422 but lost backend message/code/field information.
- Targeted Go tests passed for handlers, routers, staff authentication, staff bookings, visitor access and visitor submission. The initial default-cache access failure was resolved by using a workspace-local Go build cache. These checks do not establish browser-to-backend integration.
- Existing frontend production-contract tests use supplied fixture responses. Their success does not establish compatibility with this Go backend.
- No live database, email, upload, browser-to-backend or device flow was exercised. Database integration tests require `VMS_TEST_DATABASE_DSN`; no database mutations or migrations were requested or performed.

## Recommended implementation order

1. Agree on the owning contract for canonical roles, visitor email verification, reschedule proposals, required identity images/numbers, staff-created bookings, and restricted visits. These are behavior differences, not just serialization changes.
2. Align sessions, permission semantics and error normalization; preserve server authorization.
3. Align officer/room catalogs and visitor form/upload/submission/recovery, then add visitor verification and cookie-based access.
4. Align staff list/detail pagination, supported filters/metrics, approval/rejection and reception credentials; refetch details after compact mutations.
5. Implement the agreed rescheduling/cancellation/admin/report/restricted-visit contracts and missing backend features.
6. Exercise the production bundles against the real backend and a disposable database. Keep fixture tests, but add compatibility checks that consume actual backend responses.

This review changes no application source or backend files.

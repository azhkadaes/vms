# Frontend-driven API review

Status: working evidence and decision register, not a frozen replacement contract.
Date: 16 November 2026, WITA.

## Authority and scope

- User selected complete planning/review before application edits. Contract documents are authorized; application changes are not.
- Cover the full current reachable frontend, including employee and role administration. Older Phase 1 exclusions do not determine the inventory.
- User reaffirmed strict frontend derivation after initially selecting live availability. Therefore no new availability endpoint is included. Record unavailable functionality as a gap.
- User confirmed representative face photo is required for public registration, matching the visible form. Optional transport-schema fields are a mismatch, not permission to omit it.
- Existing dirty files are preserved. This review does not replace the existing canonical OpenAPI or claim backend implementation.
- SIHP is a structural reference, not a source of VMS business rules or response formats.

## Source hierarchy for this review

Trace reachable route → component/control → hook → adapter/validator. Distinguish visible behavior, transport expectations, preview behavior, and older documents. Ask about contradictions rather than silently adopting a source. A frontend guard does not prove backend authorization.

## Confirmed corrections to the supplied endpoint list

| User item | Current evidence | Contract treatment |
| --- | --- | --- |
| Pejabat/unit dropdown; get jabatan | `frontendx/packages/data/src/contracts.ts` Catalogs includes officers with jobTitle/directorate/deputy | Reuse catalog response; no independent jabatan API demonstrated |
| Available dates/times | `apps/visitor/src/pages/Booking.tsx` date minimum; `validation.ts` future start and 08:00–16:00 WITA | No availability call exists; do not invent one |
| Cek PIN/get booking code | `apps/visitor/src/pages/Status.tsx` uses code plus registration email | One lookup operation with both proofs; no PIN operation |
| Repeated PA/Protocol/Reception counts | `read-model.ts`, `Dashboard.tsx` | Shared scoped list and dashboard metrics, not one endpoint per card or role |
| Protocol VIP dashboard | Shared `/dashboard` route | Role-dependent presentation, no separate VIP dashboard resource |
| Post pengunjung plus detail | `validation.ts`, `repositories.ts` | Representative and named roster embedded in booking creation; no standalone visitor creation demonstrated |
| Approved/check-in/out/current date | `Reception.tsx`, `attendance.ts` | Separate queue query, credential resolution, identity confirmation and attendance commands |
| Terlewat | `attendance.ts`, `wire.ts` | Derived display status `missed`; not a stored seventh state |
| Laporan | `DashboardAnalytics.tsx`, `VisitLog.tsx` | Aggregate counts and individual visit log serve different visible needs |
| Post role policy | `management.ts` | PATCH an existing role's permissions with version; no create-new-role UI found |
| Super Admin booking list | `main.tsx` VisitRoute | Renders visit log; do not assume operational list contract is identical |

## Reachable operation inventory

Paths below include `/v1`. This inventory traces source workflows with approved target changes noted inline. The [operation reference](FRONTEND-DRIVEN-REST-API-2026-11-16.md) defines the current target routes.

| Operation | Consumer/input | Required result/use |
| --- | --- | --- |
| GET /public/catalogs | Public registration | officers `{id,name,vip,active,jobTitle?,directorate?,deputy?}`, rooms, privacy notice `{version,id,en}` |
| POST /public/attachments | ND or representative photo upload; multipart `file,reference` | token, expiresAt, serverTime; distinct document/photo validation must be specified |
| POST /public/bookings | Public registration; representative/contact, officerId,date,time,purpose,identityType,visitors,optional ND token, required photo token, privacy acknowledgment/version, idempotencyKey | Public booking receipt with code and status; roster includes representative first |
| POST /public/submissions/resolve | Uncertain submission; idempotencyKey,email | Submission resolution; prevents duplicate registration after a lost response |
| POST /public/lookup | code,email | Public detail, schedule, timing, permitted history and reasons |
| POST /public/bookings/{id}/cancel | code,email,version | Updated public booking; visible cancellation eligibility |
| POST /public/bookings/{id}/admission | code,email,version | QR payload, expiresAt,serverTime |
| GET /staff/catalogs | Staff destination/room controls | Scoped reference data; client-selected role is not authority |
| GET /staff/bookings | view,q,status,sort,dir,date,vip,page,pageSize,mode | `{items,total,totalScoped,page,pageSize,metrics?}`; dashboard requires metrics |
| GET /staff/bookings/{id} | Authorized detail | Role-projected staff booking, version, timing, permitted notes/history |
| POST /staff/bookings | Standard or restricted creation plus idempotencyKey | Created staff booking; different input shapes, not unrelated visitor creation |
| POST /staff/bookings/{id}/approve | Approval form plus version | Updated booking; enforce schedule/room conflicts atomically |
| POST /staff/bookings/{id}/reject | rejectionReason,internalRejectionNote,version | Updated booking; separate public reason from internal note |
| POST /staff/bookings/{id}/reschedule | New schedule/reason and version | Updated booking/timing; old admission credential invalidation obligation |
| POST /staff/bookings/{id}/cancel | reason,internalCancellationNote,version | Updated booking; scope and lifecycle checked server-side |
| POST /staff/bookings/{id}/update | Restricted edit and version | Updated permitted restricted projection |
| POST /staff/resolve | credential from code or QR | Eligible role-projected record for reception confirmation |
| POST /staff/bookings/{id}/attendance | Explicit `check_in` or `check_out`, credential and expected version after identity confirmation | Updated arrival projection. Only `check_out` allows records still checked in after an earlier scheduled date. The current adapter's two action paths will migrate to this target route. |
| POST /staff/attachments | Staff document/photo upload | Upload token/expiry/server time |
| GET /staff/bookings/{id}/attachment | Permitted document display/download | Validated binary; reception proof binding unresolved |
| GET /staff/bookings/{id}/visitor-photo | Permitted photo display | Image binary; reception proof binding unresolved |
| GET /staff/reports/summary | from,to,status | filter,scope,generatedAt,canExport,revision,counts,total |
| GET /staff/reports/export | Same filters plus revision | filename,csv,revision; separate export permission |
| GET /staff/reports/visits | from,to,q,status,kind,officer,room,sort,dir,page,pageSize | Filtered rows, summary,total,paging,revision,generatedAt,canExport,officers,rooms |
| GET /staff/reports/visits/export | Same filter plus revision | Complete matching snapshot; UI creates downloadable report |
| GET /staff/employees | q,page,pageSize | items,total,totalAll,page,pageSize,passwordMinLength |
| GET /staff/employees/{id} | Employee detail/editor | Employee profile,roles,assignment,active,version |
| POST /staff/employees | Employee form and initial password policy | Created employee; no password returned |
| PATCH /staff/employees/{id} | Profile,roles,assignment,active,version | Updated employee |
| GET /staff/roles | Peran & akses | Existing role definitions with permissions,assignablePermissions,editable,version |
| PATCH /staff/roles/{id} | permissions,version | Updated definition; no arbitrary role creation |
| POST /staff/sessions | email,password | Current login adapter then reads session; exact successful login payload not consumed |
| GET /staff/session | Session bootstrap/refresh | Active or role-selection-required principal, permissions,assignments,password policy |
| POST /staff/session/role | role | Server-confirmed active principal; discard old private caches |
| DELETE /staff/session | Logout | Success/204 accepted by shared client |
| POST /staff/password | currentPassword,newPassword | Success followed by session workflow |

Supporting source: `frontendx/packages/data/src/{repositories,read-model,management,attachments,http,contracts,wire,validation,reports,visit-log,session-contract}.ts`, `session.tsx`, and app routes. Integration adapters exist but `/integration` redirects; they are not included solely because unused code exists. Verify any alternate reachable controls before freezing that exclusion.

## Endpoint redundancy review

Reviewed all 41 original method/path operations. The approved attendance merge leaves 40 in the current target. The counts below cover that entire target. Further candidates are proposals and have no effect on the current operation reference.

| Area | Operations | Consolidation assessment |
| --- | ---: | --- |
| Public and staff catalogs | 2 | Share the reference-data query and serializer where projections overlap. Keep separate routes because public registration uses a public projection and privacy notice, while staff options are authorized and scoped. |
| Public and staff uploads | 2 | Each route already handles Nota Dinas and visitor photos via `purpose`. Share upload validation and storage logic. The public and staff routes keep their distinct authorization and reference ownership. |
| Public booking workflows | 5 | Creation, submission recovery, booking lookup, cancellation and QR issuance each produce a distinct result or side effect. Recovery uses email with an idempotency key. Lookup and visitor actions use email with the booking code. Share proof validation across lookup, cancel and admission where appropriate. |
| Staff booking reads and creation | 3 | Dashboard, operational list and reception queues already share `GET /staff/bookings`. Keep focused booking detail separate for its authorized projection. Staff creation already accepts ordinary and Terbatas variants. GET and POST share the collection path and have distinct purposes. |
| Staff booking decisions and edits | 5 | Approve and reject are a good pair for one `POST /staff/bookings/{id}/decision`; both decide a Pending request and return the updated booking. Require `action: approve` or `reject` and validate each action's own fields and permission. Rescheduling, cancellation and Terbatas edits have distinct eligibility, side effects and data. |
| Reception resolve and attendance | 2 | Approved merge uses one attendance endpoint with explicit action. Keep resolve as the credential lookup before the operator confirms identity. Resolving a booking does not record attendance. |
| Existing booking file reads | 4 | Strongest further candidate. Two file kinds can share `GET /staff/bookings/{id}/files/{fileKind}` for authorized staff and `POST` on the same path for reception with `{credential}`. Allow only attachment and visitor-photo. Keep GET scope separate from the reception POST proof and preserve each file's content type. Four operations would become two. |
| Reports and exports | 4 | Reuse filters, scoped queries and snapshot logic. Summary and visit log return different data. Exports require an explicit revision and export permission and may return a complete snapshot. Keep the read/export boundary. |
| Employee and role administration | 6 | Employee reads and writes already share collection/detail paths by method. Role permission editing is a different resource. Share pagination and version checks across suitable handlers. |
| Staff session and password | 7 | Session read and logout already share a path by method. Sign-in, refresh, activity renewal, role selection and password change each have distinct inputs and session effects. Share session logic and keep focused operations. |
| **Current target** | **40** | No separate endpoint needed for each dashboard count, visit queue, upload purpose or attendance direction. |

Further recommendation: parameterize file reads first. This reduces the target to 38 operations with no visible workflow change. A combined approve/reject decision would bring it to 37 and should retain separate input schemas and permissions. A generic status-write endpoint would conceal those safeguards and is unnecessary for the current workflows.

### Shared dropdown data

| Frontend consumer | Shared request | Reuse rule |
| --- | --- | --- |
| Public registration Pejabat and privacy notice | `GET /v1/public/catalogs` | `Booking` and `PrivacyNotice` both call `useCatalogs`. Reuse the anonymous cache entry for both controls. |
| Staff Pejabat and Ruangan in appointment creation, approval, rescheduling and Terbatas forms | `GET /v1/staff/catalogs` | These components call the same `useCatalogs` hook. Use the same staff catalog entry for the same session and role. Forms may narrow the options for their UI. The backend still checks eligibility on submit. |
| Employee editor Pejabat assignment | `GET /v1/staff/catalogs` | Reuse the same scoped Pejabat collection. The editor filters assignment choices locally. |
| Employee role and public identity type | Existing frontend options | Role assignment uses the fixed `roles` list. Identity type uses `KTP`, `Paspor` and `SIM`. Neither dropdown requires its own GET. `GET /v1/staff/roles` remains the role-policy management read. |
| Visit-log Pejabat and Ruangan filters | Options in `GET /v1/staff/reports/visits` | Reuse the underlying Pejabat and Ruangan records. Options must reflect report scope and can include historical inactive references, so the active form catalog is not a drop-in replacement. The existing report response supplies options without a second catalog request. |

`useCatalogs` keys the result by session and treats it as fresh for 60 seconds. Calling the hook from several controls in the same app and scope can reuse the same cache entry. Public and staff use separate projections, and staff reuse is scoped to the active session and role. See [catalog hook](../../frontendx/packages/data/src/hooks.ts#L19) and [catalog route selection](../../frontendx/packages/data/src/repositories.ts#L65).

### One operational booking GET across pages

All rows below go to `GET /v1/staff/bookings`. Different query strings select the view and page. The server scopes the results and applies filters before pagination. Dashboard metrics must cover the authorized dataset for each metric, including the already approved cross-date `inside` count.

| Screen | Example query |
| --- | --- |
| Dashboard | `view=dashboard&date=2026-11-18&page=1&pageSize=50` |
| Booking list | `view=list&status=approved&page=1&pageSize=7` |
| Today's bookings | `view=list&date=2026-11-18&page=1&pageSize=7` |
| Reception check-in queue | `view=reception&mode=checkin&date=2026-11-18&page=1&pageSize=25` |
| Reception checkout queue | `view=reception&mode=checkout&page=1&pageSize=25` |

The production hook keys results by session and filter. Consumers with equivalent parameters can share cached data. A new filter, page, view or role needs its corresponding result, since rows, totals and permitted fields can differ. The target adapter should normalize equivalent queries to one key and use one refresh path per active query. Current booking reads may refetch on focus or reconnect. These observations come from source and do not establish network request counts. Booking detail uses its own ID route for the authorized detail projection. Visit logs and analytics use their report endpoints because their projections, counts and export revisions differ from the operational list. See [shared booking hook](../../frontendx/packages/data/src/hooks.ts#L28) and [request builder](../../frontendx/packages/data/src/read-model.ts#L27).

## Common data observations

- JSON adapters expect direct camelCase records/pages, not SIHP's snake_case success envelope.
- Code is eight digits; phone, code, and service identity remain strings.
- Stored statuses: pending, approved, rejected, cancelled, checked_in, checked_out. `missed` is a time-derived display value.
- WITA calendar date/time and server timing are separate from UTC event timestamps. Current client timing uses numeric milliseconds for serverTime/opensAt/endsAt.
- Mutations use expected version. Public creation uses idempotencyKey and an uncertainty-recovery flow.
- Public and staff projections differ. Restricted is not interchangeable with VIP. General search/count/export must respect projection and scope before aggregation.
- Dashboard metrics are `{today,pending,arrivals,inside,completed}`. Their date/scope semantics require endpoint-level definitions; do not count only the returned page.
- Public purpose is 5–1000 characters; representative/roster names 2–120; phone 8–20 digits; identity choices KTP/Paspor/SIM; roster maximum 20 including representative. These are observed validation constraints, not database type decisions.

## Open decisions and checks

1. RESOLVED: user confirmed current login/session/logout/password-change/role-switching workflows, multiple assigned roles and one active role. Older single-role Phase 1 is conflicting. Cookie attributes, expiration, revocation and CSRF enforcement remain technical contract details requiring evidence; the workflow decision does not invent them.
2. Required photo is confirmed. Need distinguish public requirement from staff creation and ensure upload purpose can be validated with current multipart shape.
3. RESOLVED policy: reception file reads require verified code/QR proof, explicitly confirmed by user. Production adapter omits proof; fix is deferred. Proof transport still needs specification; no existing inspected file implementation establishes parity.
4. Checkout queue verified: `Reception.tsx` applies `date=today()` only to check-in; checkout filters `checked_in` without a date. Preserve this behavior. The user's current-date-only checkout proposal would hide overnight occupants.
5. Confirm each role's read versus decision authority; frontend role guards cannot establish enforcement.
6. Trace report reachability and row/count/export filters before treating analytics and visit logs as redundant.
7. Exact sorting, stable ties, optional/null semantics, validation error mapping, size limits and concurrent command outcomes need final contract tables.

## Verification evidence and limitations

Executed during this review: `npm run typecheck` passed; `npm test` passed 106/106 tests. The npm PowerShell wrapper printed a denied optional roaming-prefix lookup, then both commands ran successfully with exit code 0.

These results establish frontend type/test checks only. No live UI click-through, live backend acceptance, database concurrency, scanner/camera, email delivery, or production readiness is claimed. No application files changed by this review.

## SIHP comparison

Inspected reference source contains Swagger 2.0, handler annotations, public/admin routers, dashboard overview, explicit finalize command and repository transaction. Reuse organization and operation-to-consumer mapping, not commodity rules or authentication scope. SIHP response/pagination differs from VMS. No Go test files were found by the reference investigation; its README command is not test evidence.

The earlier VMS contract refers to uploaded `apicontract-improved.md` and `apicontract revisi2.md`; original attachments have not been read in this review. Existing documentation's description is secondary evidence only.

## Reception contract detail verified after inventory

Check-in queue: `GET /v1/staff/bookings?view=reception&mode=checkin&date=YYYY-MM-DD&page=1&pageSize=25`. Check-out queue uses `mode=checkout` and omits date. The component filters approved arrivals for today and all checked-in departures respectively. Server filtering must produce matching totals, not leave unauthorized/irrelevant rows for client filtering.

Both code and QR first call `POST /v1/staff/resolve` with `{credential}`. The result must have the expected stored status (`approved` for arrival or `checked_in` for departure). The UI then requires the operator to check identity confirmation. The subsequent mutation sends `{credential,version}` to `/checkin` or `/checkout`; the checkbox value itself is not sent. The server cannot treat a local checkbox as independent evidence that physical identity was verified. Whether a durable operator attestation is required is a business decision, not an invented payload field.

On 401/403/404/409 during attendance, the UI discards resolved identity and credential and requires resolution again. A changed version from queue refresh also invalidates confirmation. Server time eligibility (`canCheckIn/canCheckOut`) is displayed, but the command must recheck it atomically.

## Standard versus restricted staff creation fields

Standard input: public-style representative/contact/identity/roster fields plus `duration` (15–240 integer minutes; input schema defaults 60), optional `room` (max 200), `visitorInstructions` (max 2000), `coordinationNote` (max 2000), document/photo names and upload tokens. Staff photo requiredness is not inferred from the confirmed public requirement.

Restricted input is a strict alternate schema: `visitKind=restricted`, required date/time, optional room, officerId, name, organization, serviceId, visitorInstructions, coordinationNote. It does not inherit public email/phone/roster/photo fields. Optional blank identity fields normalize to omission. Do not add ordinary required fields or conflate restricted with VIP.

## Shared HTTP behavior observed

The current client sends cookies (`credentials: include`); writes add `X-VMS-Request: browser`. It sends JSON unless the body is FormData; missing method defaults to GET without body and POST with body. A successful 204 is accepted; other successful JSON operations reject invalid/empty JSON. Authenticated 401 clears private cached data.

Errors consumed by UI: `{message,code?,fieldErrors?: Record<string,string[]>}` plus optional numeric `Retry-After`. Components distinguish validation, authentication, forbidden, stale/conflict and network errors. The client applies a 15-second timeout, which does not prove a timed-out mutation failed to commit. Contract recovery must preserve that distinction.

These observations do not settle cookie attributes, CSRF verification, session expiration or server-side revocation. Those must be explicitly specified with the authentication decision; the custom request header alone does not establish server protection.

## Authentication workflow decision and implementation gap

Confirmed by user on 16 November 2026: preserve all current frontend authentication workflows. A role-selection-required session must have `role=""`, `roleSelectionRequired=true`, nonempty unique `availableRoles`, and empty permissions/assignedOfficerIds. It has no operational authority until a role is selected. An active session has a recognized role included in availableRoles. Both require `active=true`, id, name, permissions and assignedOfficerIds. Optional password-change state must include a valid passwordMinLength (8–128) when mandatory.

Role switching posts `{role}` and requires a response confirming that exact active role. The frontend cancels queries, clears caches and reloads the dashboard after success. Login posts `{email,password}` then reads the current session. Logout deletes the session and clears client caches; password change posts `{currentPassword,newPassword}`. These are workflow requirements; they do not justify copying SIHP authentication unchanged.

Current backend `internal/server/rest/router/auth_router.go` registers login/refresh/logout before `BearerRequired`, then current-session/renew/role/password behind that middleware. The frontend HTTP client sends cookies and no Authorization header. This is an integration discrepancy to investigate, not evidence that the current frontend and backend authentication already work together. Refresh and renew are not added to the UI-derived inventory merely because backend routes exist.

Resolved transport choice: user selected Bearer tokens and later frontend connection changes, preserving screens. Backend login returns TokenResult `{accessToken,tokenType,accessExpiresAt,session}`. Further inspection verifies cookie-bound CSRF/Origin requirements for login/logout and refresh cookies, so this is not a cookie-free lifecycle. Required client work includes token capture/header handling and lifecycle bootstrap/CSRF alignment; renewal/persistence remains unresolved. See the detailed contract authentication section.

Subsequent confirmed decision: automatically restore login and refresh access tokens while the server session is valid; require login on session expiry. Add POST /staff/session/refresh as the supporting operation for this explicit decision. Role/password responses also return TokenResult and require adapter corrections. Session duration and activity-renewal policy remain open; source values are evidence, not silent approval.

Session limits subsequently confirmed: up to five-minute access tokens, eight-hour inactivity expiry, 24-hour absolute expiry. Actual staff activity renews inactivity within the absolute limit; background refresh does not. POST /staff/session/renew is required support for that agreed behavior. Details and acceptance cases are in the detailed contract; this closes the duration policy question without claiming implementation.

## Reception file evidence

`frontendx/tooling/preview.ts` file handler accepts GET/POST, but for Receptionist requires POST, matching booking code and approved+checkin permission or checked_in+checkout permission. Other scoped roles use GET. Restricted files are excluded. Production `loadArrivalFile` instead makes an authenticated GET without credential. Backend public/auth routers inspected here contain no booking-file route. Therefore preview and production adapter parity is not established; there is no inspected file implementation that resolves this discrepancy.

ND is PDF only; representative photo is JPEG/PNG. Both reject empty or >5 MiB files; names, when supplied, must be nonempty and at most 255 characters. Signature checks exist in the client but must also be enforced by the server. These are current source constraints, superseding older documentation describing ND image uploads for this review.

User confirmed reception proof requirement: do not freeze the bare GET entries above as sufficient for Receptionist. They describe current adapter calls with a known contract gap. A final reception-specific request must carry or securely bind verified proof as well as staff authentication and scope. The ordinary authorized staff download path is a distinct authorization case.

Proof transport subsequently confirmed: POST /v1/staff/bookings/{id}/attachment and POST /v1/staff/bookings/{id}/visitor-photo with JSON {credential} plus Bearer. Reuse the already-entered code/QR automatically; verify each request. No server-side verification grant. This adds two method-specific operations to the inventory and replaces reception's current bare GET behavior; other authorized staff GET reads remain separate.

## Role and lifecycle evidence

`domain.ts` separates read scope from command authority. Every operation additionally requires the corresponding permission; a role label alone is insufficient.

| Active role | Operational read scope | Creation/decisions |
| --- | --- | --- |
| PA | Assigned non-VIP ordinary visits and assigned restricted projections | Ordinary non-VIP visits for assigned officers; no restricted decisions |
| Protocol Officer | All operational visits under bookings permission; role-specific field projection | Ordinary VIP visits; restricted creation/updates and decisions |
| Receptionist | Attendance statuses and redacted projections | Check-in/check-out only with permission, eligibility and credential; no approval authority |
| Super Admin | Operational scope helper returns restricted visits; routed ordinary browsing uses separate visit-report model | No implied operational approval authority; administration/reporting use their own permissions |

For approve/reject, current UI requires pending. Reschedule requires approved, including derived missed only through seven days after scheduled end. Staff cancellation permits pending/approved with decision ownership; public cancellation additionally requires time strictly before start. Restricted update requires Protocol Officer, create permission and pending/approved. Check-in is inclusive from start minus 30 minutes through scheduled end; checkout is allowed while checked_in even on a later day. All must be rechecked by the server at command time, not inferred from a previously returned button state.

## Additional photo and dashboard findings

`CreateAppointment.tsx` independently requires a representative photo in both step navigation and submit; this is not merely inferred from public registration. The standard staff transport schema still makes photo fields optional. Under strict visible-frontend derivation, standard staff creation also requires the photo; restricted creation does not expose that field.

Dashboard `inside` fallback filters today's records, but its card links to all checked-in visits and checkout intentionally spans dates. Therefore the label 'Sedang berkunjung' and its destination can disagree for overnight occupants. Production requires server metrics but does not define their time scope in the response type. This is a real semantic ambiguity requiring confirmation rather than a new endpoint.

Resolved by user on 16 November 2026: `metrics.inside` counts every authorized visit with stored status `checked_in`, regardless of scheduled date. It ignores dashboard date and paging for this metric; authorization still applies. This is a count of visit records, consistent with current UI calculations, not the sum of roster members. The today-only fallback is a recorded implementation mismatch. A visitor checked in yesterday remains counted until checkout; a visit checked out today is no longer inside. No separate count endpoint is needed.

## Reporting purposes and privacy discrepancy

Aggregate analytics answers how many visits have each display status in a selected date range. `aggregateVisits` explicitly removes restricted visits before authorization/date/status aggregation. Its CSV exports summary counts, not individual identities.

Super Admin's visit log answers which visits occurred and supports row search/filtering, pagination and CSV/PDF export. `canReadVisitLog` requires Super Admin plus reports permission. `projectVisitLog` includes restricted records, including name, organization, host, schedule, room, status and attendance timestamps; it replaces their credential with a TRB reference, blanks purpose and sets visitor count to null. It does not include contact details, service IDs or internal notes. This row model differs deliberately from operational booking scope and aggregate reports.

The older documentation prohibits general export of restricted records. Current visit-log code permits projected restricted rows in the export snapshot. This privacy boundary needs explicit confirmation; neither silently removing the visible Terbatas filter nor exporting those identities based on old assumptions is appropriate. Endpoint shape can be documented while this content rule remains open.

Resolved by user on 16 November 2026: authorized Super Admin exports include the limited restricted row projection described above. The older prohibition is superseded for this report. Contacts, service IDs, admission credentials and internal notes remain excluded. Aggregate analytics still excludes restricted visits; these are distinct datasets and their totals need not match. No actual visitor data was exported during this review.

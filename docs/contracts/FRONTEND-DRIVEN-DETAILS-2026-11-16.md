# VMS frontend-driven REST contract details

Working draft, 16 November 2026 WITA. Read with the [operation inventory, evidence and decisions](FRONTEND-DRIVEN-REVIEW-2026-11-16.md). This supplements that inventory; it does not claim deployment or replace canonical OpenAPI yet.

The [operation reference](FRONTEND-DRIVEN-REST-API-2026-11-16.md) contains method, endpoint, request, response and datatype sections for all 40 target operations. The [database mapping](DATABASE-MAPPING-2026-11-16.md) compares the supplied diagram and local migrations. Complete database alignment is still under review.

## Contract boundaries

Booking responses use a nested `pejabat` object with `nama: string` for the official's display name. Read it from `pejabat.nama` through `detail_kunjungan.id_pejabat`. The UI label remains “Pejabat”. This applies to public and staff booking records, including list items, recovery responses and reception projections wherever that name is already visible. Omit `pejabat` from projections that hide the official. `officerId` retains its existing role in requests and permitted responses. The current booking adapter reads `host` and needs to read `pejabat.nama` for the target contract. The catalog and report models and the existing sort query keys are outside this booking response change.

Stored status vocabulary confirmed: database status_penerimaan and API status both use pending, approved, rejected, cancelled, checked_in, checked_out. UI labels remain Indonesian. API timing.displayStatus and report/filter enums additionally support derived missed; it is never accepted as a direct persisted lifecycle mutation.

Time representation confirmed: actual event timestamps use PostgreSQL TIMESTAMPTZ and offset-bearing ISO/RFC3339 API strings; UI converts to Asia/Makassar. Schedule date/time stay separate DATE/TIME interpreted in WITA. Existing numeric serverTime/opensAt/endsAt API fields remain epoch milliseconds; this choice does not silently change their wire types. Example: `2026-11-17T02:00:00Z` and `2026-11-17T10:00:00+08:00` represent the same instant.

Room presence semantics confirmed 17 November 2026: create with omitted/null roomId means no room; on approval, rescheduling or edit, omission preserves the current room, null clears it and a nonempty ID replaces it. Empty string is not an ID or a clearing instruction. Resolve the effective room before conflict checks; preserve the expected-version transaction. A cleared room is stored as SQL NULL, never 0 or a placeholder room. The future Go request parser must distinguish omitted, null and supplied values; a simple nullable pointer alone can lose that distinction.

Latest confirmed rule, 17 November 2026: every visit, including restricted/Terbatas, requires officerId. Room is optional for all visits and all schedule mutations, including approval and rescheduling. This explicitly supersedes current frontend behavior wherever it permits an absent restricted officer or requires an ordinary room. Backend must validate a supplied room as a valid permitted active room; omission does not justify a fabricated room. Proposed database: detail_kunjungan.id_pejabat NOT NULL; id_ruangan nullable.

Database duration decision confirmed 17 November 2026: API `duration` maps to proposed `visit_duration INTEGER` measured in minutes, with range 15–240. This supersedes the suggested TIME/DOUBLE alternatives. Owning table remains under review; no schema migration has been performed.

`detail_kunjungan` owns the officer, date, start time, duration and room foreign key. Writes and dropdown selections use `roomId`. Catalog room options are `{id:string,name:string,active:boolean}[]`. Booking responses return a derived `room` display label and `roomId` for selection. Report room filters also use IDs. See the room presence rules above for clearing and preservation.

Base prefix `/v1`; direct camelCase JSON. Public and staff responses are separate DTOs. Derive fields from reachable components, not database tables. Existing adapters define observed transport; user-confirmed corrections are marked explicitly. New behavior without a current consumer is excluded. SIHP's response envelope, snake_case and pagination are not copied.

Opaque record IDs are strings at the frontend boundary. Current frontend evidence alone does not establish UUID versus decimal backend storage. Codes and phone numbers remain strings. Dates are calendar `YYYY-MM-DD`, times `HH:mm`, interpreted in Asia/Makassar. Versions are positive integers. Absent optional fields are not automatically interchangeable with null.

## Public registration

Consumer: visitor Booking wizard and receipt. No independent visitor-resource creation is required.

Confirmed relational ownership: Kunjungan uses unique id_detail_pengunjung and id_detail_kunjungan FKs. The single booking-create operation owns their atomic creation; clients do not issue separate detail-creation requests or submit arbitrary child IDs. Bookings retain their public booking ID; child primary keys are not added to DTOs solely because they exist in storage.

Both detail records exist for every visit. In particular, restricted creation always creates detail_pengunjung and Kunjungan.id_detail_pengunjung is required. Restricted omitted identity/contact/roster/photo fields remain absent rather than fake ordinary values. Ordinary validation still requires its complete form data. This does not add those ordinary fields to the restricted request DTO or relax ordinary validation.

Confirmed persistence mapping: one `detail_pengunjung` record per visit stores representative details and ordered `nama_rombongan`; API `visitors:string[]` maps to that array. Representative is first; total 1–20; no individual companion rows. Booking creation must commit the visit and its visitor-detail record together. This does not add a separate public POST visitor endpoint. Original Pengunjung table ownership is still being reconciled, so do not assume cross-visit identity reuse.

Subsequent confirmed decision: detail_pengunjung replaces Pengunjung's role in the proposed design. Representative fields and roster have one authoritative per-visit record; no reusable profile or matching returning visitors. API paths and payloads remain unchanged. This is a design change only, not authorization to delete a table or migrate existing data.

Identity mapping confirmed: ordinary `identityType` maps to `detail_pengunjung.identity_type`, enum KTP/Paspor/SIM. No ordinary `nomor_identitas` field is stored, requested or returned. Physical identity verification at reception remains the existing workflow. Restricted `serviceId` is separate optional text and must not inherit ordinary identity-type requirements.

| Field | Input constraint / meaning |
| --- | --- |
| officerId | Required active catalog destination; submit ID, not label |
| date,time | Required future start, 08:00–16:00 WITA inclusive; no live availability query |
| purpose | Trimmed 5–1000 characters |
| purpose database mapping | Confirmed target: detail_kunjungan.keperluan VARCHAR(1000); ordinary visits only require this field |
| name | Representative, trimmed 2–120 characters |
| organization | Trimmed minimum 2 characters; maximum not established by this public schema |
| email | Valid email; normalized lowercase by transport schema |
| phone | 8–20 numeric characters, represented as text |
| identityType | KTP, Paspor or SIM; not identity document number |
| visitors | 1–20 names, each 2–120 characters; representative must be first and equal name |
| visitKind | standard; public cannot create restricted records |
| ndName,attachmentToken | Optional ND filename/upload token; filename ≤255, token 16–4096 characters |
| visitorPhotoName,visitorPhotoToken | Required by confirmed public form rule; current optional schema is a mismatch |
| ack | Must be true |
| privacyVersion | Published notice version; production form refuses submission without notice, despite optional transport schema |
| idempotencyKey | Stable reference for this submission attempt; reused for uncertainty recovery |

`POST /public/bookings` returns PublicBooking. Do not split representative and roster into separately committed operations. Transport failure after submission is an unknown outcome, not an instruction to create another booking.

`POST /public/submissions/resolve` takes `{idempotencyKey,email}`. Response is exactly one of `{outcome:"committed",booking:PublicBooking}` or `{outcome:"not_committed"}`. A failed recovery remains uncertain; it is not not_committed. The client preserves the key until the receipt is displayed or non-commitment is established. Do not replace this with code-only lookup.

`GET /public/catalogs` supplies officers and published privacy notice. Officer metadata already supports name/job title/directorate/deputy rendering. No separate GET jabatan is justified by these controls. Identity type choices are fixed application choices, not a fetched catalog.

## Public status and admission

`POST /public/lookup`: `{code,email}`. Code is eight digits and email is the registration email. POST keeps the pair out of URL history/query logs. No separate PIN or get-code endpoint is demonstrated.

PublicBooking fields consumed: id,code,name,organization,email,phone,officerId,pejabat,vip,date,time,duration,room,purpose,visitors,identityType,status,version,timing,history. `pejabat` is `{nama:string}`. Optional fields are ndName,visitorInstructions,rejectionReason,cancellationReason,rescheduled,visitKind. History uses `{at,action,publicMessage?}`. Never return internal notes, service IDs or restricted records.

`POST /public/bookings/{id}/cancel`: `{code,email,version}` → updated PublicBooking. Only pending/approved before scheduled start.

`POST /public/bookings/{id}/admission`: `{code,email,version}` → `{payload,expiresAt,serverTime}`. QR is a credential, not the same as the lookup code. The frontend refreshes eligibility; backend must recheck current state/version. Exact credential format and lifetime cannot be inferred from preview strings.

The existing approved backend plan supplies the QR rules beyond the preview. D21 caps admission expiry at the earlier of issue time plus five minutes and the scheduled check-in window end. A replacement revokes the predecessor. Successful check-in consumes the credential for admission. D96 allows the same consumed, nonrevoked credential to resolve and check out its checked-in visit after the admission expiry, including overnight. `resolve` and the file reads have no attendance side effect. Check-in and checkout still require the current version, matching proof and authorized reception action. See [approved backend decisions](../BACKEND-PLAN-2026-11-11.md).

The diagram's QR record supplies visit ID, token hash, issue time, expiry and revocation time. The physical design must also represent the issued visit version and admission consumption, either directly or through an unambiguous committed event. At checkout, validate against the current expected booking version without rejecting the consumed QR solely because check-in incremented the booking version. Physical column names and the storage layout remain proposals.

## Shared staff list and dashboard

Confirmed missed semantics: status remains approved; timing.displayStatus becomes missed only when serverTime is strictly after scheduled end without check-in. The same server-time evaluation must drive returned display values, status filters, counts and exports. Evaluate one consistent timestamp per response; do not count the same visit in both approved and missed buckets. Checked-in visits remain checked_in until checkout, regardless of scheduled end. Revision-sensitive report results must account for time-derived changes, not just row updates.

`GET /staff/bookings` supports the observed `ReadFilter` fields:

| Query | Consumer / meaning |
| --- | --- |
| view | list, dashboard or reception |
| q | List search; scope and projection must precede search |
| status | Operational list status selection; allowed values must match list UI |
| date | Today page/dashboard/check-in calendar date |
| vip | Existing list VIP filter; restricted is a separate category |
| sort,dir | List ordering; exact allowed values/ties require final schema extraction |
| page,pageSize | Paging; response schema permits 1–100 pageSize |
| mode | checkin or checkout for reception |

Response: `{items,total,totalScoped,page,pageSize,metrics?}`. `items.length ≤ pageSize`, `items.length ≤ total`, `total ≤ totalScoped`. `total` is the filtered count; totalScoped is the pre-filter authorized count. Neither reveals rows outside authority. Dashboard requires metrics; do not derive cards from a 50-row page.

Actual operational list requests use pageSize=7, default sort=priority, dir=asc, status=all, and vip=0 unless the VIP-only filter is selected. Important: vip=0 means no VIP-only filter, not non-VIP-only. vip=1 means VIP ordinary visits, excluding restricted. The PA UI does not show this filter. The current ReadFilter has no officerId/pejabat parameter: PA assignment scope is server-owned, while Super Admin report filtering separately has officer. Do not add a selectable officer query to ordinary lists solely from the initial handwritten list.

Metrics shape: `{today,pending,arrivals,inside,completed}`. Confirmed `inside` means all authorized checked-in visit records across dates, including overnight, independent of pagination/date. Other metrics must retain individually documented scope; do not make the dashboard date silently constrain inside. A visit count is not a headcount.

`GET /staff/bookings/{id}` returns one StaffBooking projection, including version and timing. Restricted projections are full, assigned-pa, reception or arrival; never fill omitted sensitive fields from another cached record.

## Staff creation and commands

Restricted identifier mapping confirmed: serviceId:string (optional, at most 100 characters) maps to detail_kunjungan.no_nd VARCHAR(100), not an integer. UI label remains No ID Dinas. Keep it separate from nota_dinas upload reference and ndName filename. Existing restricted projections determine whether the full value, last four characters or no value is returned; storage ownership does not widen access.

Decision storage confirmed: Kunjungan owns latest rejectionReason, internalRejectionNote, cancellationReason and internalCancellationNote in dedicated fields. Proposed SQL names are listed in database mapping. Audit records preserve previous events. Do not reuse visitorInstructions or coordinationNote for decisions. Staff cancellation request `reason` maps to cancellationReason; rejection request rejectionReason maps directly. Public cancellation does not collect a reason. Enforce the existing field-specific public/staff projections rather than returning all persisted fields.

Confirmed note ownership: visitorInstructions maps to detail_kunjungan.catatan; coordinationNote maps to detail_kunjungan.catatan_internal. Each is optional, maximum 2000 characters. They have no duplicate source on Kunjungan. Preserve projection rules: internal coordination is not returned to public visitors or reception. Rejection/cancellation reasons and their internal notes remain separate fields/events, not aliases of these appointment notes.

Confirmed version storage: Kunjungan.version INTEGER NOT NULL DEFAULT 1, positive. Mutations submit expected version; successful changes increment it atomically with the visit and its dependent detail records. Stale authorized updates return 409 without partial writes. Response includes the new version. Read-only operations and derived missed status do not increment the counter. This supports existing request fields without a new screen or endpoint.

`POST /staff/bookings` accepts standard or restricted creation plus idempotencyKey. Standard creation uses representative/contact/roster fields and photo required by the visible staff form, optional ND, duration 15–240 minutes, optional room, visitorInstructions and coordinationNote (each ≤2000). Restricted target input requires officerId,date,time and `visitKind=restricted`; name,organization,serviceId,room and notes remain optional. Do not require public email/phone/roster/photo for restricted creation.

| Command under /staff/bookings/{id} | Body consumed by current flow | Preconditions/result |
| --- | --- | --- |
| POST /approve | roomId (optional),duration,visitorInstructions,coordinationNote,version | Authorized decision owner; pending → approved; schedule conflict check; date/time unchanged |
| POST /reject | rejectionReason (5–1000), internalRejectionNote (≤2000), version | Authorized decision owner; pending → rejected; internal note never public |
| POST /reschedule | date,time,duration,roomId (optional),reason,visitorInstructions,coordinationNote,version | Approved only. Includes derived Missed through seven elapsed days after the original scheduled end, inclusive. Returns revised schedule with Approved status. |
| POST /cancel | reason (5–1000),internalCancellationNote (≤2000),version | Authorized decision owner; pending/approved → cancelled |
| POST /update | Restricted editor fields,version | Protocol with create permission; restricted pending/approved |
| POST /attendance | action, credential, version | `check_in` requires an eligible Approved visit. `check_out` requires Checked in, including earlier-day visits. |

The user confirmed the current frontend rule on 18 November 2026. The server derives the original scheduled end from the stored WITA date/start and duration, then evaluates eligibility using one server timestamp before replacing the schedule. Pending visits and any stored status other than Approved are ineligible. A derived Missed visit is eligible at `originalEndsAt + 7 × 24 hours`. One millisecond later it is ineligible. For example, an approved visit that ended at 11:00 WITA on 17 November remains eligible at 11:00 WITA on 24 November. The replacement start must be strictly in the future. See [the confirmed decision](../specs/frontend-driven-api-2026-11-16/issues/29-reschedule-eligibility.md).

Within one transaction, recheck authority, the submitted expected version, current eligibility and conflicts for the replacement schedule. Update `detail_kunjungan`, retain stored `approved`, increment `Kunjungan.version`, revoke superseded admission credentials and append the audit event. Return `rescheduled: true` and timing for the replacement schedule. An authorized request with stale version, ineligible state, expired reschedule window or conflicting reservations returns 409. Field validation errors use 422. The prior D100 eligibility rule is superseded for this contract.

All return the updated permitted StaffBooking. Exact approval/update form keys remain a schema-freeze check; generic unrestricted entity PATCH is not a substitute. Scope, permission, current version and lifecycle must be checked atomically. UI-disabled buttons are not backend enforcement. Double clicks and competing operators must not cause two transitions.

Approval/reschedule room is optional for every visit under the latest confirmed rule. Duration is 15–240 minutes; the current form converts an empty restricted duration to 60. The approval form does not render date/time fields, so approval must retain the requested start rather than accepting invented schedule fields. The current ordinary required-room form/helper must be corrected later.

Restricted update reconciliation: current editor submits `visitKind,date,time,room,officerId,name,organization,serviceId,visitorInstructions,coordinationNote` and the mutation hook adds version. On edit, empty name/organization/serviceId mean clear, not preserve. Current code also permits clearing officerId, but the confirmed target rejects that because every visit requires an officer. Creation normalizes optional empty identity fields to omission. The editor can change date/time as well as identity, so update must enforce schedule/conflict checks when those values change; do not implement it as an unchecked profile-only mutation.

Observed reschedule reason rule: ordinary visits require at least five trimmed characters; restricted visits permit empty reason, but a supplied reason must meet that minimum. The helper supplies no maximum; do not falsely claim one is already enforced. Latest target permits an empty room for both kinds; any supplied room must be valid catalog data.

Operational list sort keys observed: priority,name,organization,host,time,status. Direction is asc/desc; priority ignores descending in the current UI. These differ from visit-log sort keys date,name,organization,host,status,kind,visitors. Do not merge them into a single undocumented enum.

## Reception and files

Storage decision confirmed: uploaded bytes live in private file/object storage, with metadata and a private key in Uploads. No BYTEA file_data in the proposed target. Upload responses remain token/expiry/server-time metadata; authorized reads remain binary responses. No public storage URL or storage key is added to frontend DTOs. Storage/database coordination must handle partial failures; selecting this storage approach does not establish existing operational readiness.

Checkout timestamp confirmed: API checkedOutAt maps to Kunjungan.waktu_cek_out only; no second checkout-date column. Derive display/report date in WITA. A successful checkout records server time together with checked_out status, incremented version and audit event in one transaction. Unchecked-out visits expose null checkedOutAt in visit-log DTOs. Current preview derives it from history; the real contract must keep history and the authoritative timestamp consistent.

The check-in queue uses `view=reception&mode=checkin&date=YYYY-MM-DD&page=1&pageSize=25`. The checkout queue uses `mode=checkout` without date. Both first call `POST /staff/resolve` with `{credential}` to preview the visit. After confirming identity, the operator uses `POST /staff/bookings/{id}/attendance` with `{action,credential,version}`. The button supplies `action: "check_in"` or `action: "check_out"`. The identity confirmation remains a UI step.

The shared attendance command checks the permission for the requested action, matches the credential to the visit and rechecks eligibility and version in the write transaction. A successful request writes exactly one transition, one server-recorded arrival or departure time and one audit event. It returns the authorized arrival projection with the incremented version. A repeat with a stale version or an already completed transition yields 409. An invalid or missing action yields 422. Preserve consumed QR eligibility for an authorized checkout after admission expiry. Eligibility never selects the action automatically.

Reception file access requires the credential together with staff authorization, visit scope and the relevant attendance eligibility. Preview demonstrates credential-bearing POST. The production adapter currently uses GET without proof and needs the agreed POST connection change. Other staff file reads require their own scope. Restricted visits lack ordinary photo/ND exposure.

Transport now confirmed by user: reception sends `POST /staff/bookings/{id}/attachment` or `POST /staff/bookings/{id}/visitor-photo`, with Bearer authorization and JSON `{credential}`. Response is the validated file binary. The existing entered/scanned credential is reused automatically; no extra confirmation/input screen. Each request must recheck permission, scope, matching credential, permitted attendance state and file availability. It does not grant lasting access. Keep the credential out of query parameters and error messages. Production `loadArrivalFile` must change from GET to this POST in later implementation.

These POST reads are distinct from ordinary authorized staff GET reads at the same paths. Reception must not bypass proof by choosing GET. Match preview's ordinary-record restriction and reject restricted file access. Missing or inaccessible files use a non-disclosing unavailable response; do not expose whether another booking has a photo.

Upload adapter: `POST /public/attachments` or `/staff/attachments`, multipart `file,reference` → `{token,expiresAt,serverTime}`. ND: nonempty PDF ≤5 MiB. Photo: nonempty JPEG/PNG ≤5 MiB. Server must validate actual content and permitted purpose; MIME/filename/client signature checks are insufficient alone. Current production multipart lacks an explicit purpose field, while preview uses separate photo/document paths; this is an outstanding transport mismatch.

Purpose mismatch resolved by user: target multipart fields are `file` (binary), `reference` (string UUID) and `purpose` (string enum document/visitorPhoto). Both audience endpoints share this shape. document requires validated PDF content; visitorPhoto requires validated JPEG/PNG content. Response remains `{token,expiresAt,serverTime}`; token must be purpose-bound in server metadata. Booking attachmentToken accepts document tokens only; visitorPhotoToken accepts visitorPhoto tokens only. Reject a wrong-purpose token with a field error on the corresponding booking token field. Current frontend upload helper must later send its existing kind value as purpose; no additional visible control is needed.

Carry forward prior D22 and D99 for submission recovery and upload replay where compatible with the confirmed purpose and private-storage rules. Booking creates bind a durable idempotency record to the request scope, normalized payload and eligible uploads. A same-key/same-payload replay resolves to the original booking. A changed payload gives 409. Public recovery and creation serialize on the same reference, and a definitive not_committed result closes it against late creation. Upload replay regenerates the same token from its retained derivation metadata and key during its original one-hour validity. Bind purpose into reference matching alongside the existing scope/content checks. See [prior decision register](../BACKEND-PLAN-2026-11-11.md).

Ordinary authorized reads: `GET /staff/bookings/{id}/attachment` returns document binary; `/visitor-photo` returns image binary. Reception requires the stronger proof rule above. Tokens are upload references, never public URLs or permission grants for unrelated bookings.

## Analytics versus individual visit reports

`GET /staff/reports/summary?from=...&to=...&status=...` returns `{filter,scope,generatedAt,canExport,revision,counts,total}`. Range ≤366 days, inclusive calendar dates. Counts include all stored statuses plus derived missed; their sum equals total. Restricted visits are excluded. Scope values: organization,assigned,authorized,attendance.

`GET /staff/reports/export` takes identical filters plus revision → `{filename,csv,revision}`. Export permission is distinct from read permission. This is summary-only CSV.

Reachability verified: Bookings opens Reports inside the 'Ekspor laporan kunjungan' modal when reports and reportExport permissions are present. Thus summary export remains needed even though /logs redirects and analytics is embedded in the dashboard. DashboardAnalytics itself only reads summary; it does not directly export. This explains the separate report operation without inventing an extra standalone page.

Super Admin `GET /staff/reports/visits` takes from,to,q,status,kind,officer,room,sort,dir,page,pageSize. Current filters allow omitted date bounds; do not silently apply the analytics 366-day limit to this separate UI. Repeated query parameters encode multi-select values, e.g. `status=approved&status=checked_in`; do not comma-join them. OR within a selected field, AND between fields.

Room-filter decision confirmed: retain key `room` but use stable room-ID values, e.g. `room=3&room=8`. Response filter.room is the normalized ID string or array, not room names; existing empty string means no room filter. Options become `rooms:[{id:string,name:string}]`. Booking/catalog options may additionally include active; report options should not silently hide a room referenced by historical authorized rows merely because it is inactive. Visit-log rows retain room:string for display and roomId:string|null for identity. Export must use the same selected IDs and revision; do not resolve/filter by names. No new filter control or endpoint is required.

Response: `{filter,items,summary,generatedAt,revision,page,pageSize,total,canExport,officers,rooms}`. Rows contain id,reference,name,organization,officerId,host,date,time,duration,room,purpose,kind,visitors,status,checkedInAt,checkedOutAt. Nullable visitors/attendance timestamps follow the row schema, not the booking DTO.

`GET /staff/reports/visits/export` takes identical filters plus revision and returns the complete matching snapshot `{filter,items,summary,generatedAt,revision}`. The client creates CSV/PDF; no separate PDF backend endpoint is demonstrated. Snapshot item count must equal summary.total; changed revision is a conflict requiring refresh.

Confirmed restricted export rule: authorized Super Admin may export the current limited row projection, including identity label/organization/host/schedule/status. Contact details, service IDs, admission credentials and internal notes remain excluded. Restricted purpose is blank and visitors is null. Analytics excludes restricted records while the log includes them, so the two totals need not match.

## Employee and role administration

`GET /staff/employees?q=...&page=...&pageSize=...` → `{items,total,totalAll,page,pageSize,passwordMinLength}`; pageSize 1–250. `GET /staff/employees/{id}` → Employee.

The backoffice avatar opens a read-only profile for the signed-in employee. A workspace-ready session can read its own employee record. The `people` permission is required to read another employee or list the directory.

Employee: id,name,email,active,assignedOfficerIds,version; current schema accepts role or roles with at least one role; optional jobTitle,deputy,directorate,phone,nip. User confirmed multiple assigned roles with one active role, so legacy singular role is compatibility evidence, not a single-role requirement. Exact canonical response cleanup remains a schema decision.

`POST /staff/employees`: name,email,roles,active,assignedOfficerIds, optional profile fields,initialPassword,requirePasswordChange. At most one assigned officer in current editor. Initial password uses server-supplied minimum and maximum 256. `PATCH /staff/employees/{id}` updates rendered fields with version; activation/deactivation belongs here, not an invented DELETE operation. Never echo passwords.

`GET /staff/roles` → array of `{id,name,description,permissions,assignablePermissions,editable,version}`. `PATCH /staff/roles/{id}` accepts `{permissions,version}` → updated definition. Existing role names are PA, Protocol Officer, Receptionist, Super Admin. No create-new-role UI and no separate policy-dropdown request were found. Backend must reject permissions outside the allowed set regardless of UI state.

## Authentication

Confirmed current workflow supersedes the older Phase 1 single-role target. `POST /staff/sessions {email,password}` then `GET /staff/session`; `POST /staff/session/role {role}`; `DELETE /staff/session`; `POST /staff/password {currentPassword,newPassword}`.

Session parser requires active=true,id,name,permissions,assignedOfficerIds. Active role must be recognized and belong to nonempty unique availableRoles. Selection-required session has role="",roleSelectionRequired=true and no permissions/assignments. Optional roleRevision is a nonnegative safe integer. Mandatory password change requires passwordMinLength 8–128. Switching clears old private caches and must be server-authorized; never union all assigned-role permissions.

Current client sends cookies; backend routes use BearerRequired. Exact credential integration, cookie attributes, CSRF verification, expiration and revocation remain unverified. Backend refresh/renew routes do not automatically become UI contract operations.

Transport decision confirmed 16 November 2026: authenticated staff API calls use `Authorization: Bearer <accessToken>`. Preserve existing screens and workflows; change the frontend connection layer later. This overrides the current cookie-only staff request behavior, not the agreed role/session workflows.

Verified backend login DTO: `{accessToken,tokenType,accessExpiresAt,session}`. The frontend currently discards login response; it must instead validate/capture the access token and expiry, process session state and attach the token only to trusted staff API requests. Never send it to public endpoints or third-party URLs. Do not substitute a flat session response for this token result.

Important existing backend boundary: Bearer-protected business reads are distinct from login/refresh/logout lifecycle. Current Login and Logout require same-origin checks and cookie-bound `X-CSRF-Token`; Login also requires initialized anonymous cookies. Login emits a refresh cookie. Consequently selecting Bearer does not mean removing every cookie or adding only one header. Existing frontend sends `X-VMS-Request`, not X-CSRF-Token. Bootstrap, lifecycle CSRF, token renewal and reload behavior must be reconciled before integration can be claimed. The existing `/internal/staff/initialize` route is infrastructure evidence, not a new user-facing feature.

### Confirmed session restoration

User confirmed automatic restoration on reload and access-token refresh while the server session remains valid. `POST /staff/session/refresh` is therefore a necessary supporting operation for the approved workflow, not a new screen. Existing backend request is `{refreshRequestId}` with a canonical nonzero UUID, refresh cookie and X-CSRF-Token. Response is TokenResult `{accessToken,tokenType,accessExpiresAt,session}` and may rotate the refresh cookie. Invalid/expired server session returns the user to login.

Client integration obligations: keep access tokens in memory; restore via the protected refresh lifecycle after reload. Do not persist refresh secrets in JavaScript storage. Coordinate refresh attempts so simultaneous requests/tabs do not independently reuse a rotated cookie. Reuse the same refreshRequestId when recovering the same uncertain attempt; do not mint a new ID for each retry. Backend source contains a bounded recovery record and revokes sessions on detected refresh-secret reuse; concurrency/recovery behavior still needs integration tests.

Role selection and password change also return TokenResult in the existing backend. Current frontend incorrectly parses role response as a flat Session and ignores password-change response. The adapted client must capture the replacement token before subsequent requests, use result.session for UI state, clear obsolete role-scoped caches and preserve pending-role/password-change restrictions. A role switch must never leave old-role credentials paired with new-role UI state.

Refresh must not automatically replay an uncertain booking/attendance mutation. Recover the original operation using its idempotency/version rules; only retry a rejected request when non-execution is established. Token expiry alone is not evidence that an earlier write failed.

Confirmed session limits are the earliest of issue time plus five minutes, idle expiry and absolute expiry for access tokens. Idle expiry is the last accepted actual activity plus eight hours, capped by absolute expiry. Absolute expiry is session creation plus 24 hours. Equality with either expiry means expired. Staff must sign in again after server-session expiry. Refresh and current-session reads do not renew activity.

`POST /staff/session/renew` is now a documented supporting operation for the confirmed activity behavior: Bearer authorization, JSON `{}`, response Session (not TokenResult). Existing implementation limits persisted activity updates to at most once per minute. The future frontend adapter must coalesce actual interaction signals and must not use polling, token refresh, window visibility alone or unattended timers as activity. At absolute expiry, clear credentials/private state and require login. Do not silently submit a preserved form under a new identity or role after reauthentication.

Session acceptance cases: reload restores a valid session; access-token expiry refreshes without role changes; simultaneous requests share refresh; idle expiry denies refresh; absolute expiry cannot be extended by activity; background refresh leaves inactivity unchanged; role/password token replacement is adopted before later requests; logout prevents restoration. These are specified checks, not newly executed test results.

## Errors and verification obligations

Audit target choice confirmed: internal entity_type/entity_id references identify the affected object. They are distinct from actor_type/id_pegawai and are not client-writable request fields. For visit history, authorize the visit first, then select its target pair and apply the public/staff projection. Do not expose a generic arbitrary-target lookup through public booking history. Preserve events for visit detail changes under the owning visit so history does not require clients to know child table IDs.

Audit attribution confirmed: server records actor_type staff/visitor/system; staff requires authenticated employee ID, while public/system operations do not fabricate one. Neither actor_type nor id_pegawai is a client-writable command field. Commit event attribution together with the booking mutation. Public history exposes only approved event fields/messages, never raw old_values/new_values, internal notes, tokens or private staff details. This supports existing history views without adding general audit-log CRUD endpoints.

UI consumes `{message,code?,fieldErrors?: Record<string,string[]>}` and Retry-After. Validation errors must use actual form keys. Authenticated 401 clears private caches; 403 means denied; 409 requires refresh/reconciliation; 422 maps validation fields; 429 supplies retry guidance. Do not claim exact success status codes where the client only checks response.ok; freeze them explicitly in the final OpenAPI.

Acceptance cases: unauthorized access, wrong code/email, out-of-scope records, role change cache isolation, restricted field leakage, stale version, conflicting approvals, duplicate submission after response loss, check-in boundary times, overnight checkout, filtered counts beyond one page, revision mismatch on export, required photo absent, invalid content upload, and reception file request without proof. Go tests should cover handlers with table-driven cases and real persistence for concurrency invariants. No new Go tests were executed or written in this documentation-only pass.

Existing frontend verification: type check and 106 unit tests passed earlier in this review. This does not establish live frontend/backend integration or production readiness. Final endpoint field coverage and machine-readable schema reconciliation remain outstanding.

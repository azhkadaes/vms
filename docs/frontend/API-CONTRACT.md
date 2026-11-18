# New backend contract consumed by frontendx

> Historical consumer baseline. The [current audited target contract](../contracts/API-CONTRACT.md) records actual frontend callsites, backend source status and coordinated changes, including Bearer/CSRF, Room objects/roomId and token-result responses. Read its gap register before treating the readiness or implemented claims below as current.

## Database alignment update — 9 November 2026

This update supersedes the original single-role and note contracts below. The implementation is in `D:/oikn/VMS/backend`; live database/container cutover remains separate. Approved database exceptions and interface choices are recorded in `backend/docs/DATABASE-ALIGNMENT.md`.

- Business IDs are integer database keys serialized as strings. Booking codes stay eight digits. File object IDs and submission keys remain UUIDs.
- `GET /staff/session` also returns `availableRoles`, `roleSelectionRequired`, and `roleRevision`. A pending multi-role session has `role:""`, empty permissions and assignments, and `roleSelectionRequired:true`. Render the required selection dialog before mounting workspace content or queries. Never infer a role from array order.
- `POST /staff/session/role {role}` validates membership on the server and returns confirmed session authority. The sidebar dropdown offers assigned roles. Confirm discard-or-stay for unsaved drafts; after success cancel cached work and reload the workspace. Single-role sessions enter their assigned role directly. Protected operations return `409 role_required` while selection is pending.
- Employee responses support `roles:string[]`; singular `role` is a compatibility field for one-role accounts. Send `roles` from the searchable checkbox/tag picker. Optional `jobTitle`, `deputy`, `directorate`, `phone`, and `nip` are in an expandable profile section. NIP remains a string. Do not copy an arbitrary assigned role into the active-role field.
- Staff creation supports optional `visitorInstructions` and `coordinationNote` (maximum 2000 characters each). Approval, rescheduling and restricted editing accept these fields; omission retains existing values and an empty string clears one. No standalone note editor is provided.
- Visitor instructions appear on visitor details and in notification emails. Coordination notes are staff-only and excluded from public/email/reception and restricted limited projections. Existing permission/scope checks apply; separate rejection/cancellation notes retain their meaning.
- Verification is recorded in `backend/docs/evidence/2026-11-09/` and `backend/docs/DATABASE-ALIGNMENT.md`. Historical readiness statements below describe the original November 4 checkpoint, not the current implementation status.

Reviewed against frontend data consumers on 4 November 2026, WITA. Scope: D:/oikn/VMS/frontendx, visitor and backoffice. This is the **frontend-driven contract for the backend to be built**, not a claim that a service already exists. Reference backend folders and other backend implementations are outside this contract. Implementation language/database are not selected here.

Backend engineering entry point: [4 November handoff](BACKEND-HANDOFF-2026-11-04.md). Current client evidence and open response-validation items: [4 November readiness recheck](FRONTEND-READINESS-2026-11-04.md).

The request paths and payloads must follow the production adapters in this project. Runtime schemas define currently accepted wire shapes; the field requirements and screen mapping below also describe the data needed for complete screens. A permissive parser or a UI fallback does not authorize omitting required screen data. Examples are illustrative. Known frontend validation gaps are recorded below rather than treated as desired API behavior.

## Transport, identity and failures

- Default base /v1, same origin. Both apps use cookie credentials. The server sets Secure/HttpOnly/SameSite session cookies and validates Origin/CSRF protections; frontend mutations send X-VMS-Request: browser. Do not enable permissive credentialed CORS. No production caller-supplied role query/body is an authorization source.
- JSON bodies/responses except multipart upload and binary download. Successful JSON must match the audience schema; malformed successes are unavailable, not empty data. The HTTP client normalizes timeout/network/401/403/409/422/429/5xx and retains field errors, code and Retry-After seconds. Normal timeout is 15 seconds; a failed upload leaves the form intact for explicit retry with its upload reference.
- Error shape: { message: string, code?: string, fieldErrors?: Record<string,string[]> }. Use generic public unavailable/not-found responses where record existence must remain private. Only an authenticated-route 401 clears the staff session; public incorrect email/code does not log staff out. 403 hides denied cached views; 5xx can retain clearly marked prior data.
- Authenticated responses, private files and public pair-authorized records require no-store and redacted logs. Do not log passwords, secrets, raw admission/upload tokens, emails/codes as credentials or file contents. Public credential pairs are POST bodies, not URL queries.
- Date is YYYY-MM-DD and time HH:mm, interpreted in Asia/Makassar (UTC+8). Event/expiry timestamps are explicit ISO timestamps. Server timing metadata is authoritative; browser elapsed time supports conservative expiry display. Server rechecks every mutation.

## Sessions and account setup

| Method/path           | Input                         | Response/behavior                                                                                                                                                                                                                                                     |
| --------------------- | ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| POST /staff/sessions  | {email,password}              | Establish cookie session; 401 invalid credentials, 429 rate limit, 503 unavailable remain distinct.                                                                                                                                                                   |
| GET /staff/session    | Cookie                        | {active:true,id,name,role,permissions:string[],assignedOfficerIds:string[],mustChangePassword?:boolean,passwordMinLength?:number}. role is exactly one of PA, Protocol Officer, Receptionist, Super Admin. An ordered roles array is never used to infer authority.   |
| DELETE /staff/session | Cookie                        | 204; expired 401 is treated as already signed out.                                                                                                                                                                                                                    |
| POST /staff/password  | {currentPassword,newPassword} | 204 on change. While mustChangePassword=true, server blocks ordinary workspace APIs. Session metadata must supply a valid password minimum (8–128); refresh returns the updated flag. Rate-limit attempts and invalidate other sessions according to reviewed policy. |

Identity, role, permissions, assignment and password-change state form the private cache scope. Authorized reassignment must invalidate/refresh affected server sessions. The frontend refreshes session on return/reconnect and removes old scoped queries. Management writes refresh the acting session; enforcement on every server operation remains mandatory.

## Catalogs and privacy notice

GET /public/catalogs and authenticated GET /staff/catalogs return {officers:[{id,name,vip,active}],rooms:string[],privacyNotice?:{version,id,en}}. IDs/names come from service data. Staff catalogs may include inactive existing assignments so administrators can understand and remove them; public catalogs expose only permissible destinations. VIP classification is server-owned.

Production public registration requires a published, nonempty privacyNotice in both languages and binds privacyVersion to its acknowledgment. Missing policy blocks submission. Changing the version clears the checkbox. Publish organization-approved processing/retention/rights-contact content; this contract does not invent it. Staff catalogs need not carry the public notice.

## Ordinary public visits

| Method/path                         | Input                               | Output                                                                                                                                               |
| ----------------------------------- | ----------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| POST /public/bookings               | PublicBookingInput + idempotencyKey | PublicBooking; unique id and eight-digit code. Optional attachmentToken and ndName; privacyVersion and ack.                                          |
| POST /public/submissions/resolve    | {idempotencyKey,email}              | {outcome:"committed",booking:PublicBooking} or {outcome:"not_committed"}. A missing outcome closes the reference atomically against delayed creates. |
| POST /public/lookup                 | {code,email}                        | Matching ordinary PublicBooking only.                                                                                                                |
| POST /public/bookings/:id/cancel    | {code,email,version}                | Updated PublicBooking. Pending/Approved only, strictly before start, no prior check-in.                                                              |
| POST /public/bookings/:id/admission | {code,email,version}                | {payload,expiresAt,serverTime}; opaque server-issued credential for an eligible Approved visit. Every replacement revokes its predecessor.           |

Input schema is packages/data/src/validation.ts. Ordinary names/organization/contact/purpose/roster are required; representative is first in the roster; at most 20 total. Phone is 8–20 digits, preserving leading zeroes. Start must be valid/future; existing ordinary start range is 08:00–16:00 WITA. Active officer IDs and normalized payload must be validated again by the server. Pending visits may overlap; do not reserve on submission.

PublicBooking has schedule/details, version, timing and typed public history. It never contains restricted visits, internal notes, staff actor/role or raw system messages. Public history uses allowlisted action + at, with explicitly public rescheduling text; public rejection reason remains verbatim. Unknown future actions fall back safely. See wire.ts and history.ts.

Durably bind each reference to normalized input, including attachment linkage and privacy version. Same reference/same input returns the original result; changed input returns 409. One temporary transport retry uses the same request. Unknown outcomes must be resolved before edited public submission. Commit booking, reference outcome, audit and logical notification intent atomically; retries must not duplicate these. Admission issuance, expiry, single use, revocation on cancellation/rescheduling and race protection are server obligations.

## Staff list, detail, creation and decisions

GET /staff/bookings accepts bounded read models:

| view      | Query                                                        | Server behavior                                                                                                                                                                                   |
| --------- | ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| list      | q,status,sort,dir,date?,vip,page,pageSize=7                  | Search/filter/sort after scope projection, then page. sort is priority/name/organization/host/time/status; dir asc/desc. Return full authorized counts, not page-length counts.                   |
| reception | mode=checkin/checkout,date? (check-in only),page,pageSize=25 | Eligible arrivals on the WITA date, or all still checked-in visits including prior dates.                                                                                                         |
| dashboard | date,page=1,pageSize=50                                      | Bounded current-day agenda/attention samples and authoritative metrics for the full authorized scope. Include the earliest agenda rows and highest-priority pending/inside rows needed by the UI. |

Response: {items:StaffBooking[],total,totalScoped,page,pageSize,metrics?:{today,pending,arrivals,inside,completed}}. At most 100 items; never silently truncate while claiming completeness. Dashboard metrics are required. Counts and samples must share one consistent snapshot and scope. Reception excludes Pending/Rejected/Cancelled. Super Admin ordinary details are not opened through administrative rights. Production UI uses these models; repository.list is retained for the synthetic preview/contract unit tests and is not the production list UI path.

GET /staff/bookings/:id returns an authorized direct StaffBooking, independent of the current list page. 404/403 removes unavailable detail. UI list filters do not authorize detail access.

POST /staff/bookings accepts StaffAppointmentInput + idempotencyKey; ordinary and restricted variants are separate. Return Approved with server fields/timing and creation/automatic-approval audit events. Check officer/resource conflicts before saving, reserve immediately, and return 409 on conflict. A room may remain unassigned until rescheduling. Scope and catalog authorization are server decisions. Replays of the same reference must remain safe. Staff form retries retain their reference for unchanged input; do not use a request reference as a credential.

POST /staff/bookings/:id/:action supports approve/reject/reschedule/checkin/checkout/update/cancel. Every body contains version. Allowed body fields depend on action:

- approve: room/duration (optional for Terbatas). Reserve officer or occupied room atomically against Approved/Checked-in commitments.
- reject: rejectionReason (public, required) and optional internalRejectionNote. Never serialize internal text into public/email data.
- reschedule: date/time/duration/room/reason; applies immediately, checks conflicts, records audit and invalidates previous QR. Optional restricted values remain optional.
- update: restricted optional identity/target fields, preserving leading zeroes and recalculating scope; Protocol only, valid status only.
- cancel: mandatory trimmed reason (5–1000 characters) and optional internalCancellationNote (trimmed, at most 2000 characters). Persist reason as cancellationReason in visitor and authorized staff details; include only cancellationReason in the visitor cancellation notification. Persist internalCancellationNote separately and return it only to staff authorized to read cancellation notes (scoped PA/Protocol for standard visits; Protocol/Super Admin full views for restricted visits). Never include internalCancellationNote in public responses, public history, or visitor email payloads. Require explicit cancel permission and existing PA/Protocol decision scope, Pending/Approved only before any check-in. Authorized staff may act after start/end; Super Admin/Reception rights alone do not grant this. Preserve version checks, release reservations, revoke admission and audit the cancellation; public history receives only cancelled action/time. Do not expose legacy audit text as a public reason.
- checkin/checkout: the current wire body is exactly {credential,version}. The receptionist must explicitly confirm identity in the UI before sending; no identityVerified field is currently sent or required by this adapter. The service records the authenticated attendance actor and event. If a separately submitted identity attestation becomes a requirement, update this contract and the frontend together. The credential is revalidated against the record, version, status and time in the same transaction. Check-in start−30 minutes through end inclusive; derive Missed afterward if never checked in. Already checked-in visitors retain late checkout.

POST /staff/resolve {credential} returns only the authorized arrival projection. Manual and scanned credentials follow the same validation. G20 staff cancellation is POST /staff/bookings/:id/cancel {version,reason,internalCancellationNote?}; it is distinct from the public email/code cancellation route and its before-start cutoff.

Operational views refresh at 30 seconds for staff / 15 seconds for reception, pause when hidden, and refresh on manual/return/reconnect. Production refreshes active bounded queries rather than all history. Transient failures mark retained results stale; access denial hides them; forms keep their drafts during background updates. Mutations invalidate list/detail/dashboard keys.

## Restricted projections

StaffBooking is validated by wire.ts. A restricted response must name restrictedView:

| View        | Fields beyond schedule/reference/status/timing                                                                  |
| ----------- | --------------------------------------------------------------------------------------------------------------- |
| full        | Protocol/Super Admin authorized identity and code/history; internal notes require separate decision authority.  |
| assigned-pa | Name, organization, officerId/host, serviceIdLast4; no code/full ID/history or mutations.                       |
| reception   | No optional identity fields, code or internal history.                                                          |
| arrival     | Only supplied name, target and last-four ID after valid credential resolution; no organization/full ID/history. |

The client rejects known fields outside the projection allowlist. This is defense against adapter mistakes, not server authorization. Scope precedes search/count/conflict errors, direct reads, file access and exports. Restricted records/events never enter general reports/exports or public lookup.

## Protected documents (G18/G19)

POST /public/attachments and authenticated POST /staff/attachments accept multipart {file,reference}. reference is a random upload-attempt ID retained while the selected File exists; repeating it must not create duplicate blobs. The browser generates the multipart boundary. Validate PDF/JPEG/PNG, nonempty content, at most 5 MB and safe filename; inspect content/signature, malware and ownership server-side. Return success only when the file can be safely linked:

{token:string,expiresAt:ISO,serverTime:number}. token is opaque, unpredictable, short-lived and write-only. The frontend keeps it in memory, uses elapsed time for expiry and sends it with ndName in the creation payload. It is never persisted as a draft, placed in a URL or displayed as a credential. New-backend token consumption and submission idempotency must be coordinated so replaying a committed reference still works. Unlinked/failed uploads need a reviewed cleanup TTL; no retention duration is invented here.

An authorized staff detail may return ndName and attachmentAvailable:true. Filename alone grants no access. GET /staff/bookings/:id/attachment rechecks session, role/scope, file linkage and availability, audits access and returns the safe binary MIME with attachment disposition and nosniff/no-store. The frontend downloads a bounded supported Blob through a temporary object URL; no public file URL or upload token is reused for reading. Terbatas has no attachment field. Real storage and scanning are unbuilt.

## Administration (G16/G17)

All endpoints below are session-authorized and require the corresponding people/roles/integrations capability. Do not trust menu visibility. Protect administrative lockout, self-escalation and assignment scope at the server.

| Method/path                   | Input/output                                                                                                                                                                                                              |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| GET /staff/employees          | q,page,pageSize=25. {items:Employee[],total:filteredCount,totalAll:authorizedDirectoryCount,page,pageSize,passwordMinLength}. Maximum pageSize 250.                                                                       |
| GET /staff/employees/:id      | Authorized Employee snapshot for conflict reload.                                                                                                                                                                         |
| POST /staff/employees         | {name,email,role,active,assignedOfficerIds,initialPassword,requirePasswordChange}. Administrator sets the masked initial password; no password-setup email flow.                                                          |
| PATCH /staff/employees/:id    | Editable account fields + version. No initial password or change requirement is silently rewritten on ordinary edits. Active=false deactivates; no hard-delete flow.                                                      |
| GET /staff/roles              | RoleDefinition[]: {id,name,description,permissions,assignablePermissions,editable,version}. Four canonical roles; immutable/protected definitions can be displayed but not edited.                                        |
| PATCH /staff/roles/:id        | {permissions,version}; validate the allowed change and administrative lockout rules, refresh/invalidate affected sessions.                                                                                                |
| GET /staff/integrations       | Integration[] with enabled/health/version and server-defined fields. Field kind is text/secret/url/email/number, with label/required and non-secret value or secret configured marker. Never return stored secret values. |
| PATCH /staff/integrations/:id | {enabled,values,clearSecrets,version}. Omitted secret means keep; explicit clearSecrets means delete. Validate settings, credentials, endpoint allowlists and service-side security; health is server-computed.           |

Employee: {id,name,email,role,active,assignedOfficerIds,version}; no passwords/hashes. New account password minimum comes from service metadata, never a hardcoded production policy. The frontend guards edits to its own role/status; the server enforces this independently. Inactive/unavailable existing assignments are preserved until deliberately removed. Unknown validation leaves other inputs intact; 409 requires explicit discard/reload before replacing drafts.

Integration credentials exist only in the editor request while entered. Blank configured secrets remain unchanged; clear is explicit. Required-secret deletion while enabled is blocked by the UI and must be rejected by the server. The service chooses supported integration metadata; preview examples do not mandate a storage/email vendor.

## Aggregate reporting

Authenticated GET /staff/reports/summary?from&to&status returns VisitSummary from reports.ts: echoed filter, authorized scope, generatedAt, canExport, revision, counts and total. Counts sum exactly to total, include derived Missed and exclude all Terbatas before aggregation. Failed/unknown data is not zero. The current UI queries at most 366 days; scope/retention are separate policies.

GET /staff/reports/export?from&to&status&revision requires independent reportExport rights and returns {filename,csv,revision}. Verify revision and current permission/scope; 409 makes the user review a refreshed summary before another explicit export. Only date range/scope/status/count rows, never individual visitors or restricted events. An empty booking list is never used as a report source.

Report queries repeat the parameter for multiple choices, for example `status=approved&status=checked_out`. The visit log and its export also accept repeated `kind`, `officer`, and `room`. Choices within a field match any selected value, and different fields must all match. Echo one value as a string and multiple values as an array in the same order. Count, pagination, revision and export must use the same filter.

## Field-level data required by frontendx

All paths below are relative to `/v1`. `?` means optional on the wire, not nullable. Omit optional values when absent; do not replace strings, arrays, booleans or objects with null. IDs, codes, phone numbers and service IDs are strings. Versions are positive integers. JSON responses are direct objects/arrays as described here, without an additional `data`, `result` or `success` envelope.

### Booking schedule and timing

Every public or staff booking response, including a mutation result and a restricted projection, contains this schedule block:

```ts
type StoredStatus =
  "pending" | "approved" | "rejected" | "cancelled" | "checked_in" | "checked_out";
type DisplayStatus = StoredStatus | "missed";
type Schedule = {
  id: string; // nonempty stable record ID; not the visit code
  date: string; // YYYY-MM-DD, WITA
  time: string; // HH:mm, WITA
  duration: number; // integer minutes, 15–240
  room: string; // display name; may be empty when unassigned
  status: StoredStatus;
  version: number;
  rescheduled?: boolean;
  timing: {
    serverTime: number; // Unix epoch milliseconds, not seconds or ISO
    opensAt: number; // check-in opening instant, epoch milliseconds
    endsAt: number; // scheduled end, epoch milliseconds
    displayStatus: DisplayStatus;
    canCheckIn: boolean;
    canCheckOut: boolean;
    canCancel: boolean; // visitor before-start cancellation eligibility
  };
};
```

`timing.canCancel` is visitor eligibility, not staff cancellation authority. Staff cancellation may remain permitted after start/end under the staff decision rules. `opensAt` is start minus 30 minutes; `endsAt` is start plus duration. A missed visit remains stored as approved. Response status, timing and version must describe the same snapshot. The service must supply duration and room even when the creation form did not ask for them; do not require an uncollected field to create a visit. Creation defaults are service policy within the accepted range, not values the browser can infer.

### Ordinary booking responses

The ordinary visit screens need the following details in addition to Schedule:

```ts
type OrdinaryDetails = {
  code: string; // exactly eight digits, including leading zeroes
  name: string; // representative
  organization: string;
  email: string;
  phone: string;
  officerId: string; // matches a service catalog ID
  host: string; // destination display name
  vip: boolean;
  purpose: string;
  visitors: string[]; // representative first, 1–20 names including representative
  identityType: "KTP" | "Paspor" | "SIM";
  ndName?: string;
  rejectionReason?: string;
  cancellationReason?: string;
};
type PublicBooking = Schedule &
  OrdinaryDetails & {
    visitKind?: "standard";
    history: {
      at: string; // ISO event timestamp
      action:
        | "received"
        | "approved"
        | "rejected"
        | "rescheduled"
        | "cancelled"
        | "checked_in"
        | "checked_out"
        | "updated";
      publicMessage?: string; // only public rescheduling text
    }[];
  };
type StandardStaffBooking = Schedule &
  OrdinaryDetails & {
    visitKind?: "standard";
    history: { at: string; message: string }[];
    createdBy?: { name: string; role: Role };
    attachmentAvailable?: boolean;
    internalRejectionNote?: string;
    internalCancellationNote?: string;
  };
```

The service returns ordinary details for authorized list/detail/resolve consumers; unlike restricted visits, this frontend defines no reduced ordinary booking projection. Return `history:[]` when there are no visible events. Include reasons when applicable, rescheduled=true after a schedule change, and authorized creator information for staff-created visits. Staff history messages must be suitable for the authorized audience. Internal notes require the separate read authority described above. Public history is chronological, oldest first; raw staff history/actor text never substitutes for public actions. Unknown future public actions are safely displayed as updated by the current parser.

`ndName` supports the document label. Staff `attachmentAvailable:true` is meaningful only with a linked filename and authorized download; never return a token/storage URL in a booking. A filename may be present while download is unavailable. The UI displays the original identity type for physical checking; it does not collect an ordinary identity number or a required identity image.

Current validation limitation: wire.ts makes ordinary staff details/history optional to share a schema with restricted projections. The service must still supply the complete ordinary shape above. Do not use that parser permissiveness as a new partial-response contract.

### Creation inputs actually collected by the forms

```ts
type OrdinaryCreate = {
  officerId: string;
  date: string;
  time: string;
  purpose: string; // trimmed, 5–1000 characters
  name: string; // trimmed, 2–120 characters
  organization: string; // trimmed, at least 2 characters
  email: string; // valid email; public submission lowercases it
  phone: string; // 8–20 digits, leading zeroes preserved
  identityType: "KTP" | "Paspor" | "SIM";
  visitors: string[]; // 1–20, each trimmed name 2–120 characters
  ndName?: string; // trimmed, 1–255 characters
  attachmentToken?: string; // 16–4096 characters; only if a file was uploaded
};
type PublicCreate = OrdinaryCreate & {
  visitKind?: "standard";
  ack: true;
  privacyVersion: string; // required in production, published catalog version
  idempotencyKey: string;
};
type StandardStaffCreate = OrdinaryCreate & {
  visitKind?: "standard";
  idempotencyKey: string;
};
type RestrictedStaffCreate = {
  visitKind: "restricted";
  date: string;
  time: string;
  name?: string; // 2–120 characters when supplied
  organization?: string; // 2–255 characters when supplied
  officerId?: string; // active catalog ID when supplied
  serviceId?: string; // string, 1–100 characters when supplied
  idempotencyKey: string;
};
```

Only date/time are required user fields for restricted creation. The ordinary start-time range is also used by the current restricted form: 08:00–16:00 WITA and future start. Empty optional restricted creation fields normalize to omission. Staff creation does not send visitor consent. No create form sends a room, duration or end time. The backend derives host/VIP, assigns reference/code/status/version/timing/history, and returns the complete audience booking. Do not require a room ID, an identity number, a visitor count instead of the named roster, or an email verification step that these forms do not provide.

Public consent schema currently permits omitted privacyVersion for preview fixtures, but the production UI blocks submission without a published notice and sends its version. The production service must enforce consent/version independently.

### Staff actions and returned data

Every successful booking action returns the updated, complete authorized StaffBooking, not just `{id,status,version}` or an acknowledgement. Public create/lookup/cancel and committed submission recovery likewise return complete PublicBooking data. Restricted outputs retain their proper projection after writes.

| Action             | Body in addition to version                                                      | Frontend meaning                                                                                                                                                                                                    |
| ------------------ | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| approve            | room, duration                                                                   | Room is the catalog display string, duration is integer minutes. Restricted values may be omitted.                                                                                                                  |
| reject             | rejectionReason, internalRejectionNote?                                          | Public reason 5–1000 trimmed characters; internal note up to 2000.                                                                                                                                                  |
| reschedule         | date, time, duration, room, reason                                               | Future WITA schedule, duration 15–240; immediate replacement. Ordinary room and reason of at least 5 characters required; restricted room/reason may be empty, but a supplied reason must be at least 5 characters. |
| update             | visitKind:"restricted", date, time, name?, organization?, officerId?, serviceId? | Restricted editor submits its full current form. Empty strings explicitly clear optional identity/target fields. Date/time remain part of the payload; revalidate schedule/conflicts/scope.                         |
| cancel             | reason, internalCancellationNote?                                                | Public cancellation reason 5–1000 trimmed characters; private note up to 2000.                                                                                                                                      |
| checkin / checkout | credential                                                                       | Opaque QR payload or manual eight-digit visit code, passed unchanged with version. Identity confirmation occurs in the UI.                                                                                          |

Restricted update is offered through Protocol's editor for pending/approved restricted visits; the submitted date/time must pass the editor's future-start validation. Re-evaluate authorization independently for every operation. Ordinary staff appointments auto-approve on creation, so they do not require a later approve/reject step. An optional internal note may arrive as an empty string; treat it as no note without inventing public text.

### Restricted booking response details

Every restricted record contains Schedule, `visitKind:"restricted"`, a nonempty `reference` for labels/search, and a `restrictedView` from the projection table. `reference` is a display reference, not an admission credential. When an optional identity/target was not supplied, omit its fields rather than inserting ordinary placeholder personal data.

- full: optional name/organization/officerId/host/serviceId, plus the eight-digit code for manual arrival and staff history array. May include createdBy/rejectionReason/cancellationReason and separately authorized internal notes. No ordinary email/phone/roster/purpose/identityType, VIP field, or attachment fields.
- assigned-pa: supplied name/organization/officerId/host and serviceIdLast4 only. Last-four values are strings of at most four characters.
- reception: schedule/reference/status/timing only, without identity, code or history.
- arrival: supplied name/officerId/host/serviceIdLast4 after valid credential resolution, without organization, full ID, code or history.

The `full` view is for Protocol/Super Admin within authorized scope; Super Admin rights do not grant mutation authority. PA assignment is checked using officerId and assignedOfficerIds. Never send a restrictedView marker with an omitted or standard visitKind. Current parser checks depend on this discriminator and do not establish server authorization.

### Session, permissions and catalog semantics

`Role` is exactly `"PA" | "Protocol Officer" | "Receptionist" | "Super Admin"`. Session id must be the same account ID used by Employee.id so self-edit restrictions work. `name` is nonempty. Always return arrays for permissions and assignedOfficerIds, even when empty. Role is a single field, not inferred from a roles array. `mustChangePassword:true` requires integer passwordMinLength in 8–128; password inputs accept at most 256 characters.

The existing UI recognizes these permission keys:

| Key                                 | Consumer                                                                               |
| ----------------------------------- | -------------------------------------------------------------------------------------- |
| bookings                            | Booking lists, detail and operational dashboard reads                                  |
| create                              | Ordinary/restricted staff creation, combined with role and destination scope           |
| approve, reject, reschedule, cancel | Decision buttons combined with role/scope/status checks                                |
| checkin, checkout                   | Reception routes and attendance actions                                                |
| reports, reportExport               | Aggregate report view and separate export action                                       |
| people, roles, integrations         | Corresponding administration routes and writes                                         |
| audit                               | Recognized/displayed permission label; no separate audit API consumer currently exists |

Restricted update uses Protocol + create permission and valid state in the current editor. Do not invent a separately required update permission without changing that consumer. Capability keys alone never override role/assignment/projection rules.

Login accepts a 204 success or a successful JSON response; no login-response fields are consumed. The frontend then GETs /staff/session. A 200/201 with an empty or non-JSON body is rejected by the shared HTTP client. Logout/password change use 204.

Catalog officer fields id/name/vip/active are all required. Room entries are display strings used verbatim in room submissions; the frontend has no separate room-ID mapping. PrivacyNotice.id and .en are the Indonesian and English notice text, not notice record IDs. Its version identifies the published notice. Supply nonblank text in both languages. No catalog URL or pagination is consumed. Staff catalogs preserve information about existing inactive assignments; catalog IDs need not match preview fixtures.

### Bounded lists and dashboard values

For list responses, `total` is the authorized count after q/status/date/vip filtering and before pagination; `totalScoped` is the authorized count before those list filters. `page` is one-based, `pageSize` is the applied requested size. Return the requested page metadata and a matching slice, including an empty slice when beyond the result set. Detail id must equal the requested id. Do not silently return a different record or page. These correlations are service requirements even though the current reader does not yet enforce all of them.

`status` accepts all, the six stored statuses, and missed. Approved filtering excludes derived missed records. `vip=1` selects ordinary VIP visits; `vip=0` is the default with no VIP-only filter, not non-VIP-only. Search operates only on visible fields (representative/roster/organization/host/code/reference as available). Default priority order is pending first; approved not-missed/checked-in next in ascending schedule order; remaining closed/missed rows newest schedule first, with stable ID ties. Search match ranking follows selectBookings in domain.ts. Apply filters/search/order before pagination, never to one fetched page.

Dashboard metrics are authoritative counts over the authorized scope, not counts of the at-most-50 returned samples. To match Dashboard.tsx, today counts bookings on the requested WITA date; arrivals counts that date's approved, not-missed visits; inside counts that date's checked_in visits; completed counts that date's checked_out visits; pending counts all scoped pending requests, including other dates. Reception scope excludes pending/rejected/cancelled. The reception checkout endpoint separately includes checked-in visitors from earlier dates.

Dashboard items must contain a deduplicated union sufficient for the first five agenda rows and first four attention rows. For operational staff, agenda is requested-date bookings in time order and attention is all scoped pending in priority order. For reception, agenda is requested-date approved, not-missed bookings and attention is requested-date checked-in bookings in time order. Simply taking the first 50 agenda records can hide attention items and does not meet the screen requirement. A Super Admin dashboard shows administration links and does not request this read model.

### Administration and report field completeness

Employee fields are exactly id/name/email/role/active/assignedOfficerIds/version. Directory metadata additionally includes total, totalAll, page, pageSize and passwordMinLength, including when items is empty. Name is at least two characters and email valid. Employee creation requires initialPassword and requirePasswordChange; an ordinary edit omits both and sends version. No response contains either password field. The editor sends an empty assignment array for non-PA roles; unavailable current PA assignments are retained until removed deliberately.

RoleDefinition requires id/name/description/permissions/assignablePermissions/editable/version. Name is a canonical Role. Unknown permission labels can display as raw keys; do not infer capabilities from label text. Include already granted protected permissions so they remain visible; editable and assignablePermissions control which changes can be offered.

Integration requires id/name/description/enabled/health/version/fields. Health is exactly not_configured, ready or degraded. At most 30 fields; each requires key/label/kind/required. key matches `^[a-zA-Z][a-zA-Z0-9_]*$`. Non-secret value is an optional string, including for number fields; secret uses configured:boolean and never returns value. Writes send values as strings and clearSecrets as an array, including when empty. Field errors for integration inputs use the field key so the generated input can display them.

VisitSummary requires every field below, including all seven zero-valued counts, and no additional envelope:

```ts
type SummaryFilter = { from: string; to: string; status: "all" | DisplayStatus };
type VisitSummary = {
  filter: SummaryFilter; // exact requested filter
  scope: "organization" | "assigned" | "authorized" | "attendance";
  generatedAt: string; // UTC ISO datetime accepted by reports.ts
  canExport: boolean;
  revision: string; // exactly 64 lowercase hexadecimal characters
  counts: {
    pending: number;
    approved: number;
    rejected: number;
    cancelled: number;
    checked_in: number;
    checked_out: number;
    missed: number;
  };
  total: number; // sum of all counts
};
```

All counts are nonnegative integers. With a specific status filter, other status counts are zero. Date bounds are inclusive and from <= to; at most 365 elapsed days (366 dates). Scope labels come from the enum above, not arbitrary role strings. Export is exactly `{filename,csv,revision}`: filename matches `^kunjungan-ringkasan-[\d-]+\.csv$`, csv is the CSV text, and revision is the reviewed 64-character revision. Export success is JSON, not a binary CSV response. Summary/export schemas are strict, so extra keys are currently rejected. Both /analytics and /logs use this same aggregate API; /logs is a visit-report screen, not a separate audit-log API.

Upload token is 16–4096 characters, with expiresAt as a parseable ISO instant and serverTime as finite epoch milliseconds. A token must remain usable after elapsed request time; already expired uploads fail confirmation. Maximum file bytes are 5 * 1024 * 1024. Admission payload is 1–4096 characters; expiresAt/serverTime use the same units. The QR component requests admission on mount, after a booking status update, and automatically when the current QR expires. Issuance must safely handle repeated requests and revoke predecessor credentials as specified above.

Field errors use current form names, not database column names: public/staff ordinary officerId/date/time/purpose/name/organization/email/phone/identityType/visitors/ndName; consent uses ack (include a safe message for policy-version failures); restricted date/time/name/organization/officerId/serviceId; reject rejectionReason/internalRejectionNote; cancel reason/internalCancellationNote; password currentPassword/newPassword; administration its submitted input names. Unknown or cross-field errors also need a meaningful top-level message. Do not rely on fieldErrors alone to explain failure.

## Screen-to-data traceability

| Frontend consumer                    | API/data required                                                                                                                       | Owning source                                                                                |
| ------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Visitor registration/review/receipt  | Public catalogs, active destination, privacy text/version; PublicCreate, optional upload; complete PublicBooking; recovery outcome      | apps/visitor/src/pages/Booking.tsx; packages/data/src/submission.ts                          |
| Visitor status/ticket/history/cancel | Complete PublicBooking, reasons/rescheduled flag, public events, visitor timing; pair credentials/version                               | apps/visitor/src/pages/Status.tsx; features/VisitTicket.tsx; features/PublicVisitHistory.tsx |
| Visitor admission QR                 | payload/expiresAt/serverTime, tied to booking id/code/email/version                                                                     | apps/visitor/src/features/AdmissionQR.tsx                                                    |
| Backoffice login/password/navigation | Session identity, canonical role, permission keys, assignments, password metadata                                                       | packages/data/src/session.tsx; apps/admin/src/main.tsx; features/PasswordChange.tsx          |
| Operational dashboard                | Bounded agenda/attention union; five authoritative metrics; names, organization, kind, schedule, status/timing and IDs for detail links | apps/admin/src/pages/Dashboard.tsx                                                           |
| Booking/today lists and detail       | Filtered paged StaffBooking records, total/totalScoped; independent direct detail; complete ordinary or correct restricted projection   | apps/admin/src/pages/Bookings.tsx; packages/data/src/read-model.ts                           |
| Ordinary staff creation/review       | Staff catalogs, OrdinaryCreate, optional ND upload; complete auto-approved StaffBooking                                                 | apps/admin/src/features/CreateAppointment.tsx                                                |
| Restricted creation/edit             | Schedule and optional name/organization/officerId/serviceId; complete authorized restricted result                                      | apps/admin/src/features/RestrictedAppointment.tsx                                            |
| Approval/rejection/reschedule/cancel | Current version, role/scope/status/timing, rooms; action fields/reasons/notes; complete updated StaffBooking                            | apps/admin/src/features/BookingDetail.tsx                                                    |
| Reception check-in/check-out         | Bounded date/mode list, resolve credential to arrival projection, canCheckIn/canCheckOut/version; updated StaffBooking                  | apps/admin/src/pages/Reception.tsx                                                           |
| Shared visit details                 | name/code/organization/host/schedule/room/identityType/roster/purpose/ND; restricted fields only by projection                          | packages/ui/src/index.tsx (VisitIdentity); packages/ui/src/restricted-identity.tsx           |
| Protected ND download                | ndName, attachmentAvailable, authorized supported binary Blob                                                                           | apps/admin/src/features/AttachmentAccess.tsx; packages/data/src/attachments.ts               |
| Employee directory/editor            | Employee page and password minimum, staff catalogs/assignments, direct conflict reload, versioned save                                  | apps/admin/src/features/EmployeeDirectory.tsx; features/EmployeeEditor.tsx                   |
| Role management                      | Full RoleDefinition list, protected/assignable permission metadata, versioned save                                                      | apps/admin/src/features/RoleManagement.tsx                                                   |
| Integration management               | Full Integration/field metadata, safe secret markers, health, versioned string values/explicit clears                                   | apps/admin/src/features/IntegrationManagement.tsx                                            |
| Analytics/reports/export             | Independent strict VisitSummary; all counts, scope, filter echo, revision/canExport; JSON CSV export                                    | apps/admin/src/pages/Reports.tsx; packages/data/src/reports.ts                               |

Abbreviated features/ paths in the owning-source column are relative to the named app's src folder. Read adapters are packages/data/src/repositories.ts, read-model.ts, management.ts, attachments.ts and session.tsx. Schemas are validation.ts, wire.ts, management-contract.ts, session-contract.ts and reports.ts in that package. Synthetic preview records/roles are development fixtures and do not dictate real IDs, people, destinations, integrations or privacy copy.

## Operational acceptance

Frontend fixture tests exercise requests/responses and failure handling. They do not prove persistent transactions, concurrency, session cookies/CSRF at the gateway, password hashing, genuine email/storage, QR forgery resistance or physical scanning. Build and verify the new service, reviewed migrations, worker retry/recovery, privacy retention, monitoring and deployment before operational release. See DEPLOYMENT.md and VERIFICATION.md.

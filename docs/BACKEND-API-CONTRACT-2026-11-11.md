# VMS proposed API contract

> **Phase 1 supersession, 13 November 2026:** [The current Phase 1 contract](contracts/API-CONTRACT.md) replaces this document's session/refresh/CSRF/role-switch/password-workflow and editable account/role-policy scope for Phase 1. It defines 29 target operations with basic Bearer login/current-user and manually assigned single database roles. Unrelated business rules below remain historical decision evidence. Current source still uses the old session flow until the owning implementation slice replaces it.

> Current review entry point: [audited API contract and enforcement](contracts/API-CONTRACT.md). The selected business rules below are retained; current source status, frontend mismatches, exact endpoint examples and interface mappings are in that audit. The canonical OpenAPI now uses the observed `__Host-vms-*` cookies, `X-CSRF-Token` and `accessExpiresAt` token field in place of earlier labelled naming proposals. This is documentation/enforcement work only; it does not implement missing operations.

Approved planning baseline, 12 November 2026 WITA. The user explicitly approved freezing the reviewed plan, schema register and 38-operation OpenAPI, including the labelled schema/credential/SMTP/transport proposals. This approves planning only: no application edits, database/container operations, migrations or deployment. Reference-first handling and deferral of nonurgent future decisions remain in force. Documentation generator/tool selection and runtime/release gates remain deferred as recorded.

Prepared 11 November 2026, continued 12 November 2026, WITA. **Approved and frozen planning contract, not an implemented API or deployment specification.** Read with [backend plan](BACKEND-PLAN-2026-11-11.md) and [machine-readable OpenAPI](BACKEND-OPENAPI-2026-11-12.json). Existing `frontendx/API-CONTRACT.md` remains the current consumer baseline until coordinated implementation; this proposal does not silently supersede it. D01–D100 in the plan record the user's selections.

Authentication transport selected (D38): Bearer JWT access tokens held in memory plus a host-only Secure HttpOnly refresh cookie. D39 selects separate refresh and activity-renewal operations. Access JWT lifetime is 5 minutes (D41); refresh rotates on success (D42); 10-second same-request recovery is selected (D43); opaque refresh secrets are selected (D46); CSRF initialization is through staff HTML delivery (D47); D48 assigns gateway delivery with internal Go integration; D45 selects signed session-bound double-submit CSRF; related schemas are provisional. Refresh alone never extends inactivity. This remains a planning draft, not implementation authorization.

## 1. Common conventions

- Prefix `/v1`; JSON except multipart uploads and protected binary downloads. Methods are explicit; GET never approves, cancels or records attendance.
- Production uses separate configured visitor/staff hostnames (D37). Each app uses relative /v1 paths on its own origin; gateway exposes only that audience's API routes. Do not deploy a common cross-origin API hostname or shared parent-domain session cookie by default. The backend/worker codebase remains shared. Local development retains its explicitly allowed single-origin gateway.
- Direct records/arrays/page objects, without a success/data envelope. JSON camelCase. Error HTTP status is authoritative.
- Business IDs are PostgreSQL INTEGER keys in 1–2,147,483,647 (D95), encoded as canonical decimal strings, never JavaScript numbers. Reject overflow, zero, signs, fractions and leading-zero aliases before database work. Codes are exactly eight digits, including leading zeroes. UUID upload/submission/object references remain distinct from business IDs.
- `date`: real `YYYY-MM-DD`; `time`: `HH:mm`; business timezone `Asia/Makassar`. Event/expiry timestamps are RFC3339 instants; timing `serverTime/opensAt/endsAt` remain epoch **milliseconds** for current-client compatibility.
- Versions are positive integers. Every mutation of an existing versioned record requires the version it edits. Request cancellation/timeouts do not imply rollback.
- Optional absent values are omitted. Do not emit unexpected nulls. Explicit empty strings/arrays clear editable values only where documented. Immutable IDs, computed flags and permissions cannot be mass-assigned.
- Staff authority comes from a validated Bearer JWT plus the current PostgreSQL session/account/active role/permissions/assignments, not request-supplied role or stale claims alone. Protected staff resource routes require Authorization: Bearer and never fall back to the refresh cookie. Access tokens stay in memory, never URLs or persistent browser storage; designated refresh/session lifecycle routes use the protected refresh cookie. Public email/code credentials travel only in request bodies. Browser session, CSRF/Origin, no-store and log-redaction policies follow the plan.
- Reject malformed JSON/IDs/query syntax with 400; semantically invalid fields with 422. Document allowed query keys, field lengths, page bounds and sort values in OpenAPI before implementation. Do not silently coerce impossible dates or schedules.

Common failure body:

```ts
type ApiError = {
  code: string;                 // stable machine code, never a DB/vendor error
  message: string;              // safe, readable overall explanation
  fieldErrors?: Record<string, string[]>;
  requestId: string;            // correlated with redacted server diagnostics
};
```

| Status | Meaning/examples | Client recovery |
| --- | --- | --- |
| 400 | `invalid_request` | Correct request; no automatic retry. |
| 401 | `unauthenticated`, `invalid_credentials` | Staff-session loss only on authenticated requests; wrong public pair does not sign staff out. |
| 403 | `forbidden` | Hide denied private data and stop protected action. |
| 404 | `not_found` / generic public unavailable | Remove unavailable detail; do not disclose restricted record existence. |
| 409 | `version_conflict`, `schedule_conflict`, `idempotency_conflict`, `role_required`, `role_changed`, `report_changed` | Preserve draft; refresh/review, not blind mutation retry. |
| 413/415 | `payload_too_large`, `unsupported_media_type` | Preserve form; replace invalid upload/request. |
| 422 | `validation_failed`, `privacy_notice_changed`, `invalid_transition` | Associate field errors; retain remaining input. |
| 429 | `rate_limited`, integer-seconds `Retry-After` | Display cooldown; avoid repeated submissions. |
| 500/503 | `internal_error`, `service_unavailable` | Safe message; no stacktrace/SQL/provider details; resolve uncertain creates using original reference. |

Field-error keys match forms: `officerId`, `date`, `time`, `roomId`, `duration`, `name`, `organization`, `email`, `phone`, `identityType`, `visitors`, `ndName`, `ack`, `serviceId`, action-specific notes/reasons and submitted management keys. Cross-field errors also need a top-level explanation. Localization of new server messages is an implementation gate; machine event/error codes must remain language-neutral.

Rate-limit counters are shared in PostgreSQL (D26). Limits use the current authenticated principal/session and/or normalized credential identifier and trusted client-IP scope as defined per operation; proxy headers are not trusted from arbitrary callers. Limits survive API restart and apply across instances. A rejection returns 429 with Retry-After seconds and the common safe error body; do not expose counter keys or whether an account/public visit exists. Selected thresholds follow below; limiter-failure behavior is fixed by D35.

Login account threshold (D27): allow at most 5 failed password attempts for a normalized account email in a rolling 15-minute window, shared across client IPs and API instances. Treat unknown and known emails the same, with generic credential errors. Before accepting another password attempt, atomically account for in-flight attempts so concurrent requests cannot bypass the allowance; finalize failed attempts and release successful reservations without holding a DB lock during password hashing. On exhausted allowance, return 429 with Retry-After until an attempt slot recovers; do not impose a permanent account lock or extend the window merely because rejected requests continue. Keep prior failed attempts until they age out; do not silently reset them on successful login. D28 adds an independent rolling allowance of 100 admitted login attempts per trusted client IP in 15 minutes, counting both successful and failed attempts. Check/reserve the IP allowance before password hashing; reject excess traffic with 429/Retry-After until a slot recovers, without extending the window on rejected requests. Apply both limits across API instances. Values are selected initial settings, not measured production capacity; shared-office traffic must be observed before any proposed adjustment.

## 2. Shared data types and visibility

Limiter failure policy (D35): if an applicable shared counter cannot be checked/updated reliably, return 503 `service_unavailable`; do not allow the request through, report a bad password, convert reads into empty results, or assume an uncertain create rolled back. Preserve drafts/reference and show explicit retry guidance. DELETE /staff/session and basic liveness/readiness probes do not depend on the rate limiter or its budgets; logout still enforces applicable CSRF/session handling, and health responses disclose no private data. This exemption cannot make a failed DB revocation succeed: clear the browser cookie when possible but accurately report server-side revocation failure, and never claim the service is ready when required dependencies are unavailable.

Authenticated staff ceiling (D34): 120 admitted staff API requests per rolling 60 seconds per employee across sessions/tabs/API instances. Key the budget by immutable employee identity, not active role, session token, name or current IP; role switches and new sessions do not reset it. Reads, active polling, mutations, renewal and file access count; independent login/upload/sensitive-operation limits still apply where relevant. Reject excess with 429/Retry-After without turning a denied request into empty data, discarding drafts or blindly retrying mutations. Verify normal reception/staff multi-tab polling within this initial budget. D35 defines failure behavior and safe logout/health exemptions; no throttling exception grants private-data access.

Overall public request ceiling (D33): all /v1/public routes share 120 admitted requests per rolling 60 seconds per trusted client IP across instances. This bounds catalog, upload replay/invalid calls and other public traffic in addition to D29/D32 request limits and D30/D31 creation/storage allowances. Apply the overall admission before reading multipart bodies or doing expensive work; enforce the relevant operation limit as well. Do not double-charge one request within the same budget. If any applicable request budget is exhausted, return 429/Retry-After; rejected calls do not extend its blocked period. Atomic multi-budget reservation/accounting must prevent concurrency bypass and needless consumption of operation budgets when an earlier admission check rejects the request. Treat the values as selected initial settings to verify against legitimate shared-network behavior, not measured throughput guarantees.

Submission/recovery request rate (D32): POST /public/bookings and /public/submissions/resolve share 30 admitted requests in a rolling 60-second window per trusted IP across instances, including invalid input, idempotent retries and outcome resolution. Apply this request admission before expensive validation/creation, independently of D30's new-booking capacity. Excess receives 429/Retry-After without extending the window. Preserve the form and original idempotency reference; a throttled or unavailable resolve outcome is uncertain, never not_committed and never grounds to generate a fresh key. D29's credential-access budget is separate, so status/QR traffic does not consume this recovery allowance. D33 supplies the additional overall public-API admission ceiling for routes outside these operation groups.

Development upload allowance (D31): public/staff upload routes share a rolling allowance of 60 newly accepted file objects per hour per trusted IP. Actor/audience ownership remains separate even though the IP capacity budget is shared. Same-reference/same-content replay returns the existing eligible upload result without another object or allowance slot; changed-content reuse returns conflict. Reserve capacity atomically across instances and coordinate staged storage completion/failure cleanup. A new completed upload consumes its slot until it ages out even if its file is later replaced/deleted. Keep the 5 * 1024 * 1024 byte maximum; 60 maximum-size accepted files represent 300 MiB of accepted new file content per hour, not a total network/staging-storage quota. Independent request/body/concurrency limits must also cover invalid uploads and repeated body traffic. Preserve registration drafts on 429 and expose Retry-After; never claim an over-limit upload was saved. Production file-saving remains deferred.

Public-create allowance (D30): at most 30 new public bookings committed in a rolling hour per trusted client IP across instances. Resolve the idempotency reference before charging a new-booking slot; serialize new-key races and capacity reservation with the booking transaction so the 31st concurrent create is rejected without partial visit/audit/outbox writes. Rollback releases an uncommitted slot. Same-reference retries and outcome resolution never consume another new-booking slot, even when the caller's current IP differs; still recheck credentials/scope and separate request-level admission limits. Cancelled bookings do not refund their creation slot within the rolling hour. Return 429/Retry-After for an exhausted new-booking allowance. This capacity limit is not a per-request limiter; invalid or repeated calls still require independent request-level controls before expensive work.

Public credential-access rate (D29): POST /public/lookup, /public/bookings/{id}/cancel and /public/bookings/{id}/admission share 30 admitted requests in a rolling 60-second window per trusted client IP, across instances. Count both successful and unsuccessful requests; excess receives 429/Retry-After and does not extend the blocked window. Credential validity, record visibility, expected version and timing are still checked on admitted requests. This is independent of staff-login and public-create/upload budgets. Test mixed-endpoint concurrent attempts and shared-network behavior; no response discloses whether a private visit exists. Public submission recovery remains a separate operation so exhausting this budget cannot silently trigger a new booking.

```ts
type Role = "PA" | "Protocol Officer" | "Receptionist" | "Super Admin";
type Status = "pending" | "approved" | "rejected" | "cancelled" |
              "checked_in" | "checked_out";
type DisplayStatus = Status | "missed";
type Notes = { visitorInstructions?: string; coordinationNote?: string };
type Session = {
  active: true; id: string; name: string;
  role: Role | ""; availableRoles: Role[];
  roleSelectionRequired: boolean; roleRevision: number;
  permissions: string[]; assignedOfficerIds: string[];
  mustChangePassword: boolean; passwordMinLength: number; passwordMaxBytes: number;
  timing: { serverTime: number; idleExpiresAt: string; absoluteExpiresAt: string };
};
type Room = { id: string; name: string; active: boolean };
type Catalogs = {
  officers: { id: string; name: string; vip: boolean; active: boolean }[];
  rooms: Room[];
  attachmentsEnabled?: boolean; // draft capability; omitted means unavailable for new uploads
  privacyNotice?: { version: string; id: string; en: string };
};
type Schedule = {
  id: string; visitKind: "standard" | "restricted";
  date: string; time: string; duration: number;
  roomId?: string; room: string; // room is the display label, "" if unassigned
  status: Status; version: number; rescheduled?: boolean;
  timing: {
    serverTime: number; opensAt: number; endsAt: number;
    displayStatus: DisplayStatus;
    canCheckIn: boolean; canCheckOut: boolean;
    canCancel: boolean;         // PUBLIC cancellation eligibility, not staff authority
  };
};
type OrdinaryDetails = {
  code: string; name: string; organization: string; email: string; phone: string;
  officerId: string; host: string; vip: boolean; purpose: string;
  visitors: string[]; identityType: "KTP" | "Paspor" | "SIM";
};
type PublicEvent = { at: string; action: string; publicMessage?: string };
type PublicBooking = Schedule & OrdinaryDetails & {
  visitKind: "standard"; history: PublicEvent[];
  ndName?: string; visitorInstructions?: string;
  rejectionReason?: string; cancellationReason?: string;
};
type StandardStaffBooking = Schedule & OrdinaryDetails & {
  visitKind: "standard";
  ndName?: string; attachmentAvailable?: boolean;
  visitorInstructions?: string; coordinationNote?: string;
  rejectionReason?: string; internalRejectionNote?: string;
  cancellationReason?: string; internalCancellationNote?: string;
  createdBy?: { name: string; role: Role };
  history: { at: string; message: string }[];
};
```

Every successful create, detail, lookup and mutation returns the complete allowed audience shape, not just ID/status. Server computes VIP/host/room/timing. A client-provided VIP flag, name label or timing flag grants no authority. Private note fields in standard staff results appear only for actors allowed to read them; reception never receives coordination/private decision notes.

`StaffBooking` is an explicit union of StandardStaffBooking and four restricted DTOs, not one permissive bag of optional fields. Each restricted DTO has Schedule + `reference` + literal `restrictedView`; identity values below are optional because the visit may have no identity/target:

| Restricted view | Allowed additional data | Forbidden examples |
| --- | --- | --- |
| `full` | Protocol/Super Admin authorized name, organization, serviceId, officerId/host, code, history; private notes only under their separate read policy | Public exposure; implied admin decision rights |
| `assigned-pa` | name, organization, officerId/host, serviceIdLast4 | Full ID, code, private notes, history, mutation rights |
| `reception` | No optional identity/target fields | name, organization, officerId/host, ID, code, notes/history |
| `arrival` | Supplied name, officerId/host, serviceIdLast4 after valid credential resolution | organization, full ID, internal history/notes |

Restricted visits never return public contact/roster/document fields or enter general reports. Full-view notes require field-level policy; the private coordination note is not made visible merely by a full restricted-read projection. Scope is applied before search/count/aggregation and before error serialization. Do not include a conflicting booking's private data in a schedule-conflict error.

Public history uses allowlisted action codes, timestamps and explicitly public reschedule text; no raw audit messages, employee names/roles or internal notes. Unknown future action codes have a safe localized fallback. Rejection/cancellation reasons remain in their distinct fields.

## 3. Operation inventory

All paths in this table exclude `/v1`. This proposal covers **38 business HTTP operations**: 36 existing operations plus explicit activity renewal (D20) and token refresh (D39). Separate liveness/readiness probes are operational endpoints and not frontend features. The room and file-view contracts change payloads/presentation, not the route count. D39 adds a separate refresh operation; authentication payloads and delivery must reflect D38–D65; remaining contract bounds and deployment checks still apply.

| # | Method/path | Request | Success | Owner / authority |
| --- | --- | --- | --- | --- |
| 1 | POST `/staff/sessions` | `{email,password}` | 200 access-token result + Session; set refresh cookie; access JWT lifetime per D41; rotating persistent refresh per D42/D43/D46/D51 | Identity, anonymous rate-limited login |
| 2 | GET `/staff/session` | Bearer JWT | 200 Session | Identity; also available during role/password gate |
| 3 | DELETE `/staff/session` | Refresh-cookie lifecycle operation; signed double-submit CSRF + strict origin checks (D45) | 204; revoke session and clear refresh cookie | Identity; already-expired session safe to sign out |
| 4 | POST `/staff/password` | Bearer JWT; `{currentPassword,newPassword}` | 200 access-token result + Session; replacement refresh cookie | Identity; required-change allowed; revoke other sessions, replace current credentials (D52) |
| 5 | POST `/staff/session/role` | Bearer JWT; `{role:Role}` | 200 access-token result + Session | Identity; current-session assigned role, revise authority (D54) |
| 6 | GET `/public/catalogs` | None | 200 Catalogs | Public safe active destinations/notice; no staff assignments |
| 7 | GET `/staff/catalogs` | Bearer JWT | 200 Catalogs | Directory; assigned inactive metadata as permitted |
| 8 | POST `/public/bookings` | PublicCreate | 201 PublicBooking; same-key replay 200 | Visits; anonymous public-create policy |
| 9 | POST `/public/submissions/resolve` | `{idempotencyKey,email}` | 200 committed booking OR not_committed | Visits; serialize with create and close missing reference |
| 10 | POST `/public/lookup` | `{code,email}` | 200 PublicBooking | Visits; standard-only pair authorization |
| 11 | POST `/public/bookings/{id}/cancel` | `{code,email,version}` | 200 PublicBooking | Visits; public before-start cancellation |
| 12 | POST `/public/bookings/{id}/admission` | `{code,email,version}` | 200 `{payload,expiresAt,serverTime}` | Admission; eligible standard Approved visit |
| 13 | POST `/public/attachments` | Multipart `{file,reference}` | 201 `{token,expiresAt,serverTime}`; safe replay 200 | Documents; public upload policy/rate limits |
| 14 | POST `/staff/attachments` | Multipart `{file,reference}` | Same | Documents; staff creation authority |
| 15 | GET `/staff/bookings/{id}/attachment` | Bearer JWT; optional `disposition=inline|attachment` (default inline) | 200 protected bytes for web view or explicit download | Documents + visit/field scope; audit access |
| 16 | GET `/staff/bookings` | Typed view query below | 200 ReadModel | Visits/reception, permission + role/assignment/projection |
| 17 | GET `/staff/bookings/{id}` | Bearer JWT | 200 StaffBooking | Visits; independent of current list page |
| 18 | POST `/staff/bookings` | StandardStaffCreate OR RestrictedCreate | 201 StaffBooking; replay 200 | Visits; create + PA/Protocol scope; reserve immediately |
| 19 | POST `/staff/bookings/{id}/approve` | Approve | 200 StaffBooking | Visits; approve + decision scope |
| 20 | POST `/staff/bookings/{id}/reject` | Reject | 200 StaffBooking | Visits; reject + decision scope |
| 21 | POST `/staff/bookings/{id}/reschedule` | Reschedule | 200 StaffBooking | Visits; reschedule + decision scope |
| 22 | POST `/staff/bookings/{id}/update` | RestrictedUpdate | 200 restricted full DTO | Visits; Protocol + create, valid restricted edit state |
| 23 | POST `/staff/bookings/{id}/cancel` | StaffCancel | 200 StaffBooking | Visits; cancel + decision scope, before any check-in |
| 24 | POST `/staff/bookings/{id}/checkin` | `{credential,version}` | 200 authorized attendance projection | Reception; checkin + eligible role/scope/time |
| 25 | POST `/staff/bookings/{id}/checkout` | `{credential,version}` | 200 authorized attendance projection | Reception; checkout + checked-in record |
| 26 | POST `/staff/resolve` | `{credential}` | 200 authorized arrival projection | Reception; no attendance side effect |
| 27 | GET `/staff/employees` | `{q,page,pageSize}` query | 200 EmployeePage | Administration; people |
| 28 | GET `/staff/employees/{id}` | Bearer JWT | 200 Employee | Administration; people; explicit conflict reload |
| 29 | POST `/staff/employees` | EmployeeCreate | 201 Employee | Administration; people + lockout/scope policy |
| 30 | PATCH `/staff/employees/{id}` | EmployeePatch | 200 Employee | Administration; people + version + anti-escalation |
| 31 | GET `/staff/roles` | Bearer JWT | 200 RoleDefinition[] | Administration; roles |
| 32 | PATCH `/staff/roles/{id}` | `{permissions,version}` | 200 RoleDefinition | Administration; roles + assignability/lockout policy |
| 33 | GET `/staff/integrations` | Bearer JWT | 200 Integration[] | Administration; integrations; no secret values |
| 34 | PATCH `/staff/integrations/{id}` | `{enabled,values,clearSecrets,version}` | 200 Integration | Administration; integrations + safe provider config |
| 35 | GET `/staff/reports/summary` | `{from,to,status}` query | 200 Summary | Reporting; reports + reporting scope |
| 36 | GET `/staff/reports/export` | `{from,to,status,revision}` query | 200 `{filename,csv,revision}` | Reporting; reports + reportExport; recheck revision |
| 37 | POST `/staff/session/renew` | Bearer JWT + empty JSON `{}`; actual activity only | 200 Session with authoritative timing | Identity; nonexpired session, current account, fixed absolute deadline |
| 38 | POST `/staff/session/refresh` | Refresh cookie; `{refreshRequestId: UUID}`; signed double-submit CSRF header/cookie + strict origin checks (D45); initialized through staff HTML (D47) | 200 access-token result + Session; rotate refresh cookie (D42); same-request recovery per D43 | Identity; current nonexpired session/account; does not extend idle/absolute deadlines |

Session gate: a pending multi-role Session has `role:""`, `roleSelectionRequired:true`, empty permissions/assignments and nonempty availableRoles. Never choose array order as authority. One-role employees enter that role directly. While a password change is required, ordinary workspace operations remain blocked. Explicit role-switch success confirms the selected role and increments authority revision; client cancels/clears old private caches.

Session expiry (D40, superseding D19 values): the server enforces 8 hours of inactivity and an absolute 24 hours from login. Effective expiry is the earlier boundary. Background list/reception/report/session polling must not renew inactivity. D24 shows the inactivity warning 2 minutes before the authoritative deadline, with keyboard-accessible Continue session and Sign out controls. Do not announce every countdown tick. If the absolute boundary arrives first, clearly explain that signing in again is required and never offer an extension beyond it. Role switches or activity renewals never reset the absolute login timestamp. Session timing returns serverTime as epoch milliseconds and expiry fields as RFC3339 instants; the UI accounts conservatively for elapsed request time.

Activity renewal (D20): actual typing/click/keyboard input sets an activity signal in the visible staff app; coalesce signals into automatic POST /staff/session/renew calls at most once per 60 seconds per session across tabs (D25). The warning's Continue session button uses the same operation. Never send periodic renewals without new user activity, renew on background polling, or send a backlog of old events after reconnect. Coordinate multiple tabs so they do not multiply renewal traffic; recalculate deadlines from successful responses. The server validates current session/account and both expiry boundaries before updating idleExpiresAt to min(serverNow + 8 hours, absoluteExpiresAt). Expired sessions return 401 and cannot be revived. Renewal coalescing is settled by D25; the staff-wide ceiling follows D34/D35, with any additional lifecycle-specific limit still pending. A client activity signal is not trusted proof that a human is present. Renewal is allowed through role-selection/password-change gates but grants no workspace authority. Timing-only renewal must not change roleRevision, clear form drafts or invalidate the whole private query cache; actual authority changes still do. There is no extra GET required after successful renewal.

## 4. Creation, upload and action payloads

```ts
type PublicCreate = {
  visitKind: "standard"; idempotencyKey: string;
  officerId: string; date: string; time: string;
  name: string; organization: string; email: string; phone: string;
  purpose: string; identityType: "KTP" | "Paspor" | "SIM";
  visitors: string[];              // representative first, 1–20 total
  ndName?: string; attachmentToken?: string;
  privacyVersion?: string; ack: true; // D90/D91: omitted defaults; supplied must be current
};
type StandardStaffCreate = Omit<PublicCreate, "privacyVersion" | "ack"> & Notes;
type RestrictedCreate = Notes & {
  visitKind: "restricted"; idempotencyKey: string;
  date: string; time: string;
  name?: string; organization?: string; serviceId?: string; officerId?: string;
};
type Approve = Notes & { version: number; roomId?: string; duration?: number };
type Reject = {
  version: number; rejectionReason: string; internalRejectionNote?: string;
};
type Reschedule = Notes & {
  version: number; date: string; time: string;
  duration: number; roomId?: string; reason?: string;
};
type RestrictedUpdate = Notes & {
  version: number; visitKind: "restricted"; date: string; time: string;
  name?: string; organization?: string;
  serviceId?: string; officerId?: string;
};
type StaffCancel = {
  version: number; reason: string; internalCancellationNote?: string;
};
```

Validation/semantics:

- Ordinary creation requires active known official, real future WITA date/time, ordinary start 08:00–16:00, representative name 2–120 characters, trimmed organization, valid normalized email, 8–20 digit phone preserved as text, purpose 5–1000 characters and complete named roster. Representative equals visitors[0]. Public ignores/rejects private/staff-only fields. No identity number/image field.
- Restricted creation requires date/time and Protocol create authority. Name/organization/service ID/official stay optional; supplied values must be valid. Preserve service ID leading zeroes. Do not require ordinary contact/purpose/roster/upload/consent fields for this variant. Current input time range is 08:00–16:00; a broader restricted schedule policy would require a coordinated choice.
- Public creates Pending and reserves nothing. Staff standard/restricted creates Approved, reserves assigned resources immediately, and records create/auto-approve events without redundant approval UI.
- Creation forms do not collect duration/room. Service assigns 60 minutes to new public and staff visits (D13), returning an empty room label when unassigned. Staff can set 15–240 minutes during approval/rescheduling, subject to conflict checks. Do not add mandatory creation inputs for duration or room.
- Standard public approval requires valid duration/room selection; service rejects missing required selection with 422. Restricted approval may omit them. Staff-created visits may be approved without a room as previously agreed. Optional `roomId` is omitted when unassigned; for rescheduling omission clears room assignment (the full replacement schedule), while omission on approval preserves any existing room. All populated IDs require valid catalog/scope checks.
- Notes maximum 2000 characters. On approve/reschedule/restricted update, omitted notes preserve values and `""` clears. No standalone notes endpoint. Restricted update omitted identity/target fields preserve values; explicit empty strings clear optional identity/target values, even though stored/response absence is omitted. Update target changes scope and recomputes resource conflicts.
- Rejection and staff cancellation require trimmed 5–1000 character public reason; optional private note maximum 2000. Do not send private text publicly or in visitor emails. Public cancellation has no new mandatory reason field.
- Rescheduling applies immediately and invalidates old admission; no visitor accept/reject-proposal workflow. Current `validateReschedule` requires an ordinary room and reason of at least 5 trimmed characters; restricted room/reason may be absent, but a supplied reason must also have at least 5 characters. Retain that distinction; D55 sets the shared reason maximum to 1,000 code points, represented in OpenAPI. D100 settles source-state eligibility separately.
- Upload token and filename must describe one verified linked upload; filename alone never grants attachment availability. Supporting file is optional, nonempty PDF/JPEG/PNG, at most 5 * 1024 * 1024 bytes. Validate signature/content, safe filename, ownership and malware policy; deny unverified content. Same reference/same content replays safely; changed content conflicts. Upload/link/cleanup must coordinate atomically. No public file URLs or unscoped object IDs.
- Upload response token is 16–4096 characters and valid for 1 hour from issuance (D18), without sliding extension. Admission payload is 1–4096 characters, valid until min(issuance + 5 minutes, check-in window end) under D21; do not issue an already-expired credential. Single-use check-in, replacement/revocation and server-time eligibility still apply. `expiresAt` is a valid future instant relative to serverTime; client elapsed-time checks remain conservative. D17 sets unlinked development upload cleanup eligibility to 24 hours after successful upload. New linkage must occur before that deadline and before token expiry, with atomic coordination against cleanup. Expired-token validation returns 422 with `code:upload_expired` and `fieldErrors.ndName`; retain form input and reupload the selected file before a new submission. Resolve uncertain submissions first: safe replay/recovery of a previously committed submission does not re-consume an expired token. Already-linked files are excluded from abandoned-upload cleanup. Successful submission-reference retention follows D22; closure-marker/purge policy remains a separate production-retention gate.
- File saving is a draft feature (D14), using a private local-storage adapter for development (D15); production enablement remains deferred. Expose an explicit optional `attachmentsEnabled:boolean` in public/staff catalogs during coordinated draft integration; omitted/false means new uploads are unavailable. Update the frontend adapter/schema and controls together. If disabled, omit upload requirements, disable/hide upload controls with a readable explanation, and reject actual upload attempts with a safe `feature_unavailable` response; never claim the file was saved. Existing linked-file reads remain governed by their own availability and authority. Registration continues without the optional attachment.
- Authorized file access defaults to `Content-Disposition: inline`; explicit Download uses the same route with `disposition=attachment`. Send verified PDF/JPEG/PNG MIME, safe filename, no-store and nosniff in both cases. Do not expose object paths/tokens in the URL. Query selects presentation only and never changes authorization. Inline disposition asks the browser to display supported content; native viewer support/user settings may require a clear fallback. An embedded viewer needs a reviewed document-content and framing policy before real files are enabled. Current frontend download-only behavior must change to offer View plus Download. No heavy custom PDF library is selected.
- Public resolution returns `{outcome:"committed",booking:PublicBooking}` OR `{outcome:"not_committed"}`. No outcome ambiguity is treated as a success or permission to generate a new reference.
- Successful submission reference/outcome linkage is retained for the associated visit's retention lifetime (D22). Same-reference/same-input retry returns that booking under current authorization/projection, without another booking, audit creation or logical notification; changed input returns 409. Reference alone is not a read credential. Closed not-committed references remain closed to delayed creates; do not equate abandoned-upload cleanup with request-reference deletion. Future visit purging/closure-marker retention needs a separately reviewed policy that preserves duplicate prevention.

## 5. Lifecycle and authorization

| Operation | Allowed source state | Result/guard |
| --- | --- | --- |
| Public create | None | Pending; no reservation |
| Staff create | None | Approved; conflict-checked reservation |
| Approve | Pending | Approved; resource locks/version/decision authority |
| Reject | Pending | Rejected; required public reason |
| Public cancel | Pending/Approved | Cancelled strictly before start, no prior check-in |
| Staff cancel | Pending/Approved | Cancelled with explicit cancel + decision scope; may be after start/end, no prior check-in |
| Reschedule | Pending/Approved, excluding derived Missed (D100) | Preserve lifecycle state; new future schedule; reserve only when committed state requires it |
| Restricted identity/target edit | Pending/Approved | Preserve state; Protocol create authority, recompute scope and active commitments |
| Check-in | Approved | Checked-in, server time in [start−30m,end], credential current/eligible |
| Check-out | Checked-in | Checked-out; may be after end and on later dates |
| Missed display | Approved and server time > end | Derived display only, never a seventh stored status or job-driven transition |

Reschedule policy is selected by D100: Pending or Approved, excluding derived Missed. Current frontend domain.can() permits only Approved/non-Missed; add Pending deliberately during authorized implementation. Restricted Edit detail independently remains Pending/Approved with its existing future-start validation. No action is implicitly enabled solely because a DTO contains the relevant fields.

PA decides ordinary non-VIP visits for currently assigned officials; reads only assigned-pa projection for eligible restricted visits. Protocol decides ordinary VIP and manages restricted visits. Reception reads only approved/checked-in/checked-out allowed projections and performs permitted attendance actions. Super Admin administration/full restricted reads do not grant ordinary decision/document access. Permission keys remain `bookings,create,approve,reject,reschedule,cancel,checkin,checkout,reports,reportExport,people,roles,integrations,audit`; `audit` has no existing standalone API consumer. Do not invent an `update` permission for the current Protocol restricted editor.

Booking code alone is not public authorization; email/code pair is. Reception manual code resolution is separately authenticated and rate limited. Admission QR is opaque, server-issued, replaceable/revocable and one-time for admission. D96 selects a distinct checkout path: a QR consumed by successful check-in may resolve/check out that same currently checked-in visit even after its five-minute admission expiry. Admission consumption is not revocation. A revoked/replaced credential remains unusable; no repeat check-in, different-visit checkout or duplicate checkout is permitted. Revalidate current reception authority, matching visit, expected version and checked-in state in the attendance transaction. Manual eight-digit code remains an alternative. Explicit receptionist identity confirmation remains a UI step; current request body adds no invented `identityVerified` field.

## 6. Shared typed reads

```ts
type ReadModel = {
  items: StaffBooking[]; total: number; totalScoped: number;
  page: number; pageSize: number;
  metrics?: { today: number; pending: number; arrivals: number;
              inside: number; completed: number };
};
```

| view | Query | Required result |
| --- | --- | --- |
| list | q,status,sort,dir,date?,vip,page,pageSize (default 7) | Scoped searchable/filterable page, total, totalScoped |
| reception | mode=checkin/checkout,date? for check-in,page,pageSize (default 25) | Requested-day arrivals OR all still-inside records including previous dates |
| dashboard | date,page=1,pageSize (default 50) | Metrics required; deduplicated bounded sample covering agenda and attention |

Maximum 100 items; positive one-based pages and bounded pageSize. Echo requested valid page/pageSize, including an empty beyond-end page. `total` is filtered count before paging; `totalScoped` before list filters. Get sample/counts/metrics from one consistent read snapshot and authority context, using a single statement or short repeatable-read transaction without scheduling writes.

Allowed sort: priority/name/organization/host/time/status with asc/desc, stable ID ties. Scope/projection precedes search, sort, counts and page. Search only permitted representative/roster/organization/host/code/reference fields. Blank/oversized queries are normalized/rejected under documented limits, not interpolated into SQL. Preserve current `vip=1` ordinary-VIP-only and `vip=0` no-VIP-filter semantics; do not reinterpret zero as non-VIP-only. `status=approved` excludes derived Missed; `missed` selects derived Missed. All six stored statuses plus all/missed are accepted.

Dashboard: today/arrivals/inside/completed are over requested WITA date; pending is all scoped pending across dates. Operational agenda supplies first five requested-date schedule rows; attention supplies first four pending in priority order. Reception agenda is approved not-missed arrivals, attention is inside for that date. Guarantee both subsets in the bounded union; selecting the first 50 agenda records alone is insufficient. Super Admin configuration dashboard needs no operational query. Metrics count the full authorized scope rather than returned sample.

Detail ID must match requested ID; view page metadata/filter must correlate with request. Mutation results need the same authorized projection as a direct read, except credential-resolved restricted arrival information never persists into ordinary reception cache.

## 7. Administration and reporting shapes

```ts
type Employee = {
  id: string; name: string; email: string; roles: Role[];
  active: boolean; assignedOfficerIds: string[]; version: number;
  jobTitle?: string; deputy?: string; directorate?: string; phone?: string; nip?: string;
};
type EmployeePage = {
  items: Employee[]; total: number; totalAll: number;
  page: number; pageSize: number; passwordMinLength: number; passwordMaxBytes: number;
};
type EmployeeCreate = Omit<Employee,"id"|"version"> & {
  initialPassword: string; requirePasswordChange: boolean;
};
type EmployeePatch = Omit<Employee,"id">; // complete current editor payload + version
type RoleDefinition = {
  id: string; name: Role; description: string; permissions: string[];
  assignablePermissions: string[]; editable: boolean; version: number;
};
type Integration = {
  id: string; name: string; description: string; enabled: boolean;
  health: "not_configured" | "ready" | "degraded"; version: number;
  fields: {
    key: string; label: string;
    kind: "text" | "secret" | "url" | "email" | "number";
    required: boolean; value?: string; configured?: boolean;
  }[];
};
type SummaryFilter = { from: string; to: string; status: "all" | DisplayStatus };
type Summary = {
  filter: SummaryFilter; scope: "organization" | "assigned" | "authorized" | "attendance";
  generatedAt: string; canExport: boolean; revision: string;
  counts: Record<DisplayStatus,number>; total: number;
};
```

Employee roles is canonical; singular role is not needed by the new plan. Current frontend accepts roles but its outgoing types/editor logic must be checked for multi-role assignments end to end. Keep currently inactive assignments until deliberately removed. NIP/phone remain strings. Employee directory defaults pageSize 25, maximum 250; include password metadata on empty results. No stored password/hash or initialPassword in a response. Ordinary edits never reset passwords. Deactivation is versioned PATCH; no hard-delete feature is added. Bootstrap/migration commands are operational tooling, not public account-creation endpoints.

Employee assignment bound (D94): assignedOfficerIds accepts 0–100 unique decimal business IDs. Reject excess, duplicates and invalid/unauthorized references with safe assignedOfficerIds field errors; do not truncate, silently deduplicate or partially apply a replacement. An empty array clears assignments under existing role/editor rules. Preserve inactive references already retained by the editor unless deliberately removed; retaining one does not grant runtime authority. D84 retains deactivated employee/official pair rows and transactional audit. Align the future frontend selection limit, visible count and accessible error with this contract. Historical inactive pair rows are not counted as submitted current assignments.

Role descriptions are existing display DTO fields; do not reintroduce duplicated descriptions on permission-area database records. Server defines assignable/protected permissions and administrative lockout invariants. Role/assignment/account changes must affect current sessions, not just later logins.

Integration fields max 30, key `^[a-zA-Z][a-zA-Z0-9_]*$`; non-secret values are strings including number fields. Secret fields return configured only, never value. Writes omit/blank existing secrets to keep them, explicit clearSecrets deletes. Validate required settings when enabled; health is service-derived, not a user claim or a live probe on every GET. Protect configurable destinations against unauthorized network access. No extra provider-test endpoint is needed for a UI feature that does not exist.

Initial provider (D98): authenticated SMTP over verified TLS. Concrete planning schema: host (ASCII DNS/punycode name, 1–253 bytes, no URL/userinfo/path/port embedded), port (canonical decimal string 1–65535), tlsMode (implicit_tls or starttls), username (trimmed 1–255 code points), password (exact secret <=4096 UTF-8 bytes) and sender (D82 mailbox). All effective values are required to enable. Disabled configuration may be incomplete; blank/omitted configured password preserves it, clearSecrets may contain only password. PATCH values accepts only these six keys and validates the merged effective configuration transactionally. No cleartext mode, certificate bypass or STARTTLS downgrade; authenticate only after verified TLS. Port is explicit, not silently derived; conventional submission ports 465/implicit TLS and 587/STARTTLS are guidance, not a ban on another server-approved TLS endpoint. Outbound host/port allowlist and resolved-address policy are deployment-controlled, not editable around through this form. A local test sink is isolated test configuration, not a production bypass. No arbitrary endpoint URL/bucket/credential fields from preview examples are accepted for SMTP. Storage-provider settings stay deferred; the development private-local adapter uses reviewed service configuration, not invented provider fields. A deferred storage metadata record must stay disabled and cannot acquire writable arbitrary keys through the generic integration route. See [RFC 8314](https://www.rfc-editor.org/rfc/rfc8314) for SMTP TLS transport guidance.

Summary date bounds inclusive, from<=to, at most 365 elapsed days/366 dates. Return all seven nonnegative counts, whose sum equals total; specific-status filter zeros other counts. Exclude restricted records/events before aggregate. Summary/export permissions are independent of ordinary detail visibility. Scope is explicit; failure never becomes zero.

Revision is 64 lowercase hexadecimal characters binding canonical filter, authorized scope/authority and returned aggregate content. Do not hash generatedAt as a constantly changing revision. Export recomputes the same snapshot semantics; changed results/authority yield 409. Filename follows `kunjungan-ringkasan-[date-range].csv`; response remains JSON `{filename,csv,revision}` for existing frontend compatibility. CSV contains summary rows only, no visitor details. Both `/analytics` and `/logs` reuse these operations.

## 8. Coverage and non-redundancy map

| Requirements/source | Workflow | Operations | Owner/invariant | Verification status |
| --- | --- | --- | --- | --- |
| BR-AKS-01–08, later role addendum/current session code | Login, required password change, selection/switch, revocation, activity renewal | 1–5, 27–32,37–38 | Identity/admin; one active role, current scope, idle/absolute expiry | Planned: real session/role-race/expiry/denial tests |
| BR-AJU-01–09 | Catalog, public/staff submission, roster/consent, recovery | 6–9, 13–14,18 | Visits/documents; no duplicate booking/logical notification | Planned: DB/restart/replay/validation checks |
| BR-TRB-01–06 | Restricted create/read/edit/arrival | 16–18,21–26 | Policies/projection; field privacy before search/count | Planned: direct API leak matrix |
| BR-ALR-01–07 | Decisions/reservations/cancellation/Missed | 11,18–25 | Visits; versioned atomic transitions/resource locks | Planned: real concurrent writers/time boundaries |
| BR-PNG-01–03 | Visitor status/history/QR | 10–12 | Public projection, email+code, eligibility | Planned: wrong-pair/restricted/history tests |
| BR-RSP-01–05 | Manual/QR resolve, identity confirmation, attendance | 16–17,24–26 | Reception; no duplicate attendance, late checkout | Planned: simultaneous scan/replay tests |
| BR-KOM-01–03 | Notification enqueue/delivery/recovery | No extra frontend endpoint; command transactions + worker | Outbox; saved intent, safe recipients/content | Planned: commit/failure/restart/provider tests |
| BR-DAT-01–07 | Audit/history/scope/retention/reporting/refresh | 15–17,35–36 + internal audit/cleanup | Audit/reporting; no restricted general export, no invented retention | Planned; organizational retention unresolved |
| BR-UX-01–06 | Responsive accessible states, WITA, stable navigation | Error/success contracts across all | Existing UI design system; no lost input/false empty data | Planned: real browser + manual accessibility |
| BR-INT-01 | Versioned write-only integration settings | 33–34 | Administration; secret non-disclosure | Planned: safe validation/conflict/provider tests |

No additional endpoint per dashboard card, report tab, note, permission checkbox or database table. Keep separate public/staff audience boundaries and distinct action contracts because they have distinct authorization/side effects. Shared repositories/queries/transactions are internal reuse, not a reason to expose generic unrestricted CRUD.

## 9. Compatibility and freeze gates

Approved breaking delta: rooms change from string catalog entries/name writes to Room objects/roomId writes; retain read label `room`. Selected file-view delta: attachment reads default to inline web viewing in a preview dialog, with explicit disposition=attachment for Download and clear Close/X controls; outside clicks never dismiss the dialog and Escape closes it (D16). File saving remains draft with private local development storage. Additional contract tightening: explicit visitKind, restricted union/allowlists, canonical employee roles and required structured errors. Update runtime schemas, form state, adapters and fixtures together; do not silently reinterpret legacy names as IDs. Inactive renamed rooms need a safe current-selection display. Draft capability handling is documented above; production provider/scanning remain release choices.

Before a final machine-readable OpenAPI freeze: verify remaining diagram fields and transition rules against the [recovered original decisions](BACKEND-ALIGNMENT-DECISIONS-RECOVERED-2026-11-11.md); complete the field-bound register and verify the selected version/dependency compatibility. Sensitive-operation limits, CSRF/gateway strategy and request size/deadline policies follow D38–D74; actual domain/certificate/secret configuration remains a deployment prerequisite. Production scanning/provider enablement and retention/purge rules remain separate gates. Creation duration (60 minutes), local draft storage/viewer, upload token (1 hour), abandoned-upload cleanup (24 hours), session limits/renewal, QR validity (5 minutes), committed-reference retention and email retry window (up to 24 hours from first failure) are selected in D13–D23. The initial target is a fresh isolated disposable database (D12); existing-data migration is separate later work. These are review gates, not authorization to invent or implement defaults. The contract describes required frontend coverage but does not claim deployed compatibility or test passes.

## 10. Authentication lifecycle contract

D38 file-view integration: fetch protected bytes with the Bearer header, then display a temporary browser object URL in the selected preview dialog. Use the same authorized fetch for explicit Download, release object URLs when no longer needed, and never put JWTs in preview/download URLs. Session lifecycle changes must preserve focus, drafts and the selected idle-warning behavior.

Token refresh (D39): use the dedicated refresh operation after reload or when a new access token is needed, subject to the selected refresh lifecycle. Refresh never records user activity or changes idleExpiresAt/absoluteExpiresAt. Coordinate refresh requests across tabs; rotation is mandatory (D42); same-request recovery follows D43. If a token expires during genuine activity, refresh first, then renew activity using the new Bearer token only while the server session remains nonexpired. No expired session can be revived. Protected resource operations never accept the refresh cookie as fallback authorization.

Access JWT lifetime (D41): exp = min(issuedAt + 5 minutes, current idleExpiresAt, absoluteExpiresAt). Login and refresh results return accessToken, tokenType: "Bearer", accessExpiresAt (RFC3339), and session: Session. Responses use Cache-Control: no-store. Refresh may replace an expired access JWT only while the server session is valid; it cannot revive an expired session or extend inactivity. Logout/revocation/current-authority changes take effect through the server checks regardless of remaining JWT lifetime. Never interpret a network failure as evidence that a business mutation did not commit.

Refresh rotation (D42): serialize refresh transitions for the session in PostgreSQL so concurrent API instances cannot independently consume one credential. Successful refresh atomically consumes the presented credential and issues its successor with a replacement HttpOnly cookie and a new access-token result. Retain sufficient lineage for the selected reuse policy; D43 settles retry grace/recovery and consumed-credential reuse; D46 selects opaque random refresh credentials. Coordinate same-session tabs to avoid routine competing refreshes. Rotation never moves idleExpiresAt or absoluteExpiresAt. Verify races, lost responses, logout/revocation during refresh, expiry boundaries and stale-role authority before acceptance.

Refresh recovery (D43): generate refreshRequestId once per logical refresh and retain it for immediate retries; a reference alone grants no authority. Atomically bind the reference, consumed credential and successor to the committed transition. Within 10 seconds of that commit, an authenticated repeat of that transition can return the same successor while it is still current, with current session/authority checks and no new rotation or deadline extension. Account for either predecessor or already-installed successor cookie on a retry of the same transition. Keep the successor recoverable only in a short-lived encrypted record; enforce expiry at read time even if cleanup runs later. Never roll a cookie back to a superseded successor. Other confirmed reuse of a consumed credential revokes its session and invalidates its access JWTs through server checks; clear local auth and require sign-in. Random invalid credentials or request references alone must not revoke a session. A stale in-flight response must not overwrite newer client authority. Recovery timing, multi-tab races and response ordering require dedicated implementation verification.

Future authentication integration (D44): this version retains local login and adds no SSO endpoints or UI. Identity verification is separate from shared VMS session issuance and current-authority checks so a later provider adapter can reuse the browser session contract. No provider protocol, automatic account linking or global SSO logout behavior is implied by this draft.

CSRF contract (D45): login, refresh and logout require a signed CSRF cookie with a matching X-CSRF-Token header and strict configured origin validation. Authenticated token signatures bind to the stable VMS session; rotation of the refresh credential does not rotate that binding. The CSRF cookie is Secure and host-only but deliberately readable by the app; the refresh cookie remains HttpOnly. Tokens and credentials never appear in URLs/logs. Login requires a separate pre-authentication binding and replaces it after successful authentication; D47 initializes this through dynamic staff HTML delivery, not an additional API endpoint. HTML delivery follows D48; anonymous pre-login lifetime is fixed at 30 minutes (D50). Missing/invalid CSRF yields 403 without executing the action. Protected staff resource routes remain Bearer-only.

Refresh credential representation (D46): the HttpOnly refresh cookie contains an opaque cryptographically random secret, never an access JWT. Normal validation and consumed-credential lineage use hashes in the existing PostgreSQL support schema. Only the D43 recovery mechanism may retain the successor in encrypted recoverable form for its fixed 10-second window; enforce expiration even before physical cleanup. No additional API service or database is introduced by this choice.

CSRF initialization delivery (D47): the 38-operation API inventory gains no bootstrap endpoint. Initial staff HTML and deep-link reload responses establish anonymous protection or restore valid session-bound protection without issuing access tokens, rotating refresh credentials or extending inactivity. Personalized HTML responses use Cache-Control: private, no-store and bypass CDN/service-worker document caches; static assets remain separately cacheable. Successful login replaces anonymous protection. Missing CSRF during SPA use uses a controlled full page reload; it never disables the lifecycle CSRF check. D48 selects gateway delivery with an internal Go initialization call.

Cookie policy (D49): staff refresh, CSRF and anonymous-binding cookies use SameSite=Lax, Secure, host-only scope and explicit paths. Refresh and anonymous binding are HttpOnly; the separate CSRF cookie is readable. Keep signed CSRF/header matching and strict configured origin checks for login/refresh/logout. Permit ordinary external top-level GET navigation to staff HTML without treating it as a mutating API call; reject cross-site embedded initialization. Gateway/Go checks distinguish these cases using trusted request metadata. No SSO callback is introduced.

Anonymous CSRF lifetime (D50): 30 minutes from issuance, no sliding renewal on reads or failed login. Expired login protection returns 403 csrf_expired before password verification; it is not an incorrect-password failure. The frontend offers document reload through D47/D48, may preserve email, clears password and requires re-entry. Never automatically replay credentials. Valid anonymous protection is reused across tabs/reloads until expiry, then replaced. Authenticated session limits remain 8 hours inactive / 24 hours maximum.

Browser persistence (D51): refresh cookie is persistent, capped at the fixed 24-hour absolute login deadline. PostgreSQL still rejects sessions after 8 hours inactive or revocation. Reopening the browser restores CSRF via document delivery and obtains an access JWT via refresh only while valid; neither step extends inactivity. Access JWTs remain memory-only. Browser-cookie presence never proves an authorized session.

Password-change session behavior (D52): successful POST /staff/password returns the same access-token result shape used by login/refresh and replaces the current refresh cookie. Revoke all other sessions, invalidate the current session's old access/refresh credentials and recovery records, and clear mustChangePassword in the same transaction as password/audit updates. Preserve its original login/absolute expiry. Server credential-generation checks reject old JWTs even before exp. Serialize with refresh/logout and prevent stale tab responses from restoring previous authority. A lost response is an uncertain password-change outcome: never automatically replay credentials or assume rollback; guide reauthentication and preserve non-secret drafts safely.

Concurrent sessions (D53): POST /staff/sessions creates an independent browser/device session without revoking other valid sessions. Session expiry/activity and refresh rotation/recovery are scoped to that session; employee request budgets remain shared. Tabs sharing its refresh cookie share session state. D52 password change revokes other sessions; deactivation and current-authority changes still apply across sessions. D54 scopes active-role selection to the current browser session and its shared tabs; other devices remain unchanged.

Active-role scope (D54): POST /staff/session/role updates only the current session, increments its authority revision and returns an access-token result + Session. Require Bearer authorization and the existing role-switch body; return a JWT reflecting current authority. Shared tabs synchronize and discard old-scope caches/requests after the approved unsaved-change prompt. Other sessions keep their roles; employee-wide permission/assignment removals still apply immediately through current-authority checks. Old-role JWTs never execute workspace actions after the switch, and refresh never reinstates an old role. Session deadlines remain unchanged.

## 11. Field and request-schema bounds

Field-bound review evidence: current packages/data/src/validation.ts caps visitor/roster names at 120, purpose at 1000, visitor instructions and coordination notes at 2000, and cancellation reason at 1000. apps/admin/src/features/BookingDetail.tsx also applies 1000-character limits to reason controls, while packages/data/src/domain.ts validates a rescheduling minimum of 5 without a matching maximum. D55 approves the common 1,000-character reason maximum; no frontend code has been changed.

Action reason bounds (D55): cancellation, rejection and rescheduling reason strings have a maximum of 1,000 characters after the existing trimming rules. Preserve each operation's existing minimum length and conditional requiredness, including optional restricted rescheduling reasons. Backend and shared frontend schemas enforce the same maximum; over-limit requests receive 422 with the reason field error and preserve editable input. Do not silently truncate.

Organization bounds (D56): organization/instansi is at most 255 characters after existing trimming, consistently across public, staff-standard and restricted visits. Existing minimum lengths and required/optional rules remain. Match shared frontend validation and persistence; over-limit requests receive field-specific 422, never silent truncation.

Strict write schemas (D57): unknown JSON keys return 400 unexpected_field with safe fieldErrors identifying rejected paths, including nested objects and fields outside the selected standard/restricted request variant. Reject before business mutation; never silently drop misspellings or mass-assign persistence fields. Frontend adapters must construct documented DTOs rather than spread UI/entity state into requests. Existing documented semantic constraints return 422. Response consumers must tolerate compatible additive fields. D58 rejects unknown API query parameters with 400.

Query strictness (D58): endpoint/view-specific query allowlists reject unsupported keys with 400 unexpected_query_parameter and safe field details. Do not silently ignore misspelled filters or forward unrelated web-page URL parameters to APIs. This does not affect routing parameters in the web application itself. Valid filter values and server authorization scope still require independent validation.

## 12. Sensitive-operation rate limits

Password-change rate limit (D59): POST /staff/password admits at most 5 attempts per employee per rolling 15 minutes across sessions and API instances, counting success and failure. Apply in addition to D34, with D26 shared PostgreSQL counters and D35 failure policy. Authenticate the employee before charging that employee budget; atomically admit before expensive password verification, without holding a counter lock during hashing. Excess returns 429/Retry-After; denied traffic does not extend the window or permanently lock the account. A password change or new session does not reset the allowance. Do not automatically replay a throttled/uncertain password mutation or retain its password for deferred submission.

Refresh rate limit (D60): admit at most 10 POST /staff/session/refresh requests per rolling 60 seconds per identified session across tabs and API instances, counting same-request recovery retries. Also apply D34 employee-wide ceiling once identity is established. Atomically check relevant counters before any refresh transition; 429/Retry-After neither consumes credentials nor revokes the session. A rate-limit response never extends the D43 10-second recovery deadline. Limiter failure follows D35. Invalid unknown credentials cannot charge an arbitrary employee/session budget; D61 applies a separate request-level IP limit, including unidentified traffic.

Refresh IP limit (D61): POST /staff/session/refresh admits at most 100 requests per rolling 60 seconds per trusted client IP across API instances, including missing/invalid cookies and recovery retries. Check this request budget before session identification; identified callers additionally apply D60 session and D34 employee limits. Keep login budgets independent. 429/Retry-After never consumes/rotates a refresh credential, revokes a session or extends any recovery/session deadline. Apply D35 on counter failure. Shared-network legitimate traffic remains an implementation/load verification concern, not measured capacity.

## 13. Gateway and deployment contract

Gateway initialization (D48): the gateway serves staff HTML after an internal Go cookie-initialization call; this is an additional internal integration endpoint, excluded from the 38 frontend API operations. It is not publicly routable and never accepts arbitrary client-supplied internal authority headers. Go owns signing/session validation and returns separate Set-Cookie headers for the gateway to forward unchanged. Anonymous success permits login HTML, not staff API access. Dependency failures produce an unavailable document response; no CSRF bypass or idle renewal. Concrete internal route/authentication and gateway configuration must be verified before deployment.

Deployment boundary (D62): initial target is one host running gateway, API, separate worker and PostgreSQL, with private internal connections and separate public visitor/staff hostnames. Public exposure is limited to intended gateway ingress; internal initialization/database endpoints are not frontend APIs. D63 selects Docker Compose; D64 selects Nginx; D65 selects a dedicated gateway secret over verified internal TLS. This is a planning target, not a deployment or infrastructure-change authorization.

Runtime packaging (D63): initial one-host topology uses separate Docker Compose services for gateway, API, worker and PostgreSQL. Private networks and persistent volumes support repeatable deployment/isolated verification; production and development configurations remain separate. D64 selects Nginx; D65 selects a dedicated gateway secret over verified internal TLS. No public API paths change, and no container/database operation is authorized by this planning decision.

Gateway selection (D64): Nginx serves staff/visitor frontend documents and static assets, proxies permitted API routes and calls Go via an internal authorization subrequest for staff cookie initialization. Verify the selected image includes auth_request and explicitly preserve all independent cookie headers; no cached initialization or personalized HTML. This adds no public frontend API operation. D65 selects dedicated-secret authentication over verified internal TLS; actual certificate/secret provisioning remains a deployment prerequisite.

Internal gateway authentication (D65): authenticate the private initialization call with a dedicated managed gateway secret over TLS with CA/hostname verification. Gateway strips client-supplied internal headers, forwards only trusted metadata, and never exposes the secret in browser responses/logs. Go rejects missing/invalid credentials; no public route forwards this endpoint. Keep gateway secrets separate from token-signing/CSRF/recovery keys and provide a rotation procedure. Certificate/secret provisioning must be confirmed before deployment; no infrastructure changes are authorized here.

## 14. Request sizes and deadlines

JSON body limit (D66): non-upload JSON request bodies are at most 256 KiB (262,144 bytes). Nginx and Go enforce the byte cap independently of field validation and actual Content-Length claims; return 413 before business processing. The frontend handles both safe API JSON errors and gateway non-JSON 413 responses while preserving drafts. Multipart upload total-body allowance is 6 MiB (D67); the 5 MiB individual file limit remains.

Multipart body limit (D67): public/staff draft upload requests allow one file at most 5 MiB (5,242,880 bytes) and a total request body at most 6 MiB (6,291,456 bytes), including framing/metadata. Nginx and Go enforce actual bytes and independent per-file/field limits; oversized requests receive 413. Reject extra file parts and undocumented metadata; clean partial content without issuing a usable attachment token. This does not increase the file limit or enable production uploads.

JSON execution deadline (D68): ordinary JSON handler work has a 15-second server execution budget after body acceptance, propagated to use cases/database operations. This is not a total upload/body-read budget or a guarantee of response latency. Email delivery is queued. If a timeout occurs, preserve drafts and original idempotency/submission references; never assume the write rolled back or blindly replay non-idempotent actions. The frontend also handles gateway/non-JSON timeout failures. Exact timeout HTTP mapping and transport read/write budgets remain to be finalized.

Upload receive deadline (D69): the complete multipart body must arrive within 60 seconds from body reception beginning at public ingress; arriving chunks do not restart the timer. Keep the 6 MiB total / 5 MiB file caps. Incomplete uploads are rejected and partial content cleaned without issuing an attachment token. Verify absolute ingress enforcement, not only a proxy inter-read timeout or a backend timer started after gateway buffering. Any new module/buffering architecture needed to enforce it requires explicit review. This does not define the separate post-receive processing deadline.

Upload processing deadline (D70): after complete multipart body reception, validation/private local storage and metadata/token commit have a 15-second processing budget. The earlier D69 receive limit remains separate. A timeout is not proof that metadata did not commit: preserve the original upload reference and use its safe replay/recovery rules. Stage bytes privately and reconcile failures between filesystem and DB; never expose an incomplete file or issue a token before durable acceptance. Production file enablement remains deferred.

Upload replay token (D99): derive the same opaque token from a dedicated purpose-specific secret key and immutable accepted-upload metadata. Persist its key version, random derivation nonce, actor/audience binding, object/reference identity, content digest and fixed issuance/expiry values; use an unambiguous versioned canonical encoding. Keep the matching key available through the last issued token's original one-hour deadline, separately from JWT/CSRF/refresh-recovery keys. Never derive from public reference/filename alone or extend expiry on replay. After expiry, a new submission needs reupload; resolve an uncertain committed booking before that, and replay committed outcomes before new token validation. No raw/recoverable upload token is persisted or logged; a private lookup hash may be retained. This does not replace D43's separately selected encrypted refresh-successor recovery.

Upload transport (D71): Nginx streams upload bodies to Go without request buffering on those routes only. Go enforces the remaining 60-second total receive budget from trusted ingress timing, then applies the separate 15-second processing budget. Preserve the 6 MiB total / 5 MiB file limits, bounded staging and cleanup. Do not automatically retry upstream after forwarding body bytes. Acceptance must demonstrate elapsed deadline enforcement under slow trickle/chunked traffic and supported Fiber/fasthttp behavior; ordinary JSON buffering is unchanged. No additional gateway module is selected.

Framework target (D72): Fiber v3 implements this transport contract; the business payloads, direct responses and layer boundaries remain framework-independent. Adapt v2 reference middleware rather than copy it. Current official v3 guidance requires Go 1.26+; exact toolchain/dependency pins need compatibility/security verification. Explicit standard context deadlines and tested streamed-upload behavior remain mandatory.

Toolchain target (D73): implement with Go 1.27, using a consistently reviewed patch across development, CI and API/worker images. The HTTP contract remains independent of patch version; module/driver/middleware compatibility and vulnerability checks are prerequisites for implementation. No build/test or installation result is claimed.

Database target (D74): PostgreSQL 18 for the fresh isolated baseline; verify driver/migration compatibility and container-volume conventions on that major. Pin a reviewed supported minor/image before implementation. Existing-data migration remains separate and no live upgrade is authorized.

Role-assignment directorate (D75): retained as optional persistence metadata, initially unset, with no additional request/response field or access-filter behavior in this version. Employee profile directorate remains independent; replacing assigned role arrays must preserve metadata on retained assignment rows.

Room metadata (D76): persistence retains nullable ruangan/lokasi metadata, initially unset where unknown. The room catalog remains {id,name,active}; no new frontend data requirement or room-code uniqueness rule is introduced. Never derive these values from the display name or clear them through unrelated catalog updates.

Visitor snapshot ownership (D77): submitted visitor/contact data belong to that visit. Repeated bookings with matching email, phone or name create independent snapshots; no person-directory matching/profile synchronization is added. Public email/code authorization still targets a specific standard visit. Edits and roster changes cannot mutate another booking through a shared visitor record.

Restricted visitor absence (D78): a restricted visit with none of name/organization/serviceId supplied has no Pengunjung row and a nullable visitor FK. Supplying any visitor detail creates a private visit-owned snapshot. Public/standard visits still require full ordinary visitor details and roster. Wire responses continue omitting absent optional fields rather than inventing empty placeholder identities.

Text-length counting (D79): ordinary human-text min/max constraints count Unicode code points after documented trimming, not UTF-8 bytes or JavaScript UTF-16 units. Coordinate frontend validators/counters and textarea/input enforcement; native maxlength alone must not reject valid non-BMP text. Go validates UTF-8 and counts runes; database character constraints must match. Combined emoji/grapheme sequences may count as several code points. Passwords and protocol credentials/identifiers require their own explicit policies, and byte-level request limits remain independent.

Password hashing and maximum (D80): bcrypt; passwords may not exceed 72 UTF-8 bytes. Align frontend create/change/login byte validation and accessible guidance; replace the existing 256-character maximum. Never truncate, trim incidentally, or pre-hash without a separately reviewed change. Creation/password-change requests report field validation for excess; login errors remain generic. Store only private encoded salted hashes, with calibrated cost at least 10 and bounded concurrent hash work. D81 sets a 15-code-point minimum for new passwords.

New-password minimum (D81): account creation and password changes require at least 15 Unicode code points and at most 72 UTF-8 bytes (D80); both constraints apply. Session and employee-directory policy metadata return passwordMinLength: 15 and passwordMaxBytes: 72, including empty directory pages. Align all create/change controls and accessible help; do not interpret the byte ceiling as 72 arbitrary Unicode characters. Login/current-password verification uses the submitted existing credential, never a newly imposed minimum as proof of correctness.

Email scope/bound (D82): accept one valid ASCII mailbox address, at most 254 bytes after the documented surrounding-whitespace trimming. Accept valid ASCII/punycode domains; do not add automatic Unicode mailbox conversion or SMTPUTF8 support. Reject display-name/list/header-injection input, invalid syntax and excessive local/domain lengths consistently in frontend/backend. Existing selected email matching/normalization applies; do not infer ownership or merge visit snapshots from syntax validation. The diagram's varchar(255) can remain physical capacity while API/domain validation enforces max 254 bytes. No truncation.

Actor metadata storage (D83): Role, Policy and Role Policy created_by/updated_by are nullable integer employee foreign keys rather than free-text identifiers. Resolve actors from authenticated server authority; system/seed operations leave the employee FK absent and record a system actor in audit. No extra frontend field or operation is added, and clients cannot mass-assign actor metadata.

Official assignment persistence (D84): maintain one employee/official pair row regardless of active state, with a database unique pair constraint. Assignment edits deactivate omitted pairs and reactivate existing pairs rather than insert repeated historical periods. Audit changes transactionally; current server authority uses only active assignments. No new public operation or period fields.

Role-policy mapping (D85): each role has one isolated managed policy for this version. PATCH /staff/roles/{id} replaces the permitted flat permission set only through that role-owned policy; it cannot mutate a policy shared with another role. Preserve normalized Role Policy/Policy Permission entities, expected version, assignable-permission validation, anti-escalation and last-admin checks. Lock/version/audit the change atomically and invalidate stale effective-authority revisions across affected sessions. No separate policy-management API is introduced.

Directory/metadata text (D86): listed official/position/room/policy/feature/action/permission display names and optional room location/ruangan/role-assignment directorate have a maximum of 255 Unicode code points after documented trimming. Required names are nonblank; optional metadata may be absent. Existing field-specific visitor names, reasons, notes, email/password and technical keys are not overridden. No additional CRUD routes or metadata UI fields are introduced.

Integration secret bound (D87): each supplied plaintext secret is at most 4,096 UTF-8 bytes before encryption; preserve exact content rather than applying ordinary text trimming. Never return, log or echo rejected secret values. Use field-key-specific 422 errors without secret content. Omitted/blank configured secrets keep existing values under the established editor contract; explicit clearSecrets deletes them, and required secrets cannot be cleared while enabled. Retain D66 overall JSON cap and tighter provider-specific validation. Storage must allow encryption metadata/overhead beyond the plaintext limit.

Integration endpoint URL bound (D88): max 2,048 UTF-8 bytes after documented surrounding-whitespace trim. Validate the actual provider's allowed scheme, hostname/port and endpoint policy; reject embedded credentials/header injection and unsupported keys with safe field errors. The size is an application cap, not a universal URL standard or permission for arbitrary outbound requests. Do not truncate.

Integration text bound (D89): text-kind non-secret configuration values are at most 255 Unicode code points after documented trim, with tighter provider rules as applicable. Do not apply this cap to URL/email/secret fields or accept arbitrary numeric strings without provider-specific numeric syntax/range checks. Reject unsupported fields and return field-key-specific errors without echoing secrets. Structured configurations beyond this value shape require a later reviewed schema change.

Privacy version fallback (D90): public creation requires ack:true; privacyVersion is optional. When omitted, resolve and store the current configured published notice version at commit, acknowledgment time and source=server_default. A supplied version uses source=client; D91 rejects stale/unknown supplied versions before new creation. An idempotent committed replay preserves its original resolved version even after notice changes. Missing approved notice configuration does not silently create a version or authorize genuine registration. The source marker avoids representing a server default as proof of the exact notice displayed.

Supplied notice version (D91): for a new submission with privacyVersion present, require it to match the current published version; stale/unknown versions return 422 privacy_notice_changed with a safe privacyVersion field error before a visit, audit success or email intent is created. Frontend refreshes the notice, preserves non-secret form details and asks for fresh acknowledgment; do not silently remove the supplied field to trigger D90 fallback. Existing committed idempotency outcomes are resolved/replayed before revalidating mutable notice state. If payload/version changes require another reference, first resolve/close the prior uncertain reference under the selected recovery protocol. Do not resubmit automatically as a new booking.

Business-hours interpretation (D92): 08:00–16:00 WITA inclusive applies to the start only, preserving current frontend validation. A 16:00 start with the 60-minute default ends at 17:00; staff-selected duration can also extend past 16:00. Use the calculated full interval for resource overlap and admission-window end, while preserving all selected state/future-time and duration constraints. Never reject solely because the end is after 16:00.

Search text bound (D93): booking/employee q is at most 255 Unicode code points after trim; omitted/empty means no search. Excess receives a clear 400 invalid_query_parameter with q field details under query validation, without executing a broadened unfiltered query. Align frontend validation and preserve entered text rather than truncate. Authorization scope is applied before search and counts; bind/escape search expressions rather than interpolate SQL.

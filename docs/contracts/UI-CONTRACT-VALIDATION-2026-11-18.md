# UI ↔ API contract validation

18 November 2026, WITA. Documentation only. No application, migration or contract file was changed.

**Validated:** [operation reference](FRONTEND-DRIVEN-REST-API-2026-11-16.md) (40 ops), [field rules](FRONTEND-DRIVEN-DETAILS-2026-11-16.md), [approved schema target](APPROVED-SCHEMA-TARGET-2026-11-18.md) and [decision log](../specs/frontend-driven-api-2026-11-16/map.md).

**Traced against:** every reachable data-consuming component in `frontendx/apps/visitor`, `frontendx/apps/admin` and `frontendx/packages/data/src`. Each one was followed from component to hook, repository, `http.ts` and the zod parser in `wire.ts`/`reports.ts`/`visit-log.ts`. The parser is what decides whether a backend response is accepted.

**Next step:** the [backend implementation spec](../specs/backend-api-2026-11-18/spec.md) turns these findings into build decisions.

## Verdict

- **Every reachable screen has an operation.** No UI action lacks an endpoint. The only uncovered calls are the Integrations screen. It is unreachable: `/integration` redirects and the component is never imported. It stays out of scope.
- **The contract cannot yet be implemented exactly as written.** If it were, the current UI would reject these responses with a 502 parse error or mis-handle them:
  - 4 staff projection gaps
  - 3 wire-format choices
  - 1 auth status code
- **Backend status:** 8 of 40 operations exist (01, 32–36, 39, 40). The other 32 have no route, table or usecase yet.
- **Known adapter changes still pending.** Decisions 05, 08, 14, 22, 24, 30 and 31 were already agreed and deferred to the frontend: Bearer transport, reception proof POST, room IDs, upload purpose, report room IDs, `pejabat.nama`, and the single attendance route. They break the UI against a conformant backend until the frontend adapter ships. They are listed in [section 6](#6-frontend-work-not-backend), not repeated as findings.

Severity key:
- **Gap:** the UI needs data no operation defines.
- **Mismatch:** the contract and the UI disagree.
- **Ambiguity:** the backend must choose a rule. The spec adopts the default shown here.

## 1. Coverage by operation

| Op | Consumer | Backend | Verdict |
| --- | --- | --- | --- |
| 01 GET public/catalogs | Booking, PrivacyNotice | done | Example omits `rooms`, `directorate`, `deputy`, which the parser requires (V-05). KD-14. Ticket 04 corrects the example and field table |
| 02 POST public/attachments | Booking uploads | — | Covered. KD-22 |
| 03 POST public/bookings | Booking submit | — | Request table omits `visitKind`, `ndName`, `attachmentToken`. Response omits fields (V-06, V-07). KD-30 |
| 04 POST public/submissions/resolve | SubmissionRecovery | — | Covered. Outcomes underspecified (A-01) |
| 05 POST public/lookup | Status, VisitTicket, PublicVisitHistory | — | Covered. 404 semantics and history model underspecified (A-02, A-03) |
| 06 POST public/bookings/{id}/cancel | Status | — | Covered |
| 07 POST public/bookings/{id}/admission | AdmissionQR | — | Covered. Issue window and version effect unstated (A-04) |
| 08 GET staff/catalogs | CreateAppointment, RestrictedAppointment, EmployeeEditor | done | Officers lack `directorate`/`deputy` (V-05). Per-role scope undefined (A-15). KD-14. Ticket 04 documents the fields and serves the A-15 scope; the room-object parser gap (KD-14) remains |
| 09 GET staff/bookings | Bookings, Dashboard, Reception queues | — | **Gap:** no per-status tab counts (V-04). Metric and queue semantics undefined (A-09, A-10) |
| 10 GET staff/bookings/{id} | BookingDetail | — | **Gap:** ~15 consumed fields undefined (V-01). Restricted allow-list (V-02) |
| 11 POST staff/bookings | CreateAppointment, RestrictedAppointment | — | `visitKind` omitted by standard form (V-06). Duration default (A-06). KD-10 |
| 12 approve | BookingDetail | — | Covered. KD-14/15/10 |
| 13 reject | BookingDetail | — | Covered |
| 14 reschedule | BookingDetail "Edit kunjungan" | — | Covered. Doubles as a general edit (A-07). UI caps reason at 1000 |
| 15 staff cancel | BookingDetail | — | Covered. QR revocation unstated (A-08) |
| 16 update (restricted) | RestrictedAppointment edit | — | **Mismatch:** UI makes date/time read-only, contract allows edits (V-12) |
| 17 POST staff/resolve | Reception (code and QR) | — | **Gap:** returns counts only. UI renders names, purpose and ND (V-03) |
| 18 attendance | Reception confirm | — | Covered. KD-31 |
| 19 POST staff/attachments | CreateAppointment | — | Covered. KD-22 |
| 20 GET …/attachment | AttachmentAccess | — | Covered, but unreachable until `attachmentAvailable` exists (V-01) |
| 21 GET …/visitor-photo | VisitorPhoto | — | Covered (needs `visitorPhotoAvailable`, V-01) |
| 22 GET reports/summary | DashboardAnalytics, Reports modal | — | Timestamp format, filter echo (V-08, V-09). Scope semantics (A-11) |
| 23 GET reports/export | Reports modal | — | Covered. CSV format and filename pattern unstated (A-12) |
| 24 GET reports/visits | VisitLog | — | V-08, V-09, V-10. Paging rules unstated (A-13) |
| 25 GET reports/visits/export | VisitLog CSV/PDF (client-built) | — | V-08, V-09. Snapshot size vs 15 s client timeout (D-05) |
| 26–29 employees | EmployeeDirectory, EmployeeDetails, EmployeeEditor | — | Profile fields missing from tables (V-11). Rules undefined (A-16) |
| 30–31 roles | RoleManagement | — | Covered. Permission catalogue undefined (A-17) |
| 32 POST staff/sessions | Login | done | Covered (KD-05) |
| 33 GET staff/session | Session bootstrap | done | Covered. UI also handles 403 |
| 34 POST staff/session/role | RoleControls | done | Covered (KD-05: TokenResult ignored) |
| 35 DELETE staff/session | Sign out | done | Covered |
| 36 POST staff/password | PasswordChange | done | **Mismatch:** a wrong current password returns 401, which signs the user out (V-10) |
| 37–38 POST …/attachment, …/visitor-photo | ReceptionFiles | — | No production caller yet (KD-08) |
| 39–40 session refresh/renew | none yet | done | Contract-only until the Bearer adapter ships (KD-06/07) |
| *(none)* GET/PATCH staff/integrations | IntegrationManagement | — | Screen unreachable. Out of scope |

## 2. Gaps and mismatches the backend must resolve

Each row gives the evidence and the resolution the [spec](../specs/backend-api-2026-11-18/spec.md) adopts.

| ID | Finding | Evidence | Resolution |
| --- | --- | --- | --- |
| V-01 | **Staff booking fields are undefined.** No op defines `attachmentAvailable`, `visitorPhotoAvailable`, `createdBy {name, role}`, `serviceIdLast4` or `visitorCount`. The op 09–16 tables also omit `visitorInstructions`, `coordinationNote`, the four decision reasons/notes, `ndName`, `serviceId`, `reference`, `restrictedView` and `visitKind`. The UI gates the ND/photo buttons and approve/reject on them. | `wire.ts:60-82`, `BookingDetail.tsx:164-198, 654`, `ReceptionFiles.tsx:110-114`, `restricted-identity.tsx:16` | Define one StaffBooking model with these fields, projected per role ([section 4](#4-wire-invariants-the-backend-must-honour)). `createdBy` comes from the creating employee and their active role at creation. |
| V-02 | **Restricted records must contain only allow-listed keys.** The parser rejects any key outside the allow-list for the record's `restrictedView` (`reception` ⊂ `arrival` ⊂ `assigned-pa` ⊂ `full`). Sending `vip:false` or `history:[]` on a restricted record breaks the UI. | `wire.ts:83-126` | The serializer emits only allow-listed keys per view. A test asserts the exact key set for each view. |
| V-03 | **Op 17 resolve returns too little.** It returns only `visitorCount`/`visitorPhotoAvailable`. Reception renders roster names and purpose, and needs `attachmentAvailable` and `ndName` for the ND viewer. | `Reception.tsx:309`, `ui/src/index.tsx:483-501`, `ReceptionFiles.tsx:114, 139`; preview `domain.ts:317-335` | Resolve returns the reception arrival projection: StaffBooking at `arrival` (restricted) or reception scope (ordinary), with roster, purpose and file availability. |
| V-04 | **Status tab counts have no source.** Tab counts are computed from the current 7-row page, so the "missed" count is always 0. | `Bookings.tsx:300, 315` | Add `statusCounts` (`all` plus each stored status and `missed`) to op 09 `view=list`. Counts are scoped and filtered, but ignore the `status` filter. The frontend reads it in its adapter change. |
| V-05 | **Catalog fields are missing.** The public catalog parser requires `rooms`, and the op 01 example omits it. Officers' `directorate`/`deputy` are used by the employee editor and are missing from the op 01/08 tables. | `wire.ts:127-143`, `EmployeeEditor.tsx:68-72` | Document `rooms`, `directorate` and `deputy` on both catalogs. The backend already emits room objects (KD-14 covers the parser change). |
| V-06 | **Request fields are undocumented.** The op 03 table omits `visitKind:"standard"` (always sent), `ndName` and `attachmentToken`. Standard staff create omits `visitKind`. A strict decoder built from the table would reject every booking. | `validation.ts:117, 179`, `Booking.tsx:332-333` | Accept these fields. `visitKind` defaults to `standard` on op 11. Public `visitKind` must equal `standard`. |
| V-07 | **Public responses omit rendered fields.** The op 03–06 responses lack `rescheduled`, `rejectionReason`, `cancellationReason`, `visitorInstructions`, `duration` semantics and a `history` element model. | `Status.tsx:120-145`, `VisitTicket.tsx:100`, `wire.ts:15-58` | Define PublicBooking fully. History is `{at, action, publicMessage?}`, and `action` comes from `history.ts`'s vocabulary. |
| V-08 | **Report timestamps must be UTC `Z`.** Report and visit-log parsers use `z.string().datetime()`, which rejects `+08:00` (verified against zod 4.6.7). The contract allows any offset. | `reports.ts:71`, `visit-log.ts:54-55, 73` | Freeze **all** API timestamps as UTC RFC 3339 with `Z`. The backend already does this for auth. |
| V-09 | **Report filter echo must be exact.** The UI compares `filter` with what it sent: same key order, string for one value, array for 2+, same click order. "Normalized" echoes would 502. | `repositories.ts:42, 59, 168-172` | Echo multi-value filters exactly as received. Do not sort, deduplicate, or turn one value into an array. |
| V-10 | **Two status/shape mismatches.**<br>(a) Op 36 returns 401 `invalid_credentials` for a wrong current password, and any authenticated 401 signs the user out.<br>(b) The summary and visit-log parsers are `.strict()`, so any extra field (for example the planned row `roomId`) fails. | (a) `session_usecase.go:363`, `http.ts:62`<br>(b) `reports.ts:66, 77, 96`, `visit-log.ts:31-84` | (a) Return 422 with `fieldErrors.currentPassword`.<br>(b) Emit exactly the documented report fields. Add row `roomId` only when the frontend's KD-24 change ships. |
| V-11 | **Employee profile fields are missing.** Ops 26–29 omit `jobTitle`, `deputy`, `directorate`, `phone`, `nip` and the UI's length limits. The UI still accepts a legacy singular `role`. | `EmployeeDetails.tsx:19-25`, `EmployeeEditor.tsx:94-107` | Add the profile fields (all present in `vms.pegawai`). Emit `roles[]` only. |
| V-12 | **Restricted edit and schedule changes.** The op 16 details say the editor can change date/time. The UI renders them read-only and the preview rejects changes with 422. | `RestrictedAppointment.tsx:182-210`, `preview.ts:659-661` | Follow the UI. Op 16 accepts date/time only if unchanged; schedule changes go through op 14. **Confirm (D-03).** |
| V-13 | **Error body shape.** The UI needs `fieldErrors: Record<string, string[]>` with flat form keys, plus a user-facing Indonesian `message` shown verbatim. The backend emits `map[string]string` with one field and a generic message. Several 409 causes share no distinguishing `code`. | DETAILS:228; `failure.go`; `CreateAppointment.tsx:213-214` | Extend the failure type to multi-field arrays and per-code messages, and publish a code catalogue (see spec). |
| V-14 | **Authorization uses role and permission together.** The UI authorizes by active-role ownership as well as permission: attendance needs Receptionist, restricted update needs Protocol Officer, decisions need the owner role. Not-owner is 403. | `domain.ts:345-370`, `preview.ts:743-749` | The policy checks role ownership **and** permission. Out of scope → 404 (non-disclosing). In scope but not allowed → 403. Wrong state/version → 409. |

## 3. Ambiguities resolved by default

These follow strict frontend derivation: the current UI rule wins unless an approved decision overrides it. The spec builds on these defaults.

| ID | Question | Default adopted | UI evidence |
| --- | --- | --- | --- |
| A-01 | Op 04 outcomes | Unknown key → `not_committed` and the key closes. Committed key with a different email → 404, never `not_committed`. A concurrent create is serialized on the same request row, so there is no in-flight state. | `submission.ts:82-98` |
| A-02 | Op 05 "unavailable" | 404 for no match, wrong email, or a restricted visit. Rejected, cancelled and checked-out visits are still returned. Email is compared case-insensitively. | `Status.tsx`, `validation.ts:120` |
| A-03 | Public history | Visitor-visible events only: created, approved, rejected, rescheduled, cancelled, checked_in, checked_out. Oldest first. `publicMessage` is only the staff reschedule reason. | `history.ts:1-10`, `wire.ts:48-58` (**D-04**) |
| A-04 | QR issue window | **Decided by the user on 18 November 2026: the QR is static.** Issued when stored status is approved and server time < `endsAt`. Every later request returns the same payload, expiring at `endsAt`. Issuing does **not** bump the version. Cancellation, or a reschedule that changes date/time/duration, revokes it. The QR used to check in also checks out. Replaces the D21 five-minute rotation. The app works unchanged: it simply refreshes at `expiresAt`. | `AdmissionQR.tsx:26-75`, `VisitTicket.tsx:113` |
| A-05 | Timing fields | `opensAt` = start − 30 min, `endsAt` = start + duration, in epoch ms from one server timestamp per response. Missed means approved and now > `endsAt`. Cancel is allowed only strictly before start. | `attendance.ts:22-47` |
| A-06 | Duration default | 60 minutes when omitted (public booking, restricted create, blank approve). | `RestrictedAppointment.tsx:87-95`, `BookingDetail.tsx:244` |
| A-07 | Reschedule as general edit | Version and audit always change. `rescheduled:true` and QR revocation happen only when date, time or duration changes. The new start must be in the future, 08:00–16:00 WITA. The same window applies to staff create. | `BookingDetail.tsx:61-62, 228-234`, `validation.ts:50-56` |
| A-08 | Cancel side effects | Staff and public cancel revoke outstanding admission credentials. | `BookingDetail.tsx:140` |
| A-09 | Dashboard metrics | Each metric mirrors the UI fallback, except `inside` (decision 03):<br>• `today` = in-scope visits scheduled on `date`<br>• `pending` = pending visits on any date<br>• `arrivals` = visits scheduled on `date` with display status approved<br>• `completed` = checked_out visits scheduled on `date`<br>Rows: scheduled on `date`, plus pending, plus checked_in. | `Dashboard.tsx:29-50, 147-179` (**D-01**) |
| A-10 | Reception queues and list paging | • `checkin` = approved, scheduled on `date`, not missed.<br>• `checkout` = checked_in on any date.<br>• `totalScoped` ignores q/status/vip/date.<br>• A page past the end returns 200 with empty items.<br>• Search only matches fields visible in the caller's projection. | `Reception.tsx:88-92, 213`, `Bookings.tsx:101, 143, 206` |
| A-11 | Summary scope and revision | Scope by role: Super Admin → `organization`, Protocol → `authorized`, PA → `assigned`, Receptionist → `attendance`. Dates use the WITA scheduled date. `revision` = digest of the canonical filter plus the matched `(id, version, displayStatus)` set. | `reports.ts:103-139` |
| A-12 | Summary CSV | Mirror the preview generator: BOM, quoted cells, CRLF, Indonesian headers. Filename matches `^kunjungan-ringkasan-[\d-]+\.csv$`. | `reports.ts:92, 140-157` |
| A-13 | Visit-log paging | `pageSize` is one of 10/25/50, `q` is at most 200 characters, page is clamped. Empty filter values mean no filter. Restricted rows: name falls back to "Kunjungan terbatas", reference is `TRB-<id>`. | `VisitLog.tsx:65-67, 630-649`, `visit-log.ts:107` |
| A-14 | Booking conflicts | Overlap uses full WITA timestamps `[start, end)` on the same officer **or** the same non-null room, against stored approved/checked_in visits only, using scheduled intervals. It is checked under resource locks. | `scheduling.ts:11-31`, BACKEND-PLAN §6 |
| A-15 | Staff catalog scope | Active officers and rooms for every staff role with `bookings` or `people`. Officers carry `vip`, so forms narrow their own options. The employee editor also gets inactive officers already assigned. | `RestrictedAppointment.tsx:228-234`, `EmployeeEditor.tsx:493` |
| A-16 | Employee rules | • Duplicate email → 422 `email`.<br>• PA assignment: PA role only, at most one officer, non-VIP, active.<br>• You cannot change your own roles or active flag.<br>• The last active Super Admin is protected.<br>• `""` clears optional profile fields.<br>• Deactivation or role change bumps credential generation or role revision, which ends live sessions.<br>• `total` is the filtered count, `totalAll` the unfiltered count.<br>• `session.id` equals the employee id. | `EmployeeEditor.tsx:83, 493`, `management.ts` |
| A-17 | Role permissions | Catalogue = the 14 seeded keys. Assignable per role = the `domain.ts:384-398` defaults. PATCH must leave the non-assignable subset unchanged (else 422). Saving bumps role revision. | `RoleManagement.tsx:14-41` |
| A-18 | Pending session states | Role selection required → only 33, 34, 35, 39, 40. Password change required → only 33, 35, 36, 39, 40. Every business route requires a ready workspace. When both are set, the session is treated as role selection first, then password change once a role is chosen (ticket 02; a literal intersection would strand the user). | `main.tsx`, `session.tsx` |
| A-19 | Rate limits | Existing decisions D29–D33. 429 with `Retry-After`. Lookup, cancel and admission share the credential budget. | BACKEND-PLAN §2 |
| A-20 | Null vs omitted | Responses omit absent optional fields (the parsers reject `null`). `room` is `""` when there is no room. Requests: on existing-visit edits, an omitted note keeps it and `""` clears it. Room uses omit/null/ID per decision 15. | `wire.ts`, `AppointmentRoomField.tsx` |

## 4. Wire invariants the backend must honour

These are checked in parsers the backend cannot see. Each needs a serializer test.

- **IDs and formats:**
  - IDs are decimal strings. `code` is 8 digits. `duration` is an integer from 15 to 240.
  - `date` is `YYYY-MM-DD` and `time` is `HH:mm` (WITA).
  - `timing.*` values are integer epoch milliseconds. Every timestamp is UTC `Z`.
- **Optional fields:**
  - Optional fields are omitted, never `null`. `room` is always a string.
  - `checkedInAt`/`checkedOutAt` in visit-log rows are the only nullable timestamps.
- **Restricted records:** they carry `visitKind:"restricted"`, `restrictedView` and only that view's keys:
  - `reception`: schedule fields, `visitKind`, `reference`.
  - `arrival`: adds `name`, `officerId`, `host`/`pejabat`, `serviceIdLast4`.
  - `assigned-pa`: adds `organization`.
  - `full`: adds `code`, `serviceId`, `history`, `createdBy`, reasons and notes.
- **Consistency:**
  - `version` is identical across list, detail and resolve for the same visit.
  - One server timestamp drives every derived value in a response.
- **Paging:** `items.length ≤ pageSize`, `items.length ≤ total`, `total ≤ totalScoped`. `pageSize` is 1–100 (bookings) or 1–250 (employees).
- **Files:**
  - Download responses use an exact `Content-Type` with no parameters.
  - Uploads must be over 0 bytes and at most 5 MiB, with content verified by signature.
- **Errors:** `message` is shown verbatim, so it must be user-facing Indonesian. `fieldErrors` keys are flat form keys (`visitors`, not `visitors.3`).

## 5. Decisions that need the product owner

Each default is adopted in the spec until confirmed.

| ID | Decision | Default |
| --- | --- | --- |
| D-01 | Dashboard `today`/`arrivals`/`completed` are scheduled-date scoped (like the UI fallback), while `inside` spans dates (decision 03). Should `completed` count checkouts that happened on `date` instead? | Scheduled date |
| D-02 | Should op 09 add `statusCounts` so the tabs are correct (V-04), or should the frontend drop the counts? | Add `statusCounts` |
| D-03 | Restricted edit (op 16): date and time read-only, as the UI does, rather than editable as the contract says. | Read-only |
| D-04 | Should a staff reschedule reason be shown to the visitor as history `publicMessage` (as the preview does)? | Yes |
| D-05 | Should the visit-log export snapshot be capped (all-time ranges vs the 15 s client timeout)? | 10,000 rows, then 422 `export_too_large` |
| D-06 | The UI promises a notification email after staff actions (`VisitNotesFields.tsx:69`). Is the email outbox/worker in this build? | **Decided 18 November 2026:** later, via mailgrid over SMTP. See the [deferred email spec](../specs/email-notifications-2026-11-18/spec.md) |

## 6. Frontend work (not backend)

**Ticketed 19 November 2026** as [frontend adapter tickets 01–10](../specs/frontend-adapter-2026-11-19/map.md); that map shows which backend tickets each one waits for.

Already agreed, deferred adapter changes. Each breaks the UI against a conformant backend until it ships:

| Decision | Change |
| --- | --- |
| 05 / 06 / 07 | Bearer access token in memory. Refresh/renew lifecycle. Capture the TokenResult from login, role switch and password change. CSRF for lifecycle routes. |
| 08 | Reception file reads become `POST {credential}`. |
| 10 | Officer required on restricted visits. Room optional on ordinary visits. |
| 14 / 15 / 24 | Room objects and `roomId`. Report room filters by ID. Row `roomId` (strict parser). |
| 22 | Upload multipart gains `purpose`. |
| 30 | Read `pejabat.nama` instead of `host`. |
| 31 | One `/attendance` route with `action`. |

New frontend defects found by this validation:
- `AdmissionQR` retries a 409 with the same stale version and never re-runs the lookup (`AdmissionQR.tsx:40, 108-112`).
- After a stale-notice 409 on booking submit, the catalog is not refetched, so the same `privacyVersion` is resubmitted (`submission.ts:75-77`).
- Status tab counts come from the current page (`Bookings.tsx:300`). Read `statusCounts` once D-02 lands.
- The Receptionist "inside" fallback counts today only (`Dashboard.tsx:43-48`). The server metric fixes the card, not the fallback.

## 7. Contract-only items (fine, no action)

- `timing.canCancel` and `opensAt`/`endsAt` on public records: the client recomputes them, but the parser requires them.
- `roomId` and `pejabat` on bookings: the parser strips them until KD-14/30 ship.
- Staff catalog `privacyNotice` and `jobTitle`.
- The employee list `page` field.
- `roleRevision` beyond cache keys.
- The 201 vs 200 replay distinction.
- Staff cancel of pending visits: the contract allows it and the UI hides it.

Agent working reports with full line evidence were kept in the session scratchpad. This file is the durable summary.

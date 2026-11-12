# Kunjungan Terbatas feature plan

30 October G09 requirement revision: **one active role per account** replaces the multiple-assigned-role workspace-switch plan. The server assigns the active role and its permissions/officer scope; staff do not combine or self-switch production roles. An authorized administrative reassignment refreshes/invalidates the session and scoped caches, and actions retain role attribution. The development-only test-role selector remains a fixture tool. The new backend/session adapter is not implemented or certified by this requirement change; it must expose one canonical role rather than infer access from array order.

30 October G03/G05 follow-up: the user chose **adjust app** for both. The revised Terbatas form, role projections, credential-gated reception identity and general report/export exclusion are implemented in the synthetic preview. Check-in uses server time from 30 minutes before start through the scheduled end; afterward an un-checked-in Approved visit displays Terlewat/Missed, while already Checked-in visits can check out later. See [focused G03/G05 note](anti-slop/RESTRICTED-ARRIVAL-2026-10-30.md). The new D:/oikn/VMS/backend remains unbuilt; production enforcement is still pending.

Prepared 22 October 2026; revised 26 October 2026 (Asia/Makassar). The 22 October design was **implemented and verified in the synthetic UI preview**. The revised rules below were **implemented and verified in preview on 30 October**; production integration is pending. Applies to `D:/oikn/VMS/frontendx`; the future backend is a new, unbuilt D:/oikn/VMS/backend project.

Reconfirmed during the 29 October audit review. The user accepted this revised matrix and form, one active role per account (G09 revision), summary-first general reporting with Terbatas exclusion, and a check-in window from 30 minutes before start through the scheduled end. Late arrivals within that window are allowed; after the end, show Missed only if never checked in. See [the approved implementation plan](anti-slop/IMPLEMENTATION-PLAN.md). The 29 October consolidation changed documentation only; runtime changes and evidence followed under G03/G05 on 30 October.

## Superseding decision: 26 October frontend milestone

This section replaces conflicting access, form, projection and export statements in the 22 October baseline below. All sections beginning at the 22 October baseline heading are historical implementation/evidence, superseded wherever the current section differs. They are not current acceptance criteria.

- Protocol alone creates, edits, approves, rejects and reschedules Terbatas. Tanggal and Waktu mulai (WITA) are required. The form shows only four optional fields: **Nama, Instansi, No ID Dinas, Pejabat yang dituju**. A selected target may be any active pejabat, including Kepala Otorita; blank target remains valid. No ID Dinas is a text identifier, not a numeric value, so leading zeroes remain intact. Omitted fields stay absent rather than receiving fabricated values.
- A PA with a current assignment to the selected pejabat may read the visit and its schedule, Nama, Instansi and target pejabat. The PA sees only the last four characters of No ID Dinas and gets no Terbatas edit/decision actions. Blank target grants no PA access. A changed target or assignment recalculates read access; stale cached data must not remain visible.
- Super Admin may list and read full Terbatas detail, including No ID Dinas, but has no Terbatas edit/decision rights from the admin role alone. Protocol has full authorized detail. Visitor/public access remains excluded.
- Receptionist sees Approved, Checked in and Checked out Terbatas in general views, limited to reference, schedule, status and attendance controls. General visit detail hides Nama, Instansi, No ID Dinas and target pejabat. After a valid code/QR is resolved, the existing arrival-verification step may show Nama, the last four characters of No ID Dinas and Pejabat yang dituju when supplied. It must not show Instansi or the full ID. Receptionist performs ordinary check-in/out; there is no added Protocol confirmation action or acknowledgment. With all optional identity fields absent, the code/QR and existing verification step are the available match evidence; this tradeoff was accepted by the user.
- Exclude Terbatas records and events from all general visit and audit-log exports. Retain restricted audit records internally under the eventual retention policy. A restricted export would need a separate, explicit authorization and specification. Apply role-specific projections before search, pagination, counts, direct IDs and code/QR resolution; general reporting/export receives no Terbatas data.
- Frontend acceptance: date/time-only and each optional field combination; all role projections; current/changed/blank pejabat assignment; unauthorized direct IDs and counts; masked ID at every PA and Receptionist surface; receptionist gated verification and check-in/out; no Terbatas in general exports; role-switch cache clearing; keyboard, narrow-screen, typecheck, builds and focused preview tests. These checks establish preview behavior only. Backend enforcement, storage and real audit/export paths remain later integration work.

## 22 October baseline requirements (superseded where the 26 October section differs)

- Protocol creates Terbatas appointments. Receptionist cannot create them.
- Only staff with an active Protocol or Receptionist role and the required action permissions can access them. PA, Super Admin alone, other roles and public/visitor access cannot discover their existence.
- Protocol manages the appointment. Receptionist views it and performs check-in/check-out, without editing the appointment.
- **Date and start time are required. Every other appointment field is optional**, including detail/agenda, visitor name, organization, email, phone, host, room, identity information, roster and document attachment.
- Protocol starts from Kunjungan and opens **Buat janji temu**, then chooses **Kunjungan VIP** or **Kunjungan Terbatas**.
- Protocol can create and manage VIP appointments only for **Kepala Otorita**. Existing PA creation and management scope remains assigned non-VIP officers.
- For Terbatas, Protocol may optionally select **any active pejabat**, including Kepala Otorita. The host can remain blank. Selecting a host never grants that host's PA access to the restricted appointment. This destination rule was explicitly confirmed by the user on 22 October 2026.

## Preview defaults

The preview implements these defaults. They remain implementation choices rather than separately user-confirmed product requirements:

- Keep staff creation Pending, followed by Protocol approval. Receptionist sees Approved, Checked in and Checked out, matching its current status scope. Automatic approval is not included in this proposal.
- Keep date/time in WITA and the existing future-start rule. Retain the existing 60-minute default duration when none is supplied; do not require a duration field or room to save or approve Terbatas.
- Keep Terbatas staff-only: no public status lookup, visitor session, public QR view or automatic visitor email, even when an email address is optionally recorded. Reception can resolve an authorized admission credential without opening a public booking page.
- Keep the appointment kind fixed after creation in the first version. Changing to VIP could disclose a private record and needs a separate, explicit policy.
- The Dashboard shortcut should lead to the same Protocol type choice as Kunjungan.

## User flow

1. Protocol opens Kunjungan and activates Buat janji temu.
2. A keyboard-operable dropdown offers Kunjungan VIP and Kunjungan Terbatas, with short descriptions. PA keeps its existing appointment action; Receptionist and Super Admin get no creation action.
3. VIP opens the existing full form with Kepala Otorita selected and locked to the authoritative position classification. Terbatas opens a separate, shorter variant headed **Buat kunjungan terbatas**, with an optional host selector listing all active pejabat and no preselected host.
4. The Terbatas form presents Tanggal and Waktu mulai (WITA) first, marked required. Optional visitor and visit details follow. Explain: **Hanya tanggal dan waktu yang wajib diisi. Kunjungan ini hanya dapat diakses oleh Protokol dan Resepsionis.**
5. A date-and-time-only submission succeeds. Protocol sees the created detail and the existing approval action. Proposed approval does not introduce mandatory host, room, visitor, identity or contact data.
6. Receptionist sees eligible Terbatas appointments with an explicit **Terbatas** label and available details. Without a name, display **Kunjungan terbatas** plus its authorized reference and schedule; do not persist an invented visitor name or roster.
7. Receptionist resolves the appointment through the existing code/QR process, confirms the correct arrival and explicitly checks in/out. Missing optional identity/roster fields must not force completion of the appointment form; operational verification can involve Protocol.

Use a Terbatas label in authorized list, detail, Today and Reception views. Keep lifecycle status separate from appointment kind. No new route, standalone application or dependency is needed.

## Access rules

| Operation | Protocol | Receptionist | PA / Super Admin alone / other roles / visitor |
| --- | --- | --- | --- |
| Create Terbatas | Yes, with create permission | No | No |
| List / search / read Terbatas | All lifecycle states within Protocol access | Approved, Checked in, Checked out, as proposed | No |
| Edit / approve / reject / reschedule | With the relevant permission and valid lifecycle transition | No | No |
| Check in / check out | No new attendance grant | With attendance permission, credential and valid transition | No |
| Documents / internal notes | Protocol-authorized projection | Preserve existing field/document restrictions | No |

Multiple-role accounts need explicit evaluation: an active Protocol role grants Protocol scope; an active Receptionist role grants reception scope. PA or Super Admin must never grant Terbatas access by themselves. Broad administrative permissions do not bypass the restricted visit policy.

Apply access filtering **before** search, pagination, counts, summaries and serialization. Cover Kunjungan, Dashboard, Today, Reception, direct IDs, code/QR resolution, attachments, reports/exports, audit views and integrations. Inaccessible direct IDs and credentials return the same generic unavailable/not-found result as absent records. No redacted placeholder rows, hidden visit counts or private event text may reach unauthorized clients. Check conflict responses as well so they do not name or describe a restricted visit.

Retain audit events with actor, operation and timestamps, but filter access to those events using the restricted visit policy. Do not expose Terbatas references or payloads through Super Admin audit/report screens. Clear or partition query caches when the staff identity or access context changes.

## Data, validation and API contract

Introduce an explicit discriminator such as `visitKind: "standard" | "restricted"`. Existing records remain standard. The UI's VIP choice creates a standard booking for Kepala Otorita; VIP remains derived from the authoritative host classification. Terbatas authorization uses `visitKind`, independently of a missing host or the host's VIP classification. Do not repurpose the current VIP boolean as a privacy flag.

Use a distinct restricted form/input schema and a typed booking variant. Public booking types must not silently inherit a restricted record. Required fields for a restricted creation request are its kind and schedule; all non-schedule appointment fields accept omission. Server-owned ID, code, creator, timestamps, status, duration default and version are generated or resolved on the server.

For optional fields, blank means absent; supplied values still receive format/length/file validation. Empty roster means unknown, not zero confirmed attendees. Do not fabricate email, identity, visitor count or officer records to satisfy old validators. Unknown people must not inflate attendance metrics. Date/time validation checks real calendar values, valid clock ranges and the WITA instant.

Extend the typed staff-create adapter and proposed `POST /v1/staff/bookings` contract with this discriminator. Mutations resolve the staff actor and permissions from the session, never request fields. Public creation cannot request restricted kind. Preserve idempotent creation/retry, optimistic versions, atomic audit writes and generic denied reads. Partial updates distinguish omitted fields from explicit clearing; unknown or server-owned write fields are rejected.

Every endpoint that returns or mutates bookings must apply kind authorization, including visitor lookup, cancellation, reschedule responses and notification processing. Terbatas records are rejected before any public projection or visitor credential is issued. An email address or booking code does not make one public. Attached files inherit the same access boundary.

The Go backend currently has no matching staff-create route in `internal/server/rest/router/staff_booking_router.go`. Its existing staff repository joins a required visitor and officer, and its original schema requires host/representative data and a room for approved visits. Production needs a reviewed data-model migration, nullable/conditional constraints, projections, joins and approval rules that support Terbatas without relaxing standard booking validation. Inspect the current schema and legacy read/report paths before choosing the exact migration. No database migration or activation is authorized by this planning document.

## Frontend structure and state

- Reuse Kunjungan's entry action, on-demand form loading, detail, query hooks and shared controls. Add a small Protocol type menu and separate VIP/Terbatas form variants rather than weakening the existing visitor/staff schema globally.
- URL state owns form visibility and selected kind, consistent with the existing `new=1` flow; URLs contain no sensitive form values. React Hook Form owns the unsent draft. Query owns session and bookings; local state owns menu/discard/submit interactions. Derive display fallbacks rather than storing duplicate values.
- Preserve input on failed submission, map server validation to fields, prevent duplicate submits and reuse the same idempotency key for the same retry. Changing kind with edited input must handle discarding explicitly.
- Provide loading, empty, no-results, recoverable-error, success, stale-version and denied states. Generic denied states must not identify a restricted appointment. Support mobile reflow, labels, keyboard/focus return, associated inline errors and visible required-field guidance.
- No new browser persistence, worker or library is needed. Reuse the scoped list/query and defer form loading until the user opens it. Server-side access filtering is required even if the UI also scopes records.

## Build order and acceptance criteria

1. **Contract and access policy:** add kind-aware schemas, projections and authorization to the synthetic adapter. Permit an omitted Terbatas host or any active pejabat; reject inactive or nonexistent supplied hosts. Verify all roles, direct IDs/credentials, counts, reports/public routes and supplied-field validation.
2. **Protocol flow:** add the VIP/Terbatas dropdown and forms, then date/time-only create and approve. Preserve standard visitor/PA validation and Kepala Otorita-only VIP management. Exercise input retention, duplicate submit/retry, kind switching and absent details.
3. **Reception flow:** show authorized Terbatas data and labels, preserve current status scope, resolve credentials and check in/out without requiring omitted appointment fields. Verify stale/duplicate/invalid transitions and history visibility.
4. **Frontend verification:** extend the existing domain and loopback integration suites for the privacy boundary and minimal record. Run typecheck and both builds; browser-check Protocol, PA, Receptionist and Super Admin, including role-switch caches, keyboard operation and narrow layouts. Preview success is not production readiness.
5. **Production integration:** implement the Go contract, schema/projections, authenticated action rules, private documents, scoped reports/audit/integrations and notification exclusions. Verify on a disposable database, review migration effects separately, and only then consider approved live activation.

Acceptance requires both positive and negative evidence: Protocol can save with only date/time or select any active pejabat for Terbatas; inactive and nonexistent hosts are rejected; Reception can process approved Terbatas; PA/Super Admin/public callers receive no record, count, event, file or public access path. An optional host assigned to a PA never grants that PA access. Existing standard bookings continue to obey their current rules, and Protocol's VIP creation/management remains limited to Kepala Otorita.

## Source inspected for this plan

- `PRODUCT.md` and `PLAN.md`: current frontend scope and staff-created Pending default.
- `packages/data/src/domain.ts`: host-derived VIP, role scope, creation/decision helpers and staff/public projections.
- `packages/data/src/validation.ts`: existing required appointment fields.
- `tooling/preview.ts`: synthetic creation, scoping and server-generated metadata.
- `apps/admin/src/features/CreateAppointment.tsx`, `apps/admin/src/pages/Bookings.tsx`, `Dashboard.tsx` and `Reception.tsx`: current entry points and forms.
- Separate Go backend: `internal/server/rest/router/staff_booking_router.go`, `internal/repository/staff_booking/implementation/repository.go`, and booking/core-flow migrations.

## Preview implementation and evidence

The Protocol dropdown and Dashboard chooser open VIP or Terbatas forms. Terbatas supports date/time-only creation, optional active host selection and later detail editing by Protocol while Pending or Approved. Schedule changes use the separate reschedule action. Approval/rescheduling accept no room; duration defaults to 60 minutes. Kind cannot be changed through editing. Standard visitor/staff validation and Protocol VIP scope remain in place.

The loopback synthetic API filters Terbatas before returning lists or direct records, rejects unauthorized roles and public creation/lookup/cancellation, checks stale versions before restricted edits, and requires matching credentials for restricted attendance. Reception projections omit contact data, purpose, documents, creator, internal notes and event history. Approved and Pending synthetic examples are seeded; the Pending example targets a PA-assigned non-VIP host without becoming visible to that PA.

The preview uses a separate restricted schema and typed input union. Existing booking consumers retain normalized empty strings and an empty roster for absent details; display labels never become saved visitor data. A stricter stored booking union, unknown-field write rejection and database model remain production integration requirements described above.

Verification on 22 October: 23 domain/validation tests, 10 synthetic integration tests, typecheck and both builds pass. Browser verification covers date/time-only create, roomless approval, Receptionist code resolution and check-in/out, PA direct-link unavailability, Super Admin denial, the Protocol chooser and 390px/320px form reflow. See VERIFICATION.md for evidence and limitations.

The development role selector and query parameter are synthetic controls, not authentication. Production HTTP feature operations remain explicitly unavailable until the Go contract is implemented. No live database, real email, real upload or physical camera was exercised or changed.

# Spec: VMS backend for the frontend-driven REST contract

Label: spec · Status: `ready-for-agent` (test seams and ticket breakdown approved 18 November 2026). Tickets: [issues/](issues/).

Sources:
- [operation reference](../../docs/contracts/FRONTEND-DRIVEN-REST-API-2026-11-16.md)
- [field rules](../../docs/contracts/FRONTEND-DRIVEN-DETAILS-2026-11-16.md)
- [approved schema target](../../docs/contracts/APPROVED-SCHEMA-TARGET-2026-11-18.md)
- [decision map](../frontend-driven-api-2026-11-16/map.md)
- [UI ↔ contract validation](../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md) (findings `V-*`, defaults `A-*`, open decisions `D-*`)
- [Go conventions](../../docs/BACKEND-GO-CONVENTIONS.md)
- [earlier decision register D01–D100](../../docs/BACKEND-PLAN-2026-11-11.md)

## Problem Statement

The VMS frontend (public visitor app and staff app) is built, and its REST contract has been settled through 31 decision tickets. The Go backend serves only 8 of the 40 operations: the public catalog and the staff session lifecycle. Visitors cannot register, check status, cancel or get an admission QR. Staff cannot list, decide, reschedule or check visitors in and out. No one can read reports or administer employees and roles.

Validation also found that a backend built literally from the contract would still be rejected by the current UI. Staff booking projection fields are undefined. Restricted records must carry only allow-listed keys. Report timestamps must be UTC `Z`, and report filters must be echoed exactly. A wrong current password currently signs the user out.

## Solution

Implement the remaining 32 operations, and adjust the existing auth and error behaviour, inside the existing Fiber/GORM monolith. Each capability becomes a module with the house layering:

- repository, usecase and serializer interface/implementation pairs
- a flat handler
- a router file that only wires

Business rules live in one pure, heavily tested policy package. Every visit mutation runs in one database transaction that:
- rechecks the caller's authority,
- locks the visit and the affected resources in a fixed order,
- compares the expected version,
- writes the change and its audit event together.

Responses use per-audience serializers that emit exactly the keys each view allows. Work lands as tracer-bullet slices, each demoable over HTTP against the stage-1 PostgreSQL.

## User Stories

**Visitor (public app)**

1. As a visitor, I want to see active officials with their position, directorate and deputy, so that I pick the right person to visit.
2. As a visitor, I want to read the current privacy notice in Indonesian or English, so that I know how my data is used before I consent.
3. As a visitor, I want to upload my Nota Dinas as a PDF, so that the official receives my supporting letter.
4. As a visitor, I want to upload a representative photo (JPEG/PNG), so that reception can recognise me.
5. As a visitor, I want my upload rejected if it is empty, over 5 MiB, or not really the declared type, so that I fix it before submitting.
6. As a visitor, I want to submit a registration with my details, the roster of names (me first, up to 20), the official, date, time, purpose and identity type, so that I request a visit in one step.
7. As a visitor, I want a start time outside 08:00–16:00 WITA, or in the past, rejected with a clear field error, so that I choose a valid slot.
8. As a visitor, I want an 8-digit booking code and a receipt, so that I can track my request.
9. As a visitor whose connection dropped after submitting, I want the app to find out whether my registration was saved, so that I neither lose it nor book twice.
10. As a visitor, I want a retried submission with the same data to return my original booking, so that duplicates are impossible.
11. As a visitor, I want a retried submission with changed data to be refused, so that one attempt can't silently become a different booking.
12. As a visitor, I want a submission against an outdated privacy notice refused, so that my consent always matches the published notice.
13. As a visitor, I want to look up my visit with code and email (email case-insensitive), so that I see its status, schedule, room, instructions and history.
14. As a visitor, I want rejected, cancelled and completed visits to stay visible in lookup, so that I understand what happened.
15. As a visitor, I want to see the public rejection or cancellation reason, but never staff-internal notes, so that I get an explanation without leaking internal deliberation.
16. As a visitor, I want to see that my visit was rescheduled, and the reason, so that I go at the new time.
17. As a visitor, I want to cancel a pending or approved visit before it starts, so that the slot is freed.
18. As a visitor, I want a cancellation refused if staff changed my visit since I loaded it, so that I act on current information.
19. As a visitor with an approved visit, I want one static QR code that stays the same until my visit changes, so that I can save it and reception admits me quickly.
20. As a visitor, I want my QR invalidated when my visit is cancelled or moved to another time, and the same QR to check me out after I checked in with it, so that it only works for the visit it belongs to.
21. As a visitor, I want no QR issued once my visit window has ended, so that a missed visit can't be used to enter.
22. As a visitor, I want repeated wrong lookups throttled, so that nobody can guess my code.

**Pimpinan's Assistant (PA)**

23. As a PA, I want to see only ordinary non-VIP visits for the officials I'm assigned to, plus a limited view of their restricted visits, so that I handle my own officials' agenda.
24. As a PA, I want to search, filter by status/date/VIP and sort by priority, name, organization, host, time or status, so that I find visits fast.
25. As a PA, I want correct counts per status tab, including missed, so that I know how much work is waiting.
26. As a PA, I want a dashboard with today's visits, pending decisions, people inside and completed visits, so that I see my day at a glance.
27. As a PA, I want to approve a pending visit with an optional room, duration and notes, so that the visitor is confirmed.
28. As a PA, I want approval refused if the official or room is already committed at an overlapping time, so that no double booking happens.
29. As a PA, I want to reject with a public reason and a separate internal note, so that the visitor gets an explanation and the team keeps context.
30. As a PA, I want to cancel an approved visit with a reason, so that the visitor knows and their QR stops working.
31. As a PA, I want to reschedule an approved visit, including a missed one up to seven days after its original end, so that the visit can still happen.
32. As a PA, I want a notes-only edit not to mark the visit as rescheduled or revoke its QR, so that small corrections don't disrupt the visitor.
33. As a PA, I want to create a visit on behalf of a guest for my assigned non-VIP official, so that phone or walk-in requests get registered. Such visits start approved.
34. As a PA, I want my action refused when someone else changed the visit first, so that I never overwrite a decision I haven't seen.
35. As a PA, I want to see who created a visit and in which role, so that I know staff-created visits need no approval.
36. As a PA, I want to view the Nota Dinas and visitor photo of visits I'm allowed to see, so that I can verify the request.

**Protocol Officer**

37. As a Protocol Officer, I want to read all operational visits but decide only VIP and restricted ones, so that ownership follows protocol rules.
38. As a Protocol Officer, I want to create a restricted (Terbatas) visit with only official, date and time required, and optional name, organization, No ID Dinas, room and notes, so that sensitive visits are recorded minimally.
39. As a Protocol Officer, I want to edit a restricted visit's identity fields and notes (clearing optional fields with empty values), so that the record stays accurate.
40. As a Protocol Officer, I want restricted visits never to expose ordinary contact, roster or photo fields, so that their privacy rules hold.
41. As a Protocol Officer, I want ordinary visit decisions refused for non-VIP visits I don't own, so that the PA's authority is respected.

**Receptionist**

42. As a receptionist, I want a check-in queue of today's approved, not-yet-missed visits, so that I prepare arrivals.
43. As a receptionist, I want a checkout queue of everyone still checked in, including from earlier days, so that overnight visitors aren't forgotten.
44. As a receptionist, I want to resolve a typed booking code or scanned QR to the visit, roster, purpose and available files, so that I can confirm identity.
45. As a receptionist, I want check-in allowed only from 30 minutes before start until scheduled end, so that early or late arrivals are handled by staff.
46. As a receptionist, I want checkout allowed for any checked-in visit, even after the QR expired or the day changed, so that departures are always recorded.
47. As a receptionist, I want a double-clicked or competing check-in to record exactly one arrival, so that the log is correct.
48. As a receptionist, I want to open the Nota Dinas and photo only by re-presenting the visitor's code or QR, so that holding the reception role alone doesn't expose files.
49. As a receptionist, I want restricted visits to show only the reception projection, so that I see what I need and nothing more.

**Super Admin**

50. As a Super Admin, I want organization-wide analytics counts per status (including missed) for a date range up to 366 days, excluding restricted visits, so that I can report service volume.
51. As a Super Admin, I want to export those counts as CSV only if the data hasn't changed since I viewed it, so that the file matches the screen.
52. As a Super Admin, I want a visit log with search, multi-select status/kind/official/room filters, sorting and paging, including limited restricted rows, so that I can audit who visited.
53. As a Super Admin, I want a complete visit-log snapshot to export as CSV or PDF, refused if it changed since I loaded it, so that exports are consistent.
54. As a Super Admin, I want to list and search employees, so that I manage staff accounts.
55. As a Super Admin, I want to create an employee with roles, profile fields, an optional PA official assignment and an initial password, so that new staff can sign in.
56. As a Super Admin, I want to edit or deactivate employees with version protection, so that concurrent admins don't overwrite each other.
57. As a Super Admin, I want deactivation or role removal to end that employee's live sessions, so that revoked access is immediate.
58. As a Super Admin, I want to be prevented from removing my own roles or deactivating myself, and the last Super Admin protected, so that the system can't be locked out.
59. As a Super Admin, I want to edit each role's assignable permissions, but not its fixed ones, so that access can be tuned safely.
60. As a Super Admin, I want a permission change to take effect for affected sessions immediately, so that policy edits are enforced.

**Any staff member**

61. As a staff member, I want a wrong current password on the change-password form to show a field error instead of signing me out, so that I can retry.
62. As a staff member with several roles, I want only the active role's permissions applied, never a union, so that each role keeps its boundaries.
63. As a staff member, I want validation errors keyed by the form field with readable Indonesian messages, so that I can fix them.

**Operator / developer**

64. As an operator, I want every 5xx logged with request ID and a redacted cause class, so that failures can be diagnosed without leaking data.
65. As a developer, I want each operation registered in the contract pipeline and OpenAPI as it lands, so that the documented and served contracts don't drift.
66. As an operator, I want readiness to fail when a required table is missing, so that a half-migrated database isn't served.

## Implementation Decisions

### Architecture (follows the existing boilerplate)

- **Layering is unchanged.** Each module has repository, usecase and serializer interface packages with private implementations, a flat handler, and a router file that only constructs and registers. New modules: `upload`, `booking`, `report`, `employee`, `role`, plus an extended `catalog`.
- **The booking module exposes three usecase interfaces, split by caller:**
  - `PublicBookingUsecase`: create, resolve submission, lookup, cancel, issue admission.
  - `StaffBookingUsecase`: list, get, create, approve, reject, reschedule, cancel, update restricted.
  - `ReceptionUsecase`: resolve credential, attend, open file with credential.

  They share one `BookingRepository` with a transaction interface and one `BookingSerializer`.
- **The policy package is the deep module. It is pure, takes `now` as a parameter, and does no IO.** It owns:
  - role and permission constants
  - scope and ownership (mirroring the UI's `scope()`/owner rules)
  - projection choice (`full`, `assigned-pa`, `reception`, `arrival` for restricted; audience projections for ordinary)
  - the lifecycle transition table
  - WITA timing: `opensAt`, `endsAt`, missed, `canCheckIn`, `canCheckOut`, `canCancel`, and the reschedule window ending at end + 7 days inclusive
  - the schedule window (08:00–16:00 WITA, future start)
  - interval overlap

  Usecases call it; serializers never decide.
- **Transactions.** Repositories expose a unit-of-work closure. Every visit write follows the same sequence:
  1. lock the employee (authority recheck);
  2. lock the visit;
  3. lock officers, then rooms, in ascending ID order;
  4. lock admission credentials;
  5. read DB time once;
  6. compare the version and check state;
  7. run the overlap query;
  8. write, increment the version, revoke credentials if needed, append the audit event.

  Read Committed is used; a failed version guard returns 409 with no partial write. Reads use REPEATABLE READ, READ ONLY snapshots that return DB time with the rows, so every derived value in a response comes from one timestamp.
- **Authority recheck in writes.** The transaction locks the employee and reads the session snapshot on the same transaction. Validation is a `SessionValidator` extracted from the existing auth usecase. The existing package-level helper that imports internal layers is removed.
- **Staff route protection.** Bearer verification, live authority and the per-employee limiter are hoisted into router dependencies (`StaffAuth`), together with the workspace gate. Every staff business group is mounted with both. Pending states may only call the session operations listed in A-18.
- **File storage.** A `Storage` adapter interface (`Put`/`Open`/`Delete`) has a private local-filesystem implementation (D15). Object keys are server-generated UUIDs; there are no public paths or URLs.
- **Notifications.** No outbox and no email in this spec. Email comes later through mailgrid over SMTP; see the [deferred email spec](../email-notifications-2026-11-18/spec.md). It hooks in by adding one enqueue call beside each command's existing audit write.

### Transport and error contract (applies to all operations)

- **Error body.** `{code, message, requestId, fieldErrors?}`. `fieldErrors` is `Record<string, string[]>` with flat form keys. `message` is a user-facing Indonesian string per code. A code catalogue (in the conventions doc) distinguishes the 409 causes: `version_conflict`, `invalid_state`, `schedule_conflict`, `reschedule_window_closed`, `idempotency_conflict`, `notice_outdated`, `credential_expired`, `revision_conflict`.
- **Error mapping.** Out-of-scope records and proof mismatches → 404 (non-disclosing). In scope but role, ownership or permission denied → 403. Wrong state or version → 409. Field rules → 422.
- **Timestamps.** All API timestamps are UTC RFC 3339 with `Z`. Schedule fields are WITA `date`/`time`. Timing fields are epoch milliseconds.
- **Optional fields.** Responses omit absent optional fields and never send `null`, except the documented nullable `checkedInAt`/`checkedOutAt` in visit-log rows. `room` is `""` when there is no room.
- **Request tri-state.** A tri-state optional type distinguishes omitted, `null` and value, for `roomId` (decision 15) and for notes on existing-visit edits (omitted keeps, `""` clears).
- **Query strings.** Parsed by an allow-listed typed parser: unknown keys → 400, repeated keys keep their order, empty value = unset. Report filter echoes reproduce exactly what was received (V-09).
- **Op 36 change.** A wrong current password returns 422 with `fieldErrors.currentPassword`; it no longer returns 401 (V-10).

### API decisions beyond the written contract

| Area | Decision | Source |
| --- | --- | --- |
| StaffBooking model | Adds `attachmentAvailable`, `visitorPhotoAvailable`, `visitorCount`, `createdBy {name, role}`, `serviceIdLast4`, `reference`, `restrictedView`, `visitKind`, notes, reasons and `ndName`/`serviceId`, each emitted only where its view allows | V-01, V-02 |
| Op 09 | `view=list` adds `statusCounts {all, pending, approved, missed, rejected, cancelled, checked_in, checked_out}`: scoped and filtered, ignoring `status`. Metrics follow A-09; queues follow A-10 | V-04, A-09, A-10 (D-01, D-02) |
| Op 17 | Returns the reception arrival projection: roster, purpose, file availability, `ndName`, timing, version. 404 outside reception scope | V-03, A-15 |
| Ops 03 / 11 | Accept `visitKind`, `ndName`, `attachmentToken`. Op 11 `visitKind` defaults to `standard`. Missing duration → 60. Staff-created visits start approved with `createdBy` set. Photo required on public and standard staff create | V-06, A-06 |
| Op 07 | **Static QR** (user, 18 November 2026; supersedes the D21 rotation). Issued on first request when the visit is approved and `now < endsAt`. Every later request returns the same deterministic payload. Expires at `endsAt`; no version change. Revoked only by cancellation or a date/time/duration reschedule, after which the next request issues a new one | A-04, physical schema decision 4 |
| Op 18 | The QR that checked a visit in also checks it out, on any date and after expiry | D96, user 18 November 2026 |
| Op 14 | General edit. `rescheduled` and QR revocation only when date, time or duration change | A-07 |
| Ops 15 / 06 | Cancellation revokes admission credentials | A-08 |
| Op 16 | Date/time accepted only if unchanged | V-12 (D-03) |
| Ops 22–25 | Scope by role, WITA scheduled-date basis. `revision` = digest of canonical filter plus matched `(id, version, displayStatus)`. Summary CSV mirrors the preview format. Visit-log export capped at 10,000 rows → 422 `export_too_large` | A-11, A-12, A-13 (D-05) |
| Ops 26–31 | Employee rules A-16; role permission rules A-17. Deactivation, role or permission changes bump credential generation or role revision, which invalidates live tokens | A-16, A-17 |
| Rate limits | D29–D33 with `Retry-After`; lookup, cancel and admission share the credential budget | A-19 |

### Schema (strictly the user's DB diagram; one migration per table)

The build target is the [physical schema](../../docs/contracts/PHYSICAL-SCHEMA-2026-11-18.md). It follows the user's diagram column for column and departs only where an earlier agreement says so; each departure is cited per column. The user decided on 18 November 2026 to:

- **Rebuild migrations one file per table.** The four existing multi-table migrations are replaced, and every table is created in its final shape. This includes converging today's deviations:
  - `role_pegawai.id_karyawan` replaces `id_pegawai`;
  - `created_by`/`updated_by` become VARCHAR;
  - `audit_log` values become JSON, and `audit_log` gains the agreed actor/target columns;
  - `ruangan.ruangan` is dropped.

  The stage-1 database is reset by the user.
- **Keep the agreed Jabatan → Direktorat/Kedeputian hierarchy.** It carries the diagram's `nama_jabatan` and `pejabat_vip`.
- **Hold `integration_keys`.** It is not created in this build and returns with the Integrations screen.
- **Use a static admission QR.** There is one stable payload per visit until it is revoked. The diagram's `qr_kunjungan` columns are unchanged.

**Agreed departures from the diagram:**
- Pengunjung becomes per-visit `detail_pengunjung` (D09, D26, D27).
- Kunjungan gains `version` (D13) and the decision reasons (D25). It drops `tanggal_cek_out` (D17), and its English status values are stored (D23).
- In Detail Kunjungan, the room is nullable (D10) and `visit_duration` is added (D09). `keperluan` widens to 1000 (D11) and `no_nd` becomes text (D28).
- Form Documents gains `purpose` (D22) and drops `file_data` (D21).
- Every event column is `TIMESTAMPTZ` (D18).
- Contract-required columns stay on `pegawai` (status, forced password change, version) and on `role` (version).

**Values derived, not stored, so no unagreed column is added to a diagram table:** `createdBy`, `rescheduled`, the acknowledged privacy version and history messages all come from `audit_log` events. QR consumption is the visit's unrevoked credential at check-in.

**Operational records stay in the `support` schema:** sessions, limiter, upload lifecycle and booking idempotency, per the earlier plan.

The privacy notice remains configuration-published; public create compares `privacyVersion` to it. Existing legacy data is not migrated.

## Testing Decisions

**A good test** asserts externally observable behaviour: status/code/body, rows committed, audit written, keys emitted. It does not assert internal call order. Tests are named as behaviour sentences, use stdlib `testing` only, and use hand-written fakes.

**Proposed seams.** These are existing seams; no new ones. Please confirm.

1. **Policy (pure unit tables)** is where most business rules are proven. It covers:
   - the transition table per role × status × action
   - ownership and scope
   - restricted projection key sets
   - timing boundaries: `opensAt` inclusive, `endsAt` inclusive for check-in, missed at `endsAt` + 1 ms, reschedule at end + 7 days inclusive and + 7 days + 1 ms ineligible, cancel strictly before start
   - the 08:00/16:00 edges and overlap of half-open intervals

   Prior art: the existing visit policy tests.
2. **Usecase with fake repository and transaction** covers validation keys, 403/404/409 selection, the version guard, no write after an authorization failure, an audit event per mutation, and serializer output. Prior art: the auth usecase tests.
3. **HTTP router with fake usecases** (exported `Register…Routes`) covers strict decoding, the query allow-list, multipart limits, status and error-body mapping, and contract captures that `check.py --runtime` validates against the baseline. Prior art: the handler contract test and server tests.
4. **Real PostgreSQL** (`integration` build tag, stage-1 database only) covers the invariants only locks can guarantee: concurrent approvals produce one winner, overlapping approvals produce one success, an idempotent create under parallel replay commits one booking, concurrent check-in records one transition, the visit version is monotonic, resolve and create serialize on the same request, and upload replay returns the same token. Prior art: the admission limiter integration test.

Serializer tests assert the wire invariants in [validation §4](../../docs/contracts/UI-CONTRACT-VALIDATION-2026-11-18.md#4-wire-invariants-the-backend-must-honour). Wherever practical, they decode the output with the frontend's own zod schemas, run via a small Node script in the existing contract tooling. That protects V-02, V-08 and V-09 against regressions.

## Out of Scope

- Frontend adapter changes. These are already agreed (decisions 05–08, 10, 14, 15, 22, 24, 30, 31) and are their own effort. Until they ship, the UI cannot run against these endpoints end to end.
- The Integrations screen, its API and the `integration_keys` table (on hold).
- Server-side QR image rendering with piglig/go-qr. The static payload is designed so a later image endpoint renders the same QR the browser shows today.
- The email notification outbox and worker (D-06: later, through mailgrid; see the deferred email spec), and the upload cleanup worker. Upload expiry is enforced at link time, not deleted.
- Existing-data migration or cutover from legacy tables, a production storage provider, virus scanning, and deployment.
- A live availability endpoint, an audit-log browsing UI, and creation of new roles.

## Further Notes

**Open product decisions.** D-01 to D-05 in the validation file; D-06 is decided (email later, through mailgrid). Each open one has a default adopted above, so implementation can proceed and a later answer changes one ticket.

**Housekeeping.** Each slice must:
- extend the contract pipeline (`source-baseline.json`/`manifest.json`, regenerate OpenAPI, update path and migration count assertions);
- extend readiness for its tables;
- pass `make test lint contract-check`, plus the integration suite against stage-1.

The backend is a nested git repository, so commits land there.

**Tracer-bullet tickets.** These are published in [issues/](issues/) and worked blockers-first. The frontier starts at 00 and 01.

| # | Ticket | Blocked by | Delivers |
| --- | --- | --- | --- |
| [00](issues/00-rebuild-migrations-per-table.md) | Rebuild migrations, one per table | — | Existing tables re-created one file per table in the diagram's shape (`integration_keys` on hold); reference seeds; final grants; existing Go code and seed scripts adapted; stage-1 reset procedure |
| [01](issues/01-error-transport-prefactor.md) | Error and transport prefactor | — | Multi-field `fieldErrors`, code catalogue and messages, not-found → 404, cause logging, typed query parser, tri-state optional, bounded multipart reader, UTC `Z`. Op 36 → 422 |
| [02](issues/02-staff-route-authority-prefactor.md) | Staff route and authority prefactor | 00 | `StaffAuth`/`Workspace` in dependencies, `SessionValidator` with transaction authority lock (old helper removed), role and permission constants, pending-state route rules |
| [03](issues/03-contract-pipeline-rebase.md) | Contract pipeline rebase | 00, 01 | Manifest, baseline and check tooling renumbered to the 40-op contract; implemented ops 01, 32–36, 39, 40 marked; captures green |
| [04](issues/04-staff-catalogs.md) | Staff catalogs (op 08) | 01, 02, 03 | First route through the protected staff group; per-role scope; officer directorate/deputy |
| [05](issues/05-uploads.md) | Uploads (ops 02, 19) | 01, 02, 03 | `form_documents` and `support.form_document_uploads` (one migration each), storage adapter, content-verified purpose-bound tokens, replay, D31 |
| [06](issues/06-public-booking-and-recovery.md) | Public booking and recovery (ops 03, 04) | 05 | `detail_kunjungan`, `detail_pengunjung`, `kunjungan`, `support.booking_requests` (one migration each), code generation, PublicBooking, D30/D32 |
| [07](issues/07-public-lookup-and-cancel.md) | Public lookup and cancel (ops 05, 06) | 06 | Timing derivation, public history, credential budget D29 |
| [08](issues/08-admission-qr.md) | Admission QR (op 07) | 07 | `qr_kunjungan` migration (diagram columns only), static deterministic payload, revoke on cancel/reschedule |
| [09](issues/09-staff-booking-reads.md) | Staff booking reads (ops 09, 10) | 04, 06 | Views, filters, sorts, `statusCounts`, metrics, role projections, restricted allow-lists |
| [10](issues/10-staff-creation.md) | Staff creation (op 11) | 05, 09 | Standard and restricted create, starts approved, lock protocol and conflict engine |
| [11](issues/11-approve-reject-cancel.md) | Approve, reject, cancel (ops 12, 13, 15) | 08, 10 | Decisions with version, ownership, overlap, revocation |
| [12](issues/12-reschedule-and-restricted-update.md) | Reschedule and restricted update (ops 14, 16) | 11 | Seven-day window, edit semantics, revocation |
| [13](issues/13-reception-resolve-and-attendance.md) | Reception resolve and attendance (ops 17, 18) | 08, 09 | Arrival projection, single-transition attendance, consumed-QR checkout |
| [14](issues/14-booking-file-reads.md) | Booking file reads (ops 20, 21, 37, 38) | 05, 13 | Scoped GET, credential-bound reception POST, non-disclosing unavailable |
| [15](issues/15-analytics-summary-and-export.md) | Analytics summary and CSV (ops 22, 23) | 09 | Scope, counts including missed, revision, CSV |
| [16](issues/16-visit-log-and-export.md) | Visit log and export (ops 24, 25) | 09 | Multi-select filters, exact echo, restricted rows, snapshot and cap |
| [17](issues/17-employee-administration.md) | Employee administration (ops 26–29) | 02, 03, 04 | CRUD-by-PATCH, PA assignment, self and last-admin guards, session revocation |
| [18](issues/18-role-permissions.md) | Role permissions (ops 30, 31) | 02, 03 | Assignable-only edits, revision bump |

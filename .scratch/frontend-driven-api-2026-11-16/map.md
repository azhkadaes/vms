# Frontend-driven VMS REST API contract

Label: wayfinder:map

## Destination

A reviewed REST contract tracing every reachable current frontend component, field and action to an operation, with redundant requests consolidated and business ambiguities confirmed by the user. No application implementation in this effort.

## Notes

Planning-first confirmed. Full current frontend confirmed. Strict frontend derivation supersedes the briefly selected availability addition. Required public representative photo confirmed. Use backend-engineering, frontend-engineering, golang-patterns, golang-testing and wayfinder. Local Markdown tracker is used because no external tracker was configured; `/setup-matt-pocock-skills` can configure one later. User requested one-at-a-time confirmation for uncertainties. The requested contract artifact is authorized independently of implementation.

Evidence: [working review](../../docs/contracts/FRONTEND-DRIVEN-REVIEW-2026-11-16.md).

## Decisions so far

- [Use one attendance endpoint](issues/31-shared-attendance-endpoint.md): use an explicit `check_in` or `check_out` action, with the existing action-specific permissions and state rules.

- [Use pejabat.nama in booking responses](issues/30-pejabat-response-name.md): return the official's name as `pejabat.nama`. The UI label remains “Pejabat”.

- [Reconcile rescheduling eligibility](issues/29-reschedule-eligibility.md): Approved only, including derived Missed through seven elapsed days after the original scheduled end. This supersedes D100 for this contract.

- [Map restricted No ID Dinas to detail_kunjungan.no_nd](issues/28-restricted-identifier.md): optional VARCHAR(100) maps to serviceId; separate from Nota Dinas upload and ordinary identity_type.

- [Always create visitor details with visit-type validation](issues/27-restricted-visitor-detail.md): required visitor-detail FK for every visit; restricted detail fields may be absent where the form permits.

- [Reference visit details through unique foreign keys](issues/26-detail-foreign-keys.md): Kunjungan owns unique references to both detail tables; no shared editable detail records.

- [Store current decision reasons separately from appointment notes](issues/25-decision-reasons.md): dedicated Kunjungan fields hold latest values; audit retains history; visibility remains field-specific.

- [Filter visit reports by room IDs](issues/24-report-room-ids.md): room selections carry IDs; options expose ID/name; list and export share identical filtering.

- [Align stored status values with the API](issues/23-stored-status-values.md): six identical storage/API values; Indonesian labels stay in the UI; missed remains derived.

- [Bind uploads to document or visitor-photo purpose](issues/22-upload-purpose.md): add purpose to multipart requests/metadata; validate content and token use against declared purpose.

- [Store uploaded files privately outside PostgreSQL](issues/21-upload-storage.md): metadata/private key in Uploads, file bytes in private file/object storage; omit file_data from proposed schema.

- [Reference audit targets by entity type and ID](issues/20-generic-audit-target.md): generic entity_type/entity_id instead of id_kunjungan FK; target validation remains server-owned.

- [Represent staff, visitor and system audit actors](issues/19-audit-actors.md): explicit actor_type; employee ID required for staff, absent for visitor/system; server owns attribution.

- [Store actual events as timezone-aware instants](issues/18-event-timezones.md): TIMESTAMPTZ for actual events, offset-bearing API timestamps, WITA display; schedule remains DATE/TIME in WITA.

- [Keep a single checkout timestamp](issues/17-checkout-timestamp.md): waktu_cek_out supplies checkedOutAt; derive WITA date and omit duplicate tanggal_cek_out in the proposal.

- [Keep appointment notes in Detail Kunjungan](issues/16-note-ownership.md): single ownership for visitorInstructions/catatan and coordinationNote/catatan_internal; decision reasons and history remain separate.

- [Preserve omitted rooms and clear explicit nulls](issues/15-room-clearing.md): omit-to-keep on existing visits, null-to-clear; either means no room on creation.

- [Send room IDs while displaying room names](issues/14-room-id.md): submit roomId, display name, map to detail_kunjungan.id_ruangan; frontend adapter change deferred.

- [Protect booking changes with an expected version](issues/13-booking-version.md): add Kunjungan.version starting at 1; reject stale mutations and increment on committed changes.

- [Derive missed visits rather than storing another status](issues/12-derived-missed.md): keep stored approved; derive missed after scheduled end without check-in, including filters/counts/reports.

- [Preserve the frontend purpose length](issues/11-purpose-length.md): propose keperluan VARCHAR(1000), matching ordinary purpose validation of 5–1000 characters.

- [Require an officer and allow an optional room for every visit](issues/10-officer-room-requiredness.md): officer mandatory including Terbatas; room optional at all lifecycle stages, including approval.

- [Reconcile current frontend authentication with Phase 1](issues/01-auth-authority.md): user selected all current authentication workflows, multiple assigned roles and one active role; older Phase 1 is conflicting.
- [Define reception proof for private files](issues/02-reception-files.md): user requires verified code/QR proof; reception role alone does not grant file access.
- [Define the dashboard occupants count](issues/03-dashboard-inside.md): count all authorized checked-in visits across dates, including overnight visits.
- [Confirm restricted visits in Super Admin report exports](issues/04-restricted-report-export.md): authorized Super Admin exports include the current limited row projection, excluding sensitive identifiers, credentials, contacts and internal notes.
- [Choose staff authentication transport](issues/05-bearer-transport.md): user selected Bearer tokens and a later frontend adapter change, preserving the screens and workflows.
- [Preserve login across reload and access-token expiry](issues/06-session-restoration.md): restore and refresh while the server session remains valid; require sign-in after session expiry.
- [Retain backend session limits and activity semantics](issues/07-session-limits.md): five-minute access maximum, eight-hour inactivity and 24-hour absolute expiry; only actual activity extends inactivity.
- [Resend reception proof for each file request](issues/08-reception-proof-transport.md): authenticated POST with the existing credential for each reception document/photo read; no verification grant.

## Not yet specified

User-requested visitor schema additions are being resolved in [Add visitor detail, identity type, duration and roster storage](issues/09-visitor-schema-additions.md). These are documentation proposals, not executed migrations.

Full per-operation error examples, authorization/concurrency acceptance cases, and remaining component coverage gaps after the detailed audit.

## Out of scope

Application edits, deployment, new live availability UI/API, and copying SIHP business semantics.

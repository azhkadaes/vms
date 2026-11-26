# Current approved schema direction

As of 18 November 2026, WITA. Documentation only; no application or database changes. This summary consolidates explicit user choices. It does not declare outstanding constraints resolved or supersede the supplied diagram silently.

References: [API operation reference](FRONTEND-DRIVEN-REST-API-2026-11-16.md), [detailed contract](FRONTEND-DRIVEN-DETAILS-2026-11-16.md), [diagram/source comparison](DATABASE-MAPPING-2026-11-16.md), [decision tickets](../specs/frontend-driven-api-2026-11-16/map.md).

## Visit and representative ownership

| Owner | Approved responsibility | API mapping |
| --- | --- | --- |
| Kunjungan | Visit lifecycle, booking code and record version | id,code,status,version |
| Kunjungan decision fields | Latest rejection/cancellation reasons and internal decision notes; earlier events retained in Audit Log | rejectionReason,internalRejectionNote,cancellationReason,internalCancellationNote |
| detail_kunjungan | Officer, date, start time, duration, optional room, purpose and appointment notes | officerId,date,time,duration,roomId,purpose,visitorInstructions,coordinationNote |
| pejabat.nama | Display name joined through detail_kunjungan.id_pejabat | Booking response `pejabat.nama` as a string where visible. UI label “Pejabat”. |
| detail_pengunjung | One per-visit representative snapshot and ordered name array; replaces Pengunjung's role | name,organization,email,phone,identityType,visitors,photo reference |
| Audit Log | Server-attributed events with generic entity target | Filtered history; never raw public audit payload |

No reusable visitor profile, matching of returning visitors, separate companion rows, or standalone public visitor-creation endpoint is required. Visit and dependent detail mutations must commit atomically. The confirmed detail relationships and visit-type requirements are described below.

FK layout now confirmed: Kunjungan holds unique references id_detail_pengunjung and id_detail_kunjungan to the corresponding detail IDs. Detail rows cannot be shared across visits. Create them and the visit in one transaction. Unique references alone do not prevent orphan detail rows; Storage-level enforcement and deletion policy still need specification.

Visitor-detail existence now confirmed: id_detail_pengunjung is NOT NULL and UNIQUE for every visit, including Terbatas. Always create the row; do not fabricate missing restricted identity/contact/roster/photo values. Ordinary requiredness is conditional on visit type, not a universal NOT NULL requirement for each shared detail column. The schedule-detail FK is likewise required. Enforce cross-table visit-type rules transactionally; a simple CHECK on the detail row cannot inspect its parent visit by itself. Exact optional-field null/empty serialization and deletion policy remain to specify.

## Approved field types and rules

Restricted identifier confirmed: detail_kunjungan.no_nd is nullable VARCHAR(100), mapped to serviceId (UI No ID Dinas). Preserve letters/leading zeros. It is not a Nota Dinas document number and is separate from nota_dinas, the upload reference. No detail_pengunjung.service_id column is proposed.

| Database field / concept | Type / rule | API representation |
| --- | --- | --- |
| Kunjungan.version | INTEGER NOT NULL DEFAULT 1, positive | version:integer; expected on mutation |
| detail_kunjungan.id_pejabat | Required FK for every visit, including Terbatas | officerId:string |
| detail_kunjungan.id_ruangan | Nullable FK; single room owner | roomId:string or null; room is derived display label |
| detail_kunjungan.visit_duration | INTEGER minutes, 15–240 | duration:integer |
| detail_kunjungan.tanggal | DATE, WITA calendar | date:YYYY-MM-DD string |
| detail_kunjungan.waktu_mulai | TIME interpreted in Asia/Makassar | time:HH:mm string |
| detail_kunjungan.keperluan | VARCHAR(1000); ordinary input trimmed 5–1000 characters | purpose:string |
| detail_kunjungan.catatan | Sole visitor-instruction owner; optional, max 2000 in API | visitorInstructions:string |
| detail_kunjungan.catatan_internal | Sole coordination-note owner; optional, max 2000 in API | coordinationNote:string; excluded from public/reception |
| detail_pengunjung.identity_type | Enum KTP/Paspor/SIM for ordinary visits | identityType:string |
| detail_pengunjung.nama_rombongan | Ordered name array; proposed physical TEXT[] | visitors:string[], ordinary 1–20, representative first |
| Kunjungan.waktu_cek_out | Single actual checkout timestamp, TIMESTAMPTZ | checkedOutAt:offset-bearing timestamp or null |
| Other actual events | TIMESTAMPTZ for check-in, creation and audit time | ISO/RFC3339 with Z/offset, displayed in WITA |
| Audit Log.actor_type | staff/visitor/system, server assigned | Internal attribution; not writable by API clients |
| Audit Log.id_pegawai | Required for staff; nullable for non-staff | Internal actor identity |
| Audit Log.entity_type/entity_id | Generic target pair; server validated | Internal association; exact SQL ID type not frozen |

Do not create ordinary identity-number storage. Restricted No ID Dinas is separate optional text stored in detail_kunjungan.no_nd as confirmed above. Do not require ordinary identity/contact/roster/photo fields for restricted visits merely because the representative table is shared.

## Status, counts and concurrency

- Stored status_penerimaan values are confirmed as pending, approved, rejected, cancelled, checked_in and checked_out, identical to API status. Indonesian labels remain UI presentation; no translation layer for stored status is required in the target.
- missed/terlewat is derived: approved and server time strictly after scheduled end, without check-in. It is not a seventh stored state or a scheduled status-write job.
- Rescheduling requires stored approved status. A derived missed visit remains eligible through seven elapsed days after its original scheduled end, inclusive. The successful replacement preserves approved status and increments the version.
- Inside count includes all authorized checked-in visits, including overnight; it counts visits, not roster members.
- Reception records arrivals and departures through one attendance command. The request action determines which permission, lifecycle transition and timestamp apply. The visit, version and event commit together.
- Mutations compare expected version and increment it atomically on success. Stale authorized updates return 409; failures and read-only time derivations do not increment version.
- Staff event timestamps come from the server, never from user-submitted timestamps. Checkout persists status/time/version/audit consistently.

## Room semantics

Room is optional at every stage, including approval and rescheduling. Supplied room IDs must be valid; no placeholder room and no room-name-to-ID matching.

| Operation | Omitted roomId | roomId:null | Supplied roomId |
| --- | --- | --- | --- |
| Create | No room | No room | Set validated room |
| Existing-visit schedule mutation | Preserve room | Clear room | Replace with validated room |

Room catalogs use id/name/active objects; dropdown shows name and sends ID. Empty string is invalid as a selected ID. Check conflicts against the effective officer/time/room; absence of room does not disable officer conflict checking. Report room filters also use IDs: preserve existing room query key, submit repeated IDs for multi-selection, and return options as id/name objects. The existing empty-string no-filter sentinel is not a selected room ID. List and export use identical ID filters.

## Files and privacy

- Upload purpose confirmed: multipart `purpose` is document or visitorPhoto and is retained in Uploads metadata/token binding. document accepts PDF; visitorPhoto accepts JPEG/PNG. Reject invalid content and cross-purpose token use. Existing file size limits remain unchanged.

- Ordinary public and staff forms require the representative photo; restricted form does not.
- Reception file reads use POST with already-entered credential plus staff Bearer; revalidate each request. No session verification grant and no proofless reception GET bypass.
- Authorized other staff file GETs remain separately scoped.
- Super Admin visit-log export may include the agreed limited Terbatas projection; no contacts, service IDs, admission credentials or internal notes.
- Aggregate analytics excludes restricted visits, independently of the visit log.
- File storage confirmed: private file/object storage holds bytes; Uploads holds metadata and a private storage key. Omit file_data from proposed schema. Storage provider and file lifecycle details remain to specify; no public paths/URLs are returned by ordinary DTOs.

## Authentication

Multiple assigned roles, one active role; retain login/session/logout/password change and role selection/switching. Use Bearer access tokens with existing cookie/CSRF lifecycle support. Restore after reload and refresh while server session is valid. Access tokens last up to five minutes, idle expiry is eight hours, absolute expiry is 24 hours. Actual user activity extends idle expiry only; background refresh does not. All are contract decisions; current frontend adapter still requires changes.

## Removed duplicate concepts in proposed diagram only

- Pengunjung's original role is replaced by per-visit detail_pengunjung.
- Kunjungan.id_ruangan duplicate is omitted; detail_kunjungan owns the room.
- Kunjungan.catatan/catatan_internal duplicates are omitted; detail_kunjungan owns appointment notes.
- tanggal_cek_out duplicate is omitted; derive WITA date from waktu_cek_out.
- Ordinary nomor_identitas is omitted; keep identity_type.
- Stored terlewat is omitted; derive display status.

This is not permission to delete existing tables, columns or data. Existing-data migration and cleanup require their own concrete reviewed plan.

## Outstanding alignment work

Physical details still to specify include upload lifecycle columns, QR version and consumption storage, audit target types, and defaults and nullability for individual columns. The detailed contract records the QR lifetime, late checkout rules and confirmed rescheduling boundary. Endpoint-specific errors and full example/schema conformance remain under review.

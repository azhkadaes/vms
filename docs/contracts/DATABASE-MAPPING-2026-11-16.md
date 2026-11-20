# API to database mapping

Status: updated against the second, readable diagram supplied by the user. Local SQL and diagram are separate evidence sources and differ. No schema changes made. Conflicts below prevent a claim of complete alignment.

## Evidence boundaries

The first attachment exposed only table names. The second attachment `codex-clipboard-9e75849b-062a-4bec-949c-749cc9cf28e5.png` exposes columns and relationships and is now the supplied design reference. Labels typed simply as `string` do not establish an exact SQL storage type or length. The original editable schema is still unavailable, so ambiguous labels and missing constraints remain explicit gaps.

Local references inspected: the migrations in backend/migration/files as they stood on 16 November 2026. On 18 November they were replaced by one schemas prerequisite plus one migration per table, `202611180001_create_schemas.sql` through `202611180023_create_support_refresh_recovery.sql`, built from [the physical schema](PHYSICAL-SCHEMA-2026-11-18.md); that document supersedes the column comparisons below where they differ. Migrations are source definitions, not evidence that a running database matches them. The attached design may differ.

SIHP reference inspected: docs/swagger.yaml, pkg/dto and route/handler definitions in the SIHP workspace. Reuse method/path/parameters/typed-response documentation structure. Previously referenced uploaded SIHP Markdown originals are not available in this review, so exact typography/section parity cannot be certified.

## Verified local column mappings

| API field | Local SQL source | SQL data type | API data type | Verification / caveat |
| --- | --- | --- | --- | --- |
| officer.id / officerId | vms.pejabat.id | INTEGER | string | Catalog serializer converts integer to decimal text |
| Booking `pejabat.nama` / catalog `officers[].name` | vms.pejabat.nama | VARCHAR(255) NOT NULL | string | Booking references the selected official through `detail_kunjungan.id_pejabat`. Preserve response visibility by audience. |
| officer.active | vms.pejabat.status_aktif | BOOLEAN NOT NULL | boolean | Catalog availability |
| officer.vip | pejabat.id_jabatan → jabatan.pejabat_vip | INTEGER FK → BOOLEAN | boolean | Joined classification, not necessarily stored on pejabat |
| officer.jobTitle | vms.jabatan.nama | VARCHAR(255) NOT NULL | string | FK pejabat.id_jabatan |
| officer.directorate/deputy | jabatan organization FKs → direktorat/kedeputian.nama | INTEGER FK → VARCHAR(255) | string or omitted | Added organization migration; not visible as tables in attachment |
| room.id | vms.ruangan.id | INTEGER | string in backend DTO | Frontend currently uses room names, see mismatch below |
| room name | vms.ruangan.nama | VARCHAR(255) NOT NULL | string | Names not declared unique in inspected room migration |
| room.active | vms.ruangan.status_aktif | BOOLEAN NOT NULL | boolean | Backend room DTO includes active |
| employee.id | vms.pegawai.id | INTEGER | string | Record identifier, not NIP |
| employee.name | vms.pegawai.nama | VARCHAR(255) NOT NULL | string | SQL trimmed length 2–255 |
| employee.email | vms.pegawai.email | VARCHAR(254), UNIQUE, NOT NULL | string | SQL requires lowercased trimmed email; max bytes 254 |
| employee.jobTitle | vms.pegawai.jabatan | VARCHAR(255), nullable | optional string | Null-to-omitted serialization must be checked |
| employee.deputy | vms.pegawai.kedeputian | VARCHAR(255), nullable | optional string | Employee profile text, distinct from officer organization joins |
| employee.directorate | vms.pegawai.direktorat | VARCHAR(255), nullable | optional string | Do not substitute role_pegawai.direktorat silently |
| employee.phone | vms.pegawai.nomor_hp | VARCHAR(255), nullable | optional string | Not numeric |
| employee.nip | vms.pegawai.nip | VARCHAR(255), nullable | optional string | Preserve leading zeros |
| employee.active | vms.pegawai.status_aktif | BOOLEAN NOT NULL | boolean | Server authority |
| employee.version | vms.pegawai.version | INTEGER >0 | integer | Expected version for updates |
| employee.roles[] | role_pegawai.id_pegawai/id_role → role.nama | INTEGER FKs → VARCHAR(255) | array of role-name strings | Unique employee/role pair; multiple roles supported |
| assignedOfficerIds[] | pegawai_pejabat.id_pegawai/id_pejabat/status_aktif | INTEGER FKs, BOOLEAN | array of ID strings | Later partial unique index permits at most one active assignment per employee |
| role.id/name/version | vms.role.id/nama/version | INTEGER/VARCHAR(255)/INTEGER | string/string/integer | Four constrained role names |
| role.permissions[] | role_policy → policy → policy_permission → permission.permission_key | FKs and VARCHAR(64) | string[] | Active links and active policy must be respected; not a JSON array column |
| permission association | permission.id_feature/id_action | INTEGER FKs | flattened permission key | No separate UI CRUD required for each junction |
| session.availableRoles | role_pegawai → role | relational query | role-name string[] | Not a single employee.role column |
| session active role | support.staff_sessions.active_role_id | nullable INTEGER FK | role name / empty selection state | Different from the assigned-role collection |
| session expiry | support.staff_sessions idle_expires_at/absolute_expires_at | TIMESTAMPTZ | ISO timestamp string | Separate serverTime milliseconds is computed serialization |
| initial/current/new password | no plaintext column | bcrypt encoded hash in support.employee_credentials | input string only | Never return encoded_hash or password |

## Unverified visit/storage mappings

| Frontend/API data | Diagram entity candidate | Status |
| --- | --- | --- |
| Booking id/code/status/date/time/duration/version | Kunjungan and/or Detail Kunjungan | Cannot determine owning columns or SQL types |
| Representative name/email/phone/organization/identityType | Pengunjung and/or Detail Kunjungan | Cannot determine per-person vs per-visit ownership, length/nullability |
| visitors[] named roster and ordering | Pengunjung + Detail Kunjungan | Junction/cardinality and representative-first persistence unverified |
| officerId and room reference on booking | Pejabat/Ruangan joined to visit | Visit FK names and nullability unverified |
| visitKind restricted, serviceId and restricted privacy data | Kunjungan/Detail Kunjungan | Column existence and storage protections unverified |
| ND/photo upload tokens and metadata | Uploads | Column types, purpose binding, lifecycle, storage references unverified |
| QR payload/version/expiry/revocation | QR Kunjungan | Persisted credential vs generated token strategy unverified |
| public/staff event history, actor and reasons | Audit Log and/or visit event table | Event shape, visibility and relationships unverified |
| integration secrets | Integration Keys | Unreachable UI does not justify an endpoint; secret storage unknown |
| booking idempotency and uncertain submission resolution | No readable table identified | Supporting persistence required; cannot claim covered by diagram |

Computed fields do not require matching columns: dashboard counts, report totals, displayStatus=missed, timing eligibility, canExport, joined labels and revision hashes. Document their input tables/derivation once columns are available. API DTOs need not mirror table layouts; do not invent one endpoint per table.

## Concrete compatibility gaps

1. Room catalog: frontend expects rooms:string[] and submits room name. Backend serializer returns rooms:{id,name,active}[]. Local SQL has integer room IDs and no unique-name constraint. Current frontend parser cannot consume that backend shape. Name-to-ID lookup is ambiguous if duplicates exist. Needs explicit resolution after readable design is available.
2. Frontend employee email schema lacks the local SQL 254-byte cap. A contract must represent the actual cap and map failures to email, not claim frontend and database already match.
3. Current role schema accepts singular role or roles[]; confirmed business rule is multiple assigned roles with one active role. Canonical DTO and storage join must agree without silently choosing the first role.
4. Required standard/public photo is confirmed, but its current transport schema is optional and diagram Uploads columns are unreadable.
5. Auth requires support.staff_sessions and employee_credentials plus refresh recovery support. These are not visibly established by the supplied table-name diagram. Do not claim the design supports approved login semantics until their inclusion is confirmed.

## Completion criteria

For every request/response leaf field: record API datatype, required/nullability, validation, owning column or derivation, SQL datatype, FK relation where applicable, and verified/mismatch/unverified status. Obtain readable SQL/DBML/original diagram before filling unresolved visit mappings. Preserve approved frontend behavior and ask about any incompatible schema decision; do not alter migrations or quietly drop fields to force a match.

## Readable diagram mapping (supersedes the earlier unreadable-image limitations)

The earlier local-SQL table remains useful but must not be mistaken for this diagram. The following diagram mapping supersedes earlier claims that visit column names cannot be read.

| API field / concept | Diagram source | Diagram datatype | JSON datatype | Match / unresolved point |
| --- | --- | --- | --- | --- |
| booking.id | Kunjungan.id | int NOT NULL PK | string | Decimal-string representation consistent with other frontend IDs |
| booking.code | Kunjungan.kode_kunjungan | char(8) UNIQUE | string | Frontend additionally requires exactly eight digits |
| name | Pengunjung.nama | string | string | Representative linkage via Kunjungan.id_pengunjung |
| organization | Pengunjung.asal_instansi | string | string | Length not defined in diagram |
| email | Pengunjung.email | varchar(255) | string | Visitor column differs from employee local SQL limit |
| phone | Pengunjung.nomor_telpon | string | string | Text preserves leading zeros |
| representative photo | Pengunjung.foto → Uploads.id | int reference | upload token on write; availability boolean on read | Token lookup must resolve FK; token is not raw upload ID |
| visitors[] | Pengunjung.nama_rombongan | string | string[] | A single string does not define ordered named-person records; mapping unresolved |
| identityType | no visible matching column | absent | enum string | KTP/Paspor/SIM cannot be stored as nomor_identitas |
| identity number | Pengunjung.nomor_identitas | int NOT NULL | not collected by ordinary frontend | Required database field conflicts with current UI; never insert fabricated zero |
| officerId | Detail Kunjungan.id_pejabat | int NOT NULL FK | string, optional for restricted | Restricted creation permits no officer; NOT NULL conflicts |
| date | Detail Kunjungan.tanggal | date | YYYY-MM-DD string | WITA calendar interpretation |
| time | Detail Kunjungan.waktu_mulai | time | HH:mm string | WITA wall time; no timezone stored by plain time |
| purpose | Detail Kunjungan.keperluan | varchar(255) | string, current UI max 1000 | Length conflict; no silent truncation |
| duration | no visible matching column | absent | integer minutes | Needed for conflicts, end time and check-in/missed eligibility |
| room | Kunjungan.id_ruangan and Detail Kunjungan.id_ruangan → Ruangan.id | int / FK label | frontend currently string label | Duplicate ownership; choose authoritative FK before writes |
| visitKind/vip | Kunjungan.tipe_kunjungan | ENUM(vip,umum,terbatas) | visitKind standard/restricted plus vip boolean | Candidate mapping umum→standard,false; vip→standard,true; terbatas→restricted; restricted VIP boolean must not be guessed |
| status | Kunjungan.status_penerimaan | ENUM with pending,disetujui,ditolak,dibatalkan,checked in,checked out,terlewat | pending/approved/rejected/cancelled/checked_in/checked_out | Explicit translation required; persisted terlewat conflicts with derived frontend missed |
| checkedInAt | Kunjungan.waktu_cek_in | timestamp | ISO timestamp string or null | Timezone/nullable constraints not stated |
| checkedOutAt | Kunjungan.waktu_cek_out and tanggal_cek_out | timestamp | ISO timestamp string or null | Duplicate checkout timestamps need ownership clarification |
| created time | Kunjungan.tanggal_dibuat | timestamp | ISO timestamp | Server owned |
| visitorInstructions / coordinationNote | catatan/catatan_internal on Kunjungan and Detail Kunjungan | string | optional string | Duplicate owners and public/internal semantics need confirmation |
| rejection/cancellation reasons | no dedicated visible columns | absent / possible notes | string | Do not merge distinct history/reason fields without approval |
| version | no visible booking version | absent | positive integer | Required optimistic concurrency unsupported by visible design |
| ND upload | Detail Kunjungan.nota_dinas → Uploads | int reference | attachmentToken/ndName | FK reference, filename derived from upload metadata |
| ND number | Detail Kunjungan.no_nd | int | no separate current form field | Cannot substitute filename or token; leading-zero/letter risk |
| restricted serviceId | no unambiguous matching field | absent | optional string | nomor_identitas is required integer on visitor; not an equivalent mapping |
| upload metadata | Uploads.nama,storage_path,mime_type,ukuran_file_bytes,file_data | string,varchar,varchar,int,bytea | filename:string; binary file | ukuran_file_bytes repeated; path/blob storage policy unspecified |
| QR credential | QR Kunjungan.token_hash | varchar UNIQUE | payload:string | Stored hash is not returned as QR token; hash comparison/generation required |
| QR lifecycle | QR Kunjungan.id_kunjungan,tanggal_dibuat,tanggal_expire,dicabut_pada | int FK,timestamp,datetime,timestamp | booking ID and timestamps | Version/schedule binding not visible |
| history | Audit Log.jenis_kegiatan,tanggal,deskripsi,old_values,new_values | string,timestamp,string,JSON nullable | filtered history objects | No visible booking FK; public filtering/event ownership unresolved |
| audit actor | Audit Log.id_pegawai | int NOT NULL | server actor | Anonymous public actions lack employee; do not fabricate staff actor |
| officer label/title/VIP | Pejabat.nama,nama_jabatan,pejabat_vip,status_aktif | string,string,bool,bool | name,jobTitle,vip,active | nama repeated in diagram; local SQL instead uses jabatan join |
| employee profile | Pegawai.nama,jabatan,kedeputian,direktorat,email,nomor_hp,NIP | string except email varchar(255) | strings | active/version/password policy absent in diagram but present locally |
| roles[] | Role Pegawai.id_karyawan,id_role → Pegawai/Role | int NOT NULL | role-name string[] | id_karyawan naming differs from diagram Pegawai and local id_pegawai |
| assignedOfficerIds | Pegawai Pejabat.id_pegawai,id_pejabat,status_aktif | int,int,boolean | string[] | One-active-assignment constraint not visible; exists in local migration |
| role permissions | Role Policy → Policy → Policy Permission → Permission | integer FKs | permission-key string[] | Derived join, not a new endpoint for each table |
| permission key | Permission.permission_key | string | string enum | Permission joins Feature and Action |
| auth session/token | no visible session/credential tables | absent | TokenResult/Session | Local support tables required for confirmed login behavior |

### Decision priorities

1. Preserve UI-required fields and propose schema additions, or reduce frontend scope? Awaiting user choice.
2. Resolve duplicate room/notes/checkout ownership, roster cardinality, restricted nullability and required identity number.
3. Confirm enum translation and derived missed semantics; do not store an extra state silently.
4. Confirm purpose length and ID/document-number types. Explicit API validation must match final storage capacity.
5. Define supporting version/idempotency/session/upload/QR lifecycle storage. Their absence in this diagram is not permission to drop confirmed behavior.

All proposed schema corrections remain documentation only. Existing migrations and application files are unchanged.

## User-requested additions, 17 November 2026

Rescheduling eligibility confirmed 18 November 2026. Derive the cutoff from the existing `detail_kunjungan.tanggal`, `waktu_mulai` and `visit_duration` in WITA. Require stored `Kunjungan.status_penerimaan = approved` and allow a derived Missed visit through the original end plus seven elapsed days, inclusive. On success, atomically update the schedule detail, increment `Kunjungan.version`, revoke prior admission credentials and append an audit event. The stored status remains approved. This rule supersedes D100 for the current contract.

Restricted identifier fully resolved: detail_kunjungan.no_nd is optional VARCHAR(100), API serviceId, UI No ID Dinas. This supersedes the diagram's integer type and the earlier proposed visitor-detail owner. It represents the restricted visitor identifier, not a Nota Dinas document number. Keep nota_dinas upload FK and ndName filename separate. Preserve letters/leading zeros and existing role-specific masking/omission. No migration performed.

Latest user direction for restricted identifier: map the existing API serviceId (UI label No ID Dinas) to detail_kunjungan.no_nd, not detail_pengunjung.service_id. The supplied diagram defines no_nd as int and its name could also mean Nota Dinas number. Confirm that this column intentionally represents the restricted No ID Dinas, and whether its target type should be VARCHAR(100) to preserve current frontend letters/leading zeros. Until confirmed, do not conflate it with nota_dinas (upload FK), ndName (filename) or an independently collected document number. No DDL or API rename executed.

Restricted detail existence confirmed: Kunjungan.id_detail_pengunjung is a required unique FK for all visit types. Terbatas gets a real detail_pengunjung row even when optional visitor details are absent. Ordinary representative name/contact/identity_type/ordered roster/photo requirements remain enforced for ordinary creation; they must not force fake data into restricted records. Shared nullable columns alone do not enforce ordinary rules: validate the complete visit aggregate within the transaction, with appropriate database enforcement where feasible. Do not label absent restricted headcount as zero people. Response projection continues to omit inaccessible fields, independently of storage nullability.

Detail FK direction confirmed: proposed Kunjungan.id_detail_pengunjung REFERENCES detail_pengunjung(id) UNIQUE and Kunjungan.id_detail_kunjungan REFERENCES detail_kunjungan(id) UNIQUE. Replace the old id_pengunjung target accordingly; do not also add reverse owning FKs. A detail row cannot be shared by two visits. UNIQUE/FK alone does not guarantee every detail row is referenced: create both detail records and visit atomically, roll back failures, and define any deletion policy separately. Schedule detail is required for visit timing; restricted visitor-detail row/field nullability remains to finalize. This is documentation, not executed SQL.

Decision-reason ownership confirmed: Kunjungan stores latest rejection/cancellation reasons and internal decision notes; Audit Log preserves earlier events. Proposed SQL names/types: alasan_penolakan VARCHAR(1000) → rejectionReason; catatan_internal_penolakan VARCHAR(2000) → internalRejectionNote; alasan_pembatalan VARCHAR(1000) → cancellationReason; catatan_internal_pembatalan VARCHAR(2000) → internalCancellationNote. These fields may be absent before the relevant decision. Current command validation requires 5–1000 trimmed characters for a submitted staff reason, with optional internal note up to 2000; public cancellation has no reason input and must not fabricate one. Store lifecycle/version/current reason/audit event atomically. These are separate from detail_kunjungan.catatan/catatan_internal. Response projection remains role/event specific; storage does not authorize disclosure. Exact omission/clearing behavior follows the applicable command and remains to be consolidated in final schemas.

Report room filtering resolved 18 November 2026: selected room filter values refer to detail_kunjungan.id_ruangan, not Ruangan.nama. Retain existing API filter key room; single ID or repeated room parameters for multiple IDs. Return room options as {id,name}; labels are display-only and duplicate names do not merge rooms. Preserve an explicit no-filter state distinct from filtering visits without a room; the current UI has no dedicated no-room filter. List/export and response filter echo use the same normalized IDs.

Stored status vocabulary resolved 18 November 2026: Kunjungan.status_penerimaan uses exactly pending, approved, rejected, cancelled, checked_in, checked_out, identical to API status. Replace diagram's Indonesian enum labels in the proposed target only; keep Indonesian display labels in the frontend. No stored missed/terlewat value. Existing-data translation, if needed, must preserve semantics and be separately reviewed; no data rewritten.

Upload purpose resolved 18 November 2026: add Uploads.purpose constrained to document/visitorPhoto. API multipart purpose has the same values. document maps to Nota Dinas attachment; visitorPhoto maps to the representative photo. Bind token to recorded purpose; validation must check stored metadata/content, not merely a caller-supplied MIME header. attachmentToken must resolve to document; visitorPhotoToken must resolve to visitorPhoto. This does not add a third purpose or change existing size limits. Physical constraint syntax remains part of the proposed DDL; no migration executed.

Upload storage resolved 18 November 2026: private file/object storage holds photo/ND bytes. Uploads retains metadata and a private storage key (diagram storage_path may serve this purpose); omit file_data from the revised design. Do not expose the key as a public URL or accept arbitrary storage paths from clients. API upload tokens resolve to authorized metadata records; downloads stream through the controlled API after permission/proof checks. Provider and orphan-file/expiry lifecycle remain to specify. No files moved, deleted or provisioned.

Audit target decision recorded 18 November 2026: use entity_type/entity_id, not a dedicated id_kunjungan FK. Actor fields identify who acted; entity fields identify the affected record. Visit mutations, including their detail changes, associate history with the owning visit's target pair; employee/role events use their respective targets. Server validates controlled type and identifier, scopes history reads to the authorized target, and commits event with mutation. Generic references cannot provide a normal FK spanning multiple tables; do not claim database referential integrity without an additional mechanism. Exact SQL ID type remains open because integer business IDs and UUID sessions coexist. No new user-facing endpoint is required.

Audit actors confirmed: propose Audit Log.actor_type constrained to staff/visitor/system and nullable id_pegawai FK. Staff events require the authenticated employee ID; visitor/system events carry no fabricated staff ID. Server derives actor type/identity from the verified operation context. Enforce the conditional relationship in persistence. Visitor means an operation authorized by the public booking proof, not a verified legal identity. Keep credentials, raw tokens and passwords out of audit payloads. The booking/event FK, public event projection and retention remain separate design details; no raw audit endpoint is added solely because the table exists.

Event timezone types confirmed: actual check-in, checkout, creation and audit-event columns use PostgreSQL TIMESTAMPTZ in the proposed schema. API event fields use ISO/RFC3339 strings with explicit UTC Z or numeric offset; render in Asia/Makassar. Scheduled detail_kunjungan.tanggal remains DATE and waktu_mulai remains TIME, interpreted together in WITA; visit_duration remains integer elapsed minutes. TIMESTAMPTZ stores an instant, not the original timezone label. Existing naive timestamps require evidence of their original timezone before conversion; no conversion executed.

Checkout timestamp ownership confirmed: Kunjungan.waktu_cek_out is the single actual checkout timestamp, mapped to checkedOutAt. Omit tanggal_cek_out from the proposed design; derive calendar date using Asia/Makassar, not host timezone. Before checkout, the API visit-log value is null; after successful checkout it is the server-recorded event time. Persist status, timestamp, version and audit event atomically. Exact SQL timezone type still needs specification. Do not silently alter stored conflicting timestamps or assume no existing data.

Note ownership confirmed: API visitorInstructions → detail_kunjungan.catatan; coordinationNote → detail_kunjungan.catatan_internal. Omit duplicate note columns from the proposed Kunjungan target. Both API fields have maximum 2000 characters under current frontend validation; exact SQL storage declaration remains to be frozen. Public responses may expose visitorInstructions only through their permitted projection; coordinationNote is excluded from public and reception responses. Rejection/cancellation public reasons, internal decision notes and audit events must not overwrite these appointment notes. No migration executed.

Room presence decision confirmed: detail_kunjungan.id_ruangan is nullable. On creation, omitted/null API roomId maps to SQL NULL. On existing-visit mutations, omission performs no room assignment, explicit null sets SQL NULL, and supplied ID replaces the FK after validation. Empty string and zero are invalid IDs. Recheck conflicts against the resulting officer/time/room in the same transaction as version increment. No schema/code changes executed.

Room transport resolved: user confirmed API roomId while displaying room names. Catalog options use `{id:string,name:string,active:boolean}` from Ruangan; chosen roomId maps to detail_kunjungan.id_ruangan INTEGER FK. Booking responses retain room:string for the existing displayed label and add roomId for stable selection; the server derives the label. Do not accept an independently editable room name in commands. Null/omitted roomId clearing semantics and migration of report room-name filters still need explicit specification. This corrects the current string-array catalog adapter without changing the screen or executing code edits.

Booking version subsequently approved: `Kunjungan.version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0)`, mapped to API `version: integer`. On a visit mutation, compare the submitted expected version and increment in the same transaction as all affected Kunjungan/detail_kunjungan/detail_pengunjung updates. If no authorized row matches the expected version, return the appropriate unavailable/conflict outcome without partial writes. For a known stale authorized record, use HTTP 409 and require refresh. A read or time-derived transition to missed does not increment version because it does not mutate the record. A failed mutation does not increment it. No migration executed.

User requested adding `detail_pengunjung`, `identity_type enum`, `visit_duration time / double`, and `nama_rombongan array`. Preserve the frontend fields while specifying these additions. The structural details below are not silently approved.

| Addition | API mapping | Established frontend behavior | Still to decide |
| --- | --- | --- | --- |
| detail_pengunjung | Per-visit representative/contact snapshot | One detail record per visit; replaces Pengunjung's role in proposed diagram | Exact FK layout and restricted-visit nullability |
| identity_type enum | identityType:string | KTP, Paspor, SIM; owned by detail_pengunjung | Confirmed: omit ordinary nomor_identitas; restricted serviceId is separate |
| visit_duration | duration:integer | Whole minutes, minimum 15, maximum 240 | INTEGER minutes confirmed by user; owning table still to finalize |
| nama_rombongan array | visitors:string[] | Representative first, 1–20 names, each 2–120 characters | Owner confirmed: detail_pengunjung; proposed physical type TEXT[] |

Do not store the same roster both as independent person rows and a separately editable name array without defining the authoritative source. Do not interpret TIME as elapsed duration. Record version, restricted nullability and other existing diagram conflicts remain unresolved; this answer does not close them.

Duration confirmed 17 November 2026: SQL `visit_duration INTEGER`, JSON `duration` integer, unit minutes, range 15–240. Proposed constraint: `CHECK (visit_duration BETWEEN 15 AND 240)`. End time derives from WITA start plus elapsed minutes. Column owner/default/nullability remain separate questions; no migration executed.

Visitor detail decision confirmed 17 November 2026: one `detail_pengunjung` record per visit, storing that visit's representative details and ordered `nama_rombongan` array. No separately persisted row per companion. API `visitors` remains `string[]`; the first element equals representative `name`. Preserve order; equal names do not imply the same person and must not be silently deduplicated.

Proposed constraints, not yet executed: exactly one detail record per owning visit; non-null one-dimensional name array for ordinary registration; 1–20 elements; each trimmed name 2–120 characters; no null elements; first element matches representative. Restricted visits require a separate optional-field rule because their UI does not collect an ordinary roster. The approved ordinary roster does not authorize fabricating names for restricted visits.

The original `Pengunjung` table already contains representative/contact fields. Its future purpose remains unresolved: do not duplicate those fields as independently editable sources in both tables or silently create a reusable visitor directory. The next decision must state which table is authoritative and whether old columns are retained, moved or references only.

Resolved 17 November 2026: user selected replacement of Pengunjung's role with per-visit detail_pengunjung in the proposed diagram. No reusable visitor-profile entity or cross-visit identity matching is required. Earlier Pengunjung mappings above describe the supplied diagram, not the revised target. Target representative name, organization, email, phone, photo reference, identity type and ordered roster belong to detail_pengunjung. Exact identity-number handling remains open because the ordinary form does not collect it.

Relationship proposal: Kunjungan's existing id_pengunjung reference becomes a reference to the visit's detail_pengunjung record, with uniqueness to prevent sharing between visits. Exact FK direction/name and restricted optionality must be frozen in the revised schema; the approved decision is one-to-one ownership, not destructive DDL. Creating/updating a visit and its detail must be transactional. Existing records, if any, need a separately approved migration plan; no DROP, data movement or application changes performed.

Identity decision confirmed 17 November 2026: ordinary detail_pengunjung stores identity_type with values KTP, Paspor, SIM; no ordinary identity-number storage or API field. The diagram's required Pengunjung.nomor_identitas is omitted from the revised target, not filled with a placeholder or converted into the enum. Restricted No ID Dinas remains optional text, independent of this enum; its owning column and restricted-row nullability remain to be finalized. This decision does not authorize deleting existing identity data.

Schedule ownership confirmed 17 November 2026: detail_kunjungan is the single owner of id_pejabat, tanggal, waktu_mulai, visit_duration INTEGER minutes and id_ruangan. Omit the duplicate Kunjungan.id_ruangan from the revised design. API officerId/date/time/duration/room resolve through Kunjungan's detail-kunjungan relationship. Room and schedule changes must update that one authoritative record transactionally and recheck conflicts/version. This is a documentation proposal, not an executed column drop.

Outstanding schedule boundary: diagram id_pejabat is NOT NULL, while the Terbatas form permits no officer; Terbatas also allows no room. Ordinary public submission likewise does not choose a room; staff supplies one during approval. Column nullability and lifecycle validation must reflect those phases rather than inserting placeholder officer/room IDs.

Boundary resolved by explicit user confirmation, 17 November 2026: officer is required for ALL visits, including Terbatas; room is optional at ALL stages, including ordinary approval. Keep detail_kunjungan.id_pejabat NOT NULL and id_ruangan nullable. Earlier passages describing mandatory ordinary approval room or optional restricted officer are source-baseline observations, not the revised target. Required later frontend corrections: restricted create/edit cannot omit or clear officerId; approval/reschedule must accept blank room for ordinary visits. When a room is supplied, validate the reference and check room conflicts; officer conflict checks still apply without a room. No migration or application change performed.

Purpose length resolved 17 November 2026: user confirmed detail_kunjungan.keperluan VARCHAR(1000), replacing the diagram's VARCHAR(255) in the proposed target. API purpose remains string, trimmed length 5–1000 for ordinary visits. Reject excessive input; never silently truncate. Restricted creation does not collect purpose, so the ordinary requiredness rule must not be imposed on restricted records. No ALTER TABLE executed.

Missed status resolved 17 November 2026: omit terlewat from the proposed stored status_penerimaan enum. Preserve the six lifecycle states; displayStatus=missed derives when stored status maps to approved and serverTime > scheduled start + visit_duration. At exact end the visit is still eligible for check-in; checked_in never becomes missed. Database labels may still require explicit Indonesian/English translation, independently of this decision. API status stays approved while timing.displayStatus is missed. List/report filters and counts must use the derived display status where exposed, without double-counting approved and missed. No scheduler or status-update job is required. Existing terlewat rows, if any, require separate reconciliation; none were inspected or changed.

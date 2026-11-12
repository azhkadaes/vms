# VMS schema and field evidence

Approved planning baseline, 12 November 2026 WITA. The user explicitly approved freezing the reviewed plan, schema register and 38-operation OpenAPI, including the labelled schema/credential/SMTP/transport proposals. This approves planning only: no application edits, database/container operations, migrations or deployment. Reference-first handling and deferral of nonurgent future decisions remain in force. Documentation generator/tool selection and runtime/release gates remain deferred as recorded.

Planning evidence, 11 November 2026. This is a transcription/reconciliation register, not applied DDL. The user selected completing field/schema evidence checks before final review. No application edits or database changes are authorized.

## Source and reading method

Original attachment: C:/Users/ASUS/AppData/Local/Temp/codex-clipboard-5ad7e512-35e5-4f86-afcd-af4abe1e4699.png (7248 x 3828 RGBA). Its transparent background obscured black field text in the default viewer. A white-background inspection rendering and regional crops now make the fields readable; the original was not modified. See [inspection rendering](planning-evidence/database-diagram-white-background.png).

Read with [33 recovered original decisions](BACKEND-ALIGNMENT-DECISIONS-RECOVERED-2026-11-11.md), [current plan](BACKEND-PLAN-2026-11-11.md) and [API draft](BACKEND-API-CONTRACT-2026-11-11.md). Diagram text is evidence of the original model; selected exceptions take precedence. A field labelled string without a length does not establish a varchar limit, default or nullability. An FK alone does not establish a UNIQUE constraint.

## Transcribed diagram fields and reconciliation

Every listed table has id: int NOT NULL as its primary key. Names below preserve diagram spelling, including inconsistent or duplicated names.

| Diagram entity | Other visible fields | Reconciliation with selected requirements |
| --- | --- | --- |
| File Storage | nama: string; storage_path: varchar; tipe_file: varchar; ukuran_file: varchar | Original decisions 11/25/31/32: optional visit attachment, integer byte size, integer metadata ID plus UUID-derived private object location; no visitor-required file link. |
| Pengunjung | id_filestorage: int not null (FK); nama: string; nomor_identitas: int not null; asal_instansi: string; nomor_telpon: string; email: varchar(255); jumlah_pengunjung: int | Original decisions 2/10/11/12/14: separate visitor/visit/detail entities, ordinary identity type only with number uncollected/nullable, optional restricted identity, separate named roster and derived count; move optional file relationship to visit. |
| Detail Kunjungan | id_pejabat: int NOT NULL (FK); id_ruangan: int NOT NULL (FK); tanggal: date; waktu_mulai: time; waktu_selesai: time; keperluan: enum; keperluan_lain: varchar(255); catatan: string; catatan: string | Original decisions 9/14/19/20/26/27: nullable official for restricted visits, nullable room under approved assignment rules, start instant + duration, free-text purpose, two distinct public-instruction/internal-coordination fields. |
| Pejabat | id_jabatan: int not null (FK); nama: string; status_aktif: bool | Original decision 7: Jabatan owns VIP classification. Retain active official lifecycle; do not move VIP into an independent conflicting flag on Pejabat. |
| Jabatan | nama: string; pejabat_vip: bool | Preserve position reference and derived VIP projection. Exact directory name bounds/uniqueness not specified by diagram. |
| Pegawai Pejabat | id_pegawai: int not null (FK); id_pejabat: int not null (FK); status_aktif: boolean | Original decision 29: retain inactive assignments. D84 selects one unique employee/official pair row, toggled active/inactive with audit history. |
| Pegawai | nama: string; jabatan: string; kedeputian: string; direktorat: string; email: varchar(255); nomor_hp: string; NIP: string | Original decision 18: optional administrative profile fields; frontend adds account active/password-change/credential/version requirements. Password hash must never be exposed. |
| Ruangan | nama: string; lokasi: string; ruangan: string; status_aktif: bool | Original decision 8 retains room fields; D10 API catalog currently exposes id/name/active. D76 retains ruangan and lokasi as nullable metadata, without unconfirmed code/number semantics or extra API fields. |
| Role Pegawai | id_karyawan: int not null (FK to Pegawai); id_role: int not null (FK); direktorat: string | Original decisions 5/6 retain multi-role assignment; D54 active role belongs to session. id_karyawan is a naming inconsistency, not evidence of a new Karyawan entity. D75 retains role-level direktorat as nullable assignment metadata, initially unset, without access filtering or current UI exposure. |
| Audit Log | id_pegawai: int NOT NULL (FK); jenis_kegiatan: string; tanggal: timestamp; deskripsi: string; old_values: JSON NULL; new_values: JSON NULL; ip_address: varchar(255) | Original decisions 15/16: retain filtered description/IP/before-after, explicit actor type and nullable employee link for public/system actions. No raw credentials/secrets. |
| Kunjungan | id_pengunjung: int NOT NULL (FK); id_detail_kunjungan: int NOT NULL (FK); booking_code: char(6) UNIQUE; status_penerimaan: ENUM; tanggal_cek_in: timestamp; tanggal_cek_out: timestamp; cek_in_by: string; cek_out_by: string | Original decisions 3/21/24: eight-digit code, separate attendance events with derived detail and six current statuses. Restricted identity nullability and visit/visitor cardinality need explicit physical reconciliation. |
| Role | nama: string; created_at: timestamp; created_by: varchar; updated_at: timestamp; updated_by: varchar | Preserve role entity; four current frontend role labels. Actor-reference storage semantics/lengths are not specified by diagram. |
| Role Policy | id_policy: int NOT NULL (FK); id_role: int NOT NULL (FK); created_at: timestamp; created_by: varchar; updated_at: timestamp; updated_by: varchar; status_aktif: bool | Original decision 4 retains policy graph. Pair uniqueness, inactive-link treatment and audit actor types need explicit constraints. |
| Policy | nama: string; created_at: timestamp; created_by: varchar; updated_at: timestamp; updated_by: varchar; status_aktif: bool | Preserve policy graph; effective authority must account for active relationships. No user-facing policy-management feature is inferred from storage alone. |
| Integration Keys | nama: string; api_key: string; status_aktif: bool; tanggal_dibuat: timestamp | Original decision 22 retains provider settings/encrypted secrets model as exception; do not store plaintext API keys by copying diagram. |
| QR Kunjungan | id_kunjungan: int NOT NULL (FK); token_hash: varchar UNIQUE; tanggal_dibuat: timestamp; tanggal_expire: date; dicabut_pada: timestamp | Original decisions 17/30: exact timezone-aware expiry and nullable revoked-at. D21 selects five-minute admission QR validity, capped by check-in window; single-use check-in and replacement revocation. |
| Feature | nama: string; nama: string | Original decision 28 resolves duplicated field to one name, no separate description. |
| Permission | id_feature: int not null (FK); id_action: int not null (FK); permission_key: string; nama: string | Preserve feature/action permission graph; keys need explicit uniqueness and bounds. Public API must expose only authorized/assignable permission projections. |
| Policy Permission | id_permission: int not null (FK); id_policy: int not null (FK) | Preserve link entity. Pair uniqueness and deletion/reference constraints still need specification. |
| Action | nama: string | Preserve action dictionary; no CRUD endpoints inferred merely from table existence. |

## Relationship evidence

Visible foreign-key directions: Pengunjung -> File Storage; Detail Kunjungan -> Pejabat/Ruangan; Pejabat -> Jabatan; Pegawai Pejabat -> Pegawai/Pejabat; Role Pegawai -> Pegawai/Role; Audit Log -> Pegawai; Kunjungan -> Pengunjung/Detail Kunjungan; Role Policy -> Role/Policy; QR Kunjungan -> Kunjungan; Permission -> Feature/Action; Policy Permission -> Policy/Permission.

Crow-foot link notation is visible for several assignment/permission/QR links, and single-end markers appear on visitor/visit/detail links. Do not freeze one-to-one UNIQUE constraints from line appearance alone: reconcile restricted nullability, visit-owned personal data, the roster and repeated visitor submissions first. A one-to-many QR history remains compatible with exactly one currently usable credential enforced transactionally. Approved file-link/attendance exceptions replace their original relationship placement. Physical on-delete/on-update behavior is not specified by the image.

## Frontend field bounds already evidenced

Sources: frontendx/packages/data/src/validation.ts, management-contract.ts, wire.ts, domain.ts and admin BookingDetail.tsx; current code is consumer evidence, not proof of backend behavior.

| Field/group | Current evidence or selected rule | Freeze status |
| --- | --- | --- |
| Business IDs | Positive integer IDs encoded as decimal strings | Selected; physical integer width/constraints must match diagram/approved exceptions. |
| Booking code | Exactly eight decimal digits, leading zeroes preserved | Selected original exception. |
| Visitor/roster name | Trimmed, 2–120 in frontend validation | Retain existing behavior; D79 selects Unicode code-point counting and coordinated frontend changes. |
| Employee name | Trimmed, 2–255 in frontend employee input | Retain existing behavior; D79 selects Unicode code-point counting. |
| Organization | Ordinary minimum 2; restricted optional, supplied minimum 2; max 255 | D56 settles shared maximum. |
| Visitor phone | Exactly 8–20 ASCII digits | Existing validation; employee phone is a separate optional administrative string max 255. |
| Email | Trimmed email syntax; public-create lowercases; diagram varchar(255); frontend has no explicit maximum | D82 selects ASCII mailbox syntax and max 254 bytes; retain existing selected normalization consistently. |
| Identity type | KTP, Paspor or SIM for ordinary bookings; no ordinary identity number | Selected existing rule. |
| Restricted serviceId | Optional trimmed 1–100 when present | Existing validation; not a business record ID or ordinary identity number. |
| Roster | 1–20 names including representative first; UI companion array max 19 | Selected; conversion belongs in adapters, not a second conflicting API shape. |
| Purpose | Trimmed 5–1,000 in ordinary validation | Existing rule. Restricted creation does not require purpose. |
| Date/time | Real date; future start; frontend start window 08:00–16:00 WITA inclusive | Existing validation; D92 confirms the window constrains start only; calculated end may be later. |
| Duration | Creation defaults 60; staff approval/rescheduling 15–240 minutes | D13. |
| Action reasons | Existing minimum/requiredness retained, shared max 1,000 | D55. |
| Visitor instructions / coordination note | Optional trimmed, each max 2,000 | Existing rule; field visibility/edit lifecycle remain selected. |
| Rejection/cancellation internal notes | Optional trimmed, each max 2,000 | Existing rule; separate from public reasons. |
| Document display name | Optional, trimmed 1–255 when supplied | Existing rule; not a storage path. |
| Attachment token | Frontend permits length 16–4,096 | Representation/maximum must match issued opaque token contract; D18 one-hour expiry. |
| Privacy version | Optional in current input schema, length 1–120 | D90 keeps version optional with current-server fallback; ack:true remains required. D91 rejects stale/unknown supplied versions. |
| Employee profile | jobTitle/deputy/directorate/phone/nip optional trimmed max 255 | Existing administrative input; no implicit authorization scope. |
| Employee roles | Assigned-role enum array, at least one; legacy singular role still accepted by current frontend | New contract makes roles canonical; D57 requires coordinated outgoing adapters. |
| Password | Current frontend max 256 and metadata 8–128 are consumer evidence only | D80/D81 settle bcrypt, minimum 15 code points and maximum 72 UTF-8 bytes for new passwords; coordinated frontend change required. |
| Assigned officer IDs | Current frontend array of strings has no explicit maximum | D94 selects 0–100 unique valid/authorized decimal IDs; no truncation/partial update; preserve D84 inactive pair history. |
| Integration fields | At most 30 described fields; known field keys; variable value strings | Per-field URL/secret/numeric/email bounds need backend schema/allowlist. |
| Pagination | Booking page max 100, employee page max 250 | Existing contract; strict query keys D58, request caps D66/D67. |

## Semantic question register

G01 resolved by D75: Role Pegawai.direktorat is optional assignment metadata, initially unset, independent of employee profile directorate. No assignment-level directorate UI/API or access filter is added. Retained assignment rows preserve metadata across edits.

Further review candidates: G02 room nama/ruangan meaning; G03 visitor/detail cardinality and restricted absence; G04 audit actor references; G05 remaining string bounds/counting and email/password semantics; G06 policy/link active-state and uniqueness constraints. These are not all equivalent to new features and must be checked against recovered decisions before asking the user.

G02 resolved by D76: retain ruangan/lokasi as optional room metadata; keep the public/staff room catalog id/name/active. No automatic duplication from nama, no added UI/API field and no new uniqueness rule for an unconfirmed room code.

G03 ownership resolved by D77: visit-owned Pengunjung snapshots, no matching by personal attributes; each snapshot is referenced by at most one visit. Proposed physical enforcement is a unique visit reference to the snapshot, alongside transactional aggregate creation. D78 resolves restricted absence with no visitor row and a nullable visit FK; supplying any visitor details creates the visit-owned snapshot. No DDL has been applied.

G03 restricted absence resolved by D78: optional visit-to-visitor link for identity-free restricted visits; ordinary visits require a complete snapshot. Exact handling of later clearing all optional restricted identity fields must preserve audit/history and retention policy, not silently hard-delete historical personal data.

G05 text-counting resolved by D79: ordinary human-text bounds count code points after documented trimming; passwords and technical identifiers remain separately reviewed. Frontend native maxlength/Zod string-length checks require alignment for non-BMP text.

G05 password algorithm/maximum resolved by D80: bcrypt, 72 UTF-8 bytes maximum, no truncation or unselected pre-hashing. Existing max-256 frontend validation must change; D81 selects minimum 15 Unicode code points for new passwords. Ordinary human-text counting D79 does not override this byte ceiling.

G05 new-password bounds resolved by D81/D80: minimum 15 code points, maximum 72 UTF-8 bytes, with both constraints represented in policy metadata. Existing login credentials are still actually verified.

G05 email scope/bound resolved by D82: valid ASCII mailbox, maximum 254 bytes, including ASCII/punycode domain support; Unicode mailbox names deferred. Diagram varchar(255) storage does not authorize a 255-byte API address.

G04 actor metadata resolved by D83: Role/Policy/Role Policy created_by and updated_by use nullable integer employee FKs, replacing diagram varchar types. System/seed identity is retained in audit rather than a fabricated employee row. Audit Log still follows original decisions 15/16. Employee lifecycle/deletion constraints must preserve referential and historical integrity; no employee hard-delete feature is added.

G06 official-assignment history resolved by D84: UNIQUE(id_pegawai,id_pejabat), one retained row, activation state plus audit. Separate gap: the frontend PATCH role permissions is a flat per-role set, but Role Policy/Policy Permission permit shared reusable policies. Editing a shared policy could affect another role; ownership/mapping semantics must be selected before freezing that write contract.


G06 policy mapping resolved by D85: one active role-owned policy per role, no shared active policies in this version. Proposed constraints: UNIQUE Role Policy(role,policy), unique active role link per role, unique active role link per policy, and UNIQUE Policy Permission(policy,permission). Effective permissions require active Role Policy and active Policy; missing/inactive policy grants no permissions. Keep the owned policy stable during ordinary permission edits, updating its link set transactionally under the role lock/version. Role assignment and permission dictionaries likewise require unique valid pairs/keys; these enforce selected set semantics rather than introducing new features. Seed configuration must supply every initial role with its isolated active policy. UI policy-management remains absent.

G05 short text bounds resolved by D86: max 255 Unicode code points for listed directory/display/metadata fields, preserving all prior field-specific constraints. This completes the previously unspecified diagram short-text limits without changing the catalog projection.

G05 integration secret maximum resolved by D87: 4,096 UTF-8 plaintext bytes, preserved exactly and encrypted; overall request cap still applies. Provider field dictionaries, endpoint URL bounds and non-secret field validation remain to be completed against the actual supported adapters. Preview examples do not select providers (frontendx/API-CONTRACT.md explicitly states this).

G05 integration endpoint bound resolved by D88: 2,048 UTF-8 bytes, plus provider schema/allowlists. Non-secret generic text maximum remains open; email follows D82 and secrets D87. Numeric fields require typed provider ranges rather than arbitrary free-text acceptance.

G05 generic integration text bound resolved by D89: 255 code points after trim, provider-specific tightening allowed. URL/email/secret rules remain separate. The existing field kind number is presentation metadata; each supported adapter must define actual integer/decimal syntax and allowed range before exposing such a field. No generic unbounded number field is authorized.

G05 privacy omission resolved by D90: current published server version on omission, mandatory acknowledgment, persist source/client-or-default and commit-time association. Already committed retries retain the original resolved version. D91 rejects supplied stale/unknown versions before new creation; genuine-collection approval remains separate.

G05 supplied privacy version resolved by D91: stale/unknown explicit versions require reconfirmation, not omission fallback. A committed replay retains its original acknowledgment association; preserve and resolve the original reference before any new submission.

G05 start/end-window semantics resolved by D92: start 08:00–16:00 WITA inclusive, no 16:00 end cap; duration and full-interval resource/admission checks remain.

G05 search bound resolved by D93: q max 255 code points after trim, empty/absent means no search, clear error on excess and server scope before filtering/counts.

G07 employee assignment bound resolved by D94: submitted assignedOfficerIds contains 0–100 unique decimal business IDs. Current source management-contract.ts has no item maximum; EmployeeEditor.tsx exposes non-VIP active officials plus retained existing inactive assignments, and preserves unavailable IDs until deliberately removed. Validate new references against current catalog/authority; a retained inactive assignment is not effective runtime authority. Reject invalid, duplicate or excessive sets atomically. D84 inactive historical pair rows are independent of the submitted current-set bound.

## Physical constraint reconciliation, continuation 11 November

This register separates constraints implied by selected rules from proposed implementation details. It is not applied DDL. `Settled` means traceable to an existing decision/contract; `proposal` requires inclusion in the consolidated review and is not a new user selection. Naming and generated constraint names remain implementation details. The diagram has no physical deletion policy and no demonstrated workload sizing.

### Common representation and integrity

- Settled by D95: all 20 business entity PKs use positive PostgreSQL INTEGER, range 1–2,147,483,647; foreign-key columns use the same underlying type. Wire IDs remain canonical decimal strings. This follows diagram int rather than sihp-be/entity/base/base.go, which uses UUID keys. Reject zero, signs, decimal fractions, leading-zero aliases and overflow for business IDs; eight-digit booking codes and serviceId remain separate.
- Settled: eight-digit booking codes preserve leading zeroes and are globally unique in retained visits. UUID submission/upload/object references remain distinct; never coerce a booking code or restricted serviceId into a numeric ID.
- Settled: versioned aggregate writes require a positive expected version and atomically increment the stored version; IDs/actor metadata/computed counts cannot be assigned by clients. The safe upper bound of JSON numeric versions and exhaustion handling still needs explicit representation in OpenAPI/physical review.
- Settled: FK existence is necessary but insufficient for active/authorized catalog membership; enforce role/scope/lifecycle rules in the transactional use case. A CHECK on one row cannot establish a cross-table invariant by itself.
- Proposal: explicit NOT NULL on required values and explicit NULL on optional metadata/FKs, no empty placeholder visitor or fabricated employee. Do not default missing required input into a valid value. Server-owned defaults such as initial status, duration 60 and version initialization are selected behavior, independent of SQL DEFAULT syntax.
- Proposal: RESTRICT/NO ACTION for business references, immutable IDs and no routine hard deletion. Keep historical links when an official/employee/room becomes inactive. No CASCADE from a directory row into visits, attendance, audit or files. Any future authorized retention purge needs its own ordered deletion/anonymization plan; this proposal does not select retention periods or disable later privacy obligations.

### Entity constraints and cardinality

| Entity | Settled aggregate/cardinality/constraints | Physical detail still proposed or pending |
| --- | --- | --- |
| File Storage | Integer metadata PK, unique UUID private object identifier; byte size positive and <= 5 MiB for accepted upload; optional visit link, private storage and link/cleanup coordination | Unique linked file per visit and at most one visit per accepted upload back the single-file contract; stage/accepted/link/cleanup representation belongs to support/storage design. MIME allowlist PDF/JPEG/PNG. Display name <=255; object paths never client authority. |
| Pengunjung | Visit-owned snapshot; referenced by at most one visit (D77). No row for identity-free restricted visit (D78). No uniqueness on name/email/phone and no cross-booking matching | UNIQUE nullable Kunjungan visitor FK; ordinary required-field completeness checked transactionally. Optional restricted identity clearing preserves audit/history; retention semantics remain a production gate. Ordinary identity number remains NULL. |
| Detail Kunjungan | Separate visit-owned detail with start instant/duration; optional official for restricted and optional room under selected rules; distinct instructions/coordination notes <=2000 each | UNIQUE visit-to-detail FK and required detail FK proposed to enforce one detail per visit. Timestamptz plus duration; full interval [start,end), Asia/Makassar projection. Cross-row resource conflict remains transactional, not a simple UNIQUE(start). |
| Pejabat | Required Jabatan FK; active state; VIP derived through Jabatan; display name <=255 | No uniqueness on display name is evidenced. Deactivation does not erase references or VIP history; historical classification behavior needs review before claiming snapshot semantics. |
| Jabatan | Owns VIP classification; required nonblank name <=255 | Position-name uniqueness not established by diagram; do not silently merge same-named positions. |
| Pegawai Pejabat | Required employee/official FKs; UNIQUE pair across active/inactive states; reactivate same row, audit changes (D84) | Active state required. Effective authority additionally checks current employee/official/role eligibility. Assignment replacement is transactional and submitted set <=100 (D94). |
| Pegawai | Required name/email/account state/credential linkage; profile fields nullable <=255; no password/hash in responses; deactivation rather than hard-delete API | Normalized login identity uniqueness required to avoid ambiguous authentication; exact private credential storage placement and version width remain to freeze. No uniqueness on optional NIP/phone/name is selected. |
| Ruangan | Stable integer identity, active state, required name <=255; nullable ruangan/lokasi <=255 (D76/D86) | No room-code/name uniqueness is inferred; do not derive optional metadata from name. Preserve metadata on unrelated updates. |
| Role Pegawai | Required employee/role FKs, unique pair/set semantics; optional assignment directorate <=255 (D75); one selected active role per session, not per employee | Retained links preserve metadata. Do not add a UNIQUE(employee) constraint that prevents multiple roles. Removal storage history is not specified by D84, which applies to official assignments. |
| Audit Log | Explicit actor kind; nullable employee FK for public/system; filtered descriptions/IP/before-after; transactionally coupled to successful writes | Actor-kind enum/check and IP storage representation need physical specification; do not invent public employee IDs or log credentials. Audit append-only permissions and retention remain operational review items. |
| Kunjungan | Required detail; nullable unique visitor link; standard completeness/restricted absence; unique 8-digit code; six stored statuses only; positive version; optional file link | Visit kind required to enforce variant policy. Approved/checked-in commitments follow selected full-interval rules. Derived Missed is never a persisted seventh status. Privacy acknowledgment association belongs to ordinary public creation only. |
| Role | Four current role names; versioned permission aggregate; actor metadata nullable employee FKs (D83) | Unique canonical role key/name required for unambiguous role matching. Description is a display DTO, not a new Feature description. No role CRUD added. |
| Role Policy | Required role/policy FKs; unique pair, one active policy per role and active owner per policy (D85); nullable actor employee FKs (D83) | Partial unique indexes proposed for active links. Active link alone grants nothing if Policy inactive; seed each role with an isolated policy. |
| Policy | Required nonblank name <=255; active flag; nullable actor employee FKs | No shared active role-owned policy (D85); no separate user-facing CRUD. Missing/inactive policy grants no permissions. Name uniqueness not independently established. |
| Integration Keys | Provider settings/encrypted secrets exception; secret plaintext <=4096 bytes, URL <=2048 bytes, generic text <=255 code points; no secret responses | Provider identifier/key allowlist and typed numeric ranges still need supported-provider evidence. Ciphertext column must hold encryption overhead. No default plaintext api_key column. |
| QR Kunjungan | Required visit FK; unique credential hash; issuance/expiry instants, nullable revocation instant; 5-minute validity capped by admission window; history retained | One currently usable admission credential enforced under visit lock; replacement revokes predecessor. Exact opaque format/hash encoding and checkout-eligible credential path remain to freeze, not inferred from hash uniqueness. |
| Feature | One nonblank name <=255; no duplicate description field | Stable dictionary identity/unique machine key mapping must be reconciled with permission keys; no CRUD inferred. |
| Permission | Required Feature/Action FKs; unique machine permission key; display name <=255; dictionary/assignability controlled by server | Feature/action pair uniqueness proposed for one permission per combination. Current 14 API permission keys bound the emitted set; machine-key syntax/length must be explicit. |
| Policy Permission | Required policy/permission FKs; unique pair (D85); role-owned set changed atomically | Link deletion is part of selected set replacement, not permission/dictionary deletion. FK deletion policy must not propagate into unrelated roles. |
| Action | Nonblank name <=255; server dictionary | Canonical action identity/key uniqueness requires dictionary mapping; display-name uniqueness is not proven by diagram. |

### Lifecycle evidence checked in this continuation

Current frontend domain.ts enumerates exactly pending/approved/rejected/cancelled/checked_in/checked_out. attendance.ts derives Missed only for approved and now > end, opens check-in at start minus 30 minutes, includes end in check-in eligibility and permits checkout whenever checked_in. It allows public cancellation only while Pending/Approved and strictly before start. These independent source checks agree with the API lifecycle table; they are not runtime/backend verification.

Current management-contract.ts supplies an unbounded roles array of four enum values and unbounded permission-string arrays. Set semantics imply unique roles (1–4), permissions restricted to the server's current dictionary, and no grants from unsupported keys. OpenAPI should encode the existing finite sets and uniqueItems; a new permission must require an intentional contract update rather than being accepted as an arbitrary string.

Sources rechecked: frontendx/packages/data/src/{management-contract,management,validation,wire,domain,session-contract,attendance,history}.ts; frontendx/apps/admin/src/features/EmployeeEditor.tsx; sihp-be/internal/entity/base/base.go. PostgreSQL INTEGER/BIGINT ranges verified against [PostgreSQL 18 numeric types](https://www.postgresql.org/docs/18/datatype-numeric.html). No SQL, service, container, migration or application test was executed.

### D96 QR consumption versus revocation

Selected 12 November: retain a distinct admission-consumed timestamp/state linked to the successful check-in event; do not encode consumption solely by setting dicabut_pada. The consumed, nonrevoked credential remains a checkout-only proof for its own checked-in visit even after admission expiry. Resolve must apply the correct status-specific eligibility rather than always reject expired admission tokens. Revocation/replacement blocks both admission and checkout. Successful checkout and version/state checks prevent another checkout. Manual code remains available. Credential history is private and retained with the visit under the existing retention gate; no raw token is stored in audit or ordinary projections. This is a storage extension supporting the selected operation, not an added endpoint.

### D97/D98 version and provider reconciliation

D97 resolves version width: positive PostgreSQL INTEGER, 1–2,147,483,647; mutation at maximum fails without increment/wrap/reset, requiring an explicit upgrade plan. The same bounded JSON numeric version applies to visits/employees/roles/integrations; session authority revision is a separate server counter requiring safe numeric serialization.

D98 selects authenticated SMTP, not the generic preview endpoint examples. The concrete field dictionary is host/port/tlsMode/username/password/sender; see the API contract for bounds, strict keys, merge/clear semantics and verified TLS. Port is a typed canonical decimal string 1–65535 despite the editor number kind. Only password is secret and clearable. Effective merged configuration must be complete before enablement. Integration settings remain encrypted/redacted; exact encryption/key lifecycle belongs to security implementation review. Storage provider is deliberately deferred, not an unresolved invitation to accept arbitrary fields.

### D99 upload replay and remaining credential shapes

D99 selects deterministic secret-key derivation for accepted-upload replay. Immutable support metadata contains a cryptographically random nonce, key version, accepted file/content identity, request reference, audience/owner binding and fixed issuance/expiry. Retain the matching dedicated key through all its live upload-token expiries. Never persist a recoverable upload token or regenerate it using a newer key/expiry. Store a private token digest for lookup/linking; a committed booking replay remains independent of a now-expired upload credential. Refresh rotation still follows D43's separate 10-second encrypted successor recovery.

Proposed technical representation for consolidated review (not additional user selections): 32 random bytes for admission/refresh secrets; unpadded base64url; versioned purpose prefixes; SHA-256 token digests encoded as 64 lowercase hex characters for unique private lookup. High-entropy random secrets do not need password hashing; employee credentials still use D80 bcrypt. Upload token MAC uses HMAC-SHA-256 over versioned, length-delimited immutable metadata with a separate key and bounded key-version ID. Credential prefixes prevent confusion with an eight-digit manual code. No meaningful person/booking identifier is embedded in opaque browser tokens. Reject unsupported formats/purposes, malformed encoding and excess before lookup; normal responses never expose digests or object metadata. Exact prefix/cookie/header names are routine review conventions, not business choices already approved.

Credential timestamps are timezone-aware instants. Required created/expiry values have expiry>created; revocation/consumption is nullable and cannot precede issuance. Token ownership, visit state, admission window, key lifetime and D96 checkout eligibility require transactional checks; cross-table correctness is not claimed from a row CHECK alone. Keep distinct consumed-for-admission and revoked states. Do not expire the checkout-only eligibility solely because admission expiry passed, or use successful checkout to authorize another attendance event.

Restricted snapshot clearing proposal: D78 prevents an empty row at initial identity-free creation. Once a real visit-owned snapshot exists, clear optional current fields without hard-deleting historical evidence or matching another visitor; retain the established row and audit cleared field changes under the retention gate. This distinguishes later empty current details from inventing a placeholder identity at creation. Responses omit absent fields. No retention duration or purge job is selected here.

### Lifecycle reconciliation completed for current operations

Current domain.ts can() allows reschedule only for Approved with displayStatus not Missed, whereas the earlier planning table proposed Pending/Approved. This was surfaced explicitly, not treated as inherited behavior. D100 selects Pending/Approved excluding Missed, requiring a coordinated frontend policy change. Pending remains nonreserving; Approved retains full-interval locks/conflict checks. Future new start and current decision scope remain required. Restricted Edit detail independently permits Pending/Approved through Protocol create authority and validates the submitted future schedule (BookingDetail.tsx and API-CONTRACT.md staff-action addendum). Other transition guards remain in API section 5.

### Field-storage/default/deletion proposals for consolidated review

These fill implementation details without claiming they came from the diagram or a separately selected business rule:

| Field class | Physical proposal / enforcement |
| --- | --- |
| Business id and FK | INTEGER PK generated by identity; CHECK id>0; matching INTEGER FKs, NOT NULL except documented optional links/actors. Identity gaps are normal; business codes are unrelated. |
| Visit/employee/role/integration version | INTEGER NOT NULL DEFAULT 1, CHECK 1..2147483647 (D97). Conditional compare/increment; exhausted counter fails before update/audit/outbox, never wraps. |
| Stored visit status/kind | Six-value PostgreSQL status ENUM NOT NULL; kind standard/restricted NOT NULL. No generic default permitting the wrong audience; creation use case writes Pending/public or Approved/staff explicitly. Kind is immutable through action APIs. |
| Visit schedule | Detail start_at TIMESTAMPTZ NOT NULL, duration INTEGER NOT NULL DEFAULT 60, CHECK 15..240. Wire date/time derived in Asia/Makassar. Purpose nullable for restricted, otherwise trimmed5..1000; instructions/coordination nullable max2000. Public reject/cancel reasons and internal notes are distinct persisted fields. |
| Ordinary/restricted visitor fields | Name nullable globally, otherwise2..120; organization nullable globally, otherwise2..255; email nullable globally, otherwise D82; phone nullable globally, otherwise8..20 ASCII digits. Ordinary aggregate requires all ordinary fields; restricted only its allowed optional identity fields, no ordinary contact placeholders. Restricted serviceId is text1..100 preserving zeros; ordinary identity type enum KTP/Paspor/SIM; identity number remains NULL/uncollected. |
| Roster and count | Separate ordered visit-owned roster extension with1..20 ordinary names, name2..120, unique(visit,ordinal), representative first. Do not UNIQUE(name); different people may share names. Restricted has no roster. Count is derived, replacing Pengunjung.jumlah_pengunjung rather than a second writable field. |
| Attendance | Separate retained event extension, required visit/actor/time/action, unique(visit,attendance kind) for one check-in and one checkout. Check-in precedes checkout under visit lock; derived actors/timestamps replace Kunjungan duplicated string/timestamp attendance columns. Timestamps TIMESTAMPTZ; valid actor FK cannot be supplied by client. |
| Nullable metadata | Employee profile and D75/D76 metadata are nullable VARCHAR(255); absence is NULL, editable clear normalizes empty to absence. Required names use nonblank trimmed VARCHAR(255), except tighter named visitor/employee limits. Do not invent UNIQUE on display names, NIP or phone. |
| Dictionary/link keys | Unique canonical role identity; current4 role names. Permission key unique within fixed14-key server dictionary, stored VARCHAR(64) with ASCII machine-key validation; permission Feature/Action unique pair. Feature/Action names nonblank <=255, dictionaries seeded explicitly. Policy Permission and Role Pegawai unique pairs. Role/permission arrays are unique finite sets, not unbounded arbitrary strings. |
| Link activation/defaults | Official assignment unique pair persists, boolean active NOT NULL explicitly written; removed pairs deactivate (D84). Role Policy active/policy active NOT NULL explicitly written; unique active role owner/policy (D85). No implicit active=true granting authority. Role-assignment removal and policy-permission replacement may delete the specific join rows transactionally with audit; parent rows/references never cascade-delete. Retained Role Pegawai rows preserve directorate metadata. |
| Actor/timestamps | Role/Policy/Role Policy created_by/updated_by nullable INTEGER employee FKs; created_at/updated_at TIMESTAMPTZ NOT NULL. Server/database creation time initialization, explicit update timestamps; clients cannot assign actors. Audit captures system/public kinds with no fabricated employee. |
| Audit fields | Required server-owned action code VARCHAR(64), recorded_at TIMESTAMPTZ, safe generated description TEXT; IP remains nullable VARCHAR(255), canonical validated trusted address only, not a forwarded-header dump. Filtered before/after JSONB nullable, allowlisted per operation and bounded by affected finite fields/request limits. Never full raw request bodies, secrets or tokens. No new audit endpoint is inferred. |
| File fields | Name VARCHAR(255); MIME allowlist stored in VARCHAR(255); byte_size INTEGER CHECK 1..5242880; UUID object_id UNIQUE NOT NULL, object location derived rather than public storage_path. File relationship nullable on visit, one accepted upload links at most one visit. Filename/MIME/content are validated, never path authority. |
| QR fields | Visit FK NOT NULL; token_hash CHAR(64) UNIQUE NOT NULL for proposed SHA-256 hex representation; created_at/expires_at TIMESTAMPTZ NOT NULL, expires>created; revoked_at and consumed_at nullable TIMESTAMPTZ. D96 consumption links successful check-in and permits checkout-only use after admission expiry; revocation blocks both. |
| Integration storage | Integer identity/version, name<=255, enabled explicitly set; strict SMTP non-secret JSONB settings plus authenticated encrypted secret envelope/private key version. Never store diagram plaintext api_key. JSONB object shape is constrained by server schema; enabling validates complete merged settings. Ciphertext capacity includes authentication tag/nonce/key metadata beyond4096 plaintext bytes. |
| Technical references | Native UUID for upload/submission/object references; unique scoped request reference plus immutable payload digest. Opaque upload/admission/refresh formats are versioned/purpose-bound and bounded within existing wire maxima; keys/encoded digests never enter display fields. Named internal events/permission keys are bounded ASCII machine codes, not human text. |
| FK update/delete | NO ACTION/RESTRICT, immutable parent IDs; no directory delete operation. Deactivate referenced directory/account rows. Explicit join-set updates do not delete their parents. No cascading visit/audit/attendance/file loss. Any future retention purge/anonymization requires separate approval and reviewed ordered handling of support/idempotency history. |

Defaults do not substitute for required request validation. Cross-entity invariants (ordinary completeness, representative/roster count, one detail/snapshot owner, current active policy, required enabled SMTP settings) require aggregate transaction logic plus the stated unique/FK backstops. A future schema/migration implementation must verify its actual DDL against this register; none exists or has run in this planning assignment. Directory classification/history changes have no new CRUD route in this version; introducing one requires a separate history/authority review rather than silently recomputing past business decisions.

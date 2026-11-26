# Physical schema and per-table migrations

18 November 2026, WITA. This is the build target for the [backend spec](../specs/backend-api-2026-11-18/spec.md). It follows the user's database diagram strictly. It departs from the diagram only where an earlier agreement says otherwise; every departure is cited in the **Source** column.

Decisions recorded today (user, 18 November 2026):

1. **Rebuild the migrations, one file per table.** The four existing multi-table migrations are replaced. Each table is created directly in its final shape. The stage-1 database holds only synthetic data; the user resets its schema and re-runs migrations.
2. **Keep the agreed Jabatan hierarchy.** The diagram's `Pejabat.nama_jabatan` and `Pejabat.pejabat_vip` live on `jabatan` (`nama`, `pejabat_vip`). `jabatan` links to `direktorat`/`kedeputian`, as agreed on 15 November ([employee organization](EMPLOYEE-ORGANIZATION.md)).
3. **Integration Keys is on hold.** The diagram's table is not created in this build. It returns with the Integrations screen and API.
4. **The admission QR is static.** Each visit has one stable QR payload until a reschedule or cancellation revokes it. It replaces the earlier five-minute rotating credential (D21). The QR used to check in can be presented again to check out. A stable payload lets a later server-side renderer (piglig/go-qr) produce the same image every time; rendering stays client-side in this build.

Source abbreviations:
- **Diagram**: the user's DB design.
- **Dnn**: decision ticket nn in the [decision map](../specs/frontend-driven-api-2026-11-16/map.md).
- **Org**: the 15 November organization agreement.
- **Contract**: required by the agreed REST operations.
- **Plan**: the earlier [backend plan](../BACKEND-PLAN-2026-11-11.md), which agreed that operational records live in a separate `support` schema.

## Rules applied to every migration

- **One file per table:** `migration/files/<timestamp>_create_<schema>_<table>.sql`, in the order below.
- **Up/Down:**
  - Up creates the table, its indexes, its reference seed (if any) and its runtime grants.
  - Down revokes those grants and drops only that table, plus any ENUM type the table introduced.
- **Prerequisite migration:** one migration before all tables creates the `vms` and `support` schemas and their usage grants. It is the only non-table migration.
- **Type mapping from the diagram:**

  | Diagram type | SQL type | Basis |
  | --- | --- | --- |
  | `int NOT NULL` PK | `INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY CHECK (id > 0)` | existing convention |
  | `int` | `INTEGER` | |
  | `string`, and `varchar` without a length | `VARCHAR(255)`, with a trimmed-length `CHECK` where the API constrains it | project default |
  | `varchar(n)`, `char(n)` | as written, unless a decision sets a different length | |
  | `bool`/`boolean` | `BOOLEAN` | |
  | `date`, `time` | `DATE`, `TIME` (WITA wall time) | D18 |
  | `timestamp`, `datetime` | `TIMESTAMPTZ` | D18 |
  | `JSON` | `JSON` | |
  | `ENUM` | a named PostgreSQL `ENUM` type | |

  ENUM types are also used where an agreement calls a column an enum (D09 `identity_type`). Other agreed value sets use `VARCHAR … CHECK (… IN (…))`.
- **Diagram duplicates:** `Pejabat.nama` and `Form Documents.ukuran_file_bytes` each appear twice in the diagram. Each becomes a single column.
- **Foreign keys:** `ON UPDATE RESTRICT ON DELETE RESTRICT`, and an index on every FK column.
- **Audit columns:** `created_by`/`updated_by` are `VARCHAR(255)` per the diagram. They hold the acting employee ID as decimal text, or `system` for seeds and migrations. They cannot be FKs.
- **Grants:** each table's grants cover what the runtime role `vms_api` needs for every operation in the spec, and nothing more. Later tickets need no grant migrations.

## `vms` schema (business tables)

### 1. `kedeputian` (Org)
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Org |
| kode | VARCHAR(16) | NOT NULL UNIQUE | Org |
| nama | VARCHAR(255) | NOT NULL UNIQUE, trimmed 1–255 | Org |

Seed: the 7 deputy groups from the existing migration. Grant: SELECT.

### 2. `direktorat` (Org)
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Org |
| id_kedeputian | INTEGER | NOT NULL FK → kedeputian | Org |
| nama | VARCHAR(255) | NOT NULL UNIQUE | Org |

Seed: the 21 directorates. Grant: SELECT.

### 3. `jabatan` (Org; carries the diagram's `nama_jabatan` and `pejabat_vip`)
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Org |
| nama | VARCHAR(255) | NOT NULL | Diagram `Pejabat.nama_jabatan` |
| pejabat_vip | BOOLEAN | NOT NULL | Diagram `Pejabat.pejabat_vip` |
| id_direktorat | INTEGER | FK → direktorat, nullable | Org |
| id_kedeputian | INTEGER | FK → kedeputian, nullable | Org |

Table check: not both organization FKs. Grant: SELECT.

### 4. `pejabat`
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| id_jabatan | INTEGER | NOT NULL FK → jabatan | Org |
| nama | VARCHAR(255) | NOT NULL, trimmed 1–255 | Diagram |
| status_aktif | BOOLEAN | NOT NULL | Diagram |

Grant: SELECT.

### 5. `ruangan`
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| nama | VARCHAR(255) | NOT NULL, trimmed 1–255 | Diagram |
| lokasi | VARCHAR(255) | nullable | Diagram |
| status_aktif | BOOLEAN | NOT NULL | Diagram |

The existing extra `ruangan` column is **removed**: it is not in the diagram and has no agreement. Grant: SELECT.

### 6. `pegawai`
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| nama | VARCHAR(255) | NOT NULL, trimmed 2–255 | Diagram |
| jabatan | VARCHAR(255) | nullable | Diagram |
| kedeputian | VARCHAR(255) | nullable | Diagram |
| direktorat | VARCHAR(255) | nullable | Diagram |
| email | VARCHAR(255) | NOT NULL UNIQUE, stored lower-case and trimmed | Diagram |
| nomor_hp | VARCHAR(255) | nullable | Diagram |
| nip | VARCHAR(255) | nullable | Diagram `NIP` |
| status_aktif | BOOLEAN | NOT NULL | Contract (employee `active`, ops 26–29) |
| must_change_password | BOOLEAN | NOT NULL | Contract (`requirePasswordChange`, op 36 flow) |
| version | INTEGER | NOT NULL DEFAULT 1, > 0 | Contract (employee `version`) |

Grants: SELECT, INSERT, UPDATE.

### 7. `role`
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| nama | VARCHAR(255) | NOT NULL UNIQUE, CHECK IN (PA, Protocol Officer, Receptionist, Super Admin) | Diagram; fixed role set (Contract ops 30–31) |
| version | INTEGER | NOT NULL DEFAULT 1, > 0 | Contract (role `version`) |
| created_at | TIMESTAMPTZ | NOT NULL | Diagram |
| created_by | VARCHAR(255) | NOT NULL | Diagram |
| updated_at | TIMESTAMPTZ | NOT NULL | Diagram |
| updated_by | VARCHAR(255) | NOT NULL | Diagram |

Seed: the 4 roles. Grants: SELECT, UPDATE (version and audit columns).

### 8. `pegawai_pejabat`
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| id_pegawai | INTEGER | NOT NULL FK → pegawai | Diagram |
| id_pejabat | INTEGER | NOT NULL FK → pejabat | Diagram |
| status_aktif | BOOLEAN | NOT NULL | Diagram |

Unique `(id_pegawai, id_pejabat)`; partial unique: one active row per pegawai (Org). Grants: SELECT, INSERT, UPDATE.

### 9. `role_pegawai`
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| id_karyawan | INTEGER | NOT NULL FK → pegawai | Diagram (renamed from the existing `id_pegawai`) |
| id_role | INTEGER | NOT NULL FK → role | Diagram |
| direktorat | VARCHAR(255) | nullable | Diagram |

Unique `(id_karyawan, id_role)`. Grants: SELECT, INSERT, DELETE.

### 10–11. `feature`, `action`
Columns: `id` PK and `nama VARCHAR(255) NOT NULL` (Diagram). Seed: the features and actions behind the 14 permission keys. Grant: SELECT.

### 12. `permission`
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| id_feature | INTEGER | NOT NULL FK → feature | Diagram |
| id_action | INTEGER | NOT NULL FK → action | Diagram |
| permission_key | VARCHAR(255) | NOT NULL UNIQUE, CHECK IN the 14 keys | Diagram; key set (Contract) |
| nama | VARCHAR(255) | NOT NULL | Diagram |

Unique `(id_feature, id_action)`. Seed: the 14 permissions. Grant: SELECT.

### 13. `policy`
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| nama | VARCHAR(255) | NOT NULL | Diagram |
| created_at, updated_at | TIMESTAMPTZ | NOT NULL | Diagram |
| created_by, updated_by | VARCHAR(255) | NOT NULL | Diagram |
| status_aktif | BOOLEAN | NOT NULL | Diagram |

Seed: one default policy per role. Grants: SELECT, UPDATE.

### 14. `role_policy`
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| id_policy | INTEGER | NOT NULL FK → policy | Diagram |
| id_role | INTEGER | NOT NULL FK → role | Diagram |
| created_at, updated_at | TIMESTAMPTZ | NOT NULL | Diagram |
| created_by, updated_by | VARCHAR(255) | NOT NULL | Diagram |
| status_aktif | BOOLEAN | NOT NULL | Diagram |

Partial uniques: one active row per role and one per policy, as today. Seed: link each role to its default policy. Grant: SELECT.

### 15. `policy_permission`
Columns: `id` PK, `id_permission` NOT NULL FK → permission, `id_policy` NOT NULL FK → policy (Diagram). Unique `(id_policy, id_permission)`.

Seed: the default grants per role, taken from the frontend's role defaults:
- PA and Protocol Officer: bookings, create, approve, reject, reschedule, cancel, reports, reportExport
- Receptionist: bookings, checkin, checkout
- Super Admin: bookings, people, roles, integrations, reports, reportExport, audit

Grants: SELECT, INSERT, DELETE.

### 16. `audit_log`
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| actor_type | VARCHAR(16) | NOT NULL, CHECK IN (staff, visitor, system) | D19 |
| id_pegawai | INTEGER | FK → pegawai; required when `actor_type = staff`, null otherwise | Diagram; nullability D19 |
| entity_type | VARCHAR(32) | NOT NULL, CHECK IN (kunjungan, pegawai, role) | D20 |
| entity_id | INTEGER | NOT NULL | D20 |
| jenis_kegiatan | VARCHAR(255) | NOT NULL | Diagram |
| tanggal | TIMESTAMPTZ | NOT NULL | Diagram, D18 |
| deskripsi | VARCHAR(255) | NOT NULL | Diagram |
| old_values | JSON | nullable | Diagram |
| new_values | JSON | nullable | Diagram |
| ip_address | VARCHAR(255) | nullable | Diagram |

Index on `(entity_type, entity_id, tanggal)`. Grants: INSERT, SELECT (visit history and derived fields).

### 17. `integration_keys` (on hold)
Not created in this build (decision 3 above). When it returns, it follows the diagram:
- `id` PK
- `nama`
- `api_key`, stored encrypted and never plaintext
- `status_aktif`
- `tanggal_dibuat` (TIMESTAMPTZ)

It takes migration slot 17 in the order.

### 18. `form_documents` (diagram "Form Documents"; the agreements call it "Uploads") — ticket 05
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| nama | VARCHAR(255) | NOT NULL, original file name, trimmed 1–255 | Diagram |
| storage_path | VARCHAR(255) | NOT NULL UNIQUE; private server-generated storage key, never a client path or URL | Diagram; meaning per D21 |
| mime_type | VARCHAR(255) | NOT NULL, CHECK IN (application/pdf, image/jpeg, image/png) | Diagram |
| ukuran_file_bytes | INTEGER | NOT NULL, 1–5,242,880 | Diagram (one column) |
| purpose | VARCHAR(16) | NOT NULL, CHECK IN (document, visitorPhoto); document ⇒ pdf, visitorPhoto ⇒ jpeg/png | D22 |

`file_data` is omitted (D21). Grants: SELECT, INSERT.

### 19. `detail_kunjungan` — ticket 06
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| id_pejabat | INTEGER | NOT NULL FK → pejabat | Diagram; D10 |
| id_ruangan | INTEGER | FK → ruangan, **nullable** | Diagram; nullability D10 |
| id_formdocuments | INTEGER | FK → form_documents, nullable, UNIQUE; the Nota Dinas | Diagram |
| tanggal | DATE | NOT NULL (WITA) | Diagram |
| waktu_mulai | TIME | NOT NULL (WITA) | Diagram |
| visit_duration | INTEGER | NOT NULL, 15–240 minutes | D09 |
| keperluan | VARCHAR(1000) | nullable; ordinary visits require 5–1000 trimmed (usecase) | Diagram; length D11 |
| catatan | VARCHAR(2000) | nullable; visitor instructions | Diagram; D16 |
| catatan_internal | VARCHAR(2000) | nullable; coordination note | Diagram; D16 |
| no_nd | VARCHAR(100) | nullable; restricted No ID Dinas | Diagram; type D28 |

Grants: SELECT, INSERT, UPDATE.

### 20. `detail_pengunjung` (replaces the diagram's Pengunjung, D09/D26/D27) — ticket 06
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram (Pengunjung) |
| nama | VARCHAR(255) | nullable; trimmed 2–120 when present | Diagram |
| asal_instansi | VARCHAR(255) | nullable; trimmed ≥2 when present | Diagram |
| nomor_telpon | VARCHAR(255) | nullable; 8–20 digits when present | Diagram |
| email | VARCHAR(255) | nullable; lower-case | Diagram |
| identity_type | ENUM `identity_type` (KTP, Paspor, SIM) | nullable | D09 |
| nama_rombongan | TEXT[] | nullable; 1–20 elements, none null, each 2–120 | D09 |
| jumlah_pengunjung | INTEGER | nullable; equals `cardinality(nama_rombongan)` when the roster is present | Diagram |
| id_formdocuments | INTEGER | FK → form_documents, nullable, UNIQUE; the representative photo | Photo reference D09; name follows the diagram's FK convention |

`nomor_identitas` is omitted (D09). Requiredness depends on visit kind and is enforced in the create transaction (D27). Grants: SELECT, INSERT, UPDATE.

### 21. `kunjungan` — ticket 06
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| id_detail_pengunjung | INTEGER | NOT NULL UNIQUE FK → detail_pengunjung | Diagram `id_pengunjung`, retargeted by D26/D27 |
| id_detail_kunjungan | INTEGER | NOT NULL UNIQUE FK → detail_kunjungan | Diagram; UNIQUE D26 |
| kode_kunjungan | CHAR(8) | NOT NULL UNIQUE, eight digits | Diagram |
| tipe_kunjungan | ENUM `tipe_kunjungan` (umum, vip, terbatas) | NOT NULL; snapshot at creation | Diagram |
| status_penerimaan | ENUM `status_penerimaan` (pending, approved, rejected, cancelled, checked_in, checked_out) | NOT NULL | Diagram; values D23 |
| version | INTEGER | NOT NULL DEFAULT 1, > 0 | D13 |
| waktu_cek_in | TIMESTAMPTZ | nullable | Diagram, D18 |
| waktu_cek_out | TIMESTAMPTZ | nullable | Diagram, D17, D18 |
| tanggal_dibuat | TIMESTAMPTZ | NOT NULL | Diagram, D18 |
| alasan_penolakan | VARCHAR(1000) | nullable | D25 |
| catatan_internal_penolakan | VARCHAR(2000) | nullable | D25 |
| alasan_pembatalan | VARCHAR(1000) | nullable | D25 |
| catatan_internal_pembatalan | VARCHAR(2000) | nullable | D25 |

`tanggal_cek_out` is omitted (D17). Table checks:
- `checked_in` ⇒ check-in time set and checkout time null;
- `checked_out` ⇒ both set, and checkout ≥ check-in;
- other statuses ⇒ neither set.

Grants: SELECT, INSERT, UPDATE.

### 22. `qr_kunjungan` — ticket 08
| Column | Type | Rule | Source |
| --- | --- | --- | --- |
| id | PK | | Diagram |
| id_kunjungan | INTEGER | NOT NULL FK → kunjungan | Diagram |
| token_hash | VARCHAR(255) | NOT NULL UNIQUE; SHA-256 hex of the payload | Diagram |
| tanggal_dibuat | TIMESTAMPTZ | NOT NULL | Diagram |
| tanggal_expire | TIMESTAMPTZ | NOT NULL, after `tanggal_dibuat`; the visit's scheduled end (end of the check-in window) | Diagram; decision 4 above |
| dicabut_pada | TIMESTAMPTZ | nullable | Diagram |

Partial unique index: one unrevoked credential per visit. Grants: SELECT, INSERT, UPDATE (`dicabut_pada`).

**Static credential lifecycle** (decision 4):

- **Issue:** the first admission request for an approved, not-yet-ended visit creates the row. Every later request returns the **same** payload and the same `tanggal_expire` without writing anything.
- **Payload:** derived deterministically from the credential row and a server secret, e.g. credential ID plus an HMAC over credential ID and visit ID. The payload is never stored; only its hash is. Re-deriving it keeps the QR static without keeping a plaintext secret.
- **Revoke:**
  - **Reschedule** with a changed date, time or duration, and **cancellation**, set `dicabut_pada`. The next admission request issues a new static credential with the new scheduled end.
  - A notes-only or room-only edit keeps the credential.
- **Check-in:** check-in accepts the credential until `tanggal_expire`.
- **Checkout:** the same credential checks the visit out while it is `checked_in`, on any date and after `tanggal_expire`. A checked-out visit can no longer be resolved for attendance.

## `support` schema (operational records, Plan)

The first six tables already exist and are rebuilt one per file without column changes:

| # | Table | Purpose | Ticket |
| --- | --- | --- | --- |
| 23 | `rate_limit_buckets` | Limiter buckets (D29–D33) | 00 |
| 24 | `rate_limit_events` | Limiter events | 00 |
| 25 | `employee_credentials` | Password hashes | 00 |
| 26 | `staff_sessions` | Sessions, active role, expiry | 00 |
| 27 | `refresh_lineage` | Refresh-secret reuse detection | 00 |
| 28 | `refresh_recovery` | Uncertain refresh recovery | 00 |
| 29 | `form_document_uploads` | Upload lifecycle bound to `form_documents.id`: audience, client reference UUID, owner digest, sha256, token hash (unique), `created_at`, `expires_at` (+1 h, D18), `linked_at`, uploader employee. Unique `(audience, owner digest, reference)` | 05 |
| 30 | `booking_requests` | Booking idempotency and submission recovery (D22): audience, owner digest, reference UUID, payload digest, email digest, state (committed / closed), visit FK. Unique `(audience, owner digest, reference)` | 06 |

## Values derived instead of stored

No column was added to a diagram table without an agreement. These API values are therefore derived:

| API value | Derived from |
| --- | --- |
| `createdBy {name, role}` | The visit's creation `audit_log` event: the staff actor's `pegawai.nama`, and the active role recorded in `new_values` |
| `rescheduled` | A reschedule event exists in `audit_log` for the visit |
| Acknowledged privacy version | Recorded in the creation event's `new_values` |
| History `publicMessage` (D-04) | The allow-listed `reason` key of a reschedule event's `new_values`. Raw values are never exposed |
| QR used for check-in / checkout (D96, decision 4) | The visit's single unrevoked static credential. Issuing requires `approved`, so the credential that checked the visit in is the one that checks it out |
| `visitorCount` | `detail_pengunjung.jumlah_pengunjung` |
| `vip` / `visitKind` | `kunjungan.tipe_kunjungan` (umum → standard, not VIP; vip → standard, VIP; terbatas → restricted) |
| End time, missed, eligibility | `tanggal` + `waktu_mulai` + `visit_duration` in WITA, evaluated at server time |

## Migration order

1. **Prerequisite:** schemas and usage grants.
2. **Ticket 00 (vms tables):** 1–16 in the numbered order above. 17 is on hold.
3. **Ticket 00 (support tables):** 23–28.
4. **Ticket 05:** 18 `form_documents`, then 29 `form_document_uploads`.
5. **Ticket 06:**
   1. 19 `detail_kunjungan`
   2. 20 `detail_pengunjung`
   3. 21 `kunjungan`
   4. 30 `booking_requests`
6. **Ticket 08:** 22 `qr_kunjungan`.

Stage-1 reset is done by the user, not by an agent:
1. Drop the `vms` and `support` schemas, including the migration ledger, as `vms_migrator`.
2. Run `vms-migrate -mode up`.
3. Re-run the synthetic catalog seed (updated for the removed `ruangan.ruangan` column).

Credentials and the database volume are kept.

# Database and migration implications — Phase 1

This revision executes **no database action and creates no SQL migration**. Existing migration history is source evidence; it is not proof of a deployed database. The current [Phase 1 contract](API-CONTRACT.md) overrides the earlier session and editable-role-policy requirements for this phase.

SIHP's timestamped sql-migrate Up/Down layout, explicit FK/unique/index constraints and repository-owned transactions remain the reference. No new account service, duplicate users table or GORM AutoMigrate is introduced.

## Current SQL source

Updated 19 November 2026: on 18 November the user replaced the earlier multi-table migrations with one schemas prerequisite plus one migration per table, built from [the physical schema](PHYSICAL-SCHEMA-2026-11-18.md). The stage-1 database is reset once by the user.

| Migration | Table | Concrete requirement |
| --- | --- | --- |
| 202611180001_create_schemas.sql | none | Creates the `vms` and `support` schemas |
| 202611180002_create_vms_kedeputian.sql | vms.kedeputian | Organization reference: deputy groups, verified names seeded |
| 202611180003_create_vms_direktorat.sql | vms.direktorat | Organization reference: directorates under a deputy group |
| 202611180004_create_vms_jabatan.sql | vms.jabatan | Position with VIP flag, linked to one directorate or deputy group |
| 202611180005_create_vms_pejabat.sql | vms.pejabat | Officials |
| 202611180006_create_vms_ruangan.sql | vms.ruangan | Rooms |
| 202611180007_create_vms_pegawai.sql | vms.pegawai | Employee account and profile |
| 202611180008_create_vms_role.sql | vms.role | Roles |
| 202611180009_create_vms_pegawai_pejabat.sql | vms.pegawai_pejabat | PA official assignment; one active row per employee |
| 202611180010_create_vms_role_pegawai.sql | vms.role_pegawai | Employee roles |
| 202611180011_create_vms_feature.sql | vms.feature | Permission catalogue |
| 202611180012_create_vms_action.sql | vms.action | Permission catalogue |
| 202611180013_create_vms_permission.sql | vms.permission | Permission catalogue |
| 202611180014_create_vms_policy.sql | vms.policy | Role policies |
| 202611180015_create_vms_role_policy.sql | vms.role_policy | Role policies |
| 202611180016_create_vms_policy_permission.sql | vms.policy_permission | Role policies |
| 202611180017_create_vms_audit_log.sql | vms.audit_log | Audit events |
| 202611180018_create_support_rate_limit_buckets.sql | support.rate_limit_buckets | Shared rolling public/staff/operation admissions across instances and restarts |
| 202611180019_create_support_rate_limit_events.sql | support.rate_limit_events | Same |
| 202611180020_create_support_employee_credentials.sql | support.employee_credentials | Local verifier credentials isolated from the profile |
| 202611180021_create_support_staff_sessions.sql | support.staff_sessions | Revocation, role, idle, absolute and refresh state |
| 202611180022_create_support_refresh_lineage.sql | support.refresh_lineage | Consumed-credential reuse detection |
| 202611180023_create_support_refresh_recovery.sql | support.refresh_recovery | Encrypted same-request successor recovery for 10 seconds |

The source has 22 tables: 14 original diagram business tables, two organization reference tables, and six support tables. Later slices add their tables one file each. The organization extension and one-active-assignment constraint are documented in [EMPLOYEE-ORGANIZATION.md](EMPLOYEE-ORGANIZATION.md). Their continued presence does not make all of them Phase 1 dependencies.

## Authentication and manual provisioning

| Concern | Phase 1 use |
| --- | --- |
| Account/profile | Reuse vms.pegawai; name/email/status_aktif identify and enable the account |
| Credentials | Reuse support.employee_credentials; existing encoded bcrypt hash only, no plaintext password |
| Fixed roles | Reuse vms.role and vms.role_pegawai; exactly one existing role per employee, including Super Admin |
| PA scope | Reuse active vms.pegawai_pejabat assignments and current official non-VIP classification |
| Multiple role rows | Current schema allows several distinct roles. Controlled provisioning must select one; the API denies ambiguous authority instead of picking/unioning. No new role-table redesign is required by this contract revision |
| Permissions | Fixed backend mapping in the contract; vms.feature/action/permission/policy/role_policy/policy_permission are not evaluated or exposed for Phase 1 |
| Access JWT | Signed fixed-expiry credential with required server lifetime configuration; no staff_sessions row, token history, rotation or refresh recovery dependency |
| Logout | Browser-only clearing; no revocation write. Existing token copies remain usable until expiry while account is active |
| Account/password/role writes | Controlled operator setup only; no Phase 1 employee create/edit, role assignment/policy, forced-password or reset API |
| Legacy state | support.staff_sessions, refresh_lineage and refresh_recovery remain existing source/deployment history; no Phase 1 dependency and no drop authorized |

No fabricated employee ID, seed email/password, credential or live database state is supplied here. Operator provisioning uses the exact fixed role labels and current encoded credential format. Existing must_change_password and policy metadata can remain unused while their workflows are deferred.

The earlier DB-PROFILE finding was stale: the inspected identity migration already defines jabatan, kedeputian, direktorat, nomor_hp and nip. Read-only employee profile projection does not justify adding duplicate columns or a profile table.

## Retained business schema work

| Requirement / affected feature | Reuse or future change | Justification and constraints |
| --- | --- | --- |
| Ordinary/restricted bookings, status, detail and history (F02,F05,F21–28) | Remaining diagram Pengunjung, Detail Kunjungan, Kunjungan; current Audit Log | Visit-owned visitor snapshot, separate detail and visit, unique eight-digit code, positive version, six stored states. Restricted identity-free visit has no fabricated Pengunjung row; nullable FK. Ordinary snapshot and named roster complete atomically. Do not deduplicate people by email/name. |
| Roster order/count (F02,F21) | Previously selected named roster child relation owned by visit/visitor aggregate | Concrete1–20 ordered names including representative; derive count, do not create one table per form step. Physical table/column naming finalized in visit migration; no new contact-directory entity. |
| Reservation conflicts / dashboards / filters | Use Detail Kunjungan date/start/duration and nullable Pejabat/Ruangan links | Lock resource rows in fixed type/ID order and query complete overlapping committed intervals; add justified query indexes with owning slice. No reservations/dashboard/filter/cache table is required by the current selected design. Pending reserves nothing. |
| Attendance (F33,F34) | Previously selected attendance event relation + existing visit status and Audit Log | Need one successful checkin/checkout, actor and timestamp history. Unique transition backstops; no duplicate generic scan/identity/image tables. Checkout may occur after end; checked-in commitments retain approved interval semantics. |
| QR (F07,F31–34) | Remaining diagram QR Kunjungan | Token hashes, exact timestamptz expiry, consumed/revoked state and visit binding. One usable admission credential, replacement/revocation and post-checkin checkout rule. No public token values stored in display fields. |
| Optional documents (F29,F45) | Remaining diagram File Storage; optional visit link; selected staged upload metadata/reference | Integer metadata ID, UUID private object location, MIME/byte size and verification/owner/link state. Shared metadata/reference relation may satisfy upload replay/cleanup; do not create separate upload, preview and download entities unless existing columns cannot establish ownership/lifecycle. Token1h; unlinked cleanup24h; already linked files excluded. Production capability remains disabled. |
| Submission recovery/idempotency (F04,F21,F22) | Approved durable scoped submission/outcome support relation | Unique audience/principal/reference, immutable normalized input digest, committed visit link or minimal closed marker. Must serialize absent-reference resolve with delayed create and survive restart. Reuse one structure for ordinary/restricted creates; no operation-specific replay tables. Retain linkage for visit lifetime; no invented closure purge. |
| Email reliability (F02,F21,F24–28) | Approved PostgreSQL outbox support relation | Visit/audit/logical notification commit together, retry through worker up to selected24h after first failure. Store durable intent/delivery state; attempts may be columns/history as justified, not a speculative endpoint/mailbox schema. No external broker or generic inbox feature. |
| SMTP metadata/secrets (F40) | Remaining diagram Integration Keys, expanded to reviewed encrypted provider config/version | Six-key SMTP adapter dictionary and encrypted exact secret; reuse same provider record/fields. No separate email/settings screen tables or writable arbitrary preview storage provider. |
| Reports/export (F41,F42) | Read aggregate from ordinary visits/attendance and current authority | Derive seven counts and revision; no report/CSV/export-history/dashboard tables required. |


These business support relations retain their earlier concrete visit, file, recovery and notification requirements. They are not session-auth work and are not removed merely by simplifying login. Before adding physical structures, check whether existing columns/relations satisfy the invariant. Endpoint/DTO/control counts never justify a table by themselves.

Future authorized implementation appends timestamped Up/Down migrations and extends existing migration sources. Do not run a destructive Down or change deployed history as part of this contract edit. Canonical IDs/FKs remain positive INTEGER; optional relationships stay explicit; no cascaded deletion of visit/audit/file history. Retention/purge and production attachment-provider/scanning choices remain release decisions.

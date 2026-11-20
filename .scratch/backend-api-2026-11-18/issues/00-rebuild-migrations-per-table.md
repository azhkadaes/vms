# 00: Rebuild migrations, one per table

**What to build:** The database is described by one migration per table and matches the user's DB diagram exactly. It departs from the diagram only where an earlier agreement says so.

The four existing multi-table migrations are replaced by:
- a schemas prerequisite;
- one file per existing table, created directly in its final shape (`vms` tables 1–16 and `support` tables 23–28 in the physical schema).

The diagram's `integration_keys` (slot 17) is on hold and is not created.

**Convergence with the diagram:**
- `role_pegawai.id_karyawan` replaces `id_pegawai`;
- `created_by`/`updated_by` become VARCHAR;
- `audit_log` gets JSON values plus the agreed `actor_type` (staff/visitor/system), nullable staff `id_pegawai` and `entity_type`/`entity_id`;
- `pegawai.email` becomes VARCHAR(255);
- `ruangan.ruangan` is dropped.

**Reference seeds** live in their table's migration: deputy groups, directorates, the 4 roles, features, actions, the 14 permissions, and one default policy per role with its links and default permissions. **Grants** cover every runtime need in the spec, so later tickets need no grant migrations.

**Existing code** adapts to the converged columns: the session and auth repositories, audit writes, the catalog query, the readiness check, the stage-1 seed script, and the migration and contract-manifest table lists. A written stage-1 reset procedure is included; the user runs it.

Context:
- [physical schema](../../../docs/contracts/PHYSICAL-SCHEMA-2026-11-18.md): rules, tables 1–17 and 23–28, migration order, reset procedure
- [spec](../spec.md): Schema
- [conventions](../../../docs/BACKEND-GO-CONVENTIONS.md): §8

**Blocked by:** None (can start immediately).

**Status:** done (commit 402e94f on vms/backend-api)

- [x] **Old migrations:** the 4 existing files are removed. The migration source holds one prerequisite plus one file per table, in the documented order, each with Up and Down.
- [x] **Shape:** every table has exactly the columns, types, nullability and checks listed in the physical schema. A test compares the applied catalog (`information_schema`) against the expected column list per table.
- [x] **Down migrations:** each table's Down drops only that table and its grants. A full up → down → up cycle on a scratch database succeeds.
- [x] **Seeds:**
  - [x] 7 deputy groups, 21 directorates, 4 roles, 14 permissions
  - [x] default policies giving each role the frontend's default permission set
  - [x] `created_by` = `system`
- [x] **Grants:** match the physical schema per table
- [x] **`audit_log` constraints:** rejects a staff row without `id_pegawai` and a visitor or system row with one. Session lifecycle audits now write `entity_type = pegawai`.
- [x] **Existing behaviour unchanged:**
  - [x] Login, refresh, role switch, password change and catalog work against the rebuilt schema
  - [x] existing unit tests and the integration suite pass
  - [x] the migration-count, readiness and manifest assertions are updated
- [x] **Stage-1 reset procedure:** written for the user to run. It covers:
  - [x] dropping both schemas as the migrator;
  - [x] migrating up;
  - [x] re-seeding the synthetic catalog, with the seed updated for `ruangan`.

  No agent runs it, and credentials and the volume are preserved.

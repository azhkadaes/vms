\set ON_ERROR_STOP on
-- User-run stage-1 reset (docs/LOCAL-DEVELOPMENT.md): drops every VMS table, the migration
-- ledger and all synthetic rows. Roles, passwords, database grants and the volume are kept.
BEGIN;
DO $vms$
BEGIN
  IF current_database() <> 'vms_stage1' OR current_user <> 'vms_migrator' THEN
    RAISE EXCEPTION 'Schema reset is restricted to vms_stage1 as vms_migrator';
  END IF;
END
$vms$;
DROP SCHEMA IF EXISTS vms CASCADE;
DROP SCHEMA IF EXISTS support CASCADE;
COMMIT;

-- +migrate Up
-- The only non-table migration. sql-migrate may already have created support for its
-- ledger, and a fresh stage-1 volume may predate this file, so both are idempotent.
CREATE SCHEMA IF NOT EXISTS vms;
CREATE SCHEMA IF NOT EXISTS support;
GRANT USAGE ON SCHEMA vms, support TO vms_api;

-- +migrate Down
-- support keeps the migration ledger, so only the (by now empty) vms schema is dropped.
REVOKE USAGE ON SCHEMA vms, support FROM vms_api;
DROP SCHEMA vms;

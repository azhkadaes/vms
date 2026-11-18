#!/usr/bin/env bash
set -euo pipefail

export VMS_INIT_MIGRATOR_PASSWORD="$(cat /run/secrets/db_migrator_password)"
export VMS_INIT_API_PASSWORD="$(cat /run/secrets/db_api_password)"
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<'SQL'
\getenv migrator_password VMS_INIT_MIGRATOR_PASSWORD
\getenv api_password VMS_INIT_API_PASSWORD
SET log_statement TO 'none';
SET log_min_error_statement TO 'panic';
CREATE ROLE vms_migrator LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT PASSWORD :'migrator_password';
CREATE ROLE vms_api LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT PASSWORD :'api_password';
REVOKE ALL ON DATABASE vms_stage1 FROM PUBLIC;
GRANT CONNECT ON DATABASE vms_stage1 TO vms_migrator,vms_api;
-- The migrator creates the vms and support schemas itself (prerequisite migration and ledger).
GRANT CREATE ON DATABASE vms_stage1 TO vms_migrator;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
ALTER DATABASE vms_stage1 SET timezone TO 'UTC';
SQL
unset VMS_INIT_MIGRATOR_PASSWORD VMS_INIT_API_PASSWORD

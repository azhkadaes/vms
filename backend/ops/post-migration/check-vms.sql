\set ON_ERROR_STOP on
BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL search_path TO pg_catalog;
SET LOCAL statement_timeout TO '30s';
SET LOCAL lock_timeout TO '5s';
SET LOCAL vms_check.expected_database TO :'expected_database';

DO $check$
DECLARE
    expected_tables CONSTANT text[] := ARRAY[
        'kedeputian', 'direktorat', 'jabatan', 'pejabat', 'ruangan', 'pegawai',
        'role', 'pegawai_pejabat', 'role_pegawai', 'feature', 'action', 'permission',
        'policy', 'role_policy', 'policy_permission', 'audit_log', 'form_documents',
        'detail_kunjungan', 'detail_pengunjung', 'kunjungan', 'qr_kunjungan'
    ];
    problems text;
BEGIN
    IF current_database() <> current_setting('vms_check.expected_database')
       OR current_setting('server_version_num')::int / 10000 <> 18 THEN
        RAISE EXCEPTION 'Connect to the specified VMS database on PostgreSQL 18';
    END IF;

    SELECT string_agg(expected.name, ', ' ORDER BY expected.name) INTO problems
    FROM unnest(expected_tables) AS expected(name)
    LEFT JOIN pg_class AS tbl
      ON tbl.oid = to_regclass(format('vms.%I', expected.name))
    WHERE tbl.oid IS NULL OR tbl.relkind <> 'r'
       OR NOT EXISTS (
           SELECT 1 FROM pg_constraint WHERE conrelid = tbl.oid AND contype = 'p'
       )
       OR EXISTS (
           SELECT 1 FROM pg_constraint WHERE conrelid = tbl.oid AND NOT convalidated
       )
       OR EXISTS (
           SELECT 1 FROM pg_index WHERE indrelid = tbl.oid AND NOT (indisready AND indisvalid)
       );
    IF problems IS NOT NULL THEN
        RAISE EXCEPTION 'vms tables missing or with invalid constraints/indexes: %', problems;
    END IF;

    IF (SELECT count(*) FROM vms.role WHERE nama IN ('PA', 'Protocol Officer', 'Receptionist', 'Super Admin')) <> 4
       OR (SELECT count(*) FROM vms.permission WHERE permission_key IN (
           'bookings', 'create', 'approve', 'reject', 'reschedule', 'cancel', 'checkin',
           'checkout', 'people', 'roles', 'integrations', 'reports', 'reportExport', 'audit'
       )) <> 14 THEN
        RAISE EXCEPTION 'One or more built-in VMS roles or permission keys are missing';
    END IF;

    RAISE NOTICE 'vms: verified % business tables and the built-in roles and permission keys', cardinality(expected_tables);
END
$check$;

COMMIT;

\set ON_ERROR_STOP on
BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL search_path TO pg_catalog;
SET LOCAL statement_timeout TO '30s';
SET LOCAL lock_timeout TO '5s';
SET LOCAL vms_check.expected_database TO :'expected_database';

DO $check$
DECLARE
    expected_tables CONSTANT text[] := ARRAY[
        'rate_limit_buckets', 'rate_limit_events', 'employee_credentials',
        'staff_sessions', 'refresh_lineage', 'refresh_recovery',
        'form_document_uploads', 'booking_requests', 'schema_migrations'
    ];
    expected_migrations CONSTANT text[] := ARRAY[
        '202611180001_create_schemas.sql',
        '202611180002_create_vms_kedeputian.sql',
        '202611180003_create_vms_direktorat.sql',
        '202611180004_create_vms_jabatan.sql',
        '202611180005_create_vms_pejabat.sql',
        '202611180006_create_vms_ruangan.sql',
        '202611180007_create_vms_pegawai.sql',
        '202611180008_create_vms_role.sql',
        '202611180009_create_vms_pegawai_pejabat.sql',
        '202611180010_create_vms_role_pegawai.sql',
        '202611180011_create_vms_feature.sql',
        '202611180012_create_vms_action.sql',
        '202611180013_create_vms_permission.sql',
        '202611180014_create_vms_policy.sql',
        '202611180015_create_vms_role_policy.sql',
        '202611180016_create_vms_policy_permission.sql',
        '202611180017_create_vms_audit_log.sql',
        '202611180018_create_support_rate_limit_buckets.sql',
        '202611180019_create_support_rate_limit_events.sql',
        '202611180020_create_support_employee_credentials.sql',
        '202611180021_create_support_staff_sessions.sql',
        '202611180022_create_support_refresh_lineage.sql',
        '202611180023_create_support_refresh_recovery.sql',
        '202611180024_create_vms_form_documents.sql',
        '202611180025_create_support_form_document_uploads.sql',
        '202611180026_create_vms_detail_kunjungan.sql',
        '202611180027_create_vms_detail_pengunjung.sql',
        '202611180028_create_vms_kunjungan.sql',
        '202611180029_create_support_booking_requests.sql',
        '202611180030_create_vms_qr_kunjungan.sql'
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
      ON tbl.oid = to_regclass(format('support.%I', expected.name))
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
        RAISE EXCEPTION 'support tables missing or with invalid constraints/indexes: %', problems;
    END IF;

    SELECT string_agg(expected.id, ', ' ORDER BY expected.id) INTO problems
    FROM unnest(expected_migrations) AS expected(id)
    WHERE NOT EXISTS (SELECT 1 FROM support.schema_migrations WHERE id = expected.id);
    IF problems IS NOT NULL THEN
        RAISE EXCEPTION 'Missing migration history entries: %', problems;
    END IF;

    SELECT string_agg(applied.id, ', ' ORDER BY applied.id) INTO problems
    FROM support.schema_migrations AS applied
    WHERE applied.id <> ALL(expected_migrations);
    IF problems IS NOT NULL THEN
        RAISE EXCEPTION 'Unexpected migration history entries: %', problems;
    END IF;

    RAISE NOTICE 'support: verified % tables including the ledger and all % migration history entries',
        cardinality(expected_tables), cardinality(expected_migrations);
END
$check$;

COMMIT;

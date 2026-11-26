# VMS post-migration checks

Review [the 30 ordered migrations](../../migration/files/) and the two check files before approving a migration run. The checks are separate from the migration runner. Each opens a read-only transaction and stops with an error if its checks fail.

| File | Checks |
| --- | --- |
| [check-vms.sql](check-vms.sql) | All 21 business tables, primary keys, validity of existing constraints and indexes, and built-in roles and permission keys |
| [check-support.sql](check-support.sql) | All eight support tables and the migration ledger, primary keys, validity of existing constraints and indexes, and the exact 30 expected migration IDs |

After approval and successful migration, run the two checks from the backend directory against the same PostgreSQL 18 database. Use the migration role because the application role has limited table access. The following example uses the published stage-1 port. Use your configured host and port and authenticate through psql.

```powershell
$vmsTarget = 'vms_stage1'
psql -X -h 127.0.0.1 -p 55450 -U vms_migrator -d $vmsTarget -v "expected_database=$vmsTarget" -f ops/post-migration/check-vms.sql
psql -X -h 127.0.0.1 -p 55450 -U vms_migrator -d $vmsTarget -v "expected_database=$vmsTarget" -f ops/post-migration/check-support.sql
```

The local Docker stack keeps PostgreSQL inside the container. For that stack, run both checks against the already-running database service.

```powershell
foreach ($check in 'check-vms.sql', 'check-support.sql') {
    Get-Content -Raw "ops/post-migration/$check" |
        docker compose -f docker-compose.yml -f docker-compose.local.yml exec -T db sh -c 'PGPASSWORD="$(cat /run/secrets/db_migrator_password)" psql -X -h 127.0.0.1 -U vms_migrator -d vms_stage1 -v expected_database=vms_stage1 -f -'
    if ($LASTEXITCODE -ne 0) { throw "$check failed" }
}
```

Both commands should report a verification notice and exit successfully. These checks are tied to the reviewed migration set. Update the lists when a migration adds a table or changes the ledger. The [schema integration suite](../../migration/schema_integration_test.go) provides detailed column, constraint, index and grant checks on a disposable database.

The [OIKN organization data migration review](OIKN-ORGANIZATION-REVIEW.md) covers the separate catalog seed prepared from official profiles.

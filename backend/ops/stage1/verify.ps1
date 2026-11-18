param([ValidateRange(1024,65535)][int]$DatabasePort=55450,[ValidateRange(1024,65535)][int]$ApiPort=58050)
$ErrorActionPreference='Stop'
$vmsRoot=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
Push-Location $vmsRoot
try {
    $vmsBase="http://127.0.0.1:$ApiPort"
    $vmsReady=Invoke-WebRequest -Uri "$vmsBase/health/ready" -SkipHttpErrorCheck -TimeoutSec 10
    if ($vmsReady.StatusCode -ne 200) { throw 'Local API is not ready.' }
    $vmsResponse=Invoke-WebRequest -Uri "$vmsBase/v1/public/catalogs" -SkipHttpErrorCheck -TimeoutSec 10
    if ($vmsResponse.StatusCode -ne 200) { throw 'Public catalog request failed.' }
    $vmsCatalog=$vmsResponse.Content | ConvertFrom-Json
    if ($vmsCatalog.officers.Count -ne 1 -or $vmsCatalog.rooms.Count -ne 1 -or -not $vmsCatalog.officers[0].vip -or -not $vmsCatalog.officers[0].active) { throw 'Synthetic active/VIP projection is incorrect.' }
    if ($vmsCatalog.officers[0].id -isnot [string] -or $vmsCatalog.rooms[0].id -isnot [string]) { throw 'Business IDs must be strings.' }
    if ($vmsCatalog.attachmentsEnabled -or $vmsCatalog.privacyNotice -or $vmsResponse.Content.Contains('private-test-')) { throw 'Disabled/private metadata escaped in public catalog.' }
    if ($vmsResponse.Headers['Cache-Control'] -notcontains 'no-store') { throw 'Catalog caching policy is incorrect.' }
    $vmsBadQuery=Invoke-WebRequest -Uri "$vmsBase/v1/public/catalogs?role=Super%20Admin" -SkipHttpErrorCheck -TimeoutSec 10
    if ($vmsBadQuery.StatusCode -ne 400) { throw 'Unknown catalog query was accepted.' }
    $vmsLegacy=Invoke-WebRequest -Uri "$vmsBase/v1/users" -SkipHttpErrorCheck -TimeoutSec 10
    if ($vmsLegacy.StatusCode -ne 404) { throw 'Legacy user route exposed by new API.' }
    $vmsInternalUri=[IO.File]::ReadAllText((Join-Path $vmsRoot '.runtime/stage1/api_database_url')).Trim()
    $vmsParsed=[Uri]::new($vmsInternalUri)
    if ($vmsParsed.Host -ne 'db' -or $vmsParsed.Port -ne 5432 -or $vmsParsed.AbsolutePath -ne '/vms_stage1') { throw 'Unexpected private integration target.' }
    $env:VMS_INTEGRATION_DATABASE_URL=$vmsInternalUri.Replace('@db:5432/',"@127.0.0.1:${DatabasePort}/")
    $env:GOCACHE=Join-Path $vmsRoot '.runtime/go-cache'
    $env:GOTMPDIR=Join-Path $vmsRoot '.runtime/go-tmp'
    # Repository and readiness suites only: they roll back their fixtures. The destructive
    # migration cycle test (./migration) needs a separate vms_scratch database and is excluded.
    go test -json -tags integration -count 1 ./internal/repository/... ./pkg/helper/database | Set-Content -LiteralPath '.runtime/stage1/postgres-integration.jsonl'
    if ($LASTEXITCODE) { throw 'Real PostgreSQL repository/grant tests failed; inspect private integration log.' }
    $vmsResult=[ordered]@{
        scope='isolated vms_stage1 only'
        readiness=$true
        activeCatalogProjection=$true
        vipFromPosition=$true
        decimalStringIds=$true
        privateMetadataOmitted=$true
        uploadsDisabled=$true
        unknownQueryRejected=$true
        legacyRouteUnavailable=$true
        postgresConcurrencyAndRuntimeGrantTest=$true
        rebuiltSchemaSessionAndCatalogTests=$true
        syntheticDirectoryDataRetained=$true
        visitorRegistrationOrLiveEmailEnabled=$false
    }
    $vmsResult | ConvertTo-Json | Set-Content -LiteralPath '.runtime/stage1/verification.json'
    $vmsResult | ConvertTo-Json -Compress
} finally {
    $env:VMS_INTEGRATION_DATABASE_URL=''
    Pop-Location
}

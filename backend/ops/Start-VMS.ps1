param([switch]$Stop, [switch]$SeedTestCatalog, [switch]$Rebuild, [switch]$NoBrowser)
$ErrorActionPreference = 'Stop'
$backend = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$privateDir = Join-Path $backend '.runtime/local'
$catalogMarker = Join-Path $privateDir 'test-catalog-seeded'
$api = 'http://127.0.0.1:58100'
$composeFiles = @('compose','-p','vms','-f','docker-compose.yml','-f','docker-compose.local.yml')

function Invoke-VmsDocker {
    & docker @args
    if ($LASTEXITCODE -ne 0) { throw 'Docker could not complete the VMS command. See the message above.' }
}

Push-Location $backend
try {
    $env:VMS_PRIVATE_DIR = './.runtime/local'
    $env:VMS_STAGE1_IMAGE = 'vms-backend:local-compose'
    $env:VMS_STAGE1_API_PORT = '58100'
    $env:VMS_STAGE1_BUILD_TARGET = 'prod'
    $env:VMS_STAGE1_BINARY_CONTEXT = '.'
    & docker info --format '{{.ServerVersion}}' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Start Docker Desktop and wait for its engine, then open Start VMS again.' }

    if ($Stop) {
        Invoke-VmsDocker @composeFiles stop
        Write-Host 'VMS stopped. Docker kept the database and files.'
        return
    }
    if (Test-Path -LiteralPath $privateDir) {
        & ./ops/stage1/prepare.ps1 -Instance local -AddMissingFeatureKeys
    } else {
        & ./ops/stage1/prepare.ps1 -Instance local
    }
    $catalogSeeded = Test-Path -LiteralPath $catalogMarker -PathType Leaf
    if ($SeedTestCatalog -or $catalogSeeded) {
        if (-not ($env:VMS_NOTICE_VERSION -or $env:VMS_NOTICE_ID -or $env:VMS_NOTICE_EN)) {
            $env:VMS_NOTICE_VERSION = 'local-test-v1'
            $env:VMS_NOTICE_ID = 'Khusus pengujian lokal. Gunakan data fiktif.'
            $env:VMS_NOTICE_EN = 'Local testing only. Use fictional data.'
        }
    }
    $image = Invoke-VmsDocker image list --quiet --filter "reference=$env:VMS_STAGE1_IMAGE"
    if ($Rebuild -or -not $image) { Invoke-VmsDocker @composeFiles build api }
    $siteImage = Invoke-VmsDocker image list --quiet --filter 'reference=vms-frontend:local-compose'
    if ($Rebuild -or -not $siteImage) { Invoke-VmsDocker @composeFiles build frontend }
    Invoke-VmsDocker @composeFiles up -d --no-build --wait --wait-timeout 120

    if ($SeedTestCatalog -and -not $catalogSeeded) {
        Get-Content -Raw -LiteralPath 'ops/stage1/seed-test-catalog.sql' |
            docker @composeFiles exec -T db psql -X -U vms_migrator -d vms_stage1 -v ON_ERROR_STOP=1 -f -
        if ($LASTEXITCODE -ne 0) { throw 'The test catalog was not added. This option requires an empty officer and room catalog.' }
        [IO.File]::WriteAllText($catalogMarker, 'Local test catalog seeded.')
    }
    $ready = $false
    for ($attempt = 0; $attempt -lt 30; $attempt++) {
        try {
            if ((Invoke-RestMethod "$api/health/ready" -TimeoutSec 2).status -eq 'ready') { $ready = $true; break }
        } catch { Start-Sleep -Milliseconds 500 }
    }
    if (-not $ready) {
        Invoke-VmsDocker @composeFiles ps -a
        Invoke-VmsDocker @composeFiles logs --tail 20 api migrate
        throw 'VMS did not pass the readiness check.'
    }
    Write-Host "VMS ready: $api/health/ready"
    Write-Host 'Visitor app: http://localhost:58101/'
    Write-Host 'Staff app:   http://localhost:58101/admin/'
    Write-Host "API docs:  $api/swagger/index.html"
    Write-Host 'Swagger username: local'
    Write-Host "Swagger password file: $(Join-Path $privateDir 'swagger_password')"
    if (-not $NoBrowser) { Start-Process 'http://localhost:58101/' }
} finally { Pop-Location }

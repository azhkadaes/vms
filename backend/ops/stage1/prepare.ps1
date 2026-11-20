param([string]$HostUserSid, [switch]$AddMissingFeatureKeys)
$ErrorActionPreference = 'Stop'
if ($PSVersionTable.PSVersion.Major -lt 7 -or -not $IsWindows) { throw 'Run this local setup with PowerShell 7 on Windows.' }
$vmsBackendRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
$vmsPrivateRoot = Join-Path $vmsBackendRoot '.runtime/stage1'
$vmsFeatureKeys = @('jwt_key','csrf_key','refresh_recovery_key','gateway_init_key','upload_token_key','admission_qr_key')
function New-VmsFeatureKeys {
    foreach ($vmsKey in $vmsFeatureKeys) {
        $vmsPath = Join-Path $vmsPrivateRoot $vmsKey
        if (Test-Path -LiteralPath $vmsPath -PathType Leaf) { continue }
        $vmsStream = [IO.File]::Open($vmsPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write)
        try {
            $vmsValue = [Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
            $vmsStream.Write([Text.Encoding]::ASCII.GetBytes($vmsValue))
        } finally { $vmsStream.Dispose() }
    }
}
if ($AddMissingFeatureKeys) {
    foreach ($vmsRequired in @('db_owner_password','db_migrator_password','db_api_password','rate_key')) {
        if (-not (Test-Path -LiteralPath (Join-Path $vmsPrivateRoot $vmsRequired) -PathType Leaf)) {
            throw 'The existing private directory must already contain the database passwords and rate key.'
        }
    }
    New-VmsFeatureKeys
    Write-Output 'Added missing feature keys. Existing keys and database credentials were preserved.'
    return
}
if (Test-Path -LiteralPath $vmsPrivateRoot) { throw 'Stage1 credential directory already exists. Review and reuse it; this script will not overwrite or rotate credentials.' }
New-Item -ItemType Directory -Path $vmsPrivateRoot -Force | Out-Null
$vmsIdentity = [Security.Principal.WindowsIdentity]::GetCurrent().User
& icacls.exe $vmsPrivateRoot /inheritance:r /grant:r ("*{0}:(OI)(CI)F" -f $vmsIdentity.Value) | Out-Null
if ($LASTEXITCODE) { throw 'Could not restrict the private credential directory.' }
if ($HostUserSid) {
    $vmsHostIdentity = [Security.Principal.SecurityIdentifier]::new($HostUserSid)
    if (-not $vmsHostIdentity.IsAccountSid()) { throw 'HostUserSid must identify the verified host account.' }
    & icacls.exe $vmsPrivateRoot /grant ("*{0}:(OI)(CI)F" -f $vmsHostIdentity.Value) | Out-Null
    if ($LASTEXITCODE) { throw 'Could not grant the verified host account access.' }
}
function New-VmsSecret {
    [Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32)).ToLowerInvariant()
}
$vmsOwnerPassword = New-VmsSecret
$vmsMigratorPassword = New-VmsSecret
$vmsApiPassword = New-VmsSecret
$vmsFiles = @{
    db_owner_password = $vmsOwnerPassword
    db_migrator_password = $vmsMigratorPassword
    db_api_password = $vmsApiPassword
    api_database_url = "postgres://vms_api:${vmsApiPassword}@db:5432/vms_stage1?sslmode=disable"
    migrator_database_url = "postgres://vms_migrator:${vmsMigratorPassword}@db:5432/vms_stage1?sslmode=disable"
    rate_key = [Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
}
foreach ($vmsFile in $vmsFiles.GetEnumerator()) {
    [IO.File]::WriteAllText((Join-Path $vmsPrivateRoot $vmsFile.Key), $vmsFile.Value, [Text.UTF8Encoding]::new($false))
}
New-VmsFeatureKeys
Write-Output 'Prepared private local VMS stage1 credentials for the caller and optional verified host account. Values were not printed. No containers or database commands were run.'

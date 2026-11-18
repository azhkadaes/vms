param(
    [Parameter(Position = 0)]
    [ValidateSet('help', 'env-setup', 'build', 'run', 'start', 'swagger', 'test', 'lint', 'migration-plan')]
    [string]$Task = 'help'
)

$ErrorActionPreference = 'Stop'

if ($Task -eq 'help') {
    Write-Output 'VMS commands: .\vms.ps1 <task>'
    Write-Output '  env-setup       Copy .env.example if .env does not exist (run this yourself).'
    Write-Output '  build           Build the native API binary.'
    Write-Output '  run             Run the API from source using your configuration.'
    Write-Output '  start           Set up missing configuration, build and run the API.'
    Write-Output '  swagger         Generate the implemented OpenAPI document.'
    Write-Output '  test            Run Go tests.'
    Write-Output '  lint            Run go vet.'
    Write-Output '  migration-plan  Inspect embedded migrations without connecting to a database.'
    exit 0
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go is required. Install the project Go toolchain and add go to PATH.'
}

$vmsArguments = switch ($Task) {
    'test'           { @('test', './...') }
    'lint'           { @('vet', './...') }
    'migration-plan' { @('run', './cmd/vms-migrate', '-mode', 'plan') }
    default          { @('run', './cmd/vms-dev', '-mode', $Task) }
}

# Resolve the project from the script, including when called from another directory.
Push-Location -LiteralPath $PSScriptRoot
try {
    & go @vmsArguments
    $vmsExitCode = $LASTEXITCODE
} finally {
    Pop-Location
}
exit $vmsExitCode

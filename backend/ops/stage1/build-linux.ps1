param()
$ErrorActionPreference = 'Stop'
$vmsBackendRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
Push-Location $vmsBackendRoot
try {
    $env:GOTOOLCHAIN = 'go1.27.4'
    $env:GOCACHE = Join-Path $vmsBackendRoot '.runtime/go-cache'
    $env:GOTMPDIR = Join-Path $vmsBackendRoot '.runtime/go-tmp'
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    New-Item -ItemType Directory -Force -Path $env:GOCACHE,$env:GOTMPDIR,'.runtime/linux' | Out-Null
    go build -trimpath -o .runtime/linux/vms-api ./cmd
    if ($LASTEXITCODE) { throw 'Linux API compilation failed.' }
    go build -trimpath -o .runtime/linux/vms-migrate ./cmd/vms-migrate
    if ($LASTEXITCODE) { throw 'Linux migration-tool compilation failed.' }
    go version .runtime/linux/vms-api .runtime/linux/vms-migrate
    if ($LASTEXITCODE) { throw 'Could not verify Linux binary build versions.' }
} finally { Pop-Location }

$ErrorActionPreference = 'Stop'

Set-Location "$PSScriptRoot\..\backend"

$env:Path = 'C:\Program Files\Go\bin;' + $env:Path

Write-Host 'Starting test dependencies (Postgres/Redis)...'
docker compose -f docker-compose.test.yml up -d

Write-Host 'Running backend tests...'
go test ./...

Write-Host 'Tests completed.'

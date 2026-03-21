$ErrorActionPreference = 'Stop'

Set-Location "$PSScriptRoot\..\backend"

$container = 'travel-planner-postgres-test'

Write-Host 'Ensuring test postgres container is running...'
docker compose -f docker-compose.test.yml up -d postgres | Out-Null

Write-Host 'Resetting database schema...'
'drop schema if exists public cascade; create schema public;' |
  docker exec -i $container psql -U travel -d travel_planner -v ON_ERROR_STOP=1 | Out-Null
if ($LASTEXITCODE -ne 0) {
  throw 'Failed to reset schema.'
}

Write-Host 'Applying up migrations...'
$upFiles = Get-ChildItem .\migrations\*.up.sql | Sort-Object Name
foreach ($file in $upFiles) {
  Write-Host "  -> $($file.Name)"
  Get-Content $file.FullName -Raw |
    docker exec -i $container psql -U travel -d travel_planner -v ON_ERROR_STOP=1 | Out-Null
  if ($LASTEXITCODE -ne 0) {
    throw "Migration failed: $($file.Name)"
  }
}

Write-Host 'Migration smoke test passed.'

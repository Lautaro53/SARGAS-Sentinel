# Operaciones diarias sobre el mismo proyecto Docker que usa el instalador.
# Uso: .\scripts\administrar-docker.ps1 -Accion estado|logs|detener
param(
    [ValidateSet('estado', 'logs', 'detener')]
    [string] $Accion = 'estado',
    [ValidateRange(1, 1000)]
    [int] $UltimasLineas = 100
)

$ErrorActionPreference = 'Stop'
$backend = Split-Path -Parent $PSScriptRoot
$compose = Join-Path $backend 'compose.yaml'
$privado = Join-Path $env:USERPROFILE 'SARGAS-privado\docker-secrets'

if (-not (Test-Path -LiteralPath $compose -PathType Leaf)) {
    throw 'No se encontro compose.yaml en el directorio del backend.'
}

foreach ($archivo in @('postgres_password.txt', 'sentinel_app_password.txt')) {
    if (-not (Test-Path -LiteralPath (Join-Path $privado $archivo) -PathType Leaf)) {
        throw 'Faltan los secretos originales de SARGAS. No se alterara la instalacion.'
    }
}

$env:SARGAS_DOCKER_SECRETS = $privado.Replace('\', '/')
Push-Location $backend
try {
    switch ($Accion) {
        'estado'  { & docker compose -f .\compose.yaml ps }
        'logs'    { & docker compose -f .\compose.yaml logs --tail=$UltimasLineas sentinel }
        'detener' { & docker compose -f .\compose.yaml stop }
    }
    if ($LASTEXITCODE -ne 0) {
        throw "Fallo la operacion '$Accion' de Docker Compose."
    }
}
finally {
    Pop-Location
}

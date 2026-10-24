
# SARGAS Sentinel - Actualizacion local segura

$ErrorActionPreference = "Stop"

$backend = Split-Path -Parent $PSScriptRoot

$actual = Join-Path $backend "sentinel.exe"
$nuevo = Join-Path $backend "sentinel-nuevo.exe"

$fecha = Get-Date -Format "yyyyMMdd_HHmmss"
$respaldo = Join-Path $backend "sentinel-backup-$fecha.exe"

# Verificar archivos
if (-not (Test-Path -LiteralPath $nuevo)) {
    throw "No existe sentinel-nuevo.exe. Ejecuta primero compilar-sentinel.ps1."
}

if (-not (Test-Path -LiteralPath $actual)) {
    throw "No existe sentinel.exe."
}

# No reemplazar ejecutables en uso
$proceso = Get-Process -Name sentinel -ErrorAction SilentlyContinue

if ($proceso) {
    throw "Sentinel esta ejecutandose. Detenelo con Ctrl+C antes de actualizar."
}

# Conservar version anterior
Copy-Item -LiteralPath $actual -Destination $respaldo

try {
    # Instalar nuevo ejecutable
    Copy-Item -LiteralPath $nuevo -Destination $actual -Force

    # Verificar integridad de la copia
    $hashNuevo = (Get-FileHash -LiteralPath $nuevo -Algorithm SHA256).Hash
    $hashActual = (Get-FileHash -LiteralPath $actual -Algorithm SHA256).Hash

    if ($hashNuevo -ne $hashActual) {
        throw "La copia del ejecutable no coincide con el original."
    }
}
catch {
    Copy-Item -LiteralPath $respaldo -Destination $actual -Force
    throw "La actualizacion fallo. Se intento restaurar el ejecutable anterior. Detalle: $_"
}

Write-Host ""
Write-Host "SARGAS Sentinel actualizado correctamente." -ForegroundColor Green
Write-Host "Respaldo: $respaldo"
Write-Host "Ejecutable: $actual"
Write-Host "Inicia Sentinel y verifica su funcionamiento."

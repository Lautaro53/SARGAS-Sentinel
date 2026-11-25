# SARGAS Sentinel - Actualizacion local segura
# Uso: .\scripts\actualizar-sentinel.ps1 [-Compilar]
param([switch]$Compilar)

$ErrorActionPreference = 'Stop'

$backend = Split-Path -Parent $PSScriptRoot
$actual = Join-Path $backend 'sentinel.exe'
$nuevo = Join-Path $backend 'sentinel-nuevo.exe'

if ($Compilar) {
    $scriptCompilacion = Join-Path $PSScriptRoot 'compilar-sentinel.ps1'
    if (-not (Test-Path -LiteralPath $scriptCompilacion -PathType Leaf)) {
        throw 'No se encontro compilar-sentinel.ps1.'
    }
    & $scriptCompilacion
}

if (-not (Test-Path -LiteralPath $nuevo -PathType Leaf)) {
    throw 'No existe sentinel-nuevo.exe. Ejecuta primero compilar-sentinel.ps1 o utiliza -Compilar.'
}

# No modificar el ejecutable mientras Sentinel esta en uso.
if (Get-Process -Name sentinel -ErrorAction SilentlyContinue) {
    throw 'Sentinel esta ejecutandose. Detenelo con Ctrl+C antes de actualizar.'
}

# Primer uso local: instalar el binario sin crear un respaldo inexistente.
if (-not (Test-Path -LiteralPath $actual -PathType Leaf)) {
    $hashNuevo = (Get-FileHash -LiteralPath $nuevo -Algorithm SHA256).Hash
    $temporal = "$actual.$([guid]::NewGuid().ToString('N')).tmp"
    try {
        Copy-Item -LiteralPath $nuevo -Destination $temporal -ErrorAction Stop
        if ((Get-FileHash -LiteralPath $temporal -Algorithm SHA256).Hash -ne $hashNuevo) { throw 'La copia inicial no coincide.' }
        Move-Item -LiteralPath $temporal -Destination $actual -ErrorAction Stop
    } finally {
        if (Test-Path -LiteralPath $temporal) { Remove-Item -LiteralPath $temporal -Force }
    }
    Write-Host 'Ejecutable local instalado. Ejecuta .\sargas-dev.ps1 iniciar.' -ForegroundColor Green
    return
}

$hashAnterior = (Get-FileHash -LiteralPath $actual -Algorithm SHA256).Hash
$hashNuevo = (Get-FileHash -LiteralPath $nuevo -Algorithm SHA256).Hash
if ($hashAnterior -eq $hashNuevo) {
    Write-Host 'El ejecutable instalado ya coincide con sentinel-nuevo.exe. No se realizaron cambios.' -ForegroundColor Green
    return
}

$fecha = Get-Date -Format 'yyyyMMdd_HHmmss_fff'
$respaldo = Join-Path $backend "sentinel-backup-$fecha.exe"
if (Test-Path -LiteralPath $respaldo) {
    throw 'Ya existe un respaldo con ese nombre. No se reemplazara.'
}

# El respaldo debe estar verificado antes de modificar el ejecutable instalado.
Copy-Item -LiteralPath $actual -Destination $respaldo -ErrorAction Stop
if ((Get-FileHash -LiteralPath $respaldo -Algorithm SHA256).Hash -ne $hashAnterior) {
    throw "El respaldo no paso la verificacion de integridad: $respaldo"
}

try {
    Copy-Item -LiteralPath $nuevo -Destination $actual -Force -ErrorAction Stop
    if ((Get-FileHash -LiteralPath $actual -Algorithm SHA256).Hash -ne $hashNuevo) {
        throw 'La copia del ejecutable nuevo no coincide con el original.'
    }
}
catch {
    $errorActualizacion = $_
    try {
        Copy-Item -LiteralPath $respaldo -Destination $actual -Force -ErrorAction Stop
        if ((Get-FileHash -LiteralPath $actual -Algorithm SHA256).Hash -ne $hashAnterior) {
            throw 'La restauracion no coincide con el ejecutable anterior.'
        }
    }
    catch {
        throw "Fallo la actualizacion y no se pudo verificar la restauracion. Respaldo: $respaldo. Detalle: $_"
    }
    throw "Fallo la actualizacion. Se restauro el ejecutable anterior. Detalle: $errorActualizacion"
}

Write-Host ''
Write-Host 'SARGAS Sentinel actualizado correctamente.' -ForegroundColor Green
Write-Host "Respaldo: $respaldo"
Write-Host "Ejecutable: $actual"
Write-Host 'Inicia Sentinel y verifica su funcionamiento.'

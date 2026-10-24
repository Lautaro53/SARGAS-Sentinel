
# SARGAS Sentinel - Compilacion local integrada
$ErrorActionPreference = "Stop"

$backend = Split-Path -Parent $PSScriptRoot
$raiz = Split-Path -Parent $backend
$frontend = Join-Path $raiz "sargas-sentinel-web"

$origen = Join-Path $frontend "dist"
$destino = Join-Path $backend "cmd\sentinel\web\dist"
$ejecutable = Join-Path $backend "sentinel-nuevo.exe"

if (-not (Test-Path "$frontend\package.json")) {
    throw "No se encontro el proyecto frontend."
}

if (-not (Test-Path "$backend\go.mod")) {
    throw "No se encontro el proyecto backend."
}

# 1. Verificar y compilar Svelte
Push-Location $frontend
try {
    if (-not (Test-Path ".\node_modules")) {
        & npm.cmd ci
        if ($LASTEXITCODE -ne 0) {
            throw "Fallo la instalacion de dependencias."
        }
    }

    & npm.cmd run check
    if ($LASTEXITCODE -ne 0) {
        throw "Fallaron las verificaciones del frontend."
    }

    & npm.cmd run build
    if ($LASTEXITCODE -ne 0) {
        throw "Fallo la compilacion del frontend."
    }
}
finally {
    Pop-Location
}

if (-not (Test-Path "$origen\index.html")) {
    throw "La compilacion no genero index.html."
}

# 2. Incorporar archivos compilados al backend
if (Test-Path $destino) {
    Remove-Item -LiteralPath $destino -Recurse -Force
}

New-Item -ItemType Directory -Path $destino -Force |
    Out-Null

Copy-Item -Path "$origen\*" -Destination $destino `
    -Recurse -Force

# 3. Compilar ejecutable Go
Push-Location $backend
try {
    & go test -count=1 ./...
    if ($LASTEXITCODE -ne 0) {
        throw "Fallaron las pruebas del backend."
    }

    & go build -o $ejecutable ./cmd/sentinel
    if ($LASTEXITCODE -ne 0) {
        throw "Fallo la compilacion de Go."
    }
}
finally {
    Pop-Location
}

Write-Host ""
Write-Host "SARGAS Sentinel compilado correctamente." `
    -ForegroundColor Green
Write-Host "Ejecutable: $ejecutable"
Write-Host "No se modifico sentinel.exe."

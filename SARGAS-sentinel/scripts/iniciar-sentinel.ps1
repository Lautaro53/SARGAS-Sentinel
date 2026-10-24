# SARGAS Sentinel - Inicio local seguro (Windows PowerShell 5.1+)
# Uso: .\scripts\iniciar-sentinel.ps1
#      .\scripts\iniciar-sentinel.ps1 -ActualizarClave
param(
    [switch]$ActualizarClave
)

$ErrorActionPreference = 'Stop'

$backend = Split-Path -Parent $PSScriptRoot
$ejecutable = Join-Path $backend 'sentinel.exe'
$carpetaPrivada = Join-Path $env:USERPROFILE 'SARGAS-privado'
$archivoCredencial = Join-Path $carpetaPrivada 'sentinel_app.credential.xml'

if (-not (Test-Path -LiteralPath $ejecutable -PathType Leaf)) {
    throw "No se encontro sentinel.exe en: $backend. Compilalo con: go build -o .\sentinel.exe .\cmd\sentinel"
}

New-Item -ItemType Directory -Path $carpetaPrivada -Force | Out-Null

try {
    if ($ActualizarClave -or -not (Test-Path -LiteralPath $archivoCredencial)) {
        Write-Host 'Ingresa la contrasena ACTUAL de PostgreSQL para sentinel_app.'
        $credencial = Get-Credential -UserName 'sentinel_app' -Message 'Clave de PostgreSQL de SARGAS Sentinel'
        if ($null -eq $credencial) {
            throw 'Se cancelo el ingreso de credenciales.'
        }
        if ($credencial.UserName -ne 'sentinel_app') {
            throw 'El usuario debe ser sentinel_app.'
        }
        $credencial | Export-Clixml -LiteralPath $archivoCredencial
        Write-Host 'Credencial guardada de forma cifrada fuera del proyecto.'
    }
    else {
        $credencial = Import-Clixml -LiteralPath $archivoCredencial
        if ($credencial -isnot [System.Management.Automation.PSCredential] -or
            $credencial.UserName -ne 'sentinel_app') {
            throw 'El archivo de credenciales no corresponde a sentinel_app. Reintenta con -ActualizarClave.'
        }
    }

    $claveCodificada = [uri]::EscapeDataString($credencial.GetNetworkCredential().Password)
    $env:DATABASE_URL = "postgres://sentinel_app:$($claveCodificada)@127.0.0.1:5432/sargas_sentinel?sslmode=disable"

    Write-Host 'Iniciando SARGAS Sentinel... (Ctrl+C para detenerlo)'
    Push-Location $backend
    try {
        & $ejecutable
        if ($LASTEXITCODE -ne 0) {
            Write-Warning 'Sentinel termino con error. Si aparece SQLSTATE 28P01, ejecuta de nuevo con -ActualizarClave.'
        }
    }
    finally {
        Pop-Location
    }
}
finally {
    Remove-Item Env:DATABASE_URL -ErrorAction SilentlyContinue
    Remove-Variable credencial, claveCodificada -ErrorAction SilentlyContinue
}

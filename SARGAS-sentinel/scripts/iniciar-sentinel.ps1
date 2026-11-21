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

# Si ya existe DATABASE_URL, utilizarla sin solicitar ni reemplazar credenciales.
# En modo local, SARGAS_DB_PORT permite cambiar el puerto PostgreSQL (por defecto 5432).
$dsnExterna = -not [string]::IsNullOrWhiteSpace($env:DATABASE_URL)

if ($dsnExterna -and $ActualizarClave) {
    throw 'Con DATABASE_URL configurada, -ActualizarClave no aplica. La credencial local no se utiliza.'
}

try {
    if (-not $dsnExterna) {
        New-Item -ItemType Directory -Path $carpetaPrivada -Force | Out-Null

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

        $puertoBD = 5432
        if (-not [string]::IsNullOrWhiteSpace($env:SARGAS_DB_PORT)) {
            if (-not [int]::TryParse($env:SARGAS_DB_PORT, [ref]$puertoBD) -or
                $puertoBD -lt 1 -or $puertoBD -gt 65535) {
                throw 'SARGAS_DB_PORT debe ser un puerto valido entre 1 y 65535.'
            }
        }

        $claveCodificada = [uri]::EscapeDataString($credencial.GetNetworkCredential().Password)
        $env:DATABASE_URL = "postgres://sentinel_app:$($claveCodificada)@127.0.0.1:$puertoBD/sargas_sentinel?sslmode=disable"
    }
    else {
        Write-Host 'Usando DATABASE_URL ya configurada (sin mostrar su contenido).'
    }

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
    if (-not $dsnExterna) {
        Remove-Item Env:DATABASE_URL -ErrorAction SilentlyContinue
    }
    Remove-Variable credencial, claveCodificada, puertoBD -ErrorAction SilentlyContinue
}

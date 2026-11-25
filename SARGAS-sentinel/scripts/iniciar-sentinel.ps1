# Inicio nativo del ejecutable, nunca de Docker.
param([switch]$ActualizarClave)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'configuracion-local.ps1')
$backend = Split-Path -Parent $PSScriptRoot
$ejecutable = Join-Path $backend 'sentinel.exe'
if (-not (Test-Path -LiteralPath $ejecutable -PathType Leaf)) {
    throw 'No existe sentinel.exe. Ejecuta .\sargas-dev.ps1 actualizar-local -Compilar desde la raiz.'
}
Confirmar-PuertoLibreLocal -Puerto 8080
Invocar-ConConexionLocal -Perfil app -ActualizarClave:$ActualizarClave -Operacion {
    $addrAnterior = [Environment]::GetEnvironmentVariable('SARGAS_HTTP_ADDR','Process')
    $puertoAnterior = [Environment]::GetEnvironmentVariable('SARGAS_PUBLIC_PORT','Process')
    Push-Location $backend
    try {
        $env:SARGAS_HTTP_ADDR = '127.0.0.1:8080'
        $env:SARGAS_PUBLIC_PORT = '8080'
        Write-Host 'Iniciando el ejecutable local (Ctrl+C para detener).'
        & $ejecutable
        if ($LASTEXITCODE -ne 0) { throw 'Sentinel termino con error.' }
    } finally {
        Pop-Location
        [Environment]::SetEnvironmentVariable('SARGAS_HTTP_ADDR',$addrAnterior,'Process')
        [Environment]::SetEnvironmentVariable('SARGAS_PUBLIC_PORT',$puertoAnterior,'Process')
    }
}

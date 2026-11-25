param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('configurar','diagnostico','dev','backend','frontend')]
    [string]$Accion,
    [switch]$ActualizarClave
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'configuracion-local.ps1')
$backend = Split-Path -Parent $PSScriptRoot
$raiz = Split-Path -Parent $backend
$frontend = Join-Path $raiz 'sargas-sentinel-web'

function Preparar-Frontend {
    if (-not (Test-Path -LiteralPath (Join-Path $frontend 'node_modules\.bin\vite.cmd') -PathType Leaf)) {
        Push-Location $frontend
        try {
            & npm.cmd ci
            if ($LASTEXITCODE -ne 0) { throw 'Fallo npm ci. Deten Vite si hay archivos bloqueados.' }
        } finally { Pop-Location }
    }
}

function Preparar-Embed {
    $destino = Join-Path $backend 'cmd\sentinel\web\dist'
    if (Test-Path -LiteralPath (Join-Path $destino 'index.html') -PathType Leaf) { return }
    Preparar-Frontend
    Push-Location $frontend
    try {
        & npm.cmd run build
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo preparar el frontend requerido por go:embed.' }
    } finally { Pop-Location }
    $origen = Join-Path $frontend 'dist'
    if (-not (Test-Path -LiteralPath (Join-Path $origen 'index.html') -PathType Leaf)) { throw 'Falta index.html compilado.' }
    # Solo se modifica el directorio generado de go:embed, no el codigo fuente.
    if (Test-Path -LiteralPath $destino) { Remove-Item -LiteralPath $destino -Recurse -Force }
    New-Item -ItemType Directory -Path $destino -Force | Out-Null
    Copy-Item -Path (Join-Path $origen '*') -Destination $destino -Recurse -Force
}

function Abrir-TerminalDev {
    $entrada = Join-Path $raiz 'sargas-dev.ps1'
    $hostPS = Join-Path $PSHOME 'powershell.exe'
    if (-not (Test-Path -LiteralPath $hostPS)) { $hostPS = Join-Path $PSHOME 'pwsh.exe' }
    $terminal = Get-Command wt.exe -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1

    if ($terminal) {
        # Una sola solicitud a Windows Terminal: ambas pestanas en la misma ventana.
        # -w 0 usa la ultima ventana activa o crea una si no existe.
        $comando = "`"$hostPS`" -NoLogo -NoProfile -NoExit -File `"$entrada`""
        $argumentos = "-w 0 new-tab -d `"$raiz`" --title `"SARGAS - Backend`" --suppressApplicationTitle $comando backend ; new-tab -d `"$raiz`" --title `"SARGAS - Frontend`" --suppressApplicationTitle $comando frontend"
        Start-Process -FilePath $terminal.Source -WorkingDirectory $raiz -ArgumentList $argumentos
        Write-Host 'Pestanas solicitadas en Windows Terminal: backend 8080 y frontend 5173.'
    } else {
        Write-Warning 'Windows Terminal (wt.exe) no disponible. Se abriran dos ventanas independientes.'
        foreach ($modo in @('backend','frontend')) {
            Start-Process -FilePath $hostPS -WorkingDirectory $raiz -ArgumentList "-NoLogo -NoProfile -NoExit -File `"$entrada`" $modo"
        }
        Write-Host 'Ventanas solicitadas: backend 8080 y frontend 5173.'
    }
}

switch ($Accion) {
    'configurar' {
        $null = Obtener-ConexionLocal -Perfil app -ActualizarClave:$ActualizarClave
        $null = Obtener-ConexionLocal -Perfil test -ActualizarClave:$ActualizarClave
        Preparar-Frontend
        Write-Host 'Configuracion local verificada. No se crearon ni modificaron bases o roles.' -ForegroundColor Green
    }
    'diagnostico' {
        foreach ($herramienta in @('go','node','npm.cmd','gcc','govulncheck')) {
            if (Get-Command $herramienta -ErrorAction SilentlyContinue) { Write-Host "OK: $herramienta" }
            else { Write-Warning "No disponible en PATH: $herramienta" }
        }
        foreach ($perfil in @('app','test')) {
            try { $null = Obtener-ConexionLocal -Perfil $perfil -SinPrompt; Write-Host "OK: PostgreSQL local ($perfil)" }
            catch { Write-Warning $_.Exception.Message }
        }
        foreach ($puerto in @(8080,5173)) {
            try { Confirmar-PuertoLibreLocal -Puerto $puerto; Write-Host "Puerto $puerto libre" }
            catch { Write-Warning $_.Exception.Message }
        }
        if (Test-Path -LiteralPath (Join-Path $frontend 'node_modules\.bin\vite.cmd')) { Write-Host 'OK: dependencias frontend presentes' }
        else { Write-Warning 'Faltan dependencias frontend. Ejecuta configurar.' }
    }
    'dev' {
        Confirmar-PuertoLibreLocal -Puerto 8080
        Confirmar-PuertoLibreLocal -Puerto 5173
        $null = Obtener-ConexionLocal -Perfil app -ActualizarClave:$ActualizarClave
        Preparar-Frontend
        Preparar-Embed
        Abrir-TerminalDev
        Write-Host 'Revisa las salidas para confirmar el inicio.'
        Write-Host 'Deten cada proceso con Ctrl+C en su terminal.'
    }
    'backend' {
        Confirmar-PuertoLibreLocal -Puerto 8080
        Preparar-Embed
        Invocar-ConConexionLocal -Perfil app -ActualizarClave:$ActualizarClave -Operacion {
            $addrAnterior = [Environment]::GetEnvironmentVariable('SARGAS_HTTP_ADDR','Process')
            $puertoAnterior = [Environment]::GetEnvironmentVariable('SARGAS_PUBLIC_PORT','Process')
            Push-Location $backend
            try {
                $env:SARGAS_HTTP_ADDR = '127.0.0.1:8080'
                $env:SARGAS_PUBLIC_PORT = '8080'
                & go run ./cmd/sentinel
                if ($LASTEXITCODE -ne 0) { throw 'El backend termino con error.' }
            } finally {
                Pop-Location
                [Environment]::SetEnvironmentVariable('SARGAS_HTTP_ADDR',$addrAnterior,'Process')
                [Environment]::SetEnvironmentVariable('SARGAS_PUBLIC_PORT',$puertoAnterior,'Process')
            }
        }
    }
    'frontend' {
        Confirmar-PuertoLibreLocal -Puerto 5173
        Preparar-Frontend
        Push-Location $frontend
        try {
            & npm.cmd run dev -- --host 127.0.0.1 --port 5173 --strictPort
            if ($LASTEXITCODE -ne 0) { throw 'Vite termino con error.' }
        } finally { Pop-Location }
    }
}

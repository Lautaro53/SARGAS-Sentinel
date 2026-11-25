# Funciones compartidas exclusivamente por el entorno nativo de Windows.
function Obtener-PuertoBDLocal {
    $puerto = 5432
    if ($env:SARGAS_DB_PORT -and (-not [int]::TryParse($env:SARGAS_DB_PORT, [ref]$puerto) -or $puerto -lt 1 -or $puerto -gt 65535)) {
        throw 'SARGAS_DB_PORT debe ser un puerto entre 1 y 65535.'
    }
    return $puerto
}

function Obtener-PsqlLocal {
    $cmd = Get-Command psql.exe -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $ruta = Join-Path $env:ProgramFiles 'PostgreSQL\18\bin\psql.exe'
    if (Test-Path -LiteralPath $ruta -PathType Leaf) { return $ruta }
    throw 'No se encontro psql.exe. Instala el cliente PostgreSQL o agregalo al PATH.'
}

function Obtener-ConexionLocal {
    param(
        [Parameter(Mandatory = $true)] [ValidateSet('app', 'test')] [string]$Perfil,
        [switch]$ActualizarClave,
        [switch]$SinPrompt
    )
    $usuario = "sentinel_$Perfil"
    $base = if ($Perfil -eq 'test') { 'sargas_sentinel_test' } else { 'sargas_sentinel' }
    $puerto = Obtener-PuertoBDLocal
    $psql = Obtener-PsqlLocal
    $privado = Join-Path $env:USERPROFILE 'SARGAS-privado'
    $archivo = Join-Path $privado "$usuario.credential.xml"
    $nueva = $ActualizarClave -or -not (Test-Path -LiteralPath $archivo -PathType Leaf)
    if ($nueva) {
        if ($SinPrompt) { throw "Falta la credencial local de $usuario. Ejecuta configurar." }
        $clave = Read-Host "Contrasena ACTUAL de PostgreSQL local para $usuario" -AsSecureString
        $credencial = [PSCredential]::new($usuario, $clave)
    } else {
        try { $credencial = Import-Clixml -LiteralPath $archivo }
        catch { throw "No se pudo leer la credencial de $usuario. Usa -ActualizarClave." }
    }
    if ($credencial -isnot [System.Management.Automation.PSCredential] -or $credencial.UserName -cne $usuario) {
        throw "Credencial incorrecta para $usuario. Usa -ActualizarClave."
    }
    # Verificar usuario y base antes de guardar o iniciar cualquier proceso Go.
    $password = $credencial.GetNetworkCredential().Password
    $anterior = [Environment]::GetEnvironmentVariable('PGPASSWORD', 'Process')
    $timeoutAnterior = [Environment]::GetEnvironmentVariable('PGCONNECT_TIMEOUT', 'Process')
    try {
        $env:PGPASSWORD = $password
        $env:PGCONNECT_TIMEOUT = '5'
        $respuesta = @(& $psql -X -w -h 127.0.0.1 -p $puerto -U $usuario -d $base -t -A -c "SELECT current_database() || '|' || current_user" 2>$null)
        if ($LASTEXITCODE -ne 0 -or ($respuesta -join '').Trim() -cne "$base|$usuario") {
            throw "No se pudo autenticar $usuario en $base (PostgreSQL local, puerto $puerto). Revisa el servidor o usa -ActualizarClave."
        }
    } finally {
        [Environment]::SetEnvironmentVariable('PGPASSWORD', $anterior, 'Process')
        [Environment]::SetEnvironmentVariable('PGCONNECT_TIMEOUT', $timeoutAnterior, 'Process')
    }
    if ($nueva) {
        New-Item -ItemType Directory -Path $privado -Force | Out-Null
        $temporal = "$archivo.$([guid]::NewGuid().ToString('N')).tmp"
        try {
            $credencial | Export-Clixml -LiteralPath $temporal
            Move-Item -LiteralPath $temporal -Destination $archivo -Force
        } finally {
            if (Test-Path -LiteralPath $temporal) { Remove-Item -LiteralPath $temporal -Force }
        }
        Write-Host "Credencial local de $usuario guardada (DPAPI)."
    }
    $codificada = [uri]::EscapeDataString($password)
    return "postgresql://${usuario}:${codificada}@127.0.0.1:${puerto}/${base}?sslmode=disable"
}

function Invocar-ConConexionLocal {
    param(
        [Parameter(Mandatory = $true)] [ValidateSet('app', 'test')] [string]$Perfil,
        [Parameter(Mandatory = $true)] [scriptblock]$Operacion,
        [switch]$ActualizarClave
    )
    $variable = if ($Perfil -eq 'test') { 'SARGAS_TEST_DATABASE_URL' } else { 'DATABASE_URL' }
    $anterior = [Environment]::GetEnvironmentVariable($variable, 'Process')
    try {
        $conexion = Obtener-ConexionLocal -Perfil $Perfil -ActualizarClave:$ActualizarClave
        [Environment]::SetEnvironmentVariable($variable, $conexion, 'Process')
        & $Operacion
    } finally {
        [Environment]::SetEnvironmentVariable($variable, $anterior, 'Process')
    }
}

function Confirmar-PuertoLibreLocal {
    param([int]$Puerto)
    $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, $Puerto)
    try { $listener.Server.ExclusiveAddressUse = $true; $listener.Start() }
    catch { throw "Puerto local $Puerto ocupado. Deten la instancia que lo utiliza (sin borrar datos) antes de continuar." }
    finally { $listener.Stop() }
}

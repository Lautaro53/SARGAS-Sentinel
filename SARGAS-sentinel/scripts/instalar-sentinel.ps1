# SARGAS Sentinel - Instalador Docker para Windows PowerShell 5.1+

$ErrorActionPreference = 'Stop'
$backend = Split-Path -Parent $PSScriptRoot
$raiz = Split-Path -Parent $backend
$frontend = Join-Path $raiz 'sargas-sentinel-web'
$compose = Join-Path $backend 'compose.yaml'
$privado = Join-Path $env:USERPROFILE 'SARGAS-privado\docker-secrets'
$archivoAdmin = Join-Path $privado 'postgres_password.txt'
$archivoApp = Join-Path $privado 'sentinel_app_password.txt'

function Nueva-Clave {
    # Caracteres compatibles con PostgreSQL y archivos de secretos de Docker.
    $alfabeto = 'abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789'
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    $resultado = New-Object System.Text.StringBuilder
    $buffer = New-Object byte[] 1
    $maximo = 256 - (256 % $alfabeto.Length)
    try {
        while ($resultado.Length -lt 32) {
            $rng.GetBytes($buffer)
            $valor = [int]$buffer[0]
            if ($valor -ge $maximo) { continue }
            [void]$resultado.Append($alfabeto[$valor % $alfabeto.Length])
        }
        return $resultado.ToString()
    }
    finally {
        $rng.Dispose()
    }
}

try {
    Write-Host ''
    Write-Host 'SARGAS Sentinel - Instalacion limpia con Docker' -ForegroundColor Cyan
    Write-Host 'No necesitas instalar npm, Node.js ni Go en Windows.'
    Write-Host ''

    # Comprobaciones antes de crear credenciales o contenedores.
    foreach ($archivo in @(
        $compose,
        (Join-Path $backend 'Dockerfile'),
        (Join-Path $frontend 'package.json'),
        (Join-Path $frontend 'package-lock.json'),
        (Join-Path $backend 'go.mod')
    )) {
        if (-not (Test-Path -LiteralPath $archivo -PathType Leaf)) {
            throw "Falta un archivo necesario: $archivo. Extrae el ZIP completo sin separar las dos carpetas."
        }
    }

    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        throw 'No se encontro Docker. Instala Docker Desktop, inicia la aplicacion y vuelve a intentar.'
    }

    Write-Host '[1/5] Comprobando Docker Desktop y Docker Compose...'
    $tipoMotor = & docker info --format '{{.OSType}}'
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker Engine no esta disponible. Inicia Docker Desktop y espera a que termine de arrancar.'
    }
    if ($tipoMotor.Trim() -ne 'linux') {
        throw 'Docker debe estar en modo contenedores Linux (Switch to Linux containers en Docker Desktop).'
    }

    $versionComposeTexto = & docker compose version --short
    if ($LASTEXITCODE -ne 0) { throw 'Docker Compose v2 no esta disponible.' }
    $coincidencia = [regex]::Match(($versionComposeTexto | Out-String), '(\d+)\.(\d+)\.(\d+)')
    if (-not $coincidencia.Success) {
        throw "No se pudo interpretar la version de Docker Compose: $versionComposeTexto"
    }
    $versionCompose = [version]::Parse(
        "$($coincidencia.Groups[1].Value).$($coincidencia.Groups[2].Value).$($coincidencia.Groups[3].Value)"
    )
    if ($versionCompose -lt [version]'2.17.0') {
        throw "Docker Compose $versionCompose es antiguo. Se necesita 2.17.0 o superior para compilar ambos proyectos."
    }

    & docker buildx version | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker Buildx no esta disponible. Actualiza Docker Desktop (el proyecto usa contextos de build adicionales).'
    }

    # Usar nombre fijo ya existente para proteger los volumenes de instalaciones previas.
    Write-Host '[2/5] Comprobando credenciales y volumen persistente...'
    $volumenBD = 'sargas-db-lab_postgres_data'
    $volumenes = @(& docker volume ls --format '{{.Name}}')
    if ($LASTEXITCODE -ne 0) { throw 'No se pudieron consultar los volumenes de Docker.' }
    $volumenExiste = $volumenes -contains $volumenBD

    $existeAdmin = Test-Path -LiteralPath $archivoAdmin -PathType Leaf
    $existeApp = Test-Path -LiteralPath $archivoApp -PathType Leaf

    if ($existeAdmin -ne $existeApp) {
        throw 'Falta uno de los dos secretos originales. No se regeneraran: recupera las credenciales de tu respaldo.'
    }
    if (-not $existeAdmin -and $volumenExiste) {
        throw "Ya existe el volumen $volumenBD, pero faltan sus credenciales. Recuperalas antes de continuar: no se tocara la base."
    }
    if ($existeAdmin) {
        if ((Get-Item -LiteralPath $archivoAdmin).Length -eq 0 -or
            (Get-Item -LiteralPath $archivoApp).Length -eq 0) {
            throw 'Uno de los secretos esta vacio. No se modificaran contrasenas existentes.'
        }
        Write-Host 'Credenciales originales reutilizadas.'
    }
    else {
        New-Item -ItemType Directory -Path $privado -Force | Out-Null
        $utf8 = [System.Text.UTF8Encoding]::new($false)
        $temporalAdmin = "$archivoAdmin.tmp"
        $temporalApp = "$archivoApp.tmp"
        try {
            [IO.File]::WriteAllText($temporalAdmin, (Nueva-Clave), $utf8)
            [IO.File]::WriteAllText($temporalApp, (Nueva-Clave), $utf8)
            Move-Item -LiteralPath $temporalAdmin -Destination $archivoAdmin -ErrorAction Stop
            Move-Item -LiteralPath $temporalApp -Destination $archivoApp -ErrorAction Stop
        }
        catch {
            # Eliminar solo archivos nuevos de esta instalacion incompleta.
            foreach ($archivo in @($temporalAdmin, $temporalApp, $archivoAdmin, $archivoApp)) {
                if (Test-Path -LiteralPath $archivo) {
                    Remove-Item -LiteralPath $archivo -Force
                }
            }
            throw
        }
        Write-Host "Credenciales nuevas creadas fuera del ZIP: $privado"
    }

    $env:SARGAS_DOCKER_SECRETS = $privado.Replace('\', '/')

    Push-Location $backend
    try {
        Write-Host '[3/5] Validando Compose...'
        & docker compose -f .\compose.yaml config --quiet
        if ($LASTEXITCODE -ne 0) { throw 'La configuracion Docker Compose es invalida.' }

        Write-Host '[4/5] Compilando e iniciando contenedores...'
        Write-Host 'Docker descargara imagenes y dependencias npm/Go durante la primera instalacion.'
        & docker compose -f .\compose.yaml up -d --build
        if ($LASTEXITCODE -ne 0) {
            throw 'No se pudo construir o iniciar SARGAS. Revisa el error de Docker mostrado arriba.'
        }

        Write-Host '[5/5] Verificando API y pagina web...'
        $api = 'http://127.0.0.1:8080/api/servicios'
        $web = 'http://127.0.0.1:8080/'
        $disponible = $false
        for ($intento = 1; $intento -le 30; $intento++) {
            try {
                $respApi = Invoke-WebRequest -Uri $api -Method Get -UseBasicParsing -TimeoutSec 3
                $respWeb = Invoke-WebRequest -Uri $web -Method Get -UseBasicParsing -TimeoutSec 3
                if ($respApi.StatusCode -eq 200 -and
                    $respWeb.StatusCode -eq 200 -and
                    $respWeb.Content -match '<html') {
                    $disponible = $true
                    break
                }
            }
            catch { }
            Start-Sleep -Seconds 2
        }
        if (-not $disponible) {
            Write-Warning 'SARGAS no respondio como se esperaba. Ultimos logs (no borraremos datos):'
            & docker compose -f .\compose.yaml ps
            & docker compose -f .\compose.yaml logs --tail=40 sentinel postgres
            throw 'La API o la web no superaron la verificacion.'
        }
        Write-Host ''
        Write-Host 'SARGAS Sentinel instalado y disponible.' -ForegroundColor Green
        Write-Host "Abrir: $web"
        Write-Host 'Los datos permanecen en el volumen de PostgreSQL.'
    }
    finally {
        Pop-Location
    }
}
catch {
    Write-Host ''
    Write-Host "ERROR: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host 'Los volumenes existentes no fueron borrados por el instalador.'
    exit 1
}

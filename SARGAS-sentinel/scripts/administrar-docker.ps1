# Administracion de la instalacion Docker existente. No construye ni crea contenedores.
[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [string]$Accion = 'ayuda',
    [ValidateRange(1, 1000)]
    [int]$UltimasLineas = 100
)

$ErrorActionPreference = 'Stop'
$proyecto = 'sargas-db-lab'

function Obtener-Contenedor {
    param([ValidateSet('sentinel', 'postgres')][string]$Servicio)
    $ids = @(& docker ps -aq --filter "label=com.docker.compose.project=$proyecto" `
        --filter "label=com.docker.compose.service=$Servicio" `
        --filter 'label=com.docker.compose.oneoff=False')
    if ($LASTEXITCODE -ne 0) { throw 'No se pudieron consultar los contenedores.' }
    if ($ids.Count -ne 1) {
        throw "Se esperaba un contenedor existente de $Servicio; encontrados: $($ids.Count). No se creara ni reemplazara ninguno."
    }
    return $ids[0]
}

function Obtener-Puerto {
    param([string]$Id)
    $json = & docker inspect --format '{{json .HostConfig.PortBindings}}' $Id
    if ($LASTEXITCODE -ne 0) { throw 'No se pudo consultar el puerto de Sentinel.' }
    $config = $json | ConvertFrom-Json
    $enlaces = @($config.'8080/tcp')
    $puerto = 0
    if ($enlaces.Count -ne 1 -or $null -eq $enlaces[0] -or
        $enlaces[0].HostIp -ne '127.0.0.1' -or
        -not [int]::TryParse([string]$enlaces[0].HostPort, [ref]$puerto) -or
        $puerto -lt 8080 -or $puerto -gt 8090) {
        throw 'El puerto publicado no coincide con una instalacion local de SARGAS. Revisar manualmente.'
    }
    return $puerto
}

function Ejecutar-Docker {
    param([string[]]$Argumentos)
    & docker @Argumentos
    if ($LASTEXITCODE -ne 0) { throw "Fallo docker $($Argumentos[0])." }
}

function Verificar-Motor {
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw 'Docker no esta disponible en PATH.' }
    $tipo = & docker info --format '{{.OSType}}'
    if ($LASTEXITCODE -ne 0 -or $tipo -ne 'linux') {
        throw 'Inicia Docker Desktop en modo contenedores Linux antes de continuar.'
    }
}

function Esperar-Postgres {
    param([string]$Id)
    for ($i = 0; $i -lt 60; $i++) {
        $salud = & docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}sin-healthcheck{{end}}' $Id
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo verificar PostgreSQL.' }
        if ($salud -eq 'healthy') { return }
        if ($salud -eq 'sin-healthcheck') { throw 'El contenedor PostgreSQL no tiene healthcheck.' }
        Start-Sleep -Seconds 2
    }
    throw 'PostgreSQL no alcanzo el estado healthy. Consulta los registros; no se eliminaran contenedores ni volumenes.'
}

function Esperar-Sentinel {
    param([int]$Puerto, [string]$Id)
    $url = "http://127.0.0.1:$Puerto"
    for ($i = 0; $i -lt 30; $i++) {
        $ejecutando = & docker inspect --format '{{.State.Running}}' $Id
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo verificar el estado de Sentinel.' }
        if ($ejecutando -ne 'true') { throw 'El contenedor Sentinel se detuvo. Consulta sargas-docker.cmd logs.' }
        try {
            $api = Invoke-WebRequest -Uri "$url/api/servicios" -UseBasicParsing -TimeoutSec 3
            $web = Invoke-WebRequest -Uri "$url/" -UseBasicParsing -TimeoutSec 3
            if ($api.StatusCode -eq 200 -and $api.Content.TrimStart().StartsWith('[') -and
                $web.StatusCode -eq 200 -and $web.Content -match '<html') {
                Write-Host "SARGAS disponible: $url/" -ForegroundColor Green
                return
            }
        } catch { }
        Start-Sleep -Seconds 2
    }
    throw 'SARGAS no supero la comprobacion HTTP. Consulta sargas-docker.cmd logs.'
}

function Iniciar-Instalacion {
    param([string]$Postgres, [string]$Sentinel, [int]$Puerto)
    Ejecutar-Docker -Argumentos @('start', $Postgres)
    Esperar-Postgres -Id $Postgres
    Ejecutar-Docker -Argumentos @('start', $Sentinel)
    Esperar-Sentinel -Puerto $Puerto -Id $Sentinel
}

function Detener-Instalacion {
    param([string]$Postgres, [string]$Sentinel)
    Ejecutar-Docker -Argumentos @('stop', $Sentinel)
    Ejecutar-Docker -Argumentos @('stop', $Postgres)
    Write-Host 'SARGAS detenido. Los datos se conservan.' -ForegroundColor Green
}

function Mostrar-Ayuda {
    Write-Host 'SARGAS Sentinel - Administracion Docker' -ForegroundColor Cyan
    Write-Host ''
    Write-Host '  .\sargas-docker.cmd ayuda                   Muestra esta ayuda'
    Write-Host '  .\sargas-docker.cmd iniciar                 Inicia PostgreSQL y Sentinel; verifica disponibilidad'
    Write-Host '  .\sargas-docker.cmd detener                 Detiene ambos contenedores; conserva los datos'
    Write-Host '  .\sargas-docker.cmd reiniciar               Detiene e inicia la instalacion existente'
    Write-Host '  .\sargas-docker.cmd estado                  Consulta contenedores activos y detenidos'
    Write-Host '  .\sargas-docker.cmd logs                    Muestra los registros de Sentinel'
    Write-Host '  .\sargas-docker.cmd logs -UltimasLineas 20   Muestra las ultimas 20 lineas'
    Write-Host '  .\sargas-docker.cmd version                 Consulta la version de la imagen instalada'
    Write-Host ''
    Write-Host 'Este script no instala ni reconstruye imagenes.'
}

function Ejecutar-Accion {
    param([string]$Operacion)
    if ($Operacion -eq 'ayuda') { Mostrar-Ayuda; return }
    Verificar-Motor
    switch ($Operacion) {
        'estado' {
            Ejecutar-Docker -Argumentos @('ps', '-a', '--filter', "label=com.docker.compose.project=$proyecto")
        }
        'logs' {
            $id = Obtener-Contenedor -Servicio sentinel
            Ejecutar-Docker -Argumentos @('logs', '--tail', [string]$UltimasLineas, $id)
        }
        'version' {
            $id = Obtener-Contenedor -Servicio sentinel

            $imagen = & docker inspect --format '{{.Image}}' $id
            if ($LASTEXITCODE -ne 0) {
                throw 'No se pudo identificar la imagen del contenedor.'
            }

            $json = & docker image inspect --format '{{json .Config.Labels}}' $imagen
            if ($LASTEXITCODE -ne 0) {
                throw 'No se pudieron consultar las etiquetas de la imagen.'
            }

            $etiquetas = $json | ConvertFrom-Json

            $version = [string]$etiquetas.'org.opencontainers.image.version'
            $origen = [string]$etiquetas.'io.sargas.build.origin'

            if ($version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+$') {
                Write-Host 'Version instalada: desconocida (imagen sin etiqueta de release).'
            } else {
                Write-Host "Version instalada: $version"
            }

            if ($origen -eq 'local') {
                Write-Host 'Origen: candidato local (no equivale a un release publicado).'
            } elseif ($origen -eq 'github-release') {
                Write-Host 'Origen: paquete verificado de GitHub Releases.'
            } elseif ($origen -eq 'paquete') {
                Write-Host 'Origen: compilacion del paquete de instalacion.'
            }

            Write-Host "Imagen del contenedor: $imagen"
            Write-Host "Puerto HTTP: $(Obtener-Puerto -Id $id)"
        }
        { $_ -in @('iniciar', 'detener', 'reiniciar') } {
            $sentinel = Obtener-Contenedor -Servicio sentinel
            $postgres = Obtener-Contenedor -Servicio postgres
            if ($Operacion -ne 'detener') { $puerto = Obtener-Puerto -Id $sentinel }
            if ($Operacion -ne 'iniciar') { Detener-Instalacion -Postgres $postgres -Sentinel $sentinel }
            if ($Operacion -ne 'detener') { Iniciar-Instalacion -Postgres $postgres -Sentinel $sentinel -Puerto $puerto }
        }
    }
}

if ($Accion -notin @('ayuda','iniciar','detener','reiniciar','estado','logs','version')) {
    Mostrar-Ayuda
    throw "Accion desconocida: '$Accion'."
}
if ($PSBoundParameters.ContainsKey('UltimasLineas') -and $Accion -ne 'logs') {
    Mostrar-Ayuda
    throw '-UltimasLineas solo corresponde a la accion logs.'
}
Ejecutar-Accion -Operacion $Accion

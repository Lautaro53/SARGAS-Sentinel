# SARGAS Sentinel - actualizaciones Docker de releases compatibles (Windows PowerShell 5.1+).
[CmdletBinding()]
param(
    [Parameter(Position = 0)][string]$Accion = 'ayuda',
    [switch]$Confirmar,
    [switch]$AceptarVersionDesconocida,
    [switch]$SoloPreparar
)
$ErrorActionPreference = 'Stop'
$raiz = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$privado = Join-Path $env:USERPROFILE 'SARGAS-privado'
$secretos = Join-Path $privado 'docker-secrets'
$proyecto = 'sargas-db-lab'
$repo = 'Lautaro53/SARGAS-Sentinel'
$patronVersion = '^v[0-9]+\.[0-9]+\.[0-9]+$'

function Mostrar-Ayuda {
    Write-Host 'SARGAS Sentinel - Actualizaciones Docker' -ForegroundColor Cyan
    Write-Host ''
    Write-Host '  .\sargas-actualizar.cmd ayuda                         Muestra esta ayuda'
    Write-Host '  .\sargas-actualizar.cmd buscar                        Consulta la ultima release estable'
    Write-Host '  .\sargas-actualizar.cmd actualizar -Confirmar         Descarga, respalda y actualiza Sentinel'
    Write-Host '  .\sargas-actualizar.cmd probar-local -SoloPreparar    Compila el candidato local sin desplegar'
    Write-Host '  .\sargas-actualizar.cmd probar-local -Confirmar       Despliega el candidato local (desarrollador)'
    Write-Host ''
    Write-Host '-AceptarVersionDesconocida permite adoptar una imagen antigua sin etiqueta solo hacia v1.2.0.'
    Write-Host 'No se actualiza PostgreSQL ni se modifica el codigo fuente.'
}
function Docker-Consulta {
    param([string[]]$Argumentos)
    $salida = @(& docker @Argumentos)
    if ($LASTEXITCODE -ne 0) { throw "Fallo docker $($Argumentos[0])." }
    return ($salida -join "`n")
}
function Docker-Ejecutar {
    param([string[]]$Argumentos)
    & docker @Argumentos | Out-Host
    if ($LASTEXITCODE -ne 0) { throw "Fallo docker $($Argumentos[0])." }
}
function Contenedor {
    param([string]$Servicio)
    $texto = Docker-Consulta @('ps','-aq','--filter',"label=com.docker.compose.project=$proyecto",'--filter',"label=com.docker.compose.service=$Servicio",'--filter','label=com.docker.compose.oneoff=False')
    $ids = @($texto -split '\r?\n' | Where-Object { $_ })
    if ($ids.Count -ne 1) { throw "Se esperaba exactamente un contenedor existente de $Servicio." }
    return (Docker-Consulta @('inspect','--format','{{json .}}',$ids[0]) | ConvertFrom-Json)
}
function Version-Imagen {
    param([string]$Imagen)
    $info = Docker-Consulta @('image','inspect','--format','{{json .Config.Labels}}',$Imagen) | ConvertFrom-Json
    $v = [string]$info.'org.opencontainers.image.version'
    if ($v -match $patronVersion) { return $v }
    return 'desconocida'
}
function Origen-Imagen {
    param([string]$Imagen)
    $info = Docker-Consulta @('image','inspect','--format','{{json .Config.Labels}}',$Imagen) | ConvertFrom-Json
    return [string]$info.'io.sargas.build.origin'
}
function Comparar-Version {
    param([string]$Nueva,[string]$Anterior)
    return ([version]$Nueva.Substring(1)).CompareTo([version]$Anterior.Substring(1))
}
function Ultima-Release {
    # No se consulta main ni se instala una prerelease.
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    $r = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -Headers @{ 'User-Agent'='SARGAS-Sentinel-Updater'; 'Accept'='application/vnd.github+json'; 'X-GitHub-Api-Version'='2022-11-28' } -TimeoutSec 30
    if ($r.draft -or $r.prerelease -or $r.tag_name -notmatch $patronVersion) { throw 'GitHub no devolvio una release estable valida.' }
    return $r
}
function Activo-Release {
    param($Release)
    $nombre = "SARGAS-Sentinel-$($Release.tag_name).zip"
    $activos = @($Release.assets | Where-Object { $_.name -ceq $nombre -and $_.state -eq 'uploaded' })
    if ($activos.Count -ne 1) { throw "La release no ofrece el paquete verificable $nombre. No se usara el ZIP automatico de GitHub." }
    $a = $activos[0]
    $url = "https://github.com/$repo/releases/download/$($Release.tag_name)/$nombre"
    if ($a.browser_download_url -cne $url -or $a.digest -notmatch '^sha256:[a-fA-F0-9]{64}$') { throw 'El activo no tiene URL y SHA-256 verificables.' }
    if ([long]$a.size -le 0 -or [long]$a.size -gt 100MB) { throw 'Tamano del paquete fuera del limite admitido.' }
    return $a
}
function Expandir-Paquete {
    param([string]$Zip,[string]$Destino)
    # Rechazar rutas absolutas, traversal, enlaces y colisiones antes de extraer.
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archivo = [IO.Compression.ZipFile]::OpenRead($Zip)
    try {
        $vistos = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
        $total = 0L
        foreach ($e in $archivo.Entries) {
            $n = $e.FullName.Replace('\','/')
            if ($n -notmatch '^[A-Za-z0-9_.@+ /-]+$' -or $n.StartsWith('/') -or $n.Contains('//') -or $n -match '(^|/)(\.{1,2}|[^/]*[. ])(/|$)' -or $n -match '(^|/)(CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])(\.|/|$)' -or $n -match '(^|/)(\.git|node_modules|Extras|docker-secrets|SARGAS-privado)(/|$)' -or $n -match '(^|/)\.env(\.|/|$)' -or $n -match '\.(exe|dll|credential\.xml|pem|key|dump|patch)$' -or (($e.ExternalAttributes -shr 16) -band 0xF000) -eq 0xA000) { throw 'El ZIP contiene una ruta o tipo de archivo no permitido.' }
            if (-not $vistos.Add($n.TrimEnd('/'))) { throw 'El ZIP contiene rutas duplicadas.' }
            $total += $e.Length
            if ($total -gt 300MB -or $vistos.Count -gt 15000) { throw 'El ZIP supera los limites de extraccion.' }
        }
    } finally { $archivo.Dispose() }
    Expand-Archive -LiteralPath $Zip -DestinationPath $Destino
}
function Hash-Texto {
    param([string]$Ruta)
    $texto = [IO.File]::ReadAllText($Ruta).Replace("`r`n","`n")
    $sha = [Security.Cryptography.SHA256]::Create()
    try { return ([BitConverter]::ToString($sha.ComputeHash([Text.Encoding]::UTF8.GetBytes($texto)))).Replace('-','').ToLowerInvariant() }
    finally { $sha.Dispose() }
}
function Verificar-Paquete {
    param([string]$Origen,[string]$Version)
    foreach ($f in @('VERSION','ACTUALIZACION.json','SARGAS-sentinel/Dockerfile','SARGAS-sentinel/go.mod','SARGAS-sentinel/go.sum','sargas-sentinel-web/package.json','sargas-sentinel-web/package-lock.json','SARGAS-sentinel/.dockerignore','sargas-sentinel-web/.dockerignore','sargas-docker.cmd','sargas-actualizar.cmd')) {
        if (-not (Test-Path -LiteralPath (Join-Path $Origen $f) -PathType Leaf)) { throw "Falta $f en el paquete." }
    }
    if ((Get-Content -LiteralPath (Join-Path $Origen 'VERSION') -Raw).Trim() -cne $Version) { throw 'VERSION no coincide con la release.' }
    $m = Get-Content -LiteralPath (Join-Path $Origen 'ACTUALIZACION.json') -Raw | ConvertFrom-Json
    if ($m.version -cne $Version -or $m.contrato -ne 'docker-v1-schema-v1' -or $m.migraciones -ne $false) { throw 'Esta version requiere un procedimiento de actualizacion diferente.' }
    # Este actualizador solo admite el contrato de despliegue y esquema de v1.2.0.
    $huellas = @{
        'SARGAS-sentinel/compose.yaml' = 'af3092754eab5dfe7d25034a86b1f35d4ba45f78435ba64a9bf063c1887e0506'
        'SARGAS-sentinel/docker/postgres/init/01-roles.sql' = 'c8763a6abfa618432a17ae4bd614fbf3da75454ce2ade0bb85440f8b941b8f9b'
        'SARGAS-sentinel/docker/postgres/init/02-schema.sql' = 'f8d7042279daa7be29c9a09cd18ccdddd6d11769c075c90cf2fb82d28c9b92c0'
        'SARGAS-sentinel/docker/postgres/init/03-permisos.sql' = '52e6869c746b562123d4bdc05b38b6ed4693f13ed8af09a5b2aaae696aec5781'
        'SARGAS-sentinel/docker/postgres/init/04-credenciales.sh' = 'fd8255a09e0e5878284b14389db1cca67a11904343c413f0b641d20d7d6cd9ba'
    }
    foreach ($f in $huellas.Keys) {
        if ((Hash-Texto (Join-Path $Origen $f)) -ne $huellas[$f]) { throw "Cambio no admitido automaticamente: $f." }
    }
    $init = @(Get-ChildItem -LiteralPath (Join-Path $Origen 'SARGAS-sentinel/docker/postgres/init') -Force)
    if ($init.Count -ne 4) { throw 'Se detectaron nuevos scripts SQL o de inicializacion.' }
    return $m
}
function Esperar-Postgres {
    param([string]$Id)
    for ($i=0; $i -lt 60; $i++) {
        if ((Docker-Consulta @('inspect','--format','{{if .State.Health}}{{.State.Health.Status}}{{end}}',$Id)) -eq 'healthy') { return }
        Start-Sleep -Seconds 2
    }
    throw 'PostgreSQL no alcanzo el estado healthy.'
}
function Esperar-API {
    param([int]$Puerto,[string]$Imagen)
    $url = "http://127.0.0.1:$Puerto"
    for ($i=0; $i -lt 30; $i++) {
        $s = Contenedor sentinel
        if ($s.Image -ne $Imagen) { throw 'El contenedor no utiliza la imagen esperada.' }
        try {
            $api = Invoke-WebRequest -Uri "$url/api/servicios" -UseBasicParsing -TimeoutSec 3
            $web = Invoke-WebRequest -Uri "$url/" -UseBasicParsing -TimeoutSec 3
            if ($s.State.Running -and $api.StatusCode -eq 200 -and $api.Content.TrimStart().StartsWith('[') -and $web.StatusCode -eq 200 -and $web.Content -match '<html') { return }
        } catch { }
        Start-Sleep -Seconds 2
    }
    throw 'La nueva instancia no supero la comprobacion HTTP.'
}
function Verificar-Instalacion {
    param($S,$PG)
    $enlaces = @($S.HostConfig.PortBindings.'8080/tcp')
    $puerto = 0
    if ($enlaces.Count -ne 1 -or $null -eq $enlaces[0] -or $enlaces[0].HostIp -ne '127.0.0.1' -or -not [int]::TryParse([string]$enlaces[0].HostPort,[ref]$puerto) -or $puerto -lt 8080 -or $puerto -gt 8090) { throw 'Puerto de instalacion no admitido.' }
    if (@($S.HostConfig.PortBindings.PSObject.Properties).Count -ne 1) { throw 'Hay puertos adicionales en Sentinel.' }
    if ($S.Config.Env -notcontains 'DATABASE_URL=postgres://sentinel_app@postgres:5432/sargas_sentinel?sslmode=disable' -or $S.Config.Env -notcontains 'SARGAS_HTTP_ADDR=0.0.0.0:8080' -or $S.Config.Env -notcontains "SARGAS_PUBLIC_PORT=$puerto") { throw 'La configuracion existente de Sentinel no coincide con el contrato admitido.' }
    if ($PG.Config.Env -notcontains 'POSTGRES_DB=sargas_sentinel' -or $PG.Config.Env -notcontains 'POSTGRES_USER=postgres' -or $PG.Config.Image -ne 'postgres:18') { throw 'La instancia PostgreSQL no coincide con la esperada.' }
    $vol = @($PG.Mounts | Where-Object { $_.Type -eq 'volume' -and $_.Name -eq 'sargas-db-lab_postgres_data' -and $_.Destination -eq '/var/lib/postgresql' })
    if ($vol.Count -ne 1) { throw 'No se identifico el volumen PostgreSQL esperado.' }
    foreach ($c in @($S,$PG)) {
        $redes = @($c.NetworkSettings.Networks.PSObject.Properties.Name)
        if ($redes.Count -ne 1 -or $redes[0] -ne 'sargas-db-lab_default') { throw 'Red Docker distinta de la esperada.' }
    }
    foreach ($nombre in @('postgres_password','sentinel_app_password')) {
        $f = Join-Path $secretos "$nombre.txt"
        if (-not (Test-Path -LiteralPath $f -PathType Leaf) -or (Get-Item -LiteralPath $f).Length -eq 0) { throw "Falta el secreto original $nombre. No se regenerara." }
        # Comprobar los mounts originales (Docker Desktop puede prefijar la ruta con /run/desktop/mnt/host).
        $ruta = $f.Replace('\','/').ToLowerInvariant()
        $rutaLinux = '/' + $ruta.Replace(':','')
        $m = @($PG.Mounts | Where-Object { $_.Destination -eq "/run/secrets/$nombre" })
        if ($m.Count -ne 1 -or $m[0].Type -ne 'bind' -or $m[0].RW) { throw 'Montaje de secreto PostgreSQL inesperado.' }
        $src = ([string]$m[0].Source).Replace('\','/').ToLowerInvariant()
        if ($src -ne $ruta -and -not $src.EndsWith($rutaLinux)) { throw 'Los secretos no corresponden a la instalacion existente.' }
        if ($nombre -eq 'sentinel_app_password') {
            $sm = @($S.Mounts | Where-Object { $_.Destination -eq "/run/secrets/$nombre" })
            if ($sm.Count -ne 1 -or $sm[0].Type -ne 'bind' -or $sm[0].RW -or $sm[0].Source -ne $m[0].Source) { throw 'El secreto de Sentinel no coincide con PostgreSQL.' }
        }
    }
    if (@($S.Config.Entrypoint).Count -ne 3 -or $S.Config.Entrypoint[0] -ne '/bin/sh' -or $S.Config.Entrypoint[1] -ne '-ec' -or $S.Config.Entrypoint[2].Trim().Replace("`r`n","`n") -ne ('export PGPASSWORD="$(cat /run/secrets/sentinel_app_password)"' + "`n" + 'exec /app/sentinel')) { throw 'Entrypoint de Sentinel distinto del contrato admitido.' }
    if ($S.State.Running -and -not $PG.State.Running) { throw 'Estado inicial incoherente: Sentinel activo y PostgreSQL detenido.' }
    if ($S.HostConfig.RestartPolicy.Name -ne 'unless-stopped' -or @($S.Config.Cmd | Where-Object { $_ }).Count -ne 0) { throw 'Configuracion de ejecucion distinta de la esperada.' }
    if (@($S.Mounts).Count -ne 1 -or $S.HostConfig.Privileged) { throw 'Sentinel tiene montajes o privilegios no admitidos.' }
    return $puerto
}
function Respaldo-BD {
    param([string]$Id,[string]$Directorio,[string]$Sufijo)
    $remoto = "/tmp/sargas-$Sufijo.dump"
    $destino = Join-Path $Directorio 'sargas_sentinel.dump'
    try {
        Docker-Ejecutar @('exec',$Id,'pg_dump','-U','postgres','-Fc','-d','sargas_sentinel','-f',$remoto)
        $lista = Docker-Consulta @('exec',$Id,'pg_restore','--list',$remoto)
        if ($lista -notmatch 'TABLE DATA') { throw 'El respaldo no contiene datos de tablas reconocibles.' }
        $lista | Set-Content -LiteralPath (Join-Path $Directorio 'contenido-respaldo.txt') -Encoding UTF8
        $hashRemoto = (Docker-Consulta @('exec',$Id,'sha256sum',$remoto)) -split '\s+' | Select-Object -First 1
        Docker-Ejecutar @('cp',"${Id}:$remoto",$destino)
        $hashLocal = (Get-FileHash -LiteralPath $destino -Algorithm SHA256).Hash
        if ((Get-Item -LiteralPath $destino).Length -le 0 -or $hashLocal -ine $hashRemoto) { throw 'El respaldo copiado no coincide con el original.' }
        $hashLocal | Set-Content -LiteralPath "$destino.sha256" -Encoding ASCII
    } finally {
        & docker exec $Id rm -f $remoto | Out-Null
    }
}

if ($Accion -eq 'ayuda') {
    Mostrar-Ayuda
    if ($Confirmar -or $AceptarVersionDesconocida -or $SoloPreparar) { throw 'Las opciones no corresponden a ayuda.' }
    return
}
if ($Accion -notin @('buscar','actualizar','probar-local')) { Mostrar-Ayuda; throw "Accion desconocida: '$Accion'." }
if (($Accion -eq 'buscar' -and ($Confirmar -or $AceptarVersionDesconocida -or $SoloPreparar)) -or ($SoloPreparar -and ($Accion -ne 'probar-local' -or $Confirmar -or $AceptarVersionDesconocida))) { Mostrar-Ayuda; throw 'Combinacion de opciones no valida.' }
if ($Accion -ne 'buscar' -and -not $SoloPreparar -and -not $Confirmar) { Mostrar-Ayuda; throw 'Para desplegar se requiere -Confirmar de forma explicita.' }
if (-not (Get-Command docker -ErrorAction SilentlyContinue) -or (Docker-Consulta @('info','--format','{{.OSType}}')) -ne 'linux') { throw 'Inicia Docker Desktop en modo Linux.' }
$origen = $raiz
$release = $null
if ($Accion -eq 'probar-local') { $version = (Get-Content -LiteralPath (Join-Path $raiz 'VERSION') -Raw).Trim() }
else { $release = Ultima-Release; $version = [string]$release.tag_name }
if ($version -notmatch $patronVersion) { throw 'Version candidata invalida.' }

# Solo preparar puede ejecutarse sin una instalacion previa. No inicia contenedores.
if (-not $SoloPreparar) {
    $s = Contenedor sentinel
    $instalada = Version-Imagen $s.Image
    $origenInstalado = Origen-Imagen $s.Image
    Write-Host "Version instalada: $instalada"
    if ($origenInstalado -eq 'local') { Write-Host 'Origen: candidato local; no equivale al paquete publicado.' }
    Write-Host "Version candidata: $version"
    if ($instalada -ne 'desconocida' -and ((Comparar-Version $version $instalada) -lt 0 -or ((Comparar-Version $version $instalada) -eq 0 -and $origenInstalado -ne 'local'))) {
        Write-Host 'No hay una version mas reciente. No se realizaran cambios.'
        return
    }
    if ($Accion -eq 'buscar') {
        if ($instalada -eq 'desconocida') { Write-Warning 'No se puede determinar si hay una actualizacion: la imagen antigua no tiene version.' }
        elseif ($version -eq $instalada) { Write-Host 'La release publicada puede reemplazar al candidato local de la misma version.' }
        else { Write-Host 'Hay una version estable mas reciente.' }
        try { $null = Activo-Release $release; Write-Host 'Paquete verificable disponible.' }
        catch { Write-Warning $_.Exception.Message }
        return
    }
    if ($instalada -eq 'desconocida' -and (-not $AceptarVersionDesconocida -or $version -ne 'v1.2.0')) { throw 'Version instalada desconocida. Solo se permite una adopcion explicita hacia v1.2.0.' }
    if ($instalada -ne 'desconocida' -and $AceptarVersionDesconocida) { throw 'La version instalada ya es conocida; no corresponde esa opcion.' }
    $pg = Contenedor postgres
    $puerto = Verificar-Instalacion $s $pg
} else { $puerto = 8080 }

# El directorio de transaccion queda fuera del repositorio, incluso si se actualiza desde un clon.
$marca = (Get-Date -Format 'yyyyMMdd-HHmmss') + '-' + [guid]::NewGuid().ToString('N').Substring(0,8)
$trabajo = Join-Path $privado "actualizaciones\$marca"
New-Item -ItemType Directory -Path $trabajo -Force | Out-Null
$lock = $null
$envAnterior = @{}
foreach ($n in @('SARGAS_DOCKER_SECRETS','SARGAS_HOST_PORT','SARGAS_BUILD_VERSION')) { $envAnterior[$n] = [Environment]::GetEnvironmentVariable($n,'Process') }
$inicioOperacion = $false
$intentoDespliegue = $false
$imagenAnterior = $null
try {
    $lock = [IO.File]::Open((Join-Path $privado 'actualizacion.lock'),[IO.FileMode]::OpenOrCreate,[IO.FileAccess]::ReadWrite,[IO.FileShare]::None)
    if ($null -ne $release) {
        $a = Activo-Release $release
        $zip = Join-Path $trabajo $a.name
        Invoke-WebRequest -Uri $a.browser_download_url -UseBasicParsing -OutFile $zip -TimeoutSec 180
        if ((Get-Item -LiteralPath $zip).Length -ne [long]$a.size -or (Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash -ine $a.digest.Substring(7)) { throw 'El paquete no coincide con el tamano o SHA-256 publicado.' }
        $origen = Join-Path $trabajo 'paquete'
        Expandir-Paquete $zip $origen
    }
    $manifest = Verificar-Paquete $origen $version
    if (-not $SoloPreparar -and $instalada -ne 'desconocida' -and $manifest.desde -notcontains $instalada) { throw "La release no declara compatibilidad desde $instalada." }
    if (-not $SoloPreparar -and $instalada -eq 'desconocida' -and $manifest.admiteLegacy -ne $true) { throw 'El paquete no permite adoptar instalaciones antiguas.' }
    $env:SARGAS_DOCKER_SECRETS = $secretos.Replace('\','/')
    $env:SARGAS_HOST_PORT = [string]$puerto
    $env:SARGAS_BUILD_VERSION = $version
    $compose = Join-Path $origen 'SARGAS-sentinel\compose.yaml'
    $nuevoTag = "sargas-sentinel:candidato-$version-$marca"
    $override = Join-Path $trabajo 'imagen-nueva.yaml'
    $origenBuild = if ($null -eq $release) { 'local' } else { 'github-release' }
    @('services:','  sentinel:',"    image: $nuevoTag",'    build:','      args:',"        SARGAS_BUILD_ORIGIN: $origenBuild") | Set-Content -LiteralPath $override -Encoding ASCII
    $envVacio = Join-Path $trabajo 'compose.env'
    [IO.File]::WriteAllText($envVacio,'')
    $base = @('compose','--env-file',$envVacio,'-p',$proyecto,'-f',$compose,'-f',$override)
    Docker-Ejecutar ($base + @('config','--quiet'))
    Write-Host 'Construyendo exclusivamente la imagen de Sentinel...'
    Docker-Ejecutar ($base + @('build','sentinel'))
    $nuevaImagen = Docker-Consulta @('image','inspect','--format','{{.Id}}',$nuevoTag)
    if ((Version-Imagen $nuevaImagen) -ne $version -or (Origen-Imagen $nuevaImagen) -ne $origenBuild) { throw 'La imagen construida no tiene la version y origen esperados.' }
    if ($SoloPreparar) { Write-Host "Imagen candidata preparada: $nuevoTag ($nuevaImagen). No se desplego."; return }

    # Revalidar tras la compilacion; no actuar si otra operacion cambio los contenedores.
    $actualS = Contenedor sentinel; $actualPG = Contenedor postgres
    if ($actualS.Id -ne $s.Id -or $actualPG.Id -ne $pg.Id -or $actualS.State.Running -ne $s.State.Running -or $actualPG.State.Running -ne $pg.State.Running) { throw 'La instalacion cambio durante la preparacion. Reintenta.' }
    $imagenAnterior = $s.Image
    $anteriorTag = "sargas-sentinel:respaldo-$marca"
    Docker-Ejecutar @('image','tag',$imagenAnterior,$anteriorTag)
    $rollback = Join-Path $trabajo 'imagen-anterior.yaml'
    @('services:','  sentinel:',"    image: $anteriorTag") | Set-Content -LiteralPath $rollback -Encoding ASCII
    $baseAnterior = @('compose','--env-file',$envVacio,'-p',$proyecto,'-f',$compose,'-f',$rollback)
    @{ proyecto=$proyecto; puerto=$puerto; postgresId=$pg.Id; imagenAnterior=$imagenAnterior; etiquetaAnterior=$anteriorTag; compose=$compose; overrideAnterior=$rollback; archivoEnv=$envVacio; secretos=$secretos; sentinelActivo=$s.State.Running; postgresActivo=$pg.State.Running } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $trabajo 'recuperacion.json') -Encoding UTF8
    $inicioOperacion = $true
    if ($s.State.Running) { Docker-Ejecutar @('stop',$s.Id) }
    if (-not $pg.State.Running) { Docker-Ejecutar @('start',$pg.Id) }
    Esperar-Postgres $pg.Id
    Write-Host 'Generando respaldo de PostgreSQL Docker...'
    Respaldo-BD $pg.Id $trabajo $marca
    $intentoDespliegue = $true
    Docker-Ejecutar ($base + @('up','-d','--no-deps','--no-build','--pull','never','--force-recreate','sentinel'))
    if ((Contenedor postgres).Id -ne $pg.Id) { throw 'Cambio inesperado del contenedor PostgreSQL.' }
    Esperar-API $puerto $nuevaImagen
    Write-Host "SARGAS actualizado a $version (origen: $origenBuild). http://127.0.0.1:$puerto/" -ForegroundColor Green
    Write-Host "Respaldo y datos de recuperacion: $trabajo"
    if ($null -ne $release) { Write-Host "Usa los comandos del paquete actualizado en: $origen" }
} catch {
    $errorOriginal = $_.Exception.Message
    if ($inicioOperacion) {
        try {
            if ((Contenedor postgres).Id -ne $pg.Id) { throw 'PostgreSQL cambio; no se intentara una recuperacion automatica.' }
            if ($intentoDespliegue) {
                Docker-Ejecutar ($baseAnterior + @('up','-d','--no-deps','--no-build','--pull','never','--force-recreate','sentinel'))
                Esperar-API $puerto $imagenAnterior
            } elseif ($s.State.Running) {
                Docker-Ejecutar @('start',$s.Id)
                Esperar-API $puerto $imagenAnterior
            }
            if (-not $s.State.Running -and $intentoDespliegue) { Docker-Ejecutar @('stop',(Contenedor sentinel).Id) }
            if (-not $pg.State.Running) { Docker-Ejecutar @('stop',$pg.Id) }
            Write-Warning 'Se recupero la imagen anterior y el estado de ejecucion original.'
        } catch { Write-Warning "No se pudo completar la recuperacion automatica: $($_.Exception.Message). Consulta $trabajo\recuperacion.json." }
    }
    throw "Actualizacion no completada: $errorOriginal. Archivos de diagnostico: $trabajo. No borres el volumen."
} finally {
    if ($null -ne $lock) { $lock.Dispose() }
    foreach ($n in $envAnterior.Keys) { [Environment]::SetEnvironmentVariable($n,$envAnterior[$n],'Process') }
}

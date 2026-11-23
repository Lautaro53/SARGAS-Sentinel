# SARGAS Sentinel - Punto de entrada para desarrollo y mantenimiento local.
# Uso: .\sargas.ps1 [accion] [-Compilar] [-ActualizarClave] [-SinRace]
[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('ayuda', 'compilar', 'iniciar', 'actualizar', 'validar', 'docker-estado', 'docker-logs', 'git-estado')]
    [string]$Accion = 'ayuda',
    [switch]$Compilar,
    [switch]$ActualizarClave,
    [switch]$SinRace,
    [ValidateRange(1, 1000)]
    [int]$UltimasLineas = 100
)

$ErrorActionPreference = 'Stop'
$raiz = $PSScriptRoot
$scripts = Join-Path (Join-Path $raiz 'SARGAS-sentinel') 'scripts'

# Evitar opciones ignoradas silenciosamente por una accion distinta.
$opcionesPorAccion = @{
    'ayuda'         = @('Accion')
    'compilar'      = @('Accion')
    'iniciar'       = @('Accion', 'ActualizarClave')
    'actualizar'    = @('Accion', 'Compilar')
    'validar'       = @('Accion', 'SinRace')
    'docker-estado' = @('Accion')
    'docker-logs'   = @('Accion', 'UltimasLineas')
    'git-estado'    = @('Accion')
}

foreach ($opcion in $PSBoundParameters.Keys) {
    if ($opcion -notin $opcionesPorAccion[$Accion]) {
        throw "El parametro -$opcion no corresponde a la accion '$Accion'."
    }
}

function Ejecutar-Script {
    param(
        [Parameter(Mandatory = $true)] [string]$Nombre,
        [object[]]$Argumentos = @()
    )

    $ruta = Join-Path $scripts $Nombre
    if (-not (Test-Path -LiteralPath $ruta -PathType Leaf)) {
        throw "No se encontro el script requerido: $Nombre"
    }
    & $ruta @Argumentos
}

switch ($Accion) {
    'ayuda' {
        Write-Host 'SARGAS Sentinel - Operaciones locales' -ForegroundColor Cyan
        Write-Host ''
        Write-Host '  .\sargas.ps1 compilar                  Compila frontend y backend (sin tests integrados)'
        Write-Host '  .\sargas.ps1 iniciar                   Inicia sentinel.exe con su configuracion'
        Write-Host '  .\sargas.ps1 iniciar -ActualizarClave  Actualiza credencial local (si aplica)'
        Write-Host '  .\sargas.ps1 actualizar               Instala sentinel-nuevo.exe con respaldo'
        Write-Host '  .\sargas.ps1 actualizar -Compilar      Compila e instala con respaldo'
        Write-Host '  .\sargas.ps1 validar                  Validacion completa; requiere BD de pruebas'
        Write-Host '  .\sargas.ps1 validar -SinRace         Diagnostico parcial; NO certifica release'
        Write-Host '  .\sargas.ps1 docker-estado             Consulta el estado de Docker Compose'
        Write-Host '  .\sargas.ps1 docker-logs               Muestra logs (opcional -UltimasLineas 200)'
        Write-Host '  .\sargas.ps1 git-estado                Consulta rama, cambios y ultimos commits'
        Write-Host ''
        Write-Host 'Instalacion Docker: .\instalar.cmd (flujo separado).'
        Write-Host 'No se realizan commits, push, merge ni etiquetas desde este script.'
    }
    'compilar' {
        Ejecutar-Script -Nombre 'compilar-sentinel.ps1'
    }
    'iniciar' {
        if ($ActualizarClave) {
            Ejecutar-Script -Nombre 'iniciar-sentinel.ps1' -Argumentos @('-ActualizarClave')
        } else {
            Ejecutar-Script -Nombre 'iniciar-sentinel.ps1'
        }
    }
    'actualizar' {
        if ($Compilar) {
            Ejecutar-Script -Nombre 'actualizar-sentinel.ps1' -Argumentos @('-Compilar')
        } else {
            Ejecutar-Script -Nombre 'actualizar-sentinel.ps1'
        }
    }
    'validar' {
        if ($SinRace) {
            Write-Warning 'Se omitira el detector de carreras: NO certifica la release.'
            Ejecutar-Script -Nombre 'validar-release.ps1' -Argumentos @('-SinRace')
        } else {
            Ejecutar-Script -Nombre 'validar-release.ps1'
        }
    }
    'docker-estado' {
        Ejecutar-Script -Nombre 'administrar-docker.ps1' -Argumentos @('-Accion', 'estado')
    }
    'docker-logs' {
        Ejecutar-Script -Nombre 'administrar-docker.ps1' -Argumentos @('-Accion', 'logs', '-UltimasLineas', $UltimasLineas)
    }
    'git-estado' {
        if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
            throw 'Git no esta disponible en el PATH.'
        }
        & git -C $raiz status --short --branch
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo consultar el estado de Git.' }
        & git -C $raiz log -5 --oneline
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo consultar el historial de Git.' }
        & git -C $raiz tag --list 'v1.2*'
        if ($LASTEXITCODE -ne 0) { throw 'No se pudieron consultar las etiquetas de Git.' }
    }
}

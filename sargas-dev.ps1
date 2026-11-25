# SARGAS Sentinel - Punto de entrada para desarrollo y mantenimiento local.
# Uso: .\sargas-dev.ps1 [accion] [-Compilar] [-ActualizarClave] [-SinRace]
[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [string]$Accion = 'ayuda',
    [switch]$Compilar,
    [switch]$ActualizarClave,
    [switch]$SinRace
)

$ErrorActionPreference = 'Stop'
$raiz = $PSScriptRoot
$scripts = Join-Path (Join-Path $raiz 'SARGAS-sentinel') 'scripts'

# Evitar opciones ignoradas silenciosamente por una accion distinta.
$opcionesPorAccion = @{
    'ayuda'         = @('Accion')
    'configurar'    = @('Accion', 'ActualizarClave')
    'diagnostico'   = @('Accion')
    'dev'           = @('Accion', 'ActualizarClave')
    'backend'       = @('Accion', 'ActualizarClave')
    'frontend'      = @('Accion')
    'actualizar-local' = @('Accion', 'Compilar')
    'compilar'      = @('Accion')
    'iniciar'       = @('Accion', 'ActualizarClave')
    'validar'       = @('Accion', 'SinRace', 'ActualizarClave')
}

function Mostrar-Ayuda {
    Write-Host 'SARGAS Sentinel - Herramientas de desarrollo local' -ForegroundColor Cyan
    Write-Host ''
    Write-Host '  .\sargas-dev.ps1 ayuda                           Muestra esta ayuda'
    Write-Host '  .\sargas-dev.ps1 configurar                      Verifica y guarda credenciales locales; prepara dependencias'
    Write-Host '  .\sargas-dev.ps1 diagnostico                     Comprueba herramientas, conexiones locales y puertos'
    Write-Host '  .\sargas-dev.ps1 dev                             Ejecuta el backend y el frontend actual'
    Write-Host '  .\sargas-dev.ps1 backend                         Ejecuta el codigo Go actual'
    Write-Host '  .\sargas-dev.ps1 frontend                        Ejecuta Vite en modo desarrollo'
    Write-Host '  .\sargas-dev.ps1 compilar                        Compila frontend y backend (sin tests integrados)'
    Write-Host '  .\sargas-dev.ps1 iniciar                         Inicia sentinel.exe con su configuracion'
    Write-Host '  .\sargas-dev.ps1 iniciar -ActualizarClave        Actualiza credencial local (si aplica)'
    Write-Host '  .\sargas-dev.ps1 actualizar-local                Instala sentinel-nuevo.exe con respaldo'
    Write-Host '  .\sargas-dev.ps1 actualizar-local -Compilar      Compila e instala con respaldo'
    Write-Host '  .\sargas-dev.ps1 validar                         Validacion completa con credencial local de pruebas'
    Write-Host '  .\sargas-dev.ps1 validar -SinRace                Diagnostico parcial; NO certifica release'
    Write-Host '  .\sargas-dev.ps1 validar -ActualizarClave        Renueva la credencial local de pruebas'
    Write-Host ''
    Write-Host 'Docker: .\sargas-instalar.cmd ayuda / .\sargas-docker.cmd ayuda'
}

if (-not $opcionesPorAccion.ContainsKey($Accion)) {
    Mostrar-Ayuda
    throw "Accion desconocida: '$Accion'."
}

foreach ($opcion in $PSBoundParameters.Keys) {
    if ($opcion -notin $opcionesPorAccion[$Accion]) {
        Mostrar-Ayuda
        throw "El parametro -$opcion no corresponde a la accion '$Accion'."
    }
}

function Ejecutar-Script {
    param(
        [Parameter(Mandatory = $true)] [string]$Nombre,
        [hashtable]$Argumentos = @{}
    )

    $ruta = Join-Path $scripts $Nombre

    if (-not (Test-Path -LiteralPath $ruta -PathType Leaf)) {
        throw "No se encontro el script requerido: $Nombre"
    }

    & $ruta @Argumentos
}

switch ($Accion) {
    'ayuda' { Mostrar-Ayuda }
    { $_ -in @('configurar','diagnostico','dev','backend','frontend') } {
        Ejecutar-Script -Nombre 'desarrollo-local.ps1' -Argumentos @{Accion = $Accion; ActualizarClave = [bool]$ActualizarClave}
    }
    'compilar' {
        Ejecutar-Script -Nombre 'compilar-sentinel.ps1'
    }
    'iniciar' {
        if ($ActualizarClave) {
            Ejecutar-Script -Nombre 'iniciar-sentinel.ps1' -Argumentos @{ActualizarClave = $true}
        } else {
            Ejecutar-Script -Nombre 'iniciar-sentinel.ps1'
        }
    }
    'actualizar-local' {
        if ($Compilar) {
            Ejecutar-Script -Nombre 'actualizar-sentinel.ps1' -Argumentos @{Compilar = $true}
        } else {
            Ejecutar-Script -Nombre 'actualizar-sentinel.ps1'
        }
    }
    'validar' {
        . (Join-Path $scripts 'configuracion-local.ps1')
        if ($SinRace) { Write-Warning 'Se omitira el detector de carreras: NO certifica la release.' }
        Invocar-ConConexionLocal -Perfil test -ActualizarClave:$ActualizarClave -Operacion {
            Ejecutar-Script -Nombre 'validar-release.ps1' -Argumentos @{SinRace = [bool]$SinRace}
        }
    }
}

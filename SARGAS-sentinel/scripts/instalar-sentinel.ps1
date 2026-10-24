
$ErrorActionPreference = "Stop"

$backend = Split-Path -Parent $PSScriptRoot

$compose = Join-Path $backend "compose.yaml"

if (-not (Test-Path -LiteralPath $compose -PathType Leaf)) {
    throw "No se encontro compose.yaml junto al proyecto SARGAS Sentinel."
}

$privado = Join-Path $env:USERPROFILE "SARGAS-privado\docker-secrets"

Write-Host "SARGAS Sentinel - Instalacion Docker"

# 1. Comprobar Docker
& docker info --format '{{.ServerVersion}}' | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "Docker Engine no esta funcionando."
}

& docker compose version | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "Docker Compose no esta disponible."
}

# 2. Preparar secretos privados
New-Item -ItemType Directory -Force -Path $privado | Out-Null

$archivoAdmin = Join-Path $privado "postgres_password.txt"
$archivoApp = Join-Path $privado "sentinel_app_password.txt"

$existeAdmin = Test-Path -LiteralPath $archivoAdmin
$existeApp = Test-Path -LiteralPath $archivoApp

if ($existeAdmin -ne $existeApp) {
    throw "Instalacion incompleta: falta uno de los secretos. No se modificaran credenciales."
}


# Impedir generar secretos nuevos si ya existe una base Docker.
# El nombre corresponde a "name: sargas-db-lab" en compose.yaml.
$volumenBD = "sargas-db-lab_postgres_data"

& docker volume inspect $volumenBD 2>$null | Out-Null
$volumenExiste = ($LASTEXITCODE -eq 0)

if (-not $existeAdmin -and $volumenExiste) {
    throw @"
Se detecto un volumen PostgreSQL existente, pero faltan las credenciales.

Volumen: $volumenBD

SARGAS no generara nuevas contrasenas porque podrian
ser incompatibles con la base existente.

Recupera los archivos de credenciales originales antes
de volver a ejecutar el instalador.
"@
}


if (-not $existeAdmin) {
    # Evitar escribir contrasenas directamente en el codigo.
    $alfabeto = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    $utf8 = New-Object System.Text.UTF8Encoding($false)

    function Nueva-Clave {
    $alfabeto = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
    $resultado = New-Object System.Text.StringBuilder
    $maximo = 256 - (256 % $alfabeto.Length)
    $buffer = New-Object byte[] 1

    while ($resultado.Length -lt 32) {
        $rng.GetBytes($buffer)
        $valor = [int]$buffer[0]

        if ($valor -ge $maximo) {
            continue
        }

        $indice = $valor % $alfabeto.Length
        [void]$resultado.Append($alfabeto[$indice])
    }

    return $resultado.ToString()
}

    try {
        [IO.File]::WriteAllText($archivoAdmin, (Nueva-Clave), $utf8)
        [IO.File]::WriteAllText($archivoApp, (Nueva-Clave), $utf8)
    }
    finally {
        $rng.Dispose()
    }

    Write-Host "Credenciales nuevas generadas."
}
else {
    if ((Get-Item $archivoAdmin).Length -eq 0 -or
        (Get-Item $archivoApp).Length -eq 0) {
        throw "Uno de los archivos de credenciales esta vacio."
    }

    Write-Host "Credenciales existentes reutilizadas."
}

# 3. Configurar Compose
$env:SARGAS_DOCKER_SECRETS = $privado.Replace("\", "/")

Push-Location $backend
try {
    & docker compose -f .\compose.yaml config --quiet
    if ($LASTEXITCODE -ne 0) {
        throw "Configuracion Compose invalida."
    }

    # 4. Compilar e iniciar
    & docker compose -f .\compose.yaml up -d --build
    if ($LASTEXITCODE -ne 0) {
        throw "No se pudo iniciar SARGAS."
    }

    # 5. Mostrar estado
    & docker compose -f .\compose.yaml ps


    # 6. Verificar que la API realmente este disponible
    Write-Host "Verificando disponibilidad de SARGAS..."

    $api = "http://127.0.0.1:8080/api/servicios"
    $disponible = $false

    for ($intento = 1; $intento -le 15; $intento++) {
        try {
            $respuesta = Invoke-WebRequest `
                -Uri $api `
                -Method Get `
                -UseBasicParsing `
                -TimeoutSec 3

            if ($respuesta.StatusCode -eq 200) {
                $disponible = $true
                break
            }
        }
        catch {
            Start-Sleep -Seconds 2
        }
    }

    if (-not $disponible) {
        Write-Host "La API no respondio correctamente." -ForegroundColor Red
        & docker compose -f .\compose.yaml logs --tail=30 sentinel
        throw "SARGAS no supero la verificacion de disponibilidad."
    }

    Write-Host "API SARGAS disponible (HTTP 200)." -ForegroundColor Green

}
finally {
    Pop-Location
}

Write-Host ""
Write-Host "Instalacion Docker ejecutada." -ForegroundColor Green
Write-Host "Abri http://127.0.0.1:8080/"
Write-Host "Los datos se conservan en el volumen de PostgreSQL."

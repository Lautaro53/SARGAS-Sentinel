# Correccion de dependencias indicada en el informe de integridad de v1.
# No altera PostgreSQL ni Docker. Requiere acceso a proxy.golang.org o mirror configurado.
$ErrorActionPreference = 'Stop'
$backend = Split-Path -Parent $PSScriptRoot
Push-Location $backend
try {
    $version = (& go version)
    if ($LASTEXITCODE -ne 0 -or $version -notmatch 'go1\.27\.(?:[3-9]|[1-9][0-9])\b') {
        throw "Se requiere Go 1.27.3 o revision posterior de Go 1.27. Detectado: $version"
    }

    & go get 'golang.org/x/text@v0.41.0'
    if ($LASTEXITCODE -ne 0) { throw 'No se pudo actualizar golang.org/x/text.' }
    & go mod tidy
    if ($LASTEXITCODE -ne 0) { throw 'Fallo go mod tidy.' }
    & go mod verify
    if ($LASTEXITCODE -ne 0) { throw 'Fallo go mod verify.' }
    Write-Host 'Dependencias de Go actualizadas. Revisar go.mod y go.sum.' -ForegroundColor Green
}
finally {
    Pop-Location
}

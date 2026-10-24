# Validacion exigente para la version 1; requiere base de PRUEBAS, nunca la principal.
# Configurar SARGAS_TEST_DATABASE_URL con la base sargas_sentinel_test.
# -SinRace es SOLO diagnostico si Windows carece de GCC; no certifica el release.
param([switch] $SinRace)

$ErrorActionPreference = 'Stop'
$backend = Split-Path -Parent $PSScriptRoot
$frontend = Join-Path (Split-Path -Parent $backend) 'sargas-sentinel-web'
$dsn = $env:SARGAS_TEST_DATABASE_URL

if ([string]::IsNullOrWhiteSpace($dsn)) {
    throw 'Falta SARGAS_TEST_DATABASE_URL. Las pruebas integradas NO pueden omitirse en un release.'
}
try { $uri = [uri]$dsn } catch { throw 'SARGAS_TEST_DATABASE_URL no es una URI valida.' }
if ($uri.Scheme -notin @('postgres', 'postgresql') -or
    $uri.AbsolutePath.Trim('/') -ne 'sargas_sentinel_test') {
    throw 'Por seguridad, la URL debe señalar exactamente a la base sargas_sentinel_test.'
}
if (-not (Test-Path -LiteralPath (Join-Path $frontend 'package.json'))) {
    throw 'No se encontro el frontend hermano sargas-sentinel-web.'
}

Push-Location $frontend
try {
    & npm.cmd ci
    if ($LASTEXITCODE -ne 0) { throw 'Fallo npm ci.' }
    & npm.cmd audit --audit-level=moderate
    if ($LASTEXITCODE -ne 0) { throw 'npm audit encontro vulnerabilidades por revisar.' }
    & npm.cmd run test:api
    if ($LASTEXITCODE -ne 0) { throw 'Fallaron pruebas de API frontend.' }
    & npm.cmd run check
    if ($LASTEXITCODE -ne 0) { throw 'Fallo Svelte Check o TypeScript.' }
    & npm.cmd run build
    if ($LASTEXITCODE -ne 0) { throw 'Fallo Vite build.' }
}
finally { Pop-Location }

# go:embed necesita los archivos compilados antes de go test ./...
$origen = Join-Path $frontend 'dist'
$destino = Join-Path $backend 'cmd\sentinel\web\dist'
if (-not (Test-Path -LiteralPath (Join-Path $origen 'index.html'))) {
    throw 'No se encontro el index.html compilado.'
}
if (Test-Path -LiteralPath $destino) { Remove-Item -LiteralPath $destino -Recurse -Force }
New-Item -ItemType Directory -Force -Path $destino | Out-Null
Copy-Item -Path (Join-Path $origen '*') -Destination $destino -Recurse -Force

Push-Location $backend
try {
    & go mod verify
    if ($LASTEXITCODE -ne 0) { throw 'Fallo go mod verify.' }
    & go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Fallo go vet.' }

    $registro = Join-Path $backend 'release-go-tests.log'
    $argsGo = @('test', '-v', '-count=1', '-coverprofile=coverage-full.out')
    if (-not $SinRace) { $argsGo += '-race' }
    $argsGo += './...'
    & go @argsGo 2>&1 | Tee-Object -FilePath $registro
    if ($LASTEXITCODE -ne 0) { throw 'Fallaron las pruebas de Go.' }

    $omitidas = @(Select-String -LiteralPath $registro -SimpleMatch -Pattern '--- SKIP:')
    if ($omitidas.Count -gt 0) {
        throw "Hubo $($omitidas.Count) pruebas omitidas; no se certifica la version."
    }

    $vuln = Get-Command 'govulncheck' -ErrorAction SilentlyContinue
    if (-not $vuln) {
        throw 'Instala govulncheck con: go install golang.org/x/vuln/cmd/govulncheck@latest'
    }
    & govulncheck ./...
    if ($LASTEXITCODE -ne 0) { throw 'govulncheck detecto avisos que requieren revision.' }
}
finally { Pop-Location }

if ($SinRace) {
    Write-Warning 'Verificacion parcial: no se ejecuto el detector de carreras.'
} else {
    Write-Host 'Validacion automatizada de release COMPLETA.' -ForegroundColor Green
}

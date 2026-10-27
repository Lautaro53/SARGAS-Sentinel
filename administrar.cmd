@echo off
setlocal
rem Uso: administrar.cmd [estado^|logs^|detener] desde la raiz SARGAS.
set "accion=%~1"
if "%accion%"=="" set "accion=estado"
if /I not "%accion%"=="estado" if /I not "%accion%"=="logs" if /I not "%accion%"=="detener" (
    echo Uso: administrar.cmd [estado^|logs^|detener]
    exit /b 2
)

powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "%~dp0SARGAS-sentinel\scripts\administrar-docker.ps1" -Accion "%accion%"
exit /b %ERRORLEVEL%

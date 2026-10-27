@echo off
setlocal
rem SARGAS Sentinel - instalacion Windows: ejecutar desde cualquier carpeta.
rem No cambia permanentemente la politica de ejecucion ni requiere PowerShell administrador.

if not exist "%~dp0SARGAS-sentinel\scripts\instalar-sentinel.ps1" (
    echo ERROR: No se encuentra SARGAS-sentinel\scripts\instalar-sentinel.ps1
    echo Conserva juntas las carpetas SARGAS-sentinel y sargas-sentinel-web.
    exit /b 1
)

powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "%~dp0SARGAS-sentinel\scripts\instalar-sentinel.ps1"
set "codigo=%ERRORLEVEL%"
if not "%codigo%"=="0" (
    echo.
    echo ERROR: La instalacion de SARGAS fallo ^(codigo %codigo%^).
    echo Revisa los mensajes anteriores. No borres los volumenes de Docker.
)
exit /b %codigo%

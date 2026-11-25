@echo off
setlocal
powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "%~dp0SARGAS-sentinel\scripts\actualizar-docker.ps1" %*
exit /b %ERRORLEVEL%

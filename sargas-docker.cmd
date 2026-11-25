@echo off
setlocal
powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "%~dp0SARGAS-sentinel\scripts\administrar-docker.ps1" %*
exit /b %ERRORLEVEL%

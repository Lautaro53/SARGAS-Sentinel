# Backend de SARGAS Sentinel

Implementación en Go de la API, motor de monitoreo HTTP y persistencia PostgreSQL.

Consultar el [README principal](../README.md) para instalación y arquitectura, y la [guía de desarrollo](../docs/desarrollo.md) para compilación y pruebas.

La compilación Docker necesita también el proyecto hermano `../sargas-sentinel-web`.

Para instalar desde un ZIP en Windows, con Docker Desktop iniciado, ejecutar desde la **raíz del repositorio**:

```powershell
.\sargas-instalar.cmd instalar
```

No se necesita Node ni Go locales, ni iniciar PowerShell como administrador. El `.cmd` invoca de forma aislada el instalador `scripts/instalar-sentinel.ps1`.

No exponer la API de administración directamente a Internet; SARGAS v1.x está diseñado para uso local.

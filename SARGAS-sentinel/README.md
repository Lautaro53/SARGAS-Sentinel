# Backend de SARGAS Sentinel

Implementación en Go de la API, motor de monitoreo HTTP y persistencia PostgreSQL.

Para la instalación, la descripción de la arquitectura, API, seguridad y pruebas de la versión 1, consultá el [README principal](../README.md).

La compilación Docker necesita también el proyecto hermano `../sargas-sentinel-web`.

Para instalar desde un ZIP en Windows, con Docker Desktop iniciado, ejecutar desde la **raiz del repositorio**:

```powershell
.\instalar.cmd
```

No se necesita Node ni Go locales, ni iniciar PowerShell como administrador. El `.cmd` invoca de forma aislada el instalador `scripts/instalar-sentinel.ps1`.

No exponer la API de administración directamente a Internet; el alcance de v1.0 es local.

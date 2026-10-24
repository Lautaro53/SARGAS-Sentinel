# Frontend de SARGAS Sentinel

Panel web desarrollado con Svelte 5, TypeScript y Vite. Muestra el estado de los servicios, sus comprobaciones e incidentes mediante la API del backend.

## Desarrollo

```powershell
npm ci
npm run dev
```

## Verificación

```powershell
npm run test:api
npm run check
npm run build
```

La distribución principal se construye con Docker Compose desde `../SARGAS-sentinel`, que incorpora el `dist` generado dentro del ejecutable Go. Para instalación y arquitectura, consultá el [README principal](../README.md).

# Desarrollo local

El desarrollo nativo utiliza PostgreSQL de Windows, separado de Docker. Ejecutar los comandos desde la raíz del repositorio. Requiere Go, Node.js/npm, el cliente PostgreSQL (`psql`), las bases locales `sargas_sentinel` y `sargas_sentinel_test` con sus roles y permisos ya preparados. Para validar un release también se requiere GCC compatible con Go race y `govulncheck`.

| Operación local | Comando desde la raíz |
|---|---|
| Ver acciones disponibles | `.\sargas-dev.ps1 ayuda` |
| Verificar y preparar configuración local | `.\sargas-dev.ps1 configurar` |
| Diagnóstico de herramientas, conexiones y puertos | `.\sargas-dev.ps1 diagnostico` |
| Iniciar Go y Vite en pestañas de Windows Terminal | `.\sargas-dev.ps1 dev` |
| Ejecutar el backend desde código fuente | `.\sargas-dev.ps1 backend` |
| Ejecutar Vite | `.\sargas-dev.ps1 frontend` |
| Compilar frontend y backend | `.\sargas-dev.ps1 compilar` |
| Instalar/actualizar el ejecutable compilado | `.\sargas-dev.ps1 actualizar-local` |
| Compilar e instalar, con respaldo si ya existe | `.\sargas-dev.ps1 actualizar-local -Compilar` |
| Iniciar el ejecutable instalado | `.\sargas-dev.ps1 iniciar` |
| Validar una candidata a release | `.\sargas-dev.ps1 validar` |

El modo desarrollo usa Go en `127.0.0.1:8080` y Vite en `127.0.0.1:5173`. Detener la instancia Docker si ocupa el puerto 8080. Si Windows Terminal no está disponible, se abren dos ventanas independientes. Para detener el modo desarrollo, usar Ctrl+C en cada pestaña o ventana. Vite actualiza el frontend; los cambios Go requieren reiniciar el backend. Si falta el frontend integrado requerido por `go:embed`, se genera una compilación inicial automáticamente.

Las credenciales nativas de `sentinel_app` y `sentinel_test` se verifican y guardan por separado mediante DPAPI en `%USERPROFILE%\SARGAS-privado`. `-ActualizarClave` vuelve a solicitar la contraseña actual de PostgreSQL (no cambia la contraseña en el servidor). Puede utilizarse con `configurar`, `backend`, `dev`, `iniciar` y `validar`. El puerto PostgreSQL local puede ajustarse mediante `SARGAS_DB_PORT` (por defecto 5432). Los comandos centralizados construyen temporalmente sus propias URL locales y restauran las variables anteriores al terminar; no utilizan los secretos Docker. `configurar` no crea bases ni roles.

## Candidatos Docker

```powershell
.\sargas-actualizar.cmd probar-local -SoloPreparar
```

Construye una imagen del código actual sin desplegarla. `probar-local -Confirmar` despliega el candidato sobre una instalación compatible; afecta Docker, no PostgreSQL nativo. El origen de la imagen se identifica como local. No utilizar `-Confirmar` sobre una instalación que deba permanecer estable sin autorización.

## Publicación

Ejecutar `sargas-dev.ps1 validar` (sin `-SinRace`) y comprobar la integración Docker antes de etiquetar. Generar el paquete a partir de la etiqueta definitiva:

```powershell
git archive --format=zip --output=SARGAS-Sentinel-v1.2.0.zip v1.2.0
```

Adjuntar el ZIP al release de GitHub y comprobar que la API informa su SHA-256. No incluir secretos, datos, dependencias locales ni parches de trabajo. Mantener `VERSION` y `ACTUALIZACION.json` coherentes; los cambios de contrato Docker o esquema SQL requieren un procedimiento de migración específico.

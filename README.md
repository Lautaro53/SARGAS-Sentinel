# SARGAS Sentinel

**Plataforma autohospedada de monitoreo de servicios HTTP/HTTPS.**

SARGAS Sentinel supervisa la disponibilidad de aplicaciones y APIs, registra tiempos de respuesta y detecta interrupciones. Su interfaz web centraliza la administración de servicios, las métricas y el historial de incidentes.

Está orientado a desarrolladores y pequeños entornos de infraestructura.

**Versión:** `v1.2.0` · **Licencia:** MIT

---

## Instalación

### Requisitos

- Windows 10 u 11.
- Docker Desktop en ejecución, con contenedores Linux, Docker Compose v2.17 o superior y Buildx.
- Conexión a Internet para descargar imágenes y dependencias.

No es necesario instalar Go, Node.js ni PostgreSQL por separado.

### Instalación rápida

1. Descargar `SARGAS-Sentinel-v1.2.0.zip` desde [GitHub Releases](https://github.com/Lautaro53/SARGAS-Sentinel/releases) y extraerlo completo.
2. Abrir PowerShell en la carpeta extraída y ejecutar:

```powershell
.\sargas-instalar.cmd instalar
```

3. Acceder a la dirección indicada al finalizar la instalación (por defecto, [http://127.0.0.1:8080/](http://127.0.0.1:8080/)). Si el puerto está ocupado, se selecciona otro disponible entre 8080 y 8090.

El instalador está destinado a instalaciones nuevas. Para una instalación existente, utilizar los comandos de administración o actualización.

### Administración

Con Docker Desktop iniciado, ejecutar desde la raíz del paquete:

| Acción | Comando |
|---|---|
| Iniciar | `.\sargas-docker.cmd iniciar` |
| Detener | `.\sargas-docker.cmd detener` |
| Reiniciar | `.\sargas-docker.cmd reiniciar` |
| Consultar estado | `.\sargas-docker.cmd estado` |
| Consultar registros de Sentinel | `.\sargas-docker.cmd logs` |
| Consultar versión instalada | `.\sargas-docker.cmd version` |

Todos los scripts muestran la ayuda al ejecutarlos sin argumentos o con `ayuda`. Los datos y el puerto de la instalación se conservan al iniciar, detener o reiniciar.

### Actualizaciones

```powershell
.\sargas-actualizar.cmd buscar
.\sargas-actualizar.cmd actualizar -Confirmar
```

El actualizador instala releases estables compatibles, verifica el paquete descargado y respalda PostgreSQL antes de sustituir Sentinel. Conserva el volumen, las credenciales y el puerto existentes.

Para instalaciones anteriores a v1.2.0, descargar primero el nuevo paquete y ejecutar sus comandos, sin utilizar el instalador. Si la versión instalada es desconocida y se ha comprobado que corresponde a una instalación anterior compatible de SARGAS, la adopción de v1.2.0 requiere:

```powershell
.\sargas-actualizar.cmd actualizar -Confirmar -AceptarVersionDesconocida
```

Después de actualizar, utilizar los scripts de la carpeta indicada por el actualizador. Conservar la carpeta de instalación original, los respaldos y las credenciales privadas. No ejecutar simultáneamente operaciones de instalación, administración o actualización.

Consultar [Respaldo y recuperación](docs/recuperacion.md) para conocer el procedimiento y sus limitaciones.

> **Seguridad:** SARGAS está diseñado para uso local. No incluye autenticación de usuarios y no debe exponerse directamente a Internet.

---

## Funcionalidades

### Monitoreo HTTP/HTTPS

- Comprobaciones automáticas y concurrentes.
- Intervalos y tiempos de espera configurables.
- Registro de códigos HTTP, latencia y errores.
- Ejecución independiente de cada servicio monitoreado.

### Gestión de servicios

- Registro y edición de servicios.
- Habilitación y pausa del monitoreo.
- Archivado sin eliminación del historial.
- Consulta de configuración, disponibilidad y resultados.

### Estados de disponibilidad

SARGAS utiliza tres estados:

| Estado | Descripción |
|---|---|
| `UP` | Servicio disponible |
| `DOWN` | Servicio no disponible |
| `UNKNOWN` | Información insuficiente |

Los cambios a `UP` o `DOWN` requieren dos comprobaciones consecutivas con el mismo resultado, evitando transiciones por fallos aislados.

### Gestión de incidentes

Los incidentes se registran automáticamente cuando se confirma una interrupción y se cierran cuando el servicio recupera su disponibilidad.

Cada incidente conserva las fechas, el servicio afectado y las evidencias de la interrupción y recuperación.

### Interfaz web

La interfaz permite consultar:

- Dashboard general de monitoreo.
- Listado y administración de servicios.
- Estados de disponibilidad.
- Gráficos interactivos de latencia con actualización automática.
- Historial de comprobaciones.
- Incidentes activos y finalizados.
- Temas claro y oscuro, búsqueda, filtros y tablas paginadas.

Si la API deja de responder, las vistas pueden mostrar los datos disponibles en la caché de sesión, indicando que están desactualizados y la última sincronización. Las operaciones administrativas requieren conexión.

### Persistencia

La información se almacena en PostgreSQL. Después de reiniciar la aplicación, SARGAS recupera las configuraciones y reanuda el monitoreo de los servicios habilitados.

---

## Arquitectura e infraestructura

| Componente | Tecnología |
|---|---|
| Backend, API REST y monitor concurrente | Go |
| Interfaz web | Svelte 5, TypeScript y Vite |
| Base de datos | PostgreSQL 18 |
| Despliegue | Docker Compose |

Docker ejecuta dos servicios: `sentinel` (backend e interfaz web integrada) y `postgres` (persistencia). La API y la web se sirven desde un único puerto local. PostgreSQL se comunica por la red interna de Docker, sin publicar su puerto en el anfitrión.

La API REST está disponible bajo `/api`. Los servicios, comprobaciones, estados e incidentes se almacenan en PostgreSQL. Las consultas históricas admiten paginación.

## Desarrollo

El desarrollo nativo utiliza herramientas y bases de datos independientes de Docker. Desde la raíz del repositorio:

```powershell
.\sargas-dev.ps1 ayuda
.\sargas-dev.ps1 configurar
.\sargas-dev.ps1 dev
```

Consultar la [guía de desarrollo](docs/desarrollo.md) para requisitos, comandos de compilación y validación.

---

## Versiones y evolución

SARGAS Sentinel sigue un desarrollo incremental, incorporando nuevas capacidades sin perder la estabilidad de las funcionalidades existentes.

| Versión | Enfoque | Objetivos |
|---|---|---|
| **v1.0.0** | Primera versión pública | Monitoreo HTTP/HTTPS, API REST, PostgreSQL, incidentes e instalación Docker |
| **v1.1.5** | Interfaz y consolidación | Renovación visual, mejoras de administración y gestión de instalación |
| **v1.2.0** | Interfaz, datos y mantenimiento | Renovación visual, caché de sesión, gráficos dinámicos y actualizaciones Docker |
| **v1.3** | Administración y actividad | Configuración ampliada y registro de eventos |
| **v1.4** | Notificaciones | Webhooks y correo electrónico |
| **v1.5** | Ampliación del monitoreo | Comprobaciones TCP, exportación y consolidación |
| **v2.0** | Monitoreo de infraestructura | Agentes Go, supervisión de recursos y contenedores |

Las versiones futuras representan objetivos de desarrollo y pueden modificarse según las necesidades del proyecto.

---

## Licencia

SARGAS Sentinel se distribuye bajo la **licencia MIT**.

Consultar el archivo [LICENSE](SARGAS-sentinel/LICENSE) para conocer sus términos.

Repositorio: [github.com/Lautaro53/SARGAS-Sentinel](https://github.com/Lautaro53/SARGAS-Sentinel)
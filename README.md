# SARGAS Sentinel

**Plataforma autohospedada de monitoreo de servicios HTTP/HTTPS.**

SARGAS Sentinel es una aplicación de monitoreo que permite supervisar la disponibilidad de servicios web, detectar interrupciones y consultar su comportamiento desde una interfaz centralizada.

El sistema ejecuta comprobaciones periódicas, registra tiempos de respuesta, identifica cambios de disponibilidad y conserva un historial de incidentes.

Desarrollado principalmente en **Go**, utiliza una arquitectura modular que separa el motor de monitoreo, la API REST, la persistencia y la interfaz web.

SARGAS busca ofrecer una solución ligera, confiable y extensible, orientada a desarrolladores y pequeños entornos de infraestructura.

**Versión:** `v1.1.5` · **Licencia:** MIT

---

## Instalación

SARGAS utiliza Docker Compose para simplificar la instalación y ejecución de sus componentes.

### Requisitos

- Windows 10 u 11.
- Docker Desktop en ejecución, con contenedores Linux.
- Docker Compose v2.17 o superior y Docker Buildx.
- Conexión a Internet durante la primera instalación.

No es necesario instalar Go, Node.js ni PostgreSQL por separado.

### Instalación rápida

**1. Descargar el proyecto**

Clonar el repositorio:

```powershell
git clone https://github.com/Lautaro53/SARGAS-Sentinel.git
cd SARGAS-Sentinel
```

También es posible descargar el proyecto desde GitHub mediante **Code → Download ZIP** y extraerlo.

**2. Ejecutar el instalador**

Con Docker Desktop iniciado, abrir PowerShell desde la raíz del proyecto y ejecutar:

```powershell
.\instalar.cmd
```

El instalador verifica las dependencias, prepara las credenciales, construye la aplicación e inicia los servicios.

**3. Acceder a SARGAS**

Al finalizar, se mostrará la dirección de acceso:

```text
SARGAS Sentinel instalado y disponible.
Abrir: http://127.0.0.1:8080/
```

El puerto predeterminado es `8080`. Si está ocupado durante una instalación nueva, SARGAS busca automáticamente otro disponible y muestra la dirección correspondiente. Las instalaciones existentes conservan su configuración de puerto cuando es posible.

### Administración

Los siguientes comandos se ejecutan desde la carpeta principal del proyecto:

| Acción | Comando |
|---|---|
| Instalar o reconstruir | `.\instalar.cmd` |
| Consultar estado | `.\administrar.cmd estado` |
| Consultar registros | `.\administrar.cmd logs` |
| Detener servicios | `.\administrar.cmd detener` |

Los datos de PostgreSQL se conservan en un volumen persistente. Las credenciales se almacenan fuera del repositorio.

> **Seguridad:** SARGAS está diseñado para uso local o en entornos de confianza. La versión actual no incluye autenticación de usuarios y no debe exponerse directamente a Internet.

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
- Gráficos de latencia.
- Historial de comprobaciones.
- Incidentes activos y finalizados.

### Persistencia

La información se almacena en PostgreSQL. Después de reiniciar la aplicación, SARGAS recupera las configuraciones y reanuda el monitoreo de los servicios habilitados.

---

## Arquitectura e infraestructura

SARGAS utiliza una arquitectura modular que separa las responsabilidades de presentación, monitoreo, comunicación y persistencia.

### Tecnologías

| Componente | Tecnología |
|---|---|
| Backend y motor de monitoreo | Go 1.27.3 |
| Frontend | Svelte 5 + TypeScript |
| Compilación frontend | Vite |
| Base de datos | PostgreSQL 18 |
| Contenedorización | Docker |
| Despliegue local | Docker Compose |
| Instalación Windows | PowerShell |

### Funcionamiento

El frontend se comunica con el backend mediante una API REST.

El backend administra los servicios y coordina el motor de monitoreo, que utiliza goroutines para realizar comprobaciones concurrentes.

Los resultados se almacenan en PostgreSQL y permiten evaluar cambios de disponibilidad, registrar incidentes y consultar información histórica.

### Infraestructura Docker

La distribución utiliza dos servicios:

| Servicio | Responsabilidad |
|---|---|
| `sentinel` | Backend Go, API REST e interfaz web |
| `postgres` | Persistencia de configuraciones y resultados |

Durante la compilación, Svelte genera los archivos estáticos de la interfaz, que se incorporan al ejecutable Go mediante `go:embed`.

Esto permite servir la interfaz y la API desde un único proceso, sin necesitar un servidor Node.js independiente en ejecución.

PostgreSQL se comunica con Sentinel mediante la red interna de Docker. Su puerto no se publica en el equipo anfitrión.

La aplicación se publica únicamente sobre la interfaz local `127.0.0.1`, utilizando el puerto configurado durante la instalación.

### Base de datos

El esquema principal se organiza en cuatro tablas:

| Tabla | Responsabilidad |
|---|---|
| `servicios` | Configuración y administración |
| `comprobaciones` | Resultados HTTP, errores y latencias |
| `estados_servicio` | Disponibilidad y contadores |
| `incidentes` | Interrupciones y recuperaciones |

Las operaciones de evaluación utilizan transacciones para mantener la consistencia de la información.

### API REST

La API se encuentra disponible bajo `/api` y permite administrar servicios y consultar estados, comprobaciones e incidentes.

Las consultas históricas admiten paginación mediante `limite` y `offset`.

---

## Versiones y evolución

SARGAS Sentinel sigue un desarrollo incremental, incorporando nuevas capacidades sin perder la estabilidad de las funcionalidades existentes.

| Versión | Enfoque | Objetivos |
|---|---|---|
| **v1.0.0** | Primera versión pública | Monitoreo HTTP/HTTPS, API REST, PostgreSQL, incidentes e instalación Docker |
| **v1.1.5** | Interfaz y consolidación | Renovación visual, mejoras de administración y gestión de instalación |
| **v1.2** | Datos y visualización | Caché, actualización de información y gráficos |
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
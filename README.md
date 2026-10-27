# SARGAS Sentinel

**Monitoreo HTTP autohospedado, con historial e incidentes.**

SARGAS Sentinel es una aplicación para supervisar servicios HTTP/HTTPS desde un panel web local. Ejecuta comprobaciones periódicas, registra las respuestas y la latencia, identifica cambios de disponibilidad y conserva el historial en PostgreSQL.

Está construida con **Go**, **Svelte 5 + TypeScript** y **PostgreSQL 18**. La distribución principal utiliza **Docker Compose**: el frontend se compila e incorpora al ejecutable Go, y la aplicación se sirve desde un único puerto HTTP.

> **Alcance:** esta versión está diseñada para uso local. No incluye autenticación de usuarios ni está preparada para exponerse directamente a Internet.

## Funcionalidades

- **Servicios:** crear, editar, habilitar, pausar y archivar servicios sin eliminar su historial.
- **Monitoreo HTTP/HTTPS:** comprobaciones concurrentes con intervalo y timeout configurables, código de respuesta y latencia.
- **Estados:** `UNKNOWN`, `UP` y `DOWN`, con confirmación mediante dos éxitos o dos fallos consecutivos.
- **Incidentes:** apertura y cierre automáticos, evidencias y tiempos de inicio y recuperación.
- **Historial:** registros persistentes de comprobaciones, estados e incidentes; consultas paginadas por API.
- **Interfaz:** Dashboard, listado de servicios, detalle con gráfico de latencia e historial global de incidentes.
- **Recuperación:** al reiniciar Sentinel se cargan desde PostgreSQL los servicios habilitados y se reanuda el monitoreo.

## Arquitectura

```text
Navegador  ── HTTP :8080 ──> Go (API REST + archivos Svelte integrados)
                                │
                                ├── Administracion de servicios
                                ├── Motor concurrente de comprobaciones HTTP
                                ├── Evaluacion de estados e incidentes
                                └── PostgreSQL 18 (volumen persistente)
```

El backend, la persistencia y la interfaz tienen responsabilidades separadas. Durante la compilación Docker, Svelte genera archivos estáticos que se integran en Go mediante `go:embed`.

## Estructura del repositorio

Los dos proyectos son carpetas hermanas y **deben mantenerse en esta disposición**:

```text
SARGAS/
├── README.md
├── .gitignore
├── SARGAS-sentinel/
│   ├── cmd/sentinel/         # Punto de entrada, servidor HTTP y web integrada
│   ├── internal/             # API, motor, repositorios, PostgreSQL
│   ├── docker/postgres/init/ # Inicializacion de base, roles y permisos
│   ├── scripts/             # Instalacion y utilidades Windows
│   ├── sql/                 # Esquema y permisos de referencia
│   ├── Dockerfile
│   └── compose.yaml
└── sargas-sentinel-web/
    ├── src/                 # Vistas, componentes, API y tipos
    ├── public/              # Recursos visuales usados por la interfaz
    ├── tests/               # Pruebas de contratos de API
    ├── package.json
    └── package-lock.json
```

El directorio personal `Extras/`, si existe, **no se publica**: se excluye mediante `.gitignore`.

## Instalacion limpia: Windows + Docker (un comando)

**Requisitos:** Windows 10/11, Docker Desktop iniciado en modo **contenedores Linux**, Docker Compose **v2.17 o posterior**, Buildx y conexion a Internet la primera vez. **No hay que instalar Node.js, npm, Go o PostgreSQL en Windows.** Docker los utiliza durante la compilacion. <https://docs.docker.com/reference/compose-file/build/#additional_contexts>

1. Descargar el ZIP del repositorio o clonar el proyecto. Extraerlo por completo; mantener `SARGAS-sentinel/` y `sargas-sentinel-web/` como carpetas hermanas.
2. Abrir Docker Desktop y esperar a que el motor este funcionando.
3. Abrir PowerShell **normal (sin Administrador)** en la carpeta `SARGAS/` y ejecutar:

   ```powershell
   .\instalar.cmd
   ```

4. Al ver "SARGAS Sentinel instalado y disponible", abrir **http://127.0.0.1:8080/**.

El `.cmd` invoca el instalador PowerShell con una politica de ejecucion valida solo para **ese proceso**; no modifica configuraciones permanentes de Windows. En equipos administrados por una organizacion, una politica de grupo puede bloquear incluso esa invocacion; debe respetarse esa restriccion, no cambiar politicas de forma global. No ejecutes scripts de origen desconocido.

**No ejecutar como Administrador por defecto.** Si Docker Desktop no permite acceso al motor, resolver primero los permisos y el grupo `docker-users` para la cuenta habitual (puede requerir asistencia de un administrador y un nuevo inicio de sesion). Ejecutar el instalador con otra cuenta podria crear secretos en otro perfil y complicar reinstalaciones.

El instalador valida Docker, conserva las credenciales anteriores o genera nuevas **solo en una instalacion sin base existente**, construye Node/Svelte y Go dentro de Docker, inicia PostgreSQL y comprueba HTTP 200 tanto en la API como en el inicio web. No modifica la politica de PowerShell, ni borra volumenes, ni solicita npm local.

**IMPORTANTE:** la primera compilacion necesita Internet para descargar las imagenes Docker y dependencias npm/Go. `npm ci` ocurre *dentro del contenedor de compilacion*, no en la maquina del usuario. Si falla por una descarga, comprobar conexion, proxy y certificados de Docker antes de modificar archivos.

### Comandos de operacion

Ejecutar desde la carpeta `SARGAS/`:

| Operacion | Comando |
|---|---|
| Instalar, iniciar o reconstruir | `.\instalar.cmd` |
| Estado de contenedores | `.\administrar.cmd estado` |
| Ultimos logs de la aplicacion | `.\administrar.cmd logs` |
| Detener los contenedores | `.\administrar.cmd detener` |
| Comprobar la API | `curl.exe -i http://127.0.0.1:8080/api/servicios` |

Las utilidades `.cmd` tambien evitan ejecutar directamente archivos `.ps1` en una sesion con restricciones habituales. Los scripts nativos de desarrollo siguen en `SARGAS-sentinel/scripts/` y **no forman parte de la instalacion de usuario**.

### Datos y reinstalaciones

Las contrasenas se guardan **fuera del repositorio** en `%USERPROFILE%\SARGAS-privado\docker-secrets`; PostgreSQL mantiene sus datos en el volumen Docker `sargas-db-lab_postgres_data`. Antes de cambiar de equipo o reinstalar Docker, guardar una copia segura de **ambos secretos y los respaldos de la base de datos**. Si existe el volumen pero faltan las contrasenas, el instalador **se detiene**: no las reemplaza.

Nunca uses `docker compose down -v` para una instalacion con datos que quieras conservar. El servicio de administracion HTTP escucha unicamente en `127.0.0.1:8080` y **no incluye autenticacion**; no publicarlo en Internet.

### Solucion de problemas

- **`docker` no reconocido:** instalar Docker Desktop y reabrir la terminal.
- **No puede conectarse al motor:** iniciar Docker Desktop; seleccionar contenedores Linux; verificar permisos Docker de tu usuario.
- **Compose antiguo o sin Buildx:** actualizar Docker Desktop. La compilacion requiere contextos adicionales para frontend y backend.
- **Error de `npm ci` / `go mod download`:** comprobar Internet, proxy, DNS y acceso a registros desde Docker; no instalar npm ni Go en Windows.
- **Puerto 8080 ocupado:** cerrar la aplicacion que lo utiliza; este instalador no cambia la exposicion local.
- **Faltan secretos y ya existe una base:** recuperar los secretos originales y no eliminar el volumen.

## API REST

La API está disponible bajo el prefijo `/api`. En la v1.0 se implementaron las rutas siguientes:

| Método | Ruta | Función |
|---|---|---|
| `GET` | `/api/servicios` | Listar servicios activos/no archivados |
| `POST` | `/api/servicios` | Registrar un servicio HTTP |
| `PUT` | `/api/servicios/{id}` | Actualizar configuración |
| `PATCH` | `/api/servicios/{id}/habilitacion` | Pausar o reanudar |
| `PATCH` | `/api/servicios/{id}/archivar` | Archivar conservando historial |
| `GET` | `/api/servicios/{id}/estado` | Estado y contadores |
| `GET` | `/api/servicios/{id}/comprobaciones` | Historial de comprobaciones |
| `GET` | `/api/servicios/{id}/incidentes` | Incidentes de un servicio |
| `GET` | `/api/incidentes` | Historial global de incidentes |

Las consultas históricas admiten `limite` (de 1 a 100; predeterminado 50) y `offset` (entero no negativo). La API valida los datos de entrada y restringe las operaciones administrativas a los orígenes previstos cuando se proporciona el encabezado `Origin`.

## Desarrollo y pruebas

**Requisitos adicionales para desarrollo nativo:** Go 1.27.3, Node.js compatible con Vite 8, npm y una instancia PostgreSQL para pruebas de integración.

Frontend:

```powershell
cd .\sargas-sentinel-web
npm.cmd ci
npm.cmd run test:api
npm.cmd run check
npm.cmd run build
```

Backend: el paquete `cmd/sentinel` utiliza `go:embed`, por lo que **antes de ejecutar `go test ./...` desde un clon limpio** debe existir `SARGAS-sentinel/cmd/sentinel/web/dist`. La compilación Docker lo resuelve automáticamente. Para desarrollo nativo en Windows puede prepararse con:

```powershell
cd ..\SARGAS-sentinel
.\scripts\compilar-sentinel.ps1
# Con el frontend integrado:
go test ./...
go vet ./...
go test -race ./...
# Para la validacion final, usar scripts/validar-release.ps1 con una base de pruebas.
```

Las pruebas de integración que consultan PostgreSQL requieren `SARGAS_TEST_DATABASE_URL` apuntando **exclusivamente** a la base de pruebas `sargas_sentinel_test`. Si no se configura, esas pruebas se omiten; no deben ejecutarse contra la base principal. El script `compilar-sentinel.ps1` genera un ejecutable nativo de Windows, pero no prepara por sí solo la base de integración.

## Datos, permisos y seguridad

PostgreSQL utiliza un volumen persistente de Docker. La inicialización del volumen ejecuta los scripts de `docker/postgres/init/` para crear el esquema y aplicar roles separados: `sentinel_owner` es propietario de los objetos y `sentinel_app` dispone de los permisos mínimos necesarios para operar (sin `DELETE` ni DDL).

Las contraseñas se generan fuera del repositorio, bajo `SARGAS-privado/docker-secrets` en el perfil de Windows. **Guardar una copia segura tanto de las credenciales como de los respaldos de la base.** Perder las credenciales no debe provocar la creación de unas nuevas sobre un volumen existente.

Por defecto la aplicación se publica solo en `127.0.0.1:8080`; PostgreSQL no publica un puerto en el host. No hay autenticación ni HTTPS de administración en la v1.0. No exponer esta instancia mediante proxies públicos, apertura de puertos o túneles sin implementar controles de seguridad adicionales.
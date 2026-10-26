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

## Inicio rápido: Windows y Docker

**Requisitos:** Docker Desktop con motor Linux iniciado, Docker Compose compatible con `additional_contexts` y PowerShell. No es necesario instalar Go, Node.js ni PostgreSQL para ejecutar la instalación Docker.

1. Descargar o clonar este repositorio conservando ambas carpetas.
2. Iniciar Docker Desktop.
3. Desde PowerShell, entrar al backend y ejecutar:

   ```powershell
   cd .\SARGAS-sentinel
   .\scripts\instalar-sentinel.ps1
   ```

4. Abrir **http://127.0.0.1:8080/**.

El instalador verifica Docker, crea credenciales privadas si es una instalación nueva, compila las imágenes, inicia la base de datos y la aplicación, y comprueba que `/api/servicios` devuelve HTTP 200. Una instalación sin servicios muestra un Dashboard vacío: es el resultado esperado.

Si Windows bloquea la ejecución de un script descargado, revisar su procedencia y, si es de confianza, usar `Unblock-File` para ese archivo. No es necesario cambiar la política de ejecución del sistema de forma permanente.

**Nota:** el instalador está diseñado para Windows. En otros sistemas operativos debe prepararse el directorio de secretos requerido por `compose.yaml` antes de utilizar Docker Compose; no se distribuye todavía un instalador multiplataforma.

## Operación diaria

Los comandos siguientes se ejecutan desde `SARGAS-sentinel/`. Si se opera con Compose directamente en una terminal nueva, primero hay que definir la variable `SARGAS_DOCKER_SECRETS` con la carpeta privada de secretos; el instalador la configura automáticamente para su propia ejecución.

| Operación | Comando |
|---|---|
| Comprobar estado | `.\scripts\administrar-docker.ps1 estado` |
| Ver registros de la aplicación | `.\scripts\administrar-docker.ps1 logs` |
| Detener los servicios | `.\scripts\administrar-docker.ps1 detener` |
| Iniciar o actualizar | `.\scripts\instalar-sentinel.ps1` |
| Comprobar la API | `curl.exe -i http://127.0.0.1:8080/api/servicios` |

Para las operaciones de `docker compose` directas puede definirse en PowerShell:

```powershell
$env:SARGAS_DOCKER_SECRETS = (
    Join-Path $env:USERPROFILE 'SARGAS-privado\docker-secrets'
).Replace('\', '/')
```

No usar `docker compose down -v` en una instalación que deba conservar sus datos: el parámetro `-v` elimina los volúmenes del proyecto.

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

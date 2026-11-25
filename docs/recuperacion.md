# Respaldo y recuperación

## Actualizaciones Docker

El actualizador conserva una etiqueta de la imagen anterior y genera un respaldo lógico de PostgreSQL antes de desplegar. Los archivos se guardan en `%USERPROFILE%\SARGAS-privado\actualizaciones\<identificador>`.

| Archivo | Contenido |
|---|---|
| `sargas_sentinel.dump` | Respaldo lógico de la base |
| `sargas_sentinel.dump.sha256` | Huella de integridad |
| `contenido-respaldo.txt` | Índice del respaldo |
| `recuperacion.json` | Referencias a la imagen anterior y configuración de recuperación |
| `imagen-anterior.yaml`, `imagen-nueva.yaml`, `compose.env` | Configuración utilizada por Docker Compose |

Si falla el despliegue, el actualizador intenta recuperar la imagen y el estado de ejecución previos. No restaura SQL automáticamente. Si la recuperación falla, conservar los archivos de diagnóstico, el volumen, las imágenes y las carpetas de instalación. No volver a ejecutar el instalador ni borrar el volumen para resolver el problema.

El respaldo lógico no incluye las contraseñas ni los roles globales. Conservar por separado los archivos de secretos de `%USERPROFILE%\SARGAS-privado\docker-secrets` y proteger los respaldos del acceso no autorizado.

## Restauración de un respaldo

Verificar primero la huella del archivo (reemplazar la ruta por la carpeta de respaldo correspondiente):

```powershell
$ruta = 'C:\ruta\al\respaldo'
$hash = (Get-FileHash "$ruta\sargas_sentinel.dump" -Algorithm SHA256).Hash
$hash -eq (Get-Content "$ruta\sargas_sentinel.dump.sha256" -Raw).Trim()
```

El resultado debe ser `True`. Ensayar la restauración en PostgreSQL 18 independiente, con un volumen nuevo, sin puertos publicados y sin acceso al volumen principal.

1. Crear una base vacía y los roles mediante `SARGAS-sentinel/docker/postgres/init/01-roles.sql`. No ejecutar el script de creación de tablas antes de `pg_restore`.
2. Copiar el dump a la instancia aislada y restaurarlo con `pg_restore -U postgres -d <base_destino> --exit-on-error --single-transaction <archivo_dump>`.
3. Restablecer el propietario y los permisos de la base, utilizando el bloque de base de datos de `03-permisos.sql` y el nombre real de destino. La restauración sobre una base existente no recupera automáticamente estos atributos.
4. Verificar recuentos, restricciones, propietarios y permisos de tablas, esquema y secuencias. Configurar las credenciales del rol de aplicación mediante los secretos privados correspondientes; `04-credenciales.sh` documenta el mecanismo de inicialización Docker.
5. Comprobar la autenticación y el funcionamiento de Sentinel antes de poner la base recuperada en servicio.

No realizar una restauración sobre una base activa ni sustituir el volumen principal sin un procedimiento de recuperación específico y un respaldo previo.

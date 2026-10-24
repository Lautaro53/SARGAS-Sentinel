
#!/bin/sh
set -eu

APP_PASSWORD_FILE="/run/secrets/sentinel_app_password"

if [ ! -s "$APP_PASSWORD_FILE" ]; then
    echo "ERROR: Falta el secreto de sentinel_app." >&2
    exit 1
fi

APP_PASSWORD="$(cat "$APP_PASSWORD_FILE")"

psql \
    --username "$POSTGRES_USER" \
    --dbname "$POSTGRES_DB" \
    -v ON_ERROR_STOP=1 \
    -v app_password="$APP_PASSWORD" <<'SQL'
ALTER ROLE sentinel_app PASSWORD :'app_password';
SQL

unset APP_PASSWORD

echo "Credenciales de sentinel_app configuradas."

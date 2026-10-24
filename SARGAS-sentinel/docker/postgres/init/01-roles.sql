
-- SARGAS Sentinel v1.0
-- Inicializacion de roles para Docker.
-- Se ejecuta una sola vez durante la creacion del volumen.

CREATE ROLE sentinel_owner
    WITH NOLOGIN
    NOSUPERUSER
    NOCREATEROLE
    NOCREATEDB
    NOREPLICATION
    NOBYPASSRLS;

CREATE ROLE sentinel_app
    WITH LOGIN
    NOSUPERUSER
    NOCREATEROLE
    NOCREATEDB
    NOREPLICATION
    NOBYPASSRLS;

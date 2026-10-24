-- SARGAS Sentinel v1.0 — Roles, propietarios y privilegios de PostgreSQL 18.
-- Elaborado a partir de \dp, \dn+ y los respaldos verificados el 22/10/2026.
--
-- PRECONDICIONES:
--   1. Existen sargas_sentinel y sargas_sentinel_test.
--   2. En ambas bases ya se ejecutó sql/schema_v1.sql.
--   3. Se ejecuta con psql como administrador postgres y permisos suficientes.
--   4. Los nombres de los roles y las bases corresponden a una instalación LOCAL.
--

\set ON_ERROR_STOP on

-- 1. Roles globales (creación condicional, sin contraseña).
DO $roles$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'sentinel_owner') THEN
        CREATE ROLE sentinel_owner NOLOGIN;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'sentinel_app') THEN
        CREATE ROLE sentinel_app LOGIN;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'sentinel_test') THEN
        CREATE ROLE sentinel_test LOGIN;
    END IF;
END
$roles$;

ALTER ROLE sentinel_owner WITH NOSUPERUSER INHERIT NOCREATEROLE NOCREATEDB NOLOGIN NOREPLICATION NOBYPASSRLS;
ALTER ROLE sentinel_app   WITH NOSUPERUSER INHERIT NOCREATEROLE NOCREATEDB LOGIN NOREPLICATION NOBYPASSRLS;
ALTER ROLE sentinel_test  WITH NOSUPERUSER INHERIT NOCREATEROLE NOCREATEDB LOGIN NOREPLICATION NOBYPASSRLS;

-- 2. Propiedad y acceso a las bases (PUBLIC mantiene TEMPORARY, no CONNECT).
ALTER DATABASE sargas_sentinel OWNER TO sentinel_owner;
REVOKE CONNECT, TEMPORARY ON DATABASE sargas_sentinel FROM PUBLIC;
GRANT TEMPORARY ON DATABASE sargas_sentinel TO PUBLIC;
REVOKE ALL PRIVILEGES ON DATABASE sargas_sentinel FROM sentinel_app, sentinel_test;
GRANT CONNECT ON DATABASE sargas_sentinel TO sentinel_app;

ALTER DATABASE sargas_sentinel_test OWNER TO sentinel_owner;
REVOKE CONNECT, TEMPORARY ON DATABASE sargas_sentinel_test FROM PUBLIC;
GRANT TEMPORARY ON DATABASE sargas_sentinel_test TO PUBLIC;
REVOKE ALL PRIVILEGES ON DATABASE sargas_sentinel_test FROM sentinel_app, sentinel_test;
GRANT CONNECT ON DATABASE sargas_sentinel_test TO sentinel_test;

-- 3. Base principal. Se mantiene el esquema public en pg_database_owner.
\connect sargas_sentinel

REVOKE ALL PRIVILEGES ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO PUBLIC;
REVOKE ALL PRIVILEGES ON SCHEMA public FROM sentinel_app, sentinel_test;
GRANT USAGE ON SCHEMA public TO sentinel_app;

-- Las secuencias IDENTITY asociadas siguen la propiedad de sus tablas.
ALTER TABLE public.comprobaciones   OWNER TO sentinel_owner;
ALTER TABLE public.estados_servicio  OWNER TO sentinel_owner;
ALTER TABLE public.incidentes       OWNER TO sentinel_owner;
ALTER TABLE public.servicios        OWNER TO sentinel_owner;

REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM PUBLIC, sentinel_app, sentinel_test;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO sentinel_app;

REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM PUBLIC, sentinel_app, sentinel_test;
GRANT SELECT, USAGE ON ALL SEQUENCES IN SCHEMA public TO sentinel_app;

-- 4. Base de pruebas. Permite DELETE, pero no DDL a sentinel_test.
\connect sargas_sentinel_test

REVOKE ALL PRIVILEGES ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO PUBLIC;
REVOKE ALL PRIVILEGES ON SCHEMA public FROM sentinel_app, sentinel_test;
GRANT USAGE ON SCHEMA public TO sentinel_test;

ALTER TABLE public.comprobaciones   OWNER TO sentinel_owner;
ALTER TABLE public.estados_servicio  OWNER TO sentinel_owner;
ALTER TABLE public.incidentes       OWNER TO sentinel_owner;
ALTER TABLE public.servicios        OWNER TO sentinel_owner;

REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM PUBLIC, sentinel_app, sentinel_test;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO sentinel_test;

REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM PUBLIC, sentinel_app, sentinel_test;
GRANT SELECT, USAGE ON ALL SEQUENCES IN SCHEMA public TO sentinel_test;

\echo 'SARGAS: configuración de roles y privilegios aplicada; verificar con \dp y \dn+.'

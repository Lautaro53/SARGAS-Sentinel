
-- SARGAS Sentinel v1.0
-- Permisos de la base principal en Docker.
-- Requiere que los roles, la base y el esquema ya existan.
-- Debe ejecutarse conectado a sargas_sentinel
-- mediante el administrador de PostgreSQL.

-- 1. Propietario de la base
ALTER DATABASE sargas_sentinel OWNER TO sentinel_owner;

-- 2. Restringir conexiones
REVOKE CONNECT, TEMPORARY
ON DATABASE sargas_sentinel FROM PUBLIC;

GRANT CONNECT
ON DATABASE sargas_sentinel TO sentinel_app;

GRANT TEMPORARY
ON DATABASE sargas_sentinel TO PUBLIC;

-- 3. Permisos del esquema public
REVOKE ALL ON SCHEMA public FROM PUBLIC;

GRANT USAGE
ON SCHEMA public TO PUBLIC;

GRANT USAGE
ON SCHEMA public TO sentinel_app;

-- 4. Propietarios de tablas
ALTER TABLE public.servicios OWNER TO sentinel_owner;
ALTER TABLE public.comprobaciones OWNER TO sentinel_owner;
ALTER TABLE public.estados_servicio OWNER TO sentinel_owner;
ALTER TABLE public.incidentes OWNER TO sentinel_owner;

-- 5. Propietarios de secuencias
ALTER SEQUENCE public.servicios_id_seq OWNER TO sentinel_owner;
ALTER SEQUENCE public.comprobaciones_id_seq OWNER TO sentinel_owner;
ALTER SEQUENCE public.incidentes_id_seq OWNER TO sentinel_owner;

-- 6. Eliminar privilegios no previstos
REVOKE ALL ON ALL TABLES IN SCHEMA public
FROM PUBLIC, sentinel_app;

REVOKE ALL ON ALL SEQUENCES IN SCHEMA public
FROM PUBLIC, sentinel_app;

-- 7. Permisos de la aplicacion
GRANT SELECT, INSERT, UPDATE
ON ALL TABLES IN SCHEMA public TO sentinel_app;

GRANT USAGE, SELECT
ON ALL SEQUENCES IN SCHEMA public TO sentinel_app;

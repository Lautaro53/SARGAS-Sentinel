--
-- PostgreSQL database dump
--

\restrict nL7rC2M4ibxL2qMOErpaRpm7K6Um0bfmKyRt5egrf9nrxrTiSOVRS2ewQxjd556

-- Dumped from database version 18.6
-- Dumped by pg_dump version 18.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: comprobaciones; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.comprobaciones (
    id bigint NOT NULL,
    servicio_id bigint NOT NULL,
    codigo_http integer,
    latencia_ms double precision NOT NULL,
    error text,
    comprobado_en timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_codigo_http CHECK (((codigo_http >= 100) AND (codigo_http <= 599))),
    CONSTRAINT chk_latencia CHECK ((latencia_ms >= (0)::double precision))
);


--
-- Name: comprobaciones_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.comprobaciones ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.comprobaciones_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: estados_servicio; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.estados_servicio (
    servicio_id bigint NOT NULL,
    estado character varying(7) DEFAULT 'UNKNOWN'::character varying NOT NULL,
    exitos_consecutivos integer DEFAULT 0 NOT NULL,
    fallos_consecutivos integer DEFAULT 0 NOT NULL,
    ultima_comprobacion_en timestamp with time zone,
    CONSTRAINT chk_estado CHECK (((estado)::text = ANY ((ARRAY['UNKNOWN'::character varying, 'UP'::character varying, 'DOWN'::character varying])::text[]))),
    CONSTRAINT chk_exitos CHECK ((exitos_consecutivos >= 0)),
    CONSTRAINT chk_fallos CHECK ((fallos_consecutivos >= 0))
);


--
-- Name: incidentes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.incidentes (
    id bigint NOT NULL,
    servicio_id bigint NOT NULL,
    iniciado_en timestamp with time zone NOT NULL,
    cerrado_en timestamp with time zone,
    evidencia_inicio text NOT NULL,
    evidencia_cierre text,
    CONSTRAINT chk_cierre_incidente CHECK ((((cerrado_en IS NULL) AND (evidencia_cierre IS NULL)) OR ((cerrado_en IS NOT NULL) AND (evidencia_cierre IS NOT NULL)))),
    CONSTRAINT chk_fechas_incidente CHECK (((cerrado_en IS NULL) OR (cerrado_en >= iniciado_en)))
);


--
-- Name: incidentes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.incidentes ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.incidentes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: servicios; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.servicios (
    id bigint NOT NULL,
    nombre character varying(120) NOT NULL,
    url text NOT NULL,
    tipo character varying(10) DEFAULT 'HTTP'::character varying NOT NULL,
    intervalo_segundos integer NOT NULL,
    timeout_segundos integer NOT NULL,
    habilitado boolean DEFAULT true NOT NULL,
    archivado_en timestamp with time zone,
    CONSTRAINT chk_intervalo CHECK ((intervalo_segundos > 0)),
    CONSTRAINT chk_timeout CHECK ((timeout_segundos > 0)),
    CONSTRAINT chk_tipo CHECK (((tipo)::text = 'HTTP'::text))
);


--
-- Name: servicios_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.servicios ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.servicios_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: comprobaciones comprobaciones_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comprobaciones
    ADD CONSTRAINT comprobaciones_pkey PRIMARY KEY (id);


--
-- Name: estados_servicio estados_servicio_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estados_servicio
    ADD CONSTRAINT estados_servicio_pkey PRIMARY KEY (servicio_id);


--
-- Name: incidentes incidentes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incidentes
    ADD CONSTRAINT incidentes_pkey PRIMARY KEY (id);


--
-- Name: servicios servicios_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.servicios
    ADD CONSTRAINT servicios_pkey PRIMARY KEY (id);


--
-- Name: idx_comprobaciones_servicio_fecha; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_comprobaciones_servicio_fecha ON public.comprobaciones USING btree (servicio_id, comprobado_en DESC);


--
-- Name: idx_incidente_activo_unico; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_incidente_activo_unico ON public.incidentes USING btree (servicio_id) WHERE (cerrado_en IS NULL);


--
-- Name: comprobaciones fk_comprobacion_servicio; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comprobaciones
    ADD CONSTRAINT fk_comprobacion_servicio FOREIGN KEY (servicio_id) REFERENCES public.servicios(id);


--
-- Name: estados_servicio fk_estado_servicio; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estados_servicio
    ADD CONSTRAINT fk_estado_servicio FOREIGN KEY (servicio_id) REFERENCES public.servicios(id);


--
-- Name: incidentes fk_incidente_servicio; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incidentes
    ADD CONSTRAINT fk_incidente_servicio FOREIGN KEY (servicio_id) REFERENCES public.servicios(id);


--
-- PostgreSQL database dump complete
--

\unrestrict nL7rC2M4ibxL2qMOErpaRpm7K6Um0bfmKyRt5egrf9nrxrTiSOVRS2ewQxjd556

<script lang="ts">
    import StatusBadge from "../components/StatusBadge.svelte";
    import MetricCard from "../components/MetricCard.svelte";

    import { onMount } from "svelte";
    import type { Servicio, EstadoServicio } from "../types/servicio";
    import {
        obtenerServicios,
        obtenerEstadoServicio,
        esServicio,
        esEstadoServicio,
    } from "../api/servicios";

    import { obtenerTodosIncidentes } from "../api/incidentes";

    const CLAVE_CACHE = "sargas.dashboard.v1";
    const CLAVE_CACHE_INCIDENTES = "sargas.dashboard.incidentes.v1";

    let servicios = $state<Servicio[]>([]);
    let estados = $state<EstadoServicio[]>([]);

    let cargando = $state(true);
    let actualizando = $state(false);
    let mensajeError = $state("");
    let ultimaSincronizacion = $state<Date | null>(null);

    let mostrandoCache = $state(false);

    let errorEstados = $state("");

    let incidentesAbiertos = $state<number | null>(null);
    let errorIncidentes = $state("");
    let incidentesDesdeCache = $state(false);
    let fechaIncidentes = $state<Date | null>(null);

    let disponibles = $derived(
        estados.filter((s) => s.habilitado && s.estado === "UP").length,
    );

    let caidos = $derived(
        estados.filter((s) => s.habilitado && s.estado === "DOWN").length,
    );

    let desconocidos = $derived(
        estados.filter((s) => s.habilitado && s.estado === "UNKNOWN").length,
    );

    let deshabilitados = $derived(estados.filter((s) => !s.habilitado).length);

    function leerCache(): {
        servicios: Servicio[];
        estados: EstadoServicio[];
        fecha: Date;
    } | null {
        try {
            const texto = sessionStorage.getItem(CLAVE_CACHE);
            if (!texto) return null;

            const datos: unknown = JSON.parse(texto);

            if (
                typeof datos !== "object" ||
                datos === null ||
                Array.isArray(datos)
            )
                return null;

            const cache = datos as Record<string, unknown>;

            if (
                cache.version !== 1 ||
                !Array.isArray(cache.servicios) ||
                !cache.servicios.every(esServicio) ||
                !Array.isArray(cache.estados) ||
                !cache.estados.every(esEstadoServicio) ||
                typeof cache.sincronizadoEn !== "string"
            )
                return null;

            const fecha = new Date(cache.sincronizadoEn);
            if (Number.isNaN(fecha.getTime())) return null;

            const ids = new Set(cache.servicios.map((s) => s.id));
            const idsEstados = new Set(cache.estados.map((e) => e.servicio_id));

            if (
                ids.size !== cache.servicios.length ||
                cache.estados.length !== cache.servicios.length ||
                idsEstados.size !== ids.size ||
                cache.estados.some((e) => !ids.has(e.servicio_id))
            )
                return null;

            return {
                servicios: cache.servicios,
                estados: cache.estados,
                fecha,
            };
        } catch {
            return null;
        }
    }

    function leerCacheIncidentes(): { total: number; fecha: Date } | null {
        try {
            const texto = sessionStorage.getItem(CLAVE_CACHE_INCIDENTES);
            if (!texto) return null;

            const valor: unknown = JSON.parse(texto);

            if (
                typeof valor !== "object" ||
                valor === null ||
                Array.isArray(valor)
            )
                return null;

            const datos = valor as Record<string, unknown>;

            if (
                datos.version !== 1 ||
                typeof datos.total !== "number" ||
                !Number.isSafeInteger(datos.total) ||
                datos.total < 0 ||
                datos.total > 50 ||
                typeof datos.sincronizadoEn !== "string"
            )
                return null;

            const fecha = new Date(datos.sincronizadoEn);
            if (Number.isNaN(fecha.getTime())) return null;

            return { total: datos.total, fecha };
        } catch {
            return null;
        }
    }

    onMount(() => {
        let activo = true;
        let consultaEnCurso = false;
        let consultaIncidentesEnCurso = false;

        const cache = leerCache();

        if (cache) {
            mostrandoCache = true;
            servicios = cache.servicios;
            estados = cache.estados;
            ultimaSincronizacion = cache.fecha;
            cargando = false;
        }

        const cacheIncidentes = leerCacheIncidentes();

        if (cacheIncidentes) {
            incidentesAbiertos = cacheIncidentes.total;
            fechaIncidentes = cacheIncidentes.fecha;
            incidentesDesdeCache = true;
        }

        async function actualizarDatos() {
            // Evitar consultas superpuestas.
            if (consultaEnCurso) return;

            consultaEnCurso = true;
            actualizando = true;

            try {
                // Consultar servicios y sus estados.
                const lista = await obtenerServicios();

                const nuevosEstados = await Promise.all(
                    lista.map((servicio) => obtenerEstadoServicio(servicio.id)),
                );

                if (!activo) return;

                const ahora = new Date();

                servicios = lista;
                estados = nuevosEstados;

                mensajeError = "";
                ultimaSincronizacion = ahora;
                mostrandoCache = false;

                try {
                    sessionStorage.setItem(
                        CLAVE_CACHE,
                        JSON.stringify({
                            version: 1,
                            servicios: lista,
                            estados: nuevosEstados,
                            sincronizadoEn: ahora.toISOString(),
                        }),
                    );
                } catch {
                    // Un fallo de almacenamiento no debe afectar
                    // los datos recibidos correctamente desde la API.
                }
            } catch (error) {
                if (!activo) return;

                mensajeError =
                    error instanceof Error
                        ? error.message
                        : "No se pudo actualizar el monitoreo";
            } finally {
                consultaEnCurso = false;

                if (activo) {
                    cargando = false;
                    actualizando = false;
                }
            }
        }

        async function actualizarIncidentes() {
            if (consultaIncidentesEnCurso) return;

            consultaIncidentesEnCurso = true;

            try {
                const lista = await obtenerTodosIncidentes();

                if (!activo) return;

                const cantidad = lista.filter(
                    (incidente) =>
                        incidente.cerrado_en === null &&
                        !incidente.servicio_archivado,
                ).length;

                const ahora = new Date();

                incidentesAbiertos = cantidad;
                fechaIncidentes = ahora;
                incidentesDesdeCache = false;
                errorIncidentes = "";

                try {
                    sessionStorage.setItem(
                        CLAVE_CACHE_INCIDENTES,
                        JSON.stringify({
                            version: 1,
                            total: cantidad,
                            sincronizadoEn: ahora.toISOString(),
                        }),
                    );
                } catch {
                    // No afectar la consulta si falla el almacenamiento.
                }
            } catch (error) {
                if (!activo) return;

                errorIncidentes =
                    error instanceof Error
                        ? error.message
                        : "No se pudieron consultar los incidentes";
            } finally {
                consultaIncidentesEnCurso = false;
            }
        }

        // Primera consulta al abrir la pantalla.
        void actualizarDatos();
        void actualizarIncidentes();

        // Repetir cada 15 segundos.
        const temporizador = window.setInterval(() => {
            void actualizarDatos();
            void actualizarIncidentes();
        }, 15_000);

        // Limpiar al desmontar el componente.
        return () => {
            activo = false;
            window.clearInterval(temporizador);
        };
    });
</script>

<div class="dashboard">
    <p class="eyebrow">SARGAS SENTINEL</p>

    <h1>Panel general</h1>

    <p class="description">Supervisa la disponibilidad de tus servicios.</p>

    <section class="metrics-section" aria-label="Resumen de servicios">
        {#if mensajeError}
            <p class="api-error" role="alert">
                No se pudo actualizar: {mensajeError}

                {#if ultimaSincronizacion !== null}
                    Se muestran los últimos datos recibidos.
                {/if}
            </p>
        {/if}

        {#if mostrandoCache}
            <p class="demo-note" role="status">
                Se muestran datos guardados de la última sincronización. La
                información todavía no fue actualizada.
            </p>
        {/if}

        {#if ultimaSincronizacion !== null}
            <p class="demo-note">
                Última sincronización:
                {ultimaSincronizacion.toLocaleString("es-UY", {
                    dateStyle: "short",
                    timeStyle: "medium",
                })}

                {#if actualizando}
                    · Actualizando...
                {/if}
            </p>
        {/if}

        {#if cargando}
            <p class="demo-note">Consultando estados de Sentinel...</p>
        {:else if ultimaSincronizacion === null}
            <p class="demo-note">No hay datos disponibles.</p>
        {:else}
            <div class="metrics-grid">
                <MetricCard
                    titulo="Servicios registrados"
                    valor={servicios.length}
                    descripcion="Total configurado"
                />

                <MetricCard
                    titulo="Disponibles"
                    valor={disponibles}
                    descripcion="Estado UP"
                    tono="up"
                />

                <MetricCard
                    titulo="No disponibles"
                    valor={caidos}
                    descripcion="Estado DOWN"
                    tono="down"
                />

                <MetricCard
                    titulo="Sin determinar"
                    valor={desconocidos}
                    descripcion="Estado UNKNOWN"
                    tono="unknown"
                />

                {#if incidentesAbiertos !== null}
                    <MetricCard
                        titulo="Incidentes abiertos"
                        valor={incidentesAbiertos}
                        descripcion={incidentesDesdeCache
                            ? "Conteo guardado · últimos 50 incidentes"
                            : "Entre los últimos 50 incidentes"}
                        tono="down"
                    />
                {/if}

                {#if incidentesDesdeCache && fechaIncidentes !== null}
                    <p class="demo-note" role="status">
                        Conteo de incidentes guardado el
                        {fechaIncidentes.toLocaleString("es-UY")}. Pendiente de
                        actualización.
                    </p>
                {/if}
            </div>

            {#if errorIncidentes}
                <p class="api-error" role="alert">
                    No se pudieron actualizar los incidentes:
                    {errorIncidentes}

                    {#if incidentesAbiertos !== null}
                        Se muestra el último conteo disponible.
                    {/if}
                </p>
            {/if}

            {#if deshabilitados > 0}
                <p class="demo-note">
                    {deshabilitados} servicio(s) deshabilitado(s), excluido(s) del
                    desglose de estados.
                </p>
            {/if}
        {/if}
    </section>

    <section class="services-panel" aria-labelledby="services-title">
        <div class="panel-header">
            <div>
                <h2 id="services-title">Servicios monitorizados</h2>
                <p>Estado y última comprobación de cada servicio.</p>
            </div>

            <span class="services-count">
                {#if ultimaSincronizacion !== null}
                    {servicios.length} registrados
                {:else}
                    Sin datos confirmados
                {/if}
            </span>
        </div>

        {#if cargando}
            <p class="table-message">Cargando servicios...</p>
        {:else if ultimaSincronizacion === null}
            <p class="table-message">No hay datos disponibles.</p>
        {:else if servicios.length === 0}
            <p class="table-message">Todavía no hay servicios registrados.</p>
        {:else}
            <div class="table-scroll">
                <table>
                    <thead>
                        <tr>
                            <th scope="col">Servicio</th>
                            <th scope="col">Estado</th>
                            <th scope="col">Monitoreo</th>
                            <th scope="col">Última comprobación</th>
                        </tr>
                    </thead>

                    <tbody>
                        {#each servicios as servicio (servicio.id)}
                            {@const estadoServicio = estados.find(
                                (e) => e.servicio_id === servicio.id,
                            )}

                            <tr>
                                <td>
                                    <strong class="service-name">
                                        {servicio.nombre}
                                    </strong>

                                    <span class="service-url">
                                        {servicio.url}
                                    </span>
                                </td>

                                <td>
                                    {#if estadoServicio}
                                        <StatusBadge
                                            estado={estadoServicio.estado}
                                        />
                                    {:else}
                                        <span class="muted">Sin datos</span>
                                    {/if}
                                </td>

                                <td>
                                    <span
                                        class="monitor-status"
                                        class:paused={!servicio.habilitado}
                                    >
                                        {servicio.habilitado
                                            ? "Activo"
                                            : "Pausado"}
                                    </span>
                                </td>

                                <td class="date-cell">
                                    {#if estadoServicio?.ultima_comprobacion_en}
                                        {new Date(
                                            estadoServicio.ultima_comprobacion_en,
                                        ).toLocaleString("es-UY", {
                                            dateStyle: "short",
                                            timeStyle: "short",
                                        })}
                                    {:else}
                                        <span class="muted"
                                            >Sin comprobaciones</span
                                        >
                                    {/if}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}

        <p class="panel-note">
            Los servicios pausados conservan su último estado conocido.
        </p>
    </section>
</div>

<style>
    .eyebrow {
        color: var(--color-primary);
        font-size: 12px;
        letter-spacing: 2px;
    }

    h1 {
        margin: 12px 0;
        font-size: 32px;
    }

    .description {
        color: var(--color-muted);
    }

    .metrics-section {
        margin-top: 32px;
    }

    .demo-note {
        margin-bottom: 12px;
        color: var(--color-muted);
        font-size: 12px;
    }

    .metrics-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
        gap: 16px;
    }

    .api-error {
        color: var(--color-down) !important;
    }

    .services-panel {
        margin-top: 24px;
        border: 1px solid var(--color-border);
        border-radius: 12px;
        background: var(--color-surface);
        overflow: hidden;
    }

    .panel-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: 16px;
        padding: 24px;
    }

    .panel-header h2 {
        margin: 0 0 8px;
        font-size: 18px;
    }

    .panel-header p,
    .services-count,
    .panel-note,
    .muted {
        color: var(--color-muted);
        font-size: 12px;
    }

    .panel-header p {
        margin: 0;
    }

    .table-scroll {
        overflow-x: auto;
    }

    table {
        width: 100%;
        min-width: 720px;
        border-collapse: collapse;
        text-align: left;
    }

    th {
        background: #15243a;
        color: var(--color-muted);
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    th,
    td {
        padding: 16px 20px;
    }

    td {
        border-top: 1px solid var(--color-border);
        font-size: 13px;
    }

    .service-name,
    .service-url {
        display: block;
    }

    .service-name {
        font-weight: 600;
    }

    .service-url {
        margin-top: 6px;
        color: var(--color-muted);
        font-size: 12px;
        overflow-wrap: anywhere;
    }

    .monitor-status {
        color: var(--color-up);
        font-weight: 500;
    }

    .monitor-status.paused {
        color: var(--color-muted);
    }

    .date-cell {
        color: var(--color-muted);
        font-variant-numeric: tabular-nums;
    }

    .table-message,
    .panel-note {
        padding: 20px 24px;
        margin: 0;
    }

    .panel-note {
        border-top: 1px solid var(--color-border);
    }

    @media (max-width: 1100px) {
        .metrics-grid {
            grid-template-columns: repeat(2, minmax(0, 1fr));
        }
    }

    @media (max-width: 520px) {
        .metrics-grid {
            grid-template-columns: 1fr;
        }
    }
</style>

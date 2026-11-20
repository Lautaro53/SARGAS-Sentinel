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

    import CheckHistory from "../components/CheckHistory.svelte";
    import { obtenerComprobaciones } from "../api/comprobaciones";
    import type { Comprobacion } from "../types/comprobacion";
    import Paginacion from "../components/Paginacion.svelte";
    import ServicioIdentidad from "../components/ServicioIdentidad.svelte";

    type Props = {
        onVerDetalle: (servicio: Servicio) => void;
    };

    let { onVerDetalle }: Props = $props();

    const CLAVE_CACHE = "sargas.dashboard.v1";
    const CLAVE_CACHE_INCIDENTES = "sargas.dashboard.incidentes.v1";

    let servicios = $state<Servicio[]>([]);
    let estados = $state<EstadoServicio[]>([]);

    let historialServicios = $state<Record<number, Comprobacion[]>>({});
    let erroresHistorial = $state<number[]>([]);

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

    const SERVICIOS_POR_PAGINA = 10;

    let paginaServicios = $state(1);

    let totalPaginasServicios = $derived(
        Math.max(1, Math.ceil(servicios.length / SERVICIOS_POR_PAGINA)),
    );

    let paginaActualServicios = $derived(
        Math.min(paginaServicios, totalPaginasServicios),
    );

    let serviciosPaginados = $derived(
        servicios.slice(
            (paginaActualServicios - 1) * SERVICIOS_POR_PAGINA,
            paginaActualServicios * SERVICIOS_POR_PAGINA,
        ),
    );

    let contenedorTabla = $state<HTMLDivElement | undefined>(undefined);

    let alturaTabla = $state<number | null>(null);

    function cambiarPaginaServicios(destino: number) {
        if (destino < 1 || destino > totalPaginasServicios) return;

        if (paginaActualServicios === 1 && contenedorTabla) {
            alturaTabla = Math.ceil(
                contenedorTabla.getBoundingClientRect().height,
            );
        }

        paginaServicios = destino;
    }

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

    async function consultarHistoriales(lista: Servicio[]): Promise<{
        historiales: Record<number, Comprobacion[]>;
        errores: number[];
    }> {
        const historiales: Record<number, Comprobacion[]> = {};
        const errores: number[] = [];

        const TAMANO_LOTE = 4;

        for (let i = 0; i < lista.length; i += TAMANO_LOTE) {
            const lote = lista.slice(i, i + TAMANO_LOTE);

            const resultados = await Promise.allSettled(
                lote.map((servicio) => obtenerComprobaciones(servicio.id, 20)),
            );

            resultados.forEach((resultado, indice) => {
                const id = lote[indice].id;

                if (resultado.status === "fulfilled") {
                    historiales[id] = resultado.value;
                } else {
                    errores.push(id);
                }
            });
        }

        return { historiales, errores };
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
            if (consultaEnCurso) return;

            consultaEnCurso = true;
            actualizando = true;

            try {
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
                const resultadoHistorial = await consultarHistoriales(lista);

                if (!activo) return;

                historialServicios = resultadoHistorial.historiales;
                erroresHistorial = resultadoHistorial.errores;
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

        void actualizarDatos();
        void actualizarIncidentes();

        const temporizador = window.setInterval(() => {
            void actualizarDatos();
            void actualizarIncidentes();
        }, 15_000);

        return () => {
            activo = false;
            window.clearInterval(temporizador);
        };
    });
</script>

<div class="dashboard">
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
            <div class="sync-row">
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
            </div>
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
                />

                <MetricCard titulo="Estado UP" valor={disponibles} tono="up" />

                <MetricCard titulo="Estado DOWN" valor={caidos} tono="down" />

                <MetricCard
                    titulo="Estado UNKNOWN"
                    valor={desconocidos}
                    tono="unknown"
                />

                {#if incidentesAbiertos !== null}
                    <MetricCard
                        titulo="Incidentes abiertos"
                        valor={incidentesAbiertos}
                        tono="down"
                    />
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
        {/if}
    </section>

    <section
        class="services-panel data-table-panel"
        aria-labelledby="services-title"
    >
        <div class="panel-header">
            <h2 id="services-title">Servicios monitorizados</h2>
        </div>

        {#if cargando}
            <p class="table-message">Cargando servicios...</p>
        {:else if ultimaSincronizacion === null}
            <p class="table-message">No hay datos disponibles.</p>
        {:else if servicios.length === 0}
            <p class="table-message">Todavía no hay servicios registrados.</p>
        {:else}
            <div
                class="table-scroll"
                bind:this={contenedorTabla}
                style:min-height={totalPaginasServicios > 1 &&
                alturaTabla !== null
                    ? `${alturaTabla}px`
                    : undefined}
            >
                <table>
                    <thead>
                        <tr>
                            <th scope="col">Servicio</th>
                            <th scope="col">Estado</th>
                            <th scope="col">Monitoreo</th>
                            <th scope="col">Última comprobación</th>
                            <th scope="col">Última latencia</th>
                            <th scope="col">Últimas 20</th>
                        </tr>
                    </thead>

                    <tbody>
                        {#each serviciosPaginados as servicio (servicio.id)}
                            {@const estadoServicio = estados.find(
                                (e) => e.servicio_id === servicio.id,
                            )}

                            {@const historial =
                                historialServicios[servicio.id] ?? []}

                            {@const ultimaComprobacion = [...historial].sort(
                                (a, b) =>
                                    new Date(b.comprobado_en).getTime() -
                                    new Date(a.comprobado_en).getTime(),
                            )[0]}

                            <tr>
                                <td>
                                    <ServicioIdentidad
                                        id={servicio.id}
                                        nombre={servicio.nombre}
                                        url={servicio.url}
                                        onVerDetalle={() =>
                                            onVerDetalle(servicio)}
                                    />
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

                                <td class="latency-cell">
                                    {#if erroresHistorial.includes(servicio.id)}
                                        <span class="muted">No disponible</span>
                                    {:else if ultimaComprobacion}
                                        <span class="latency-value">
                                            {ultimaComprobacion.latencia_ms.toLocaleString(
                                                "es-UY",
                                                {
                                                    minimumFractionDigits: 0,
                                                    maximumFractionDigits: 1,
                                                },
                                            )} ms
                                        </span>
                                    {:else if servicio.id in historialServicios}
                                        <span class="muted">—</span>
                                    {:else}
                                        <span class="muted">Consultando...</span
                                        >
                                    {/if}
                                </td>

                                <td>
                                    {#if erroresHistorial.includes(servicio.id)}
                                        <span class="muted">No disponible</span>
                                    {:else if servicio.id in historialServicios}
                                        <CheckHistory
                                            comprobaciones={historialServicios[
                                                servicio.id
                                            ]}
                                        />
                                    {:else if mensajeError}
                                        <span class="muted">No disponible</span>
                                    {:else}
                                        <span class="muted">Consultando...</span
                                        >
                                    {/if}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
            <Paginacion
                pagina={paginaActualServicios}
                totalPaginas={totalPaginasServicios}
                onCambiarPagina={cambiarPaginaServicios}
            />
        {/if}
    </section>
</div>

<style>
    .metrics-section {
        margin-top: var(--space-2);
        min-width: 0;
    }

    .demo-note {
        margin-bottom: 12px;
        color: var(--color-muted);
        font-size: 12px;
    }

    .metrics-grid {
        display: grid;
        grid-template-columns: repeat(5, minmax(0, 1fr));
        gap: 12px;
    }

    .dashboard {
        display: flex;
        flex-direction: column;
        gap: var(--space-4);
        min-width: 0;
    }

    .services-panel {
        min-width: 0;
        margin-top: 16px;
        border: 1px solid #243245;
        border-radius: var(--radius-card);
        background: #141a22;
        overflow: hidden;
    }

    .api-error {
        color: var(--color-down) !important;
    }

    .panel-header h2 {
        margin: 0 0 var(--space-2);
        line-height: 1.3;
    }

    .monitor-status.paused {
        color: var(--color-muted);
    }

    .date-cell {
        color: var(--color-muted);
        font-variant-numeric: tabular-nums;
    }

    .latency-cell {
        white-space: nowrap;
    }

    .latency-value {
        color: var(--color-text);
        font-size: 13px;
        font-variant-numeric: tabular-nums;
    }

    .sync-row {
        display: flex;
        justify-content: flex-start;
        align-items: center;
        margin-bottom: 6px;
    }

    .sync-row .demo-note {
        margin: 0;
    }

    @media (max-width: 1200px) {
        .metrics-grid {
            grid-template-columns: repeat(3, minmax(0, 1fr));
        }
    }

    @media (max-width: 760px) {
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

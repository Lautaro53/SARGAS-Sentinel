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
    import { obtenerComprobaciones, esComprobacion } from "../api/comprobaciones";
    import type { Comprobacion } from "../types/comprobacion";
    import Paginacion from "../components/Paginacion.svelte";
    import { guardarCopia, leerCopia, esLista, mensajeErrorConsulta } from "../lib/cacheSesion";
    import { leerServiciosGuardados } from "../api/servicios";
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
        estados.filter((s) => s.estado === "UP").length,
    );

    let caidos = $derived(
        estados.filter((s) => s.estado === "DOWN").length,
    );

    let desconocidos = $derived(
        estados.filter((s) => s.estado === "UNKNOWN").length,
    );

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
            if (Number.isNaN(fecha.getTime()) || fecha.getTime() > Date.now() + 60_000 ||
                Date.now() - fecha.getTime() > 24 * 60 * 60 * 1000) return null;

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
                    guardarCopia(`detalle.${id}.grafica.20`, resultado.value);
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

            const copiaServicios = leerServiciosGuardados();
            if (!copiaServicios || Date.parse(copiaServicios.sincronizadoEn) < cache.fecha.getTime()) {
                guardarCopia("servicios", cache.servicios, cache.fecha.toISOString());
            }
            for (const estado of cache.estados) {
                const clave = `detalle.${estado.servicio_id}.estado`;
                const copiaEstado = leerCopia(clave, (v): v is EstadoServicio =>
                    esEstadoServicio(v) && v.servicio_id === estado.servicio_id);
                if (!copiaEstado || Date.parse(copiaEstado.sincronizadoEn) < cache.fecha.getTime()) {
                    guardarCopia(clave, estado, cache.fecha.toISOString());
                }
            }
            const copiasHistorial: Record<number, Comprobacion[]> = {};
            for (const servicio of cache.servicios) {
                const copiaGrafica = leerCopia(`detalle.${servicio.id}.grafica.20`,
                    (v): v is Comprobacion[] => esLista(v, (c): c is Comprobacion =>
                        esComprobacion(c) && c.servicio_id === servicio.id) && v.length <= 20);
                if (copiaGrafica) copiasHistorial[servicio.id] = copiaGrafica.datos;
            }
            historialServicios = copiasHistorial;
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

                historialServicios = { ...historialServicios, ...resultadoHistorial.historiales };
                erroresHistorial = resultadoHistorial.errores;
            } catch (error) {
                if (!activo) return;

                mensajeError = mensajeErrorConsulta(error, "No se pudo actualizar el monitoreo");
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

                errorIncidentes = mensajeErrorConsulta(error, "No se pudieron consultar los incidentes");
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
        {#if mensajeError || errorIncidentes || mostrandoCache}
            <p class="stale-warning" role="status">
                {#if mensajeError}
                    {ultimaSincronizacion !== null ? "Datos guardados / sin conexión" : "No se pudo conectar a la API"}
                {:else if errorIncidentes}
                    {incidentesAbiertos !== null ? "Incidentes: se muestra el último conteo guardado" : "No se pudieron consultar los incidentes"}
                {:else}
                    Datos guardados · sincronizando con la API
                {/if}
                {#if ultimaSincronizacion !== null}
                    · última sincronización de servicios: {ultimaSincronizacion.toLocaleString("es-UY")}
                {/if}
                {#if mensajeError && errorIncidentes}
                    · también falló la consulta de incidentes.
                {/if}
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

                <MetricCard titulo="Estado UP" valor={disponibles} tono="up" ayuda="Incluye servicios pausados" />

                <MetricCard titulo="Estado DOWN" valor={caidos} tono="down" ayuda="Incluye servicios pausados" />

                <MetricCard
                    titulo="Estado UNKNOWN"
                    valor={desconocidos}
                    tono="unknown"
                    ayuda="Incluye servicios pausados"
                />

                {#if incidentesAbiertos !== null}
                    <MetricCard
                        titulo="Incidentes abiertos"
                        valor={incidentesAbiertos}
                        tono="down"
                    />
                {/if}
            </div>


        {/if}
    </section>

    <section
        class="services-panel data-table-panel"
        aria-labelledby="services-title"
    >
        <div class="panel-header">
            <h2 id="services-title">Servicios monitorizados</h2>
        </div>
        {#if erroresHistorial.length > 0 && Object.keys(historialServicios).length > 0}
            <p class="stale-warning" role="status">Algunas latencias no se pudieron actualizar. Se muestran datos guardados cuando están disponibles.</p>
        {/if}

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
                                    {#if ultimaComprobacion}
                                        <span class="latency-value">
                                            {ultimaComprobacion.latencia_ms.toLocaleString(
                                                "es-UY",
                                                {
                                                    minimumFractionDigits: 0,
                                                    maximumFractionDigits: 1,
                                                },
                                            )} ms
                                        </span>
                                    {:else if erroresHistorial.includes(servicio.id)}
                                        <span class="muted">No disponible</span>
                                    {:else if servicio.id in historialServicios}
                                        <span class="muted">—</span>
                                    {:else}
                                        <span class="muted">Consultando...</span
                                        >
                                    {/if}
                                </td>

                                <td>
                                    {#if servicio.id in historialServicios}
                                        <CheckHistory
                                            comprobaciones={historialServicios[
                                                servicio.id
                                            ]}
                                        />
                                    {:else if erroresHistorial.includes(servicio.id) || mensajeError}
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
        border: 1px solid var(--color-border);
        border-radius: var(--radius-card);
        background: var(--color-surface);
        overflow: hidden;
    }


    /* En tablets, conservar columnas legibles y desplazar solo la tabla. */
    .services-panel table {
        min-width: 930px;
        table-layout: fixed;
    }

    .services-panel th:nth-child(1) { width: 25%; }
    .services-panel th:nth-child(2) { width: 13%; }
    .services-panel th:nth-child(3) { width: 9%; }
    .services-panel th:nth-child(4) { width: 19%; }
    .services-panel th:nth-child(5) { width: 13%; }
    .services-panel th:nth-child(6) { width: 21%; }

    .services-panel :global(.service-info) {
        flex: 1;
        overflow: hidden;
    }

    .services-panel :global(.service-name) {
        width: 100%;
        overflow: hidden;
        overflow-wrap: normal;
        white-space: nowrap;
        text-overflow: ellipsis;
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

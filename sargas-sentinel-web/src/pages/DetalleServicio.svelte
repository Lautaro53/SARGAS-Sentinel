<script lang="ts">
    import { onMount } from "svelte";

    import StatusBadge from "../components/StatusBadge.svelte";
    import GraficoLatencia from "../components/GraficoLatencia.svelte";
    import { obtenerEstadoServicio } from "../api/servicios";

    import type { Servicio, EstadoServicio } from "../types/servicio";

    import type { Comprobacion } from "../types/comprobacion";
    import { obtenerComprobaciones } from "../api/comprobaciones";

    import type { Incidente } from "../types/incidente";
    import { obtenerIncidentes } from "../api/incidentes";

    type Props = {
        servicio: Servicio;
        onVolver: () => void;
    };

    let { servicio, onVolver }: Props = $props();

    let estado = $state<EstadoServicio | null>(null);
    let cargando = $state(true);
    let error = $state("");

    let comprobaciones = $state<Comprobacion[]>([]);
    let cargandoComprobaciones = $state(true);
    let errorComprobaciones = $state("");

    let incidentes = $state<Incidente[]>([]);
    let cargandoIncidentes = $state(true);
    let errorIncidentes = $state("");

    type CantidadGrafica = 20 | 50 | 100;
    let cantidadGrafica = $state<CantidadGrafica>(20);
    let ultimaSincronizacionGrafica = $state<string | null>(null);

    // La selección y el identificador controlan el ciclo de vida de la consulta.
    $effect(() => {
        const servicioId = servicio.id;
        const cantidad = cantidadGrafica;
        const controlador = new AbortController();
        let vigente = true;
        let consultando = false;

        cargandoComprobaciones = true;
        errorComprobaciones = "";

        async function actualizar() {
            if (consultando || document.hidden) return;
            consultando = true;
            try {
                const datos = await obtenerComprobaciones(servicioId, cantidad, controlador.signal);
                if (vigente) {
                    comprobaciones = datos;
                    ultimaSincronizacionGrafica = new Date().toISOString();
                    errorComprobaciones = "";
                }
            } catch (e) {
                if (vigente) {
                    errorComprobaciones = e instanceof Error
                        ? e.message : "No se pudo consultar el historial";
                }
            } finally {
                consultando = false;
                if (vigente) cargandoComprobaciones = false;
            }
        }

        void actualizar();
        const intervalo = window.setInterval(() => void actualizar(), 15_000);
        const volverAVer = () => { if (!document.hidden) void actualizar(); };
        document.addEventListener("visibilitychange", volverAVer);

        return () => {
            vigente = false;
            controlador.abort();
            window.clearInterval(intervalo);
            document.removeEventListener("visibilitychange", volverAVer);
        };
    });

    onMount(() => {
        let activo = true;

        async function cargarEstado() {
            try {
                const resultado = await obtenerEstadoServicio(servicio.id);

                if (activo) {
                    estado = resultado;
                }
            } catch (e) {
                if (activo) {
                    error =
                        e instanceof Error
                            ? e.message
                            : "No se pudo consultar el estado";
                }
            } finally {
                if (activo) {
                    cargando = false;
                }
            }
        }

        async function cargarIncidentes() {
            try {
                const datos = await obtenerIncidentes(servicio.id);

                if (activo) {
                    incidentes = datos;
                }
            } catch (e) {
                if (activo) {
                    errorIncidentes =
                        e instanceof Error
                            ? e.message
                            : "No se pudo consultar el historial de incidentes";
                }
            } finally {
                if (activo) {
                    cargandoIncidentes = false;
                }
            }
        }

        void cargarEstado();
        void cargarIncidentes();

        return () => {
            activo = false;
        };
    });
</script>

<div class="detail">
    <button class="back-button" type="button" onclick={onVolver}>
        ← Volver a servicios
    </button>

    <header class="detail-header">
        <p class="eyebrow">DETALLE DEL SERVICIO</p>
        <h1>{servicio.nombre}</h1>
        <p class="url">{servicio.url}</p>
    </header>

    {#if cargando}
        <p class="message">Consultando estado...</p>
    {:else if error}
        <p class="error" role="alert">{error}</p>
    {:else if estado}
        <section class="summary" aria-label="Estado del servicio">
            <article class="card">
                <h2>Estado actual</h2>
                <StatusBadge estado={estado.estado} />
            </article>

            <article class="card">
                <h2>Monitoreo</h2>
                <strong class:paused={!estado.habilitado}>
                    {estado.habilitado ? "Activo" : "Pausado"}
                </strong>
            </article>
        </section>

        <section class="card">
            <h2>Última comprobación</h2>

            <p>
                {estado.ultima_comprobacion_en
                    ? new Date(estado.ultima_comprobacion_en).toLocaleString(
                          "es-UY",
                      )
                    : "Sin comprobaciones registradas"}
            </p>

            <div class="counters">
                <span>Éxitos consecutivos: {estado.exitos_consecutivos}</span>
                <span>Fallos consecutivos: {estado.fallos_consecutivos}</span>
            </div>
        </section>
    {/if}

    <section class="card">
        <h2>Configuración</h2>

        <dl class="configuration">
            <div>
                <dt>Tipo</dt>
                <dd>{servicio.tipo}</dd>
            </div>

            <div>
                <dt>Intervalo</dt>
                <dd>{servicio.intervalo_segundos} segundos</dd>
            </div>

            <div>
                <dt>Timeout</dt>
                <dd>{servicio.timeout_segundos} segundos</dd>
            </div>
        </dl>
    </section>

    <section class="card" aria-label="Evolución de latencia">
        {#if cargandoComprobaciones}
            <p class="message">Cargando latencias...</p>
        {:else if errorComprobaciones && comprobaciones.length === 0}
            <p class="error" role="alert">{errorComprobaciones}</p>
        {:else if comprobaciones.length === 0}
            <p class="message">Todavía no hay comprobaciones para graficar.</p>
        {:else}
            {#if errorComprobaciones}
                <p class="refresh-warning" role="status">
                    No se pudo actualizar la gráfica. Se muestran datos de la última consulta correcta.
                </p>
            {/if}
            <GraficoLatencia
                {comprobaciones}
                cantidad={cantidadGrafica}
                ultimaSincronizacion={ultimaSincronizacionGrafica}
                onCambiarCantidad={(cantidad) => (cantidadGrafica = cantidad)}
            />
        {/if}
    </section>

    <section class="card" aria-labelledby="history-title">
        <h2 id="history-title">Historial de comprobaciones</h2>

        {#if cargandoComprobaciones}
            <p class="message">Cargando comprobaciones...</p>
        {:else if errorComprobaciones && comprobaciones.length === 0}
            <p class="error" role="alert">
                {errorComprobaciones}
            </p>
        {:else if comprobaciones.length === 0}
            <p class="message">
                Este servicio todavía no tiene comprobaciones registradas.
            </p>
        {:else}
            <div class="history-scroll">
                <table>
                    <thead>
                        <tr>
                            <th scope="col">Fecha</th>
                            <th scope="col">Código HTTP</th>
                            <th scope="col">Latencia</th>
                            <th scope="col">Error</th>
                        </tr>
                    </thead>

                    <tbody>
                        {#each comprobaciones.slice(0, 20) as comprobacion (comprobacion.id)}
                            <tr>
                                <td>
                                    {new Date(
                                        comprobacion.comprobado_en,
                                    ).toLocaleString("es-UY")}
                                </td>

                                <td>
                                    {comprobacion.codigo_http === null
                                        ? "—"
                                        : comprobacion.codigo_http}
                                </td>

                                <td>
                                    {comprobacion.latencia_ms.toFixed(2)} ms
                                </td>

                                <td class="error-detail">
                                    {comprobacion.error ?? "—"}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </section>

    <section class="card" aria-labelledby="incidents-title">
        <h2 id="incidents-title">Historial de incidentes</h2>

        {#if cargandoIncidentes}
            <p class="message">Cargando incidentes...</p>
        {:else if errorIncidentes}
            <p class="error" role="alert">
                {errorIncidentes}
            </p>
        {:else if incidentes.length === 0}
            <p class="message">
                Este servicio no tiene incidentes registrados.
            </p>
        {:else}
            <div class="history-scroll">
                <table>
                    <thead>
                        <tr>
                            <th scope="col">Inicio</th>
                            <th scope="col">Cierre</th>
                            <th scope="col">Estado</th>
                            <th scope="col">Evidencia de inicio</th>
                            <th scope="col">Evidencia de cierre</th>
                        </tr>
                    </thead>

                    <tbody>
                        {#each incidentes as incidente (incidente.id)}
                            <tr>
                                <td>
                                    {new Date(
                                        incidente.iniciado_en,
                                    ).toLocaleString("es-UY")}
                                </td>

                                <td>
                                    {incidente.cerrado_en
                                        ? new Date(
                                              incidente.cerrado_en,
                                          ).toLocaleString("es-UY")
                                        : "—"}
                                </td>

                                <td>
                                    <span
                                        class="incident-status"
                                        class:open={incidente.cerrado_en ===
                                            null}
                                    >
                                        {incidente.cerrado_en === null
                                            ? "Abierto"
                                            : "Cerrado"}
                                    </span>
                                </td>

                                <td class="error-detail">
                                    {incidente.evidencia_inicio}
                                </td>

                                <td class="error-detail">
                                    {incidente.evidencia_cierre ?? "—"}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </section>
</div>

<style>
    .detail {
        display: grid;
        gap: 24px;
    }

    .back-button {
        justify-self: start;
        border: 0;
        padding: 0;
        background: transparent;
        color: var(--color-primary);
        font: inherit;
        cursor: pointer;
    }

    .back-button:hover {
        text-decoration: underline;
    }

    .back-button:focus-visible {
        outline: 2px solid var(--color-primary);
        outline-offset: 4px;
    }

    .eyebrow {
        color: var(--color-primary);
        font-size: 12px;
        letter-spacing: 2px;
    }

    h1 {
        margin: 8px 0;
        font-size: 30px;
    }

    .url,
    .message,
    .configuration dt,
    .counters {
        color: var(--color-muted);
    }

    .url {
        overflow-wrap: anywhere;
    }

    .summary {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 16px;
    }

    .card {
        padding: 24px;
        border: 1px solid var(--color-border);
        border-radius: 12px;
        background: var(--color-surface);
    }

    .card h2 {
        margin: 0 0 18px;
        font-size: 16px;
    }

    .card strong {
        color: var(--color-up);
    }

    .card strong.paused {
        color: var(--color-muted);
    }

    .counters {
        display: flex;
        gap: 20px;
        flex-wrap: wrap;
        font-size: 13px;
    }

    .configuration {
        display: grid;
        gap: 14px;
        margin: 0;
    }

    .configuration div {
        display: flex;
        justify-content: space-between;
        gap: 16px;
    }

    .configuration dd {
        margin: 0;
    }

    .error {
        color: var(--color-down);
    }

    .history-scroll {
        overflow-x: auto;
    }

    table {
        width: 100%;
        min-width: 650px;
        border-collapse: collapse;
        text-align: left;
    }

    th,
    td {
        padding: 14px 16px;
        border-bottom: 1px solid var(--color-border);
        font-size: 13px;
    }

    th {
        color: var(--color-muted);
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
    }

    td {
        vertical-align: top;
        font-variant-numeric: tabular-nums;
    }

    .error-detail {
        max-width: 320px;
        overflow-wrap: anywhere;
        color: var(--color-muted);
    }

    .message {
        color: var(--color-muted);
    }

    .refresh-warning {
        color: var(--color-warning);
        font-size: 12px;
    }

    .incident-status {
        color: var(--color-up);
        font-weight: 600;
    }

    .incident-status.open {
        color: var(--color-down);
    }

    @media (max-width: 600px) {
        .summary {
            grid-template-columns: 1fr;
        }
    }
</style>

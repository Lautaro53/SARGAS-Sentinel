<script lang="ts">
    import StatusBadge from "../components/StatusBadge.svelte";
    import Paginacion from "../components/Paginacion.svelte";
    import DropdownMenu from "../components/ui/DropdownMenu.svelte";
    import { ArrowLeft, Globe, Activity, AlertTriangle, SlidersHorizontal, ArrowUpDown } from "@lucide/svelte";
    import GraficoLatencia from "../components/GraficoLatencia.svelte";
    import { obtenerEstadoServicio } from "../api/servicios";

    import type { Servicio, EstadoServicio } from "../types/servicio";

    import type { Comprobacion } from "../types/comprobacion";
    import { obtenerComprobaciones } from "../api/comprobaciones";

    import type { Incidente } from "../types/incidente";
    import {
        consultarHistorialComprobaciones,
        consultarHistorialIncidentes,
        type CodigoHTTP,
        type OrdenComprobaciones,
        type EstadoIncidentes,
        type OrdenIncidentes,
    } from "../api/historial";

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

    const TAMANO_PAGINA = 10;
    let paginaComprobaciones = $state(1);
    let paginaIncidentes = $state(1);
    let historialComprobaciones = $state<Comprobacion[]>([]);
    let cargandoHistorial = $state(true);
    let errorHistorial = $state("");
    let totalComprobaciones = $state(0);
    let totalIncidentes = $state(0);
    const paginasComprobaciones = $derived(Math.max(1, Math.ceil(totalComprobaciones / TAMANO_PAGINA)));
    const paginasIncidentes = $derived(Math.max(1, Math.ceil(totalIncidentes / TAMANO_PAGINA)));

    let filtroCodigo = $state<CodigoHTTP>("todos");
    let ordenComprobaciones = $state<OrdenComprobaciones>("fecha_desc");
    let latenciaMin = $state("");
    let latenciaMax = $state("");
    let filtroIncidentes = $state<EstadoIncidentes>("todos");
    let ordenIncidentes = $state<OrdenIncidentes>("fecha_desc");
    let desdeIncidentes = $state("");
    let hastaIncidentes = $state("");

    const opcionesHTTP: { value: CodigoHTTP; label: string }[] = [
        { value: "todos", label: "Todos" },
        { value: "fallidas", label: "Fallidas" },
        { value: "2xx", label: "2xx" },
        { value: "3xx", label: "3xx" },
        { value: "4xx", label: "4xx" },
        { value: "5xx", label: "5xx" },
        { value: "sin_respuesta", label: "Sin respuesta" },
    ];
    const opcionesOrdenComprobaciones: { value: OrdenComprobaciones; label: string }[] = [
        { value: "fecha_desc", label: "Más recientes" },
        { value: "fecha_asc", label: "Más antiguas" },
        { value: "latencia_desc", label: "Mayor latencia" },
        { value: "latencia_asc", label: "Menor latencia" },
    ];
    const opcionesEstadoIncidentes: { value: EstadoIncidentes; label: string }[] = [
        { value: "todos", label: "Todos" },
        { value: "abiertos", label: "Abiertos" },
        { value: "cerrados", label: "Cerrados" },
    ];
    const opcionesOrdenIncidentes: { value: OrdenIncidentes; label: string }[] = [
        { value: "fecha_desc", label: "Más recientes" },
        { value: "fecha_asc", label: "Más antiguos" },
    ];

    $effect(() => {
        const id = servicio.id;
        const pagina = paginaComprobaciones;
        const codigo = filtroCodigo;
        const orden = ordenComprobaciones;
        const minimo = latenciaMin === "" ? null : Number(latenciaMin);
        const maximo = latenciaMax === "" ? null : Number(latenciaMax);
        const controlador = new AbortController();
        let vigente = true;
        cargandoHistorial = true;
        errorHistorial = "";

        void consultarHistorialComprobaciones(id, pagina, TAMANO_PAGINA,
            { codigo, orden, latenciaMin: minimo, latenciaMax: maximo }, controlador.signal,
        ).then((resultado) => {
            if (!vigente) return;
            historialComprobaciones = resultado.registros;
            totalComprobaciones = resultado.total;
            const ultimaPagina = Math.max(1, Math.ceil(resultado.total / TAMANO_PAGINA));
            if (pagina > ultimaPagina) paginaComprobaciones = ultimaPagina;
        }).catch((e: unknown) => {
            if (!vigente || controlador.signal.aborted) return;
            errorHistorial = e instanceof Error ? e.message : "No se pudo cargar el historial";
        }).finally(() => {
            if (vigente) cargandoHistorial = false;
        });
        return () => { vigente = false; controlador.abort(); };
    });

    $effect(() => {
        const id = servicio.id;
        const pagina = paginaIncidentes;
        const estadoFiltro = filtroIncidentes;
        const orden = ordenIncidentes;
        const desde = desdeIncidentes;
        const hasta = hastaIncidentes;
        const controlador = new AbortController();
        let vigente = true;
        cargandoIncidentes = true;
        errorIncidentes = "";

        void consultarHistorialIncidentes(id, pagina, TAMANO_PAGINA,
            { estado: estadoFiltro, orden, desde, hasta }, controlador.signal,
        ).then((resultado) => {
            if (!vigente) return;
            incidentes = resultado.registros;
            totalIncidentes = resultado.total;
            const ultimaPagina = Math.max(1, Math.ceil(resultado.total / TAMANO_PAGINA));
            if (pagina > ultimaPagina) paginaIncidentes = ultimaPagina;
        }).catch((e: unknown) => {
            if (!vigente || controlador.signal.aborted) return;
            errorIncidentes = e instanceof Error ? e.message : "No se pudo cargar el historial de incidentes";
        }).finally(() => {
            if (vigente) cargandoIncidentes = false;
        });
        return () => { vigente = false; controlador.abort(); };
    });

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
        const intervalo = window.setInterval(() => void actualizar(), 5_000);
        const volverAVer = () => { if (!document.hidden) void actualizar(); };
        document.addEventListener("visibilitychange", volverAVer);

        return () => {
            vigente = false;
            controlador.abort();
            window.clearInterval(intervalo);
            document.removeEventListener("visibilitychange", volverAVer);
        };
    });

    // El estado operativo se actualiza sin retirar los datos ya mostrados.
    $effect(() => {
        const id = servicio.id;
        const controlador = new AbortController();
        let vigente = true;
        let consultando = false;
        async function actualizarEstado() {
            if (consultando || document.hidden) return;
            consultando = true;
            try {
                const resultado = await obtenerEstadoServicio(id, controlador.signal);
                if (vigente) { estado = resultado; error = ""; }
            } catch (e) {
                if (vigente && !controlador.signal.aborted) {
                    error = e instanceof Error ? e.message : "No se pudo consultar el estado";
                }
            } finally {
                consultando = false;
                if (vigente) cargando = false;
            }
        }
        void actualizarEstado();
        const intervaloEstado = window.setInterval(() => void actualizarEstado(), 5_000);
        const reanudar = () => { if (!document.hidden) void actualizarEstado(); };
        document.addEventListener("visibilitychange", reanudar);
        return () => {
            vigente = false;
            controlador.abort();
            window.clearInterval(intervaloEstado);
            document.removeEventListener("visibilitychange", reanudar);
        };
    });
</script>

<div class="detail">
    <button class="back-button" type="button" onclick={onVolver}>
        <ArrowLeft size={15} /> Volver a servicios
    </button>

    <header class="detail-header">
        <div class="identity-icon"><Globe size={22} /></div>
        <div class="identity-copy">
            <p class="eyebrow">DETALLE DEL SERVICIO</p>
            <h1>{servicio.nombre}</h1>
            <p class="url">{servicio.url}</p>
        </div>
        {#if estado}
            <StatusBadge estado={estado.estado} />
        {/if}
    </header>

    <div class="overview">
        <section class="card overview-card" aria-labelledby="status-title">
            <div class="card-heading">
                <h2 id="status-title">Estado y monitoreo</h2>
                {#if estado}
                    <span class="monitor-label" class:paused={!estado.habilitado}>
                        <span class="monitor-dot"></span>
                        {estado.habilitado ? "Activo" : "Pausado"}
                    </span>
                {/if}
            </div>
            {#if cargando}
                <p class="message">Consultando estado...</p>
            {:else if error && !estado}
                <p class="error" role="alert">{error}</p>
            {:else if estado}
                {#if error}<p class="refresh-warning" role="status">{error}</p>{/if}
                <div class="status-details">
                    <div>
                        <span class="datum-label">Última comprobación</span>
                        <strong>{estado.ultima_comprobacion_en
                            ? new Date(estado.ultima_comprobacion_en).toLocaleString("es-UY")
                            : "Sin comprobaciones"}</strong>
                    </div>
                    <div class="streaks">
                        <span><strong class="success">{estado.exitos_consecutivos}</strong> éxitos seguidos</span>
                        <span><strong class="failure">{estado.fallos_consecutivos}</strong> fallos seguidos</span>
                    </div>
                </div>
            {/if}
        </section>

        <section class="card overview-card" aria-labelledby="config-title">
            <div class="card-heading"><h2 id="config-title">Configuración</h2></div>
            <dl class="configuration">
                <div><dt>Tipo</dt><dd>{servicio.tipo}</dd></div>
                <div><dt>Intervalo</dt><dd>{servicio.intervalo_segundos} s</dd></div>
                <div><dt>Timeout</dt><dd>{servicio.timeout_segundos} s</dd></div>
            </dl>
        </section>
    </div>

    <section class="card graph-card" aria-label="Evolución de latencia">
        {#if cargandoComprobaciones && comprobaciones.length === 0}
            <p class="message">Cargando latencias...</p>
        {:else if errorComprobaciones && comprobaciones.length === 0}
            <p class="error" role="alert">{errorComprobaciones}</p>
        {:else if comprobaciones.length === 0}
            <p class="message">Todavía no hay comprobaciones para graficar.</p>
        {:else}
            {#if errorComprobaciones}
                <p class="refresh-warning" role="status">
                    No se pudo actualizar la gráfica. Se muestran los últimos datos recibidos.
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

    {#snippet iconoFiltro()}
        <SlidersHorizontal size={17} aria-hidden="true" />
    {/snippet}
    {#snippet iconoOrden()}
        <ArrowUpDown size={17} aria-hidden="true" />
    {/snippet}

    <section class="data-table-panel history-panel" aria-labelledby="history-title">
        <div class="panel-header">
            <div class="history-heading">
                <Activity size={17} />
                <h2 id="history-title">Historial de comprobaciones</h2>
            </div>
        </div>
        <div class="history-controls" aria-label="Filtros de comprobaciones">
            <div class="filter-field">
                <span class="filter-label">HTTP</span>
                <DropdownMenu
                    label="Filtrar por HTTP"
                    value={filtroCodigo}
                    options={opcionesHTTP}
                    icon={iconoFiltro}
                    menuAlign="start"
                    onChange={(valor) => { filtroCodigo = valor; paginaComprobaciones = 1; }}
                />
            </div>
            <div class="filter-field">
                <span class="filter-label">Orden</span>
                <DropdownMenu
                    label="Ordenar comprobaciones"
                    value={ordenComprobaciones}
                    options={opcionesOrdenComprobaciones}
                    icon={iconoOrden}
                    triggerText="Ordenar"
                    menuAlign="start"
                    onChange={(valor) => { ordenComprobaciones = valor; paginaComprobaciones = 1; }}
                />
            </div>
            <label class="filter-field">
                <span class="filter-label">Latencia mín. (ms)</span>
                <input aria-label="Latencia mínima en ms" type="number" min="0" step="any" placeholder="Sin mínimo"
                    value={latenciaMin} onchange={(e) => { latenciaMin = e.currentTarget.value; paginaComprobaciones = 1; }} />
            </label>
            <label class="filter-field">
                <span class="filter-label">Latencia máx. (ms)</span>
                <input aria-label="Latencia máxima en ms" type="number" min="0" step="any" placeholder="Sin máximo"
                    value={latenciaMax} onchange={(e) => { latenciaMax = e.currentTarget.value; paginaComprobaciones = 1; }} />
            </label>
            {#if cargandoHistorial && historialComprobaciones.length > 0}
                <span class="updating" role="status">Actualizando…</span>
            {/if}
        </div>
        {#if errorHistorial}
            <p class="panel-warning" role="alert">{errorHistorial}</p>
        {/if}
        {#if cargandoHistorial && historialComprobaciones.length === 0 && !errorHistorial}
            <p class="panel-message">Cargando comprobaciones...</p>
        {:else if historialComprobaciones.length === 0}
            <p class="panel-message">{errorHistorial ? "No se pudieron obtener los registros." : "No hay comprobaciones que coincidan con la consulta."}</p>
        {:else}
            <div class="table-scroll" class:history-page-stable={totalComprobaciones > TAMANO_PAGINA}>
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
                        {#each historialComprobaciones as comprobacion (comprobacion.id)}
                            <tr>
                                <td>{new Date(comprobacion.comprobado_en).toLocaleString("es-UY")}</td>
                                <td><span class="http-code" class:failure={comprobacion.codigo_http === null || comprobacion.codigo_http < 200 || comprobacion.codigo_http >= 300}>
                                    {comprobacion.codigo_http ?? "—"}
                                </span></td>
                                <td class="numeric">{comprobacion.latencia_ms.toFixed(2)} ms</td>
                                <td class="error-detail" title={comprobacion.error ?? ""}>{comprobacion.error ?? "—"}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
        {#if totalComprobaciones > 0}
            <Paginacion
                pagina={paginaComprobaciones}
                totalPaginas={paginasComprobaciones}
                deshabilitado={cargandoHistorial}
                etiqueta="comprobaciones"
                onCambiarPagina={(pagina) => (paginaComprobaciones = pagina)}
            />
        {/if}
    </section>

    <section class="data-table-panel history-panel" aria-labelledby="incidents-title">
        <div class="panel-header">
            <div class="history-heading incident-icon">
                <AlertTriangle size={17} />
                <h2 id="incidents-title">Historial de incidentes</h2>
            </div>
        </div>
        <div class="history-controls" aria-label="Filtros de incidentes">
            <div class="filter-field">
                <span class="filter-label">Estado</span>
                <DropdownMenu
                    label="Filtrar incidentes por estado"
                    value={filtroIncidentes}
                    options={opcionesEstadoIncidentes}
                    icon={iconoFiltro}
                    menuAlign="start"
                    onChange={(valor) => { filtroIncidentes = valor; paginaIncidentes = 1; }}
                />
            </div>
            <div class="filter-field">
                <span class="filter-label">Orden</span>
                <DropdownMenu
                    label="Ordenar incidentes"
                    value={ordenIncidentes}
                    options={opcionesOrdenIncidentes}
                    icon={iconoOrden}
                    triggerText="Ordenar"
                    menuAlign="start"
                    onChange={(valor) => { ordenIncidentes = valor; paginaIncidentes = 1; }}
                />
            </div>
            <label class="filter-field">
                <span class="filter-label">Desde</span>
                <input type="date" aria-label="Incidentes desde" value={desdeIncidentes}
                    onchange={(e) => { desdeIncidentes = e.currentTarget.value; paginaIncidentes = 1; }} />
            </label>
            <label class="filter-field">
                <span class="filter-label">Hasta</span>
                <input type="date" aria-label="Incidentes hasta" value={hastaIncidentes}
                    onchange={(e) => { hastaIncidentes = e.currentTarget.value; paginaIncidentes = 1; }} />
            </label>
            {#if cargandoIncidentes && incidentes.length > 0}
                <span class="updating" role="status">Actualizando…</span>
            {/if}
        </div>
        {#if errorIncidentes}
            <p class="panel-warning" role="alert">{errorIncidentes}</p>
        {/if}
        {#if cargandoIncidentes && incidentes.length === 0 && !errorIncidentes}
            <p class="panel-message">Cargando incidentes...</p>
        {:else if incidentes.length === 0}
            <p class="panel-message">{errorIncidentes ? "No se pudieron obtener los registros." : "No hay incidentes que coincidan con la consulta."}</p>
        {:else}
            <div class="table-scroll" class:history-page-stable={totalIncidentes > TAMANO_PAGINA}>
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
                                <td>{new Date(incidente.iniciado_en).toLocaleString("es-UY")}</td>
                                <td>{incidente.cerrado_en
                                    ? new Date(incidente.cerrado_en).toLocaleString("es-UY")
                                    : "—"}</td>
                                <td><span class="incident-status" class:open={incidente.cerrado_en === null}>
                                    {incidente.cerrado_en === null ? "Abierto" : "Cerrado"}
                                </span></td>
                                <td class="error-detail" title={incidente.evidencia_inicio}>{incidente.evidencia_inicio}</td>
                                <td class="error-detail" title={incidente.evidencia_cierre ?? ""}>{incidente.evidencia_cierre ?? "—"}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
        {#if totalIncidentes > 0}
            <Paginacion
                pagina={paginaIncidentes}
                totalPaginas={paginasIncidentes}
                deshabilitado={cargandoIncidentes}
                etiqueta="incidentes"
                onCambiarPagina={(pagina) => (paginaIncidentes = pagina)}
            />
        {/if}
    </section>
</div>

<style>
    .detail { display: grid; gap: 16px; min-width: 0; }
    .back-button { display: inline-flex; align-items: center; gap: 7px; justify-self: start; padding: 0; border: 0; background: transparent; color: var(--color-primary-hover); font: inherit; font-size: 13px; cursor: pointer; }
    .back-button:hover { text-decoration: underline; }
    .back-button:focus-visible { outline: 2px solid var(--color-focus); outline-offset: 4px; }
    .detail-header { display: flex; align-items: center; gap: 13px; min-width: 0; }
    .identity-icon { display: grid; place-items: center; flex: none; width: 44px; height: 44px; border: 1px solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface-elevated); color: var(--color-primary-hover); }
    .identity-copy { flex: 1; min-width: 0; }
    .eyebrow { margin: 0 0 4px; color: var(--color-primary-hover); font-size: 10px; letter-spacing: 1px; }
    h1 { margin: 0; font-size: clamp(18px, 2vw, 23px); line-height: 1.25; }
    .url { margin: 3px 0 0; color: var(--color-muted); font-size: 12px; overflow-wrap: anywhere; }
    .overview { display: grid; grid-template-columns: minmax(0, 1.15fr) minmax(0, 1fr); gap: 12px; }
    .card { min-width: 0; padding: 16px 18px; border: 1px solid var(--color-border); border-radius: var(--radius-card); background: var(--color-surface); }
    .card-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 14px; }
    .card h2 { margin: 0; font-size: 14px; font-weight: 600; }
    .overview-card { min-height: 115px; }
    .monitor-label { display: inline-flex; align-items: center; gap: 6px; color: var(--color-up); font-size: 12px; font-weight: 600; white-space: nowrap; }
    .monitor-label.paused { color: var(--color-muted); }
    .monitor-dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
    .status-details { display: grid; gap: 12px; }
    .status-details > div:first-child { display: flex; align-items: baseline; flex-wrap: wrap; gap: 7px; }
    .datum-label { color: var(--color-muted); font-size: 12px; }
    .status-details strong { font-size: 12px; font-weight: 600; font-variant-numeric: tabular-nums; }
    .streaks { display: flex; align-items: center; flex-wrap: wrap; gap: 16px; color: var(--color-muted); font-size: 12px; }
    .streaks .success { color: var(--color-up); }
    .streaks .failure { color: var(--color-down); }
    .configuration { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; margin: 0; }
    .configuration dt { color: var(--color-muted); font-size: 11px; }
    .configuration dd { margin: 6px 0 0; font-size: 14px; font-weight: 600; font-variant-numeric: tabular-nums; }
    .graph-card { padding: 18px; }
    .history-heading { display: flex; align-items: center; gap: 8px; min-width: 0; color: var(--color-primary-hover); }
    .history-heading h2 { color: var(--color-text); }
    .history-heading.incident-icon { color: var(--color-down); }
    .panel-message { margin: 0; padding: 18px 16px; color: var(--color-muted); font-size: 13px; }
    .error { color: var(--color-down); }
    .message { margin: 0; color: var(--color-muted); font-size: 13px; }
    .refresh-warning { margin: 0 0 10px; color: var(--color-warning); font-size: 12px; }
    .numeric { font-variant-numeric: tabular-nums; white-space: nowrap; }
    .http-code { font-weight: 600; font-variant-numeric: tabular-nums; color: var(--color-up); }
    .http-code.failure { color: var(--color-down); }
    .error-detail { max-width: 260px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--color-muted); }
    .incident-status { font-weight: 600; color: var(--color-up); }
    .incident-status.open { color: var(--color-down); }
    @media (max-width: 720px) { .overview { grid-template-columns: 1fr; } .detail-header { flex-wrap: wrap; } }
    @media (max-width: 480px) { .card { padding: 14px; } .configuration { gap: 6px; } .overview-card { min-height: 0; } }

    .history-panel { position: relative; z-index: 1; overflow: visible; }
    .history-panel:focus-within { z-index: 20; }
    .history-controls { display: flex; align-items: flex-end; flex-wrap: wrap; gap: 10px; padding: 10px 16px 14px; border-top: 1px solid var(--color-border); }
    .filter-field { display: grid; gap: 5px; width: 150px; min-width: 0; }
    .filter-label { color: var(--color-muted); font-size: 11px; }
    .filter-field :global(.dropdown), .filter-field :global(.dropdown .trigger) { width: 100%; }
    .filter-field :global(.dropdown .trigger) { min-height: 40px; font-size: 13px; }
    .filter-field input { box-sizing: border-box; width: 100%; min-width: 0; min-height: 40px; padding: 6px 10px; border: 1px solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-text); color-scheme: inherit; font: inherit; font-size: 12px; }
    .filter-field input:focus-visible { outline: none; border-color: var(--color-primary-hover); box-shadow: inset 0 0 0 1px var(--color-primary-hover); }
    .filter-field input[type="number"] { appearance: textfield; -moz-appearance: textfield; }
    .filter-field input[type="number"]::-webkit-inner-spin-button,
    .filter-field input[type="number"]::-webkit-outer-spin-button { -webkit-appearance: none; margin: 0; }
    .table-scroll.history-page-stable { min-height: 436px; }
    .updating { margin-left: auto; color: var(--color-muted); font-size: 11px; }
    .panel-warning { margin: 0; padding: 8px 16px; color: var(--color-down); font-size: 12px; }
</style>

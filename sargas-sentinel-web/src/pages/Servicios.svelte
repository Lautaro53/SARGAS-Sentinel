<script lang="ts">
    import { onMount } from "svelte";

    import type { Servicio } from "../types/servicio";
    import {
        obtenerServicios,
        cambiarHabilitacion,
        archivarServicio,
    } from "../api/servicios";

    import ServicioForm from "../components/ServicioForm.svelte";

    import {
        Search,
        SlidersHorizontal,
        ArrowUpDown,
        ChevronDown,
        Plus,
        Check,
        SquarePen,
        Pause,
        Play,
        Archive,
        LoaderCircle,
    } from "@lucide/svelte";

    import Paginacion from "../components/Paginacion.svelte";
    import ServicioIdentidad from "../components/ServicioIdentidad.svelte";

    import { tick } from "svelte";

    let servicios = $state<Servicio[]>([]);
    let cargando = $state(true);
    let mensajeError = $state("");

    let mostrandoFormulario = $state(false);
    let confirmacion = $state("");
    let servicioProcesando = $state<number | null>(null);
    let errorAccion = $state("");

    let servicioEnEdicion = $state<Servicio | null>(null);

    type FiltroMonitoreo = "todos" | "activos" | "pausados";

    let busqueda = $state("");
    let filtroMonitoreo = $state<FiltroMonitoreo>("todos");

    let serviciosFiltrados = $derived.by(() => {
        const termino = busqueda.trim().toLocaleLowerCase("es-UY");

        return servicios.filter((servicio) => {
            const coincideBusqueda =
                termino === "" ||
                servicio.nombre.toLocaleLowerCase("es-UY").includes(termino) ||
                servicio.url.toLocaleLowerCase("es-UY").includes(termino);

            const coincideMonitoreo =
                filtroMonitoreo === "todos" ||
                (filtroMonitoreo === "activos" && servicio.habilitado) ||
                (filtroMonitoreo === "pausados" && !servicio.habilitado);

            return coincideBusqueda && coincideMonitoreo;
        });
    });

    type OrdenServicios =
        | "nombre-asc"
        | "nombre-desc"
        | "activos"
        | "pausados"
        | "intervalo-asc"
        | "intervalo-desc"
        | "timeout-asc"
        | "timeout-desc";

    let ordenServicios = $state<OrdenServicios>("nombre-asc");
    let menuOrdenAbierto = $state(false);

    let menuFiltroAbierto = $state(false);

    let contenedorFiltro: HTMLDivElement;
    let contenedorOrden: HTMLDivElement;

    const opcionesFiltro: {
        valor: FiltroMonitoreo;
        etiqueta: string;
    }[] = [
        { valor: "todos", etiqueta: "Todos" },
        { valor: "activos", etiqueta: "Activos" },
        { valor: "pausados", etiqueta: "Pausados" },
    ];

    let etiquetaFiltro = $derived(
        opcionesFiltro.find((opcion) => opcion.valor === filtroMonitoreo)
            ?.etiqueta ?? "Todos",
    );

    function seleccionarFiltro(valor: FiltroMonitoreo) {
        filtroMonitoreo = valor;
        menuFiltroAbierto = false;
        paginaServicios = 1;
    }

    function alternarFiltro() {
        menuFiltroAbierto = !menuFiltroAbierto;
        menuOrdenAbierto = false;
    }

    function alternarOrden() {
        menuOrdenAbierto = !menuOrdenAbierto;
        menuFiltroAbierto = false;
    }

    function manejarClickExterno(evento: MouseEvent) {
        const destino = evento.target;

        if (!(destino instanceof Node)) return;

        if (menuFiltroAbierto && !contenedorFiltro?.contains(destino)) {
            menuFiltroAbierto = false;
        }

        if (menuOrdenAbierto && !contenedorOrden?.contains(destino)) {
            menuOrdenAbierto = false;
        }
    }

    function manejarEscape(evento: KeyboardEvent) {
        if (evento.key === "Escape") {
            menuFiltroAbierto = false;
            menuOrdenAbierto = false;
        }
    }

    const opcionesOrden: { valor: OrdenServicios; etiqueta: string }[] = [
        { valor: "nombre-asc", etiqueta: "Nombre A–Z" },
        { valor: "nombre-desc", etiqueta: "Nombre Z–A" },
        { valor: "activos", etiqueta: "Activos primero" },
        { valor: "pausados", etiqueta: "Pausados primero" },
        { valor: "intervalo-asc", etiqueta: "Intervalo menor" },
        { valor: "intervalo-desc", etiqueta: "Intervalo mayor" },
        { valor: "timeout-asc", etiqueta: "Timeout menor" },
        { valor: "timeout-desc", etiqueta: "Timeout mayor" },
    ];

    let serviciosOrdenados = $derived.by(() => {
        const resultado = [...serviciosFiltrados];

        const porNombre = (a: Servicio, b: Servicio) =>
            a.nombre.localeCompare(b.nombre, "es", {
                sensitivity: "base",
                numeric: true,
            });

        resultado.sort((a, b) => {
            switch (ordenServicios) {
                case "nombre-desc":
                    return -porNombre(a, b);
                case "activos":
                    return (
                        Number(b.habilitado) - Number(a.habilitado) ||
                        porNombre(a, b)
                    );
                case "pausados":
                    return (
                        Number(a.habilitado) - Number(b.habilitado) ||
                        porNombre(a, b)
                    );
                case "intervalo-asc":
                    return (
                        a.intervalo_segundos - b.intervalo_segundos ||
                        porNombre(a, b)
                    );
                case "intervalo-desc":
                    return (
                        b.intervalo_segundos - a.intervalo_segundos ||
                        porNombre(a, b)
                    );
                case "timeout-asc":
                    return (
                        a.timeout_segundos - b.timeout_segundos ||
                        porNombre(a, b)
                    );
                case "timeout-desc":
                    return (
                        b.timeout_segundos - a.timeout_segundos ||
                        porNombre(a, b)
                    );
                default:
                    return porNombre(a, b);
            }
        });

        return resultado;
    });

    const SERVICIOS_POR_PAGINA = 10;

    let paginaServicios = $state(1);

    let totalPaginasServicios = $derived(
        Math.max(
            1,
            Math.ceil(serviciosOrdenados.length / SERVICIOS_POR_PAGINA),
        ),
    );

    let paginaActualServicios = $derived(
        Math.min(paginaServicios, totalPaginasServicios),
    );

    let serviciosPaginados = $derived(
        serviciosOrdenados.slice(
            (paginaActualServicios - 1) * SERVICIOS_POR_PAGINA,
            paginaActualServicios * SERVICIOS_POR_PAGINA,
        ),
    );

    function cambiarPaginaServicios(destino: number) {
        if (destino < 1 || destino > totalPaginasServicios) return;

        paginaServicios = destino;
    }

    function seleccionarOrden(valor: OrdenServicios) {
        ordenServicios = valor;
        menuOrdenAbierto = false;
        paginaServicios = 1;
    }

    let hayFiltros = $derived(
        busqueda.trim() !== "" || filtroMonitoreo !== "todos",
    );

    function limpiarFiltros() {
        busqueda = "";
        filtroMonitoreo = "todos";
        paginaServicios = 1;
    }

    type Props = {
        onVerDetalle: (servicio: Servicio) => void;
    };

    let { onVerDetalle }: Props = $props();

    function manejarEnlaceDetalle(event: MouseEvent, servicio: Servicio) {
        if (
            event.defaultPrevented ||
            event.button !== 0 ||
            event.ctrlKey ||
            event.metaKey ||
            event.shiftKey ||
            event.altKey
        ) {
            return;
        }

        event.preventDefault();
        onVerDetalle(servicio);
    }

    onMount(() => {
        let activo = true;

        async function cargarServicios() {
            try {
                const datos = await obtenerServicios();

                if (activo) {
                    servicios = datos;
                }
            } catch (error) {
                if (activo) {
                    mensajeError =
                        error instanceof Error
                            ? error.message
                            : "No se pudieron cargar los servicios";
                }
            } finally {
                if (activo) {
                    cargando = false;
                }
            }
        }

        void cargarServicios();

        return () => {
            activo = false;
        };
    });

    function cerrarFormulario() {
        mostrandoFormulario = false;
        servicioEnEdicion = null;
    }

    async function enfocarFormulario() {
        await tick();

        const formulario = document.getElementById("formulario-servicio");
        const nombre = document.getElementById("nombre-servicio");

        formulario?.scrollIntoView({
            behavior: "smooth",
            block: "start",
        });

        if (nombre instanceof HTMLInputElement) {
            nombre.focus({ preventScroll: true });
        }
    }

    function iniciarCreacion() {
        servicioEnEdicion = null;
        mostrandoFormulario = true;
        confirmacion = "";
        void enfocarFormulario();
    }

    function iniciarEdicion(servicio: Servicio) {
        servicioEnEdicion = servicio;
        mostrandoFormulario = true;
        confirmacion = "";
        void enfocarFormulario();
    }

    function actualizarEnLista(actualizado: Servicio) {
        servicios = servicios.map((actual) =>
            actual.id === actualizado.id ? actualizado : actual,
        );

        cerrarFormulario();
        mensajeError = "";
        confirmacion = `Servicio "${actualizado.nombre}" actualizado correctamente.`;
    }

    async function alternarMonitoreo(servicio: Servicio) {
        if (servicioProcesando !== null) return;

        const nuevoEstado = !servicio.habilitado;

        servicioProcesando = servicio.id;
        errorAccion = "";
        confirmacion = "";

        try {
            await cambiarHabilitacion(servicio.id, nuevoEstado);

            servicios = servicios.map((actual) =>
                actual.id === servicio.id
                    ? { ...actual, habilitado: nuevoEstado }
                    : actual,
            );

            confirmacion = nuevoEstado
                ? `Monitoreo reanudado: ${servicio.nombre}`
                : `Monitoreo pausado: ${servicio.nombre}`;
        } catch (error) {
            errorAccion =
                error instanceof Error
                    ? error.message
                    : "No se pudo cambiar el monitoreo";
        } finally {
            servicioProcesando = null;
        }
    }

    async function archivarDesdeLista(servicio: Servicio) {
        if (servicioProcesando !== null) return;

        const confirmado = window.confirm(
            `¿Archivar "${servicio.nombre}"?\n\n` +
                "Se detendrá su monitoreo y dejará de aparecer " +
                "en el listado habitual.\n\n" +
                "Sus comprobaciones e incidentes históricos " +
                "se conservarán.\n\n" +
                "Por ahora no existe una opción para desarchivarlo.",
        );

        if (!confirmado) return;

        servicioProcesando = servicio.id;
        errorAccion = "";
        confirmacion = "";

        try {
            await archivarServicio(servicio.id);

            servicios = servicios.filter((actual) => actual.id !== servicio.id);

            if (servicioEnEdicion?.id === servicio.id) {
                cerrarFormulario();
            }

            confirmacion = `Servicio "${servicio.nombre}" archivado correctamente.`;
        } catch (error) {
            errorAccion =
                error instanceof Error
                    ? error.message
                    : "No se pudo archivar el servicio";
        } finally {
            servicioProcesando = null;
        }
    }

    function registrarEnLista(nuevo: Servicio) {
        servicios = [...servicios, nuevo];

        mostrandoFormulario = false;
        mensajeError = "";
        confirmacion = `Servicio "${nuevo.nombre}" registrado correctamente.`;
    }
</script>

<svelte:window onclick={manejarClickExterno} onkeydown={manejarEscape} />

<div class="page">
    <header class="services-heading"></header>

    {#if mostrandoFormulario}
        {#key servicioEnEdicion?.id ?? "nuevo"}
            <ServicioForm
                servicio={servicioEnEdicion}
                onCreado={registrarEnLista}
                onActualizado={actualizarEnLista}
                onCancelar={cerrarFormulario}
            />
        {/key}
    {/if}

    {#if confirmacion}
        <p class="confirmation" role="status">
            {confirmacion}
        </p>
    {/if}

    {#if errorAccion}
        <p class="action-error" role="alert">
            {errorAccion}
        </p>
    {/if}

    <div
        class="services-toolbar"
        role="search"
        aria-label="Buscar y organizar servicios"
    >
        <div class="search-field toolbar-control">
            <Search size={17} aria-hidden="true" />

            <label class="visually-hidden" for="buscar-servicios">
                Buscar servicios
            </label>

            <input
                id="buscar-servicios"
                type="search"
                placeholder="Buscar por nombre o URL..."
                autocomplete="off"
                bind:value={busqueda}
                oninput={() => (paginaServicios = 1)}
            />
        </div>

        <div class="filter-wrapper" bind:this={contenedorFiltro}>
            <button
                type="button"
                class="filter-button toolbar-control"
                aria-label={`Filtrar servicios: ${etiquetaFiltro}`}
                aria-expanded={menuFiltroAbierto}
                aria-controls="opciones-filtro"
                onclick={alternarFiltro}
            >
                <SlidersHorizontal size={17} aria-hidden="true" />
                <span>{etiquetaFiltro}</span>
                <ChevronDown size={14} aria-hidden="true" />
            </button>

            {#if menuFiltroAbierto}
                <div id="opciones-filtro" class="filter-menu">
                    {#each opcionesFiltro as opcion}
                        <button
                            type="button"
                            class:seleccionado={filtroMonitoreo ===
                                opcion.valor}
                            aria-pressed={filtroMonitoreo === opcion.valor}
                            onclick={() => seleccionarFiltro(opcion.valor)}
                        >
                            <span>{opcion.etiqueta}</span>

                            {#if filtroMonitoreo === opcion.valor}
                                <Check size={15} aria-hidden="true" />
                            {/if}
                        </button>
                    {/each}

                    {#if hayFiltros}
                        <div class="filter-menu-divider"></div>

                        <button
                            type="button"
                            class="reset-filter-option"
                            onclick={() => {
                                limpiarFiltros();
                                menuFiltroAbierto = false;
                            }}
                        >
                            <span>Limpiar filtros</span>
                        </button>
                    {/if}
                </div>
            {/if}
        </div>

        <div class="order-wrapper" bind:this={contenedorOrden}>
            <button
                type="button"
                class="order-button toolbar-control"
                aria-expanded={menuOrdenAbierto}
                aria-controls="opciones-orden"
                onclick={alternarOrden}
                onkeydown={(evento) => {
                    if (evento.key === "Escape") {
                        menuOrdenAbierto = false;
                    }
                }}
            >
                <ArrowUpDown size={17} aria-hidden="true" />
                <span>Ordenar</span>
                <ChevronDown size={14} aria-hidden="true" />
            </button>

            {#if menuOrdenAbierto}
                <div id="opciones-orden" class="order-menu">
                    {#each opcionesOrden as opcion}
                        <button
                            type="button"
                            class:seleccionado={ordenServicios === opcion.valor}
                            aria-pressed={ordenServicios === opcion.valor}
                            onclick={() => seleccionarOrden(opcion.valor)}
                        >
                            <span>{opcion.etiqueta}</span>

                            {#if ordenServicios === opcion.valor}
                                <Check size={15} aria-hidden="true" />
                            {/if}
                        </button>
                    {/each}
                </div>
            {/if}
        </div>

        <button
            type="button"
            class="create-button"
            aria-expanded={mostrandoFormulario}
            aria-controls="formulario-servicio"
            onclick={() => {
                if (mostrandoFormulario) {
                    cerrarFormulario();
                } else {
                    iniciarCreacion();
                }
            }}
        >
            <Plus size={16} aria-hidden="true" />
            <span>Nuevo servicio</span>
        </button>
    </div>

    <section class="services-panel data-table-panel">
        <div class="panel-header">
            <h2>Servicios registrados</h2>
            <span>
                {#if hayFiltros}
                    {serviciosFiltrados.length} de {servicios.length} servicios
                {:else}
                    {servicios.length} servicios
                {/if}
            </span>
        </div>

        {#if cargando}
            <p class="message">Cargando servicios...</p>
        {:else if mensajeError}
            <p class="message error" role="alert">
                {mensajeError}
            </p>
        {:else if servicios.length === 0}
            <p class="message">Todavía no hay servicios registrados.</p>
        {:else if serviciosFiltrados.length === 0}
            <div class="empty-results">
                <p>No se encontraron servicios con esos criterios.</p>
                <button type="button" onclick={limpiarFiltros}>
                    Limpiar filtros
                </button>
            </div>
        {:else}
            <div class="table-scroll">
                <table>
                    <thead>
                        <tr>
                            <th scope="col">Servicio</th>
                            <th scope="col">Tipo</th>
                            <th scope="col">Intervalo</th>
                            <th scope="col">Timeout</th>
                            <th scope="col">Monitoreo</th>
                            <th scope="col">Acciones</th>
                        </tr>
                    </thead>

                    <tbody>
                        {#each serviciosPaginados as servicio (servicio.id)}
                            <tr>
                                <td>
                                    <ServicioIdentidad
                                        id={servicio.id}
                                        nombre={servicio.nombre}
                                        url={servicio.url}
                                        mostrarUrl={true}
                                        onVerDetalle={() =>
                                            onVerDetalle(servicio)}
                                    />
                                </td>

                                <td>{servicio.tipo}</td>

                                <td>{servicio.intervalo_segundos} s</td>

                                <td>{servicio.timeout_segundos} s</td>

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

                                <td>
                                    <div class="row-actions">
                                        <button
                                            type="button"
                                            class="action-icon-btn edit-action"
                                            title={`Editar ${servicio.nombre}`}
                                            aria-label={`Editar servicio ${servicio.nombre}`}
                                            onclick={() =>
                                                iniciarEdicion(servicio)}
                                        >
                                            <SquarePen
                                                size={16}
                                                aria-hidden="true"
                                            />
                                        </button>

                                        <button
                                            type="button"
                                            class="action-icon-btn toggle-action"
                                            disabled={servicioProcesando !==
                                                null}
                                            title={servicio.habilitado
                                                ? `Pausar monitoreo de ${servicio.nombre}`
                                                : `Reanudar monitoreo de ${servicio.nombre}`}
                                            aria-label={`${servicio.habilitado ? "Pausar" : "Reanudar"} el monitoreo de ${servicio.nombre}`}
                                            aria-busy={servicioProcesando ===
                                                servicio.id}
                                            onclick={() =>
                                                void alternarMonitoreo(
                                                    servicio,
                                                )}
                                        >
                                            {#if servicioProcesando === servicio.id}
                                                <LoaderCircle
                                                    size={16}
                                                    class="loading-icon"
                                                    aria-hidden="true"
                                                />
                                            {:else if servicio.habilitado}
                                                <Pause
                                                    size={16}
                                                    aria-hidden="true"
                                                />
                                            {:else}
                                                <Play
                                                    size={16}
                                                    aria-hidden="true"
                                                />
                                            {/if}
                                        </button>

                                        <button
                                            type="button"
                                            class="action-icon-btn archive-action"
                                            disabled={servicioProcesando !==
                                                null}
                                            title={`Archivar ${servicio.nombre}`}
                                            aria-label={`Archivar servicio ${servicio.nombre}`}
                                            aria-busy={servicioProcesando ===
                                                servicio.id}
                                            onclick={() =>
                                                void archivarDesdeLista(
                                                    servicio,
                                                )}
                                        >
                                            {#if servicioProcesando === servicio.id}
                                                <LoaderCircle
                                                    size={16}
                                                    class="loading-icon"
                                                    aria-hidden="true"
                                                />
                                            {:else}
                                                <Archive
                                                    size={16}
                                                    aria-hidden="true"
                                                />
                                            {/if}
                                        </button>
                                    </div>
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
    .services-panel {
        margin-top: 32px;
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        border-radius: 12px;
        overflow: hidden;
    }

    .panel-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 16px;
        padding: 24px;
    }

    .panel-header h2 {
        margin: 0;
        font-size: 18px;
    }

    .panel-header span {
        color: var(--color-muted);
        font-size: 12px;
    }

    .table-scroll {
        overflow-x: auto;
    }

    table {
        width: 100%;
        min-width: 680px;
        border-collapse: collapse;
        text-align: left;
    }

    th,
    td {
        padding: 16px 20px;
    }

    th {
        color: var(--color-muted);
        background: var(--color-surface-elevated);
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    td {
        border-top: 1px solid var(--color-border);
        font-size: 13px;
    }

    .monitor-status {
        color: var(--color-up);
        font-weight: 500;
    }

    .monitor-status.paused {
        color: var(--color-muted);
    }

    .message {
        margin: 0;
        padding: 24px;
        color: var(--color-muted);
    }

    .error {
        color: var(--color-down);
    }

    .create-button {
        padding: 11px 16px;
        border: 1px solid var(--color-primary);
        border-radius: 8px;
        background: var(--color-primary);
        color: white;
        font: inherit;
        cursor: pointer;
    }

    .create-button:hover {
        background: var(--color-primary-hover);
    }

    .create-button:focus-visible {
        outline: 2px solid var(--color-primary);
        outline-offset: 3px;
    }

    .confirmation {
        margin-top: 16px;
        color: var(--color-up);
        font-size: 13px;
    }

    .row-actions {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .action-error {
        margin: 16px 0;
        color: var(--color-down);
        font-size: 13px;
    }

    /* Cabecera compacta */
    .services-heading {
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-wrap: wrap;
        gap: 16px;
        margin-bottom: 20px;
    }

    /* Crear servicio */
    .create-button {
        padding: 9px 14px;
        font-size: 13px;
        white-space: nowrap;
    }

    /* Búsqueda y filtros */
    .services-toolbar {
        display: flex;
        align-items: end;
        flex-wrap: wrap;
        gap: 12px;
        margin-bottom: 16px;
    }

    .search-field {
        flex: 1 1 260px;
        min-width: 0;
    }

    /* Panel y tabla */
    .services-panel {
        margin-top: 0;
    }

    .panel-header {
        padding: 14px 16px;
    }

    .panel-header h2 {
        font-size: 16px;
    }

    th,
    td {
        padding: 11px 12px;
    }

    .row-actions {
        gap: 6px;
        flex-wrap: wrap;
    }

    .empty-results {
        padding: 24px 16px;
        text-align: center;
        color: var(--color-muted);
        font-size: 13px;
    }

    .empty-results button {
        padding: 7px 12px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface-elevated);
        color: var(--color-text);
        cursor: pointer;
    }

    /* Barra compacta de Servicios */

    .services-toolbar {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 10px;
        margin-bottom: 16px;
    }

    .visually-hidden {
        position: absolute;
        width: 1px;
        height: 1px;
        padding: 0;
        margin: -1px;
        overflow: hidden;
        clip: rect(0, 0, 0, 0);
        white-space: nowrap;
        border: 0;
    }

    .services-toolbar .toolbar-control {
        display: flex;
        align-items: center;
        gap: 10px;
        height: 40px;
        padding: 0 12px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface);
        color: var(--color-primary-hover);
    }

    .services-toolbar .search-field {
        flex: 1 1 240px;
        min-width: 0;
    }

    .order-wrapper {
        position: relative;
        flex: 0 0 auto;
    }

    .services-toolbar .order-button {
        cursor: pointer;
        font: inherit;
        font-size: 13px;
        color: var(--color-text);
        white-space: nowrap;
    }

    .order-menu {
        position: absolute;
        top: calc(100% + 6px);
        right: 0;
        z-index: 20;
        width: 210px;
        padding: 5px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface-elevated);
        box-shadow: var(--shadow-card);
    }

    .order-menu button {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        width: 100%;
        padding: 9px 10px;
        border: 0;
        border-radius: 6px;
        background: transparent;
        color: var(--color-text);
        text-align: left;
        font: inherit;
        font-size: 12px;
        cursor: pointer;
    }

    .order-menu button:hover,
    .order-menu button.seleccionado {
        background: var(--color-surface);
    }

    .order-menu button.seleccionado {
        color: var(--color-primary-hover);
    }

    .services-toolbar .create-button {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 7px;
        flex: 0 0 auto;
        min-height: 40px;
        margin: 0;
        padding: 0 14px;
        font-size: 13px;
        white-space: nowrap;
    }

    /* Filtro y ordenamiento unificados */

    .filter-wrapper,
    .order-wrapper {
        position: relative;
        flex: 0 0 auto;
    }

    .services-toolbar .filter-button,
    .services-toolbar .order-button {
        display: inline-flex;
        align-items: center;
        justify-content: space-between;
        gap: 10px;
        min-width: 125px;
        height: 40px;
        cursor: pointer;
        color: var(--color-text);
        font: inherit;
        font-size: 13px;
    }

    .filter-menu,
    .order-menu {
        position: absolute;
        top: calc(100% + 6px);
        right: 0;
        z-index: 30;
        min-width: 165px;
        padding: 5px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface-elevated);
        box-shadow: var(--shadow-card);
    }

    .filter-menu button,
    .order-menu button {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        width: 100%;
        padding: 9px 10px;
        border: 0;
        border-radius: 6px;
        background: transparent;
        color: var(--color-text);
        text-align: left;
        font: inherit;
        font-size: 12px;
        cursor: pointer;
    }

    .filter-menu button:hover,
    .order-menu button:hover,
    .filter-menu button.seleccionado,
    .order-menu button.seleccionado {
        background: var(--color-surface);
    }

    .filter-menu button.seleccionado,
    .order-menu button.seleccionado {
        color: var(--color-primary-hover);
    }

    .services-toolbar .filter-button:focus-visible,
    .services-toolbar .order-button:focus-visible,
    .filter-menu button:focus-visible,
    .order-menu button:focus-visible {
        outline: 2px solid var(--color-focus);
        outline-offset: 2px;
    }

    .services-toolbar .search-field input {
        flex: 1;
        width: 100%;
        min-width: 0;
        height: 100%;
        margin: 0;
        padding: 0;
        border: none;
        border-radius: 0;
        outline: none;
        box-shadow: none;
        background: transparent;
        color: var(--color-text);
        font: inherit;
        font-size: 13px;
        appearance: none;
    }

    .services-toolbar .search-field input::placeholder {
        color: var(--color-muted);
        opacity: 1;
    }

    .services-toolbar .search-field:focus-within {
        outline: none;
        border-color: var(--color-primary-hover);
        box-shadow: 0 0 0 1px rgba(0, 102, 255, 0.2);
    }

    .services-toolbar .search-field input::-webkit-search-cancel-button {
        cursor: pointer;
    }

    .filter-menu-divider {
        height: 1px;
        margin: 5px 4px;
        background: var(--color-border);
    }

    .filter-menu .reset-filter-option {
        color: var(--color-primary-hover);
    }

    .filter-menu .reset-filter-option:hover {
        background: var(--color-surface);
    }

    /* Acciones compactas de Servicios */

    .row-actions {
        display: flex;
        align-items: center;
        flex-wrap: nowrap;
        gap: 8px;
    }

    .row-actions .action-icon-btn {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex: 0 0 34px;
        width: 34px;
        height: 34px;
        padding: 0;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
        transition:
            border-color 150ms,
            background 150ms;
    }

    .row-actions .edit-action {
        color: var(--color-primary-hover);
    }

    .row-actions .archive-action {
        color: var(--color-down);
    }

    .row-actions .edit-action:hover:not(:disabled),
    .row-actions .toggle-action:hover:not(:disabled) {
        border-color: var(--color-primary);
        background: color-mix(in srgb, var(--color-primary) 12%, transparent);
    }

    .row-actions .archive-action:hover:not(:disabled) {
        border-color: var(--color-down);
        background: color-mix(in srgb, var(--color-down) 12%, transparent);
    }

    .row-actions .action-icon-btn:focus-visible {
        outline: 2px solid var(--color-focus);
        outline-offset: 2px;
    }

    .row-actions .action-icon-btn:disabled {
        opacity: 0.45;
        cursor: not-allowed;
    }

    @keyframes action-spin {
        to {
            transform: rotate(360deg);
        }
    }

    .row-actions :global(.loading-icon) {
        animation: action-spin 1s linear infinite;
    }

    @media (max-width: 760px) {
        .services-toolbar .search-field {
            flex-basis: 100%;
        }

        .services-toolbar .create-button {
            flex: 1 1 auto;
        }
    }

    @media (max-width: 600px) {
        .services-heading {
            align-items: flex-start;
        }

        .services-toolbar {
            gap: 10px;
        }
    }
</style>

<script lang="ts">
    import { onMount, tick } from "svelte";

    import type { Servicio } from "../types/servicio";
    import {
        obtenerServicios,
        cambiarHabilitacion,
        archivarServicio,
    } from "../api/servicios";

    import ServicioForm from "../components/ServicioForm.svelte";

    import {
        SlidersHorizontal,
        ArrowUpDown,
        Plus,
        SquarePen,
        Pause,
        Play,
        Archive,
    } from "@lucide/svelte";
    import Button from "../components/ui/Button.svelte";
    import DropdownMenu from "../components/ui/DropdownMenu.svelte";
    import SearchField from "../components/ui/SearchField.svelte";

    import Paginacion from "../components/Paginacion.svelte";
    import ServicioIdentidad from "../components/ServicioIdentidad.svelte";

    let servicios = $state<Servicio[]>([]);
    let cargando = $state(true);
    let mensajeError = $state("");

    let mostrandoFormulario = $state(false);
    let confirmacion = $state("");
    let servicioProcesando = $state<number | null>(null);
    let errorAccion = $state("");
    $effect(() => {
        if (!confirmacion) return;

        const temporizador = window.setTimeout(() => {
            confirmacion = "";
        }, 4000);

        return () => window.clearTimeout(temporizador);
    });

    $effect(() => {
        if (!errorAccion) return;

        const temporizador = window.setTimeout(() => {
            errorAccion = "";
        }, 6000);

        return () => window.clearTimeout(temporizador);
    });

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

    const CLAVE_FILTRO = "sargas.servicios.filtro";
    const CLAVE_ORDEN = "sargas.servicios.orden";

    function guardarPreferencia(clave: string, valor: string) {
        try {
            localStorage.setItem(clave, valor);
        } catch {
            // La pantalla sigue funcionando cuando el almacenamiento no está disponible.
        }
    }

    const opcionesFiltro: { value: FiltroMonitoreo; label: string }[] = [
        { value: "todos", label: "Todos" },
        { value: "activos", label: "Activos" },
        { value: "pausados", label: "Pausados" },
    ];

    function seleccionarFiltro(valor: FiltroMonitoreo) {
        filtroMonitoreo = valor;
        paginaServicios = 1;
        guardarPreferencia(CLAVE_FILTRO, valor);
    }

    const opcionesOrden: { value: OrdenServicios; label: string }[] = [
        { value: "nombre-asc", label: "Nombre A–Z" },
        { value: "nombre-desc", label: "Nombre Z–A" },
        { value: "activos", label: "Activos primero" },
        { value: "pausados", label: "Pausados primero" },
        { value: "intervalo-asc", label: "Intervalo menor" },
        { value: "intervalo-desc", label: "Intervalo mayor" },
        { value: "timeout-asc", label: "Timeout menor" },
        { value: "timeout-desc", label: "Timeout mayor" },
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
        paginaServicios = 1;
        guardarPreferencia(CLAVE_ORDEN, valor);
    }

    let hayFiltros = $derived(
        busqueda.trim() !== "" || filtroMonitoreo !== "todos",
    );

    type Props = {
        onVerDetalle: (servicio: Servicio) => void;
    };

    let { onVerDetalle }: Props = $props();

    onMount(() => {
        let activo = true;

        try {
            const filtroGuardado = localStorage.getItem(CLAVE_FILTRO);
            const ordenGuardado = localStorage.getItem(CLAVE_ORDEN);
            const filtroValido = opcionesFiltro.find((opcion) => opcion.value === filtroGuardado);
            const ordenValido = opcionesOrden.find((opcion) => opcion.value === ordenGuardado);
            if (filtroValido) filtroMonitoreo = filtroValido.value;
            if (ordenValido) ordenServicios = ordenValido.value;
        } catch {
            // Se utilizan los valores predeterminados si el navegador bloquea el almacenamiento.
        }

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

<div class="page">
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

    {#snippet filtroIcono()}
        <SlidersHorizontal size={17} aria-hidden="true" />
    {/snippet}

    {#snippet ordenIcono()}
        <ArrowUpDown size={17} aria-hidden="true" />
    {/snippet}

    <div
        class="services-toolbar"
        role="search"
        aria-label="Buscar y organizar servicios"
    >
        <div class="search-container">
            <SearchField
                id="buscar-servicios"
                label="Buscar servicios"
                placeholder="Buscar por nombre o URL..."
                value={busqueda}
                onChange={(valor) => {
                    busqueda = valor;
                    paginaServicios = 1;
                }}
            />
        </div>

        <div class="monitor-filter">
            <DropdownMenu
                label="Filtrar servicios"
                value={filtroMonitoreo}
                options={opcionesFiltro}
                onChange={seleccionarFiltro}
                icon={filtroIcono}
            />
        </div>

        <DropdownMenu
            label="Ordenar servicios"
            value={ordenServicios}
            options={opcionesOrden}
            onChange={seleccionarOrden}
            triggerText="Ordenar"
            icon={ordenIcono}
        />

        <div class="create-container">
            <Button
                variant="primary"
                ariaExpanded={mostrandoFormulario}
                ariaControls="formulario-servicio"
                onClick={() => {
                    if (mostrandoFormulario) {
                        cerrarFormulario();
                    } else {
                        iniciarCreacion();
                    }
                }}
            >
                <Plus size={16} aria-hidden="true" />
                Nuevo servicio
            </Button>
        </div>
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
                <p>Probá con otra búsqueda o seleccioná «Todos».</p>
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
                                        <Button
                                            variant="accent"
                                            size="sm"
                                            iconOnly
                                            title={`Editar ${servicio.nombre}`}
                                            ariaLabel={`Editar servicio ${servicio.nombre}`}
                                            onClick={() =>
                                                iniciarEdicion(servicio)}
                                        >
                                            <SquarePen
                                                size={16}
                                                aria-hidden="true"
                                            />
                                        </Button>

                                        <Button
                                            size="sm"
                                            iconOnly
                                            disabled={servicioProcesando !==
                                                null}
                                            loading={servicioProcesando ===
                                                servicio.id}
                                            title={servicio.habilitado
                                                ? `Pausar monitoreo de ${servicio.nombre}`
                                                : `Reanudar monitoreo de ${servicio.nombre}`}
                                            ariaLabel={`${servicio.habilitado ? "Pausar" : "Reanudar"} el monitoreo de ${servicio.nombre}`}
                                            onClick={() =>
                                                void alternarMonitoreo(
                                                    servicio,
                                                )}
                                        >
                                            {#if servicio.habilitado}
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
                                        </Button>

                                        <Button
                                            variant="danger"
                                            size="sm"
                                            iconOnly
                                            disabled={servicioProcesando !==
                                                null}
                                            loading={servicioProcesando ===
                                                servicio.id}
                                            title={`Archivar ${servicio.nombre}`}
                                            ariaLabel={`Archivar servicio ${servicio.nombre}`}
                                            onClick={() =>
                                                void archivarDesdeLista(
                                                    servicio,
                                                )}
                                        >
                                            <Archive
                                                size={16}
                                                aria-hidden="true"
                                            />
                                        </Button>
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
        margin-top: 0;
    }

    .services-toolbar {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 10px;
        margin-bottom: 16px;
    }

    .search-container {
        flex: 1 1 240px;
        min-width: 0;
    }

    .monitor-filter { width: 146px; flex: 0 0 146px; }
    .monitor-filter :global(.dropdown .trigger) { width: 100%; }

    .create-container {
        flex: 0 0 auto;
    }

    .row-actions {
        display: flex;
        align-items: center;
        flex-wrap: nowrap;
        gap: 8px;
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

    .message.error,
    .action-error {
        color: var(--color-down);
    }
    .confirmation {
        margin: 16px 0;
        color: var(--color-up);
        font-size: 13px;
    }
    .action-error {
        margin: 16px 0;
        font-size: 13px;
    }

    .empty-results {
        padding: 24px 16px;
        text-align: center;
        color: var(--color-muted);
        font-size: 13px;
    }

    .confirmation,
    .action-error {
        position: fixed;
        right: 20px;
        bottom: 20px;
        z-index: 100;
        max-width: min(420px, calc(100vw - 32px));
        margin: 0;
        padding: 12px 16px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface-elevated);
        box-shadow: var(--shadow-card);
        font-size: 13px;
        line-height: 1.5;
    }

    .confirmation {
        color: var(--color-up);
        border-left: 3px solid var(--color-up);
    }

    .action-error {
        color: var(--color-down);
        border-left: 3px solid var(--color-down);
    }

    @media (max-width: 480px) {
        .confirmation,
        .action-error {
            right: 16px;
            bottom: 16px;
            left: 16px;
            max-width: none;
        }
    }

    @media (max-width: 760px) {
        .search-container {
            flex-basis: 100%;
        }
        .create-container {
            flex: 1 1 auto;
        }
        .create-container :global(button) {
            width: 100%;
        }
    }
</style>

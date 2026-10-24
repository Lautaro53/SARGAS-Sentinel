<script lang="ts">
    import { onMount } from "svelte";

    import type { Servicio } from "../types/servicio";
    import {
        obtenerServicios,
        cambiarHabilitacion,
        archivarServicio,
    } from "../api/servicios";

    import ServicioForm from "../components/ServicioForm.svelte";

    let servicios = $state<Servicio[]>([]);
    let cargando = $state(true);
    let mensajeError = $state("");

    let mostrandoFormulario = $state(false);
    let confirmacion = $state("");
    let servicioProcesando = $state<number | null>(null);
    let errorAccion = $state("");

    let servicioEnEdicion = $state<Servicio | null>(null);

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

    function iniciarCreacion() {
        servicioEnEdicion = null;
        mostrandoFormulario = true;
        confirmacion = "";
    }

    function iniciarEdicion(servicio: Servicio) {
        servicioEnEdicion = servicio;
        mostrandoFormulario = true;
        confirmacion = "";
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
    <p class="eyebrow">SARGAS SENTINEL</p>

    <h1>Servicios</h1>

    <p class="description">
        Administra los servicios supervisados por Sentinel.
    </p>

    <div class="service-actions">
        <button
            type="button"
            class="create-button"
            aria-expanded={mostrandoFormulario}
            aria-controls="formulario-servicio"
            onclick={() => {
                mostrandoFormulario = !mostrandoFormulario;
                confirmacion = "";
            }}
        >
            {mostrandoFormulario ? "Cerrar formulario" : "+ Nuevo servicio"}
        </button>
    </div>

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

    <section class="services-panel">
        <div class="panel-header">
            <h2>Servicios registrados</h2>
            <span>{servicios.length} servicios</span>
        </div>

        {#if cargando}
            <p class="message">Cargando servicios...</p>
        {:else if mensajeError}
            <p class="message error" role="alert">
                {mensajeError}
            </p>
        {:else if servicios.length === 0}
            <p class="message">Todavía no hay servicios registrados.</p>
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
                        {#each servicios as servicio (servicio.id)}
                            <tr>
                                <td>
                                    <strong class="service-name">
                                        {servicio.nombre}
                                    </strong>

                                    <span class="service-url">
                                        {servicio.url}
                                    </span>
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
                                        <a
                                            href={`/servicios/${servicio.id}`}
                                            class="edit-button"
                                            onclick={(event) =>
                                                manejarEnlaceDetalle(
                                                    event,
                                                    servicio,
                                                )}
                                        >
                                            Detalle
                                        </a>

                                        <button
                                            type="button"
                                            class="edit-button"
                                            onclick={() =>
                                                iniciarEdicion(servicio)}
                                        >
                                            Editar
                                        </button>

                                        <button
                                            type="button"
                                            class="archive-button"
                                            disabled={servicioProcesando !==
                                                null}
                                            aria-label={`Archivar servicio ${servicio.nombre}`}
                                            onclick={() =>
                                                void archivarDesdeLista(
                                                    servicio,
                                                )}
                                        >
                                            {servicioProcesando === servicio.id
                                                ? "Procesando..."
                                                : "Archivar"}
                                        </button>

                                        <button
                                            type="button"
                                            class="toggle-button"
                                            disabled={servicioProcesando !==
                                                null}
                                            aria-label={`${servicio.habilitado ? "Pausar" : "Reanudar"} el monitoreo de ${servicio.nombre}`}
                                            onclick={() =>
                                                void alternarMonitoreo(
                                                    servicio,
                                                )}
                                        >
                                            {#if servicioProcesando === servicio.id}
                                                Guardando...
                                            {:else if servicio.habilitado}
                                                Pausar
                                            {:else}
                                                Reanudar
                                            {/if}
                                        </button>
                                    </div>
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
        background: #15243a;
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    td {
        border-top: 1px solid var(--color-border);
        font-size: 13px;
    }

    .service-name,
    .service-url {
        display: block;
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

    .message {
        margin: 0;
        padding: 24px;
        color: var(--color-muted);
    }

    .error {
        color: var(--color-down);
    }

    .service-actions {
        display: flex;
        justify-content: flex-end;
        margin-top: 28px;
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
        background: #1c6ce0;
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

    .edit-button {
        padding: 8px 12px;
        border: 1px solid var(--color-border);
        border-radius: 7px;
        background: transparent;
        color: var(--color-primary);
        font: inherit;
        cursor: pointer;

        text-decoration: none;
        display: inline-flex;
        align-items: center;
        justify-content: center;
    }

    .edit-button:hover {
        background: #163968;
    }

    .edit-button:focus-visible {
        outline: 2px solid var(--color-primary);
        outline-offset: 2px;
    }

    .row-actions {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .toggle-button {
        padding: 8px 12px;
        border: 1px solid var(--color-border);
        border-radius: 7px;
        background: transparent;
        color: var(--color-text);
        font: inherit;
        cursor: pointer;
    }

    .toggle-button:hover:not(:disabled) {
        border-color: var(--color-primary);
        color: var(--color-primary);
    }

    .toggle-button:focus-visible {
        outline: 2px solid var(--color-primary);
        outline-offset: 2px;
    }

    .toggle-button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .action-error {
        margin: 16px 0;
        color: var(--color-down);
        font-size: 13px;
    }

    .archive-button {
        padding: 8px 12px;
        border: 1px solid var(--color-border);
        border-radius: 7px;
        background: transparent;
        color: var(--color-down);
        font: inherit;
        cursor: pointer;
    }

    .archive-button:hover:not(:disabled) {
        border-color: var(--color-down);
        background: rgba(240, 93, 108, 0.08);
    }

    .archive-button:focus-visible {
        outline: 2px solid var(--color-down);
        outline-offset: 2px;
    }

    .archive-button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    .archive-button {
        padding: 8px 12px;
        border: 1px solid var(--color-border);
        border-radius: 7px;
        background: transparent;
        color: var(--color-down);
        font: inherit;
        cursor: pointer;
    }

    .archive-button:hover:not(:disabled) {
        border-color: var(--color-down);
        background: rgba(240, 93, 108, 0.08);
    }

    .archive-button:focus-visible {
        outline: 2px solid var(--color-down);
        outline-offset: 2px;
    }

    .archive-button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
</style>

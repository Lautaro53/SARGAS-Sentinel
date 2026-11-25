<script lang="ts">
  import Paginacion from "../components/Paginacion.svelte";
  import DropdownMenu from "../components/ui/DropdownMenu.svelte";
  import { consultarHistorialIncidentes, type EstadoIncidentes, type OrdenIncidentes } from "../api/historial";
  import type { Incidente } from "../types/incidente";
  import { validarIncidentes } from "../api/incidentes";
  import { leerCopia, guardarCopia, esPagina, mensajeErrorConsulta } from "../lib/cacheSesion";
  const esIncidente = (v: unknown): v is Incidente => {
    try { return validarIncidentes([v]).length === 1; } catch { return false; }
  };
  import { SlidersHorizontal, ArrowUpDown } from "@lucide/svelte";

  const TAMANO_PAGINA = 10;
  let incidentes = $state<Incidente[]>([]);
  let pagina = $state(1);
  let total = $state(0);
  let filtroEstado = $state<EstadoIncidentes>("todos");
  let orden = $state<OrdenIncidentes>("fecha_desc");
  let desde = $state("");
  let hasta = $state("");
  let cargando = $state(true);
  let error = $state("");
  let fechaCopia = $state<string | null>(null);
  let usandoCopia = $state(false);
  let hayDatos = $state(false);
  let reintento = $state(0);
  let totalPaginas = $derived(Math.max(1, Math.ceil(total / TAMANO_PAGINA)));

  const opcionesEstado: { value: EstadoIncidentes; label: string }[] = [
    { value: "todos", label: "Todos" },
    { value: "abiertos", label: "Abiertos" },
    { value: "cerrados", label: "Cerrados" },
  ];
  const opcionesOrden: { value: OrdenIncidentes; label: string }[] = [
    { value: "fecha_desc", label: "Más recientes" },
    { value: "fecha_asc", label: "Más antiguos" },
  ];

  $effect(() => {
    const paginaActual = pagina;
    const estado = filtroEstado;
    const ordenActual = orden;
    const desdeFiltro = desde;
    const hastaFiltro = hasta;
    reintento;
    const clave = `incidentes.${JSON.stringify([paginaActual, estado, ordenActual, desdeFiltro, hastaFiltro])}`;
    const copia = leerCopia(clave, (v): v is { registros: Incidente[]; total: number } => esPagina(v, esIncidente));
    incidentes = copia?.datos.registros ?? [];
    total = copia?.datos.total ?? 0;
    fechaCopia = copia?.sincronizadoEn ?? null;
    hayDatos = copia !== null;
    usandoCopia = copia !== null;
    const controlador = new AbortController();
    let vigente = true;
    cargando = true;
    error = "";

    void consultarHistorialIncidentes(null, paginaActual, TAMANO_PAGINA,
      { estado, orden: ordenActual, desde: desdeFiltro, hasta: hastaFiltro }, controlador.signal,
    ).then((resultado) => {
      if (!vigente) return;
      incidentes = resultado.registros;
      total = resultado.total;
      fechaCopia = guardarCopia(clave, resultado);
      hayDatos = true;
      usandoCopia = false;
      error = "";
      const ultima = Math.max(1, Math.ceil(resultado.total / TAMANO_PAGINA));
      if (paginaActual > ultima) pagina = ultima;
    }).catch((e: unknown) => {
      if (!vigente || controlador.signal.aborted) return;
      usandoCopia = hayDatos;
      error = copia === null
        ? "No se pudo consultar el historial. No hay datos guardados para esta selección."
        : mensajeErrorConsulta(e, "No se pudieron actualizar los incidentes");
    }).finally(() => {
      if (vigente) cargando = false;
    });

    return () => { vigente = false; controlador.abort(); };
  });

  import { onMount } from "svelte";
  onMount(() => {
    const refrescar = () => { if (!document.hidden) reintento += 1; };
    window.addEventListener("online", refrescar);
    document.addEventListener("visibilitychange", refrescar);
    const intervalo = window.setInterval(refrescar, 15_000);
    return () => {
      window.removeEventListener("online", refrescar);
      document.removeEventListener("visibilitychange", refrescar);
      window.clearInterval(intervalo);
    };
  });
</script>

<div class="incidents-screen">
  <section class="data-table-panel history-panel" aria-labelledby="incidents-title">
    <div class="panel-header">
      <h2 id="incidents-title">Historial de incidentes</h2>
      <span>{!hayDatos && error ? "Sin datos disponibles" : `${total} ${total === 1 ? "registro" : "registros"}`}</span>
    </div>
    {#snippet iconoFiltro()}
      <SlidersHorizontal size={17} aria-hidden="true" />
    {/snippet}
    {#snippet iconoOrden()}
      <ArrowUpDown size={17} aria-hidden="true" />
    {/snippet}
    <div class="history-controls" aria-label="Filtros de incidentes">
      <div class="filter-field">
        <span class="filter-label">Estado</span>
        <DropdownMenu
          label="Filtrar incidentes por estado"
          value={filtroEstado}
          options={opcionesEstado}
          icon={iconoFiltro}
          menuAlign="start"
          onChange={(valor) => { filtroEstado = valor; pagina = 1; }}
        />
      </div>
      <div class="filter-field">
        <span class="filter-label">Orden</span>
        <DropdownMenu
          label="Ordenar incidentes"
          value={orden}
          options={opcionesOrden}
          icon={iconoOrden}
          triggerText="Ordenar"
          menuAlign="start"
          onChange={(valor) => { orden = valor; pagina = 1; }}
        />
      </div>
      <label class="filter-field">
        <span class="filter-label">Desde</span>
        <input type="date" aria-label="Incidentes desde" value={desde}
          onchange={(e) => { desde = e.currentTarget.value; pagina = 1; }} />
      </label>
      <label class="filter-field">
        <span class="filter-label">Hasta</span>
        <input type="date" aria-label="Incidentes hasta" value={hasta}
          onchange={(e) => { hasta = e.currentTarget.value; pagina = 1; }} />
      </label>
      {#if cargando && incidentes.length > 0}<span class="updating" role="status">Actualizando…</span>{/if}
    </div>
    {#if usandoCopia && error}
      <p class="stale-warning" role="status">Datos guardados / sin conexión · última sincronización: {fechaCopia ? new Date(fechaCopia).toLocaleString("es-UY") : "desconocida"}. {error}</p>
    {:else if error}<p class="panel-warning" role="alert">{error}</p>{/if}
    {#if cargando && incidentes.length === 0 && !error}
      <p class="panel-message">Cargando incidentes...</p>
    {:else if incidentes.length === 0}
      <p class="panel-message">{error ? "No se pudieron obtener los registros." : "No hay incidentes que coincidan con la consulta."}</p>
    {:else}
      <div class="table-scroll" class:history-page-stable={total > TAMANO_PAGINA}>
        <table>
          <thead>
            <tr>
              <th scope="col">Servicio</th>
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
                <td class="service-name">
                  {incidente.nombre_servicio}
                  {#if incidente.servicio_archivado}<span class="archived-label">Archivado</span>{/if}
                </td>
                <td>{new Date(incidente.iniciado_en).toLocaleString("es-UY")}</td>
                <td>{incidente.cerrado_en ? new Date(incidente.cerrado_en).toLocaleString("es-UY") : "—"}</td>
                <td><span class="incident-status" class:open={incidente.cerrado_en === null}>
                    {incidente.cerrado_en === null ? "Abierto" : "Cerrado"}
                </span></td>
                <td class="evidence" title={incidente.evidencia_inicio}>{incidente.evidencia_inicio}</td>
                <td class="evidence" title={incidente.evidencia_cierre ?? ""}>{incidente.evidencia_cierre ?? "—"}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
    {#if total > 0}
      <Paginacion {pagina} {totalPaginas} deshabilitado={cargando}
        etiqueta="incidentes" onCambiarPagina={(n) => (pagina = n)} />
    {/if}
  </section>
</div>

<style>
  .incidents-screen { display: grid; gap: 16px; min-width: 0; }
  .history-panel { position: relative; z-index: 1; overflow: visible; }
  .history-panel:focus-within { z-index: 20; }
  .history-controls { display: flex; align-items: flex-end; flex-wrap: wrap; gap: 10px; padding: 10px 16px 14px; border-top: 1px solid var(--color-border); }
  .filter-field { display: grid; gap: 5px; width: 150px; min-width: 0; }
  .filter-label { color: var(--color-muted); font-size: 11px; }
  .filter-field :global(.dropdown), .filter-field :global(.dropdown .trigger) { width: 100%; }
  .filter-field :global(.dropdown .trigger) { min-height: 40px; font-size: 13px; }
  .filter-field input { box-sizing: border-box; width: 100%; min-width: 0; min-height: 40px; padding: 6px 10px; border: 1px solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-text); color-scheme: inherit; font: inherit; font-size: 12px; }
  .filter-field input:focus-visible { outline: none; border-color: var(--color-primary-hover); box-shadow: inset 0 0 0 1px var(--color-primary-hover); }
  .table-scroll.history-page-stable { min-height: 436px; }
  .updating { margin-left: auto; color: var(--color-muted); font-size: 11px; }
  .panel-warning { margin: 0; padding: 8px 16px; color: var(--color-down); font-size: 12px; }
  .panel-message { margin: 0; padding: 18px 16px; color: var(--color-muted); font-size: 13px; }
  .service-name { font-weight: 600; }
  .incident-status { font-weight: 600; color: var(--color-up); }
  .incident-status.open { color: var(--color-down); }
  .evidence { max-width: 260px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--color-muted); }
  .archived-label { display: block; margin-top: 5px; color: var(--color-muted); font-size: 11px; font-weight: 400; }
</style>

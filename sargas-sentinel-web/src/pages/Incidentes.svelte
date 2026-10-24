<script lang="ts">
  import { onMount } from "svelte";
  import { obtenerTodosIncidentes } from "../api/incidentes";
  import type { Incidente } from "../types/incidente";

  let incidentes = $state<Incidente[]>([]);

  let cargando = $state(true);
  let error = $state("");

  onMount(() => {
    let activo = true;

    async function cargarDatos() {
      try {
        const listaIncidentes = await obtenerTodosIncidentes();

        if (!activo) return;

        incidentes = listaIncidentes;
      } catch (e) {
        if (!activo) return;

        error =
          e instanceof Error
            ? e.message
            : "No se pudieron cargar los incidentes";
      } finally {
        if (activo) {
          cargando = false;
        }
      }
    }

    void cargarDatos();

    return () => {
      activo = false;
    };
  });
</script>

<div class="page">
  <p class="eyebrow">SARGAS SENTINEL</p>

  <h1>Incidentes</h1>

  <p class="description">
    Consulta las interrupciones y recuperaciones de los servicios monitorizados.
  </p>

  <section class="incidents-panel">
    <div class="panel-header">
      <h2>Historial de incidentes</h2>
      <span>{incidentes.length} registros mostrados (máximo 50)</span>
    </div>

    {#if cargando}
      <p class="message">Cargando incidentes...</p>
    {:else if error}
      <p class="error" role="alert">{error}</p>
    {:else if incidentes.length === 0}
      <p class="message">Todavía no hay incidentes registrados.</p>
    {:else}
      <div class="table-scroll">
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

                  {#if incidente.servicio_archivado}
                    <span class="archived-label"> Archivado </span>
                  {/if}
                </td>

                <td>
                  {new Date(incidente.iniciado_en).toLocaleString("es-UY")}
                </td>

                <td>
                  {incidente.cerrado_en
                    ? new Date(incidente.cerrado_en).toLocaleString("es-UY")
                    : "—"}
                </td>

                <td>
                  <span
                    class="incident-status"
                    class:open={incidente.cerrado_en === null}
                  >
                    {incidente.cerrado_en === null ? "Abierto" : "Cerrado"}
                  </span>
                </td>

                <td class="evidence">
                  {incidente.evidencia_inicio}
                </td>

                <td class="evidence">
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
  .eyebrow {
    color: var(--color-primary);
    font-size: 12px;
    letter-spacing: 2px;
  }

  h1 {
    margin: 12px 0;
    font-size: 32px;
  }

  .eyebrow {
    color: var(--color-primary);
    font-size: 12px;
    letter-spacing: 2px;
  }

  h1 {
    margin: 12px 0;
    font-size: 32px;
  }

  .description,
  .message {
    color: var(--color-muted);
  }

  .incidents-panel {
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
    flex-wrap: wrap;
    gap: 12px;
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

  .message,
  .error {
    padding: 24px;
  }

  .error {
    color: var(--color-down);
  }

  .table-scroll {
    overflow-x: auto;
  }

  table {
    width: 100%;
    min-width: 1050px;
    border-collapse: collapse;
    text-align: left;
  }

  th,
  td {
    padding: 16px;
    font-size: 13px;
    vertical-align: top;
  }

  th {
    color: var(--color-muted);
    background: #15243a;
    font-size: 11px;
    text-transform: uppercase;
  }

  td {
    border-top: 1px solid var(--color-border);
  }

  .service-name {
    font-weight: 600;
  }

  .incident-status {
    color: var(--color-up);
    font-weight: 600;
  }

  .incident-status.open {
    color: var(--color-down);
  }

  .evidence {
    min-width: 230px;
    overflow-wrap: anywhere;
    font-size: 12px;
    line-height: 1.5;
  }

  .archived-label {
    display: block;
    margin-top: 6px;
    color: var(--color-muted);
    font-size: 11px;
    font-weight: 400;
  }
</style>

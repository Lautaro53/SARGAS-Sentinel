<script lang="ts">
  import { onMount } from "svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import Topbar from "./components/Topbar.svelte";
  import Dashboard from "./pages/Dashboard.svelte";
  import Servicios from "./pages/Servicios.svelte";
  import Incidentes from "./pages/Incidentes.svelte";
  import DetalleServicio from "./pages/DetalleServicio.svelte";
  import { obtenerServicios } from "./api/servicios";
  import type { Pagina } from "./types/navegacion";
  import type { Servicio } from "./types/servicio";

  type Ruta =
    | { pagina: Pagina; detalleId: null }
    | { pagina: "servicios"; detalleId: number };

  function interpretarRuta(path: string): Ruta {
    // La SPA acepta /servicios/ e /incidentes/ igual que el router Go.
    if (path.length > 1) path = path.replace(/\/+$/, "");
    if (path === "/servicios") return { pagina: "servicios", detalleId: null };
    if (path === "/incidentes") return { pagina: "incidentes", detalleId: null };
    const resultado = /^\/servicios\/([1-9]\d*)\/?$/.exec(path);
    if (resultado) {
      const id = Number(resultado[1]);
      if (Number.isSafeInteger(id)) return { pagina: "servicios", detalleId: id };
    }
    return { pagina: "dashboard", detalleId: null };
  }

  let menuAbierto = $state(false);
  let ruta = $state<Ruta>(interpretarRuta(window.location.pathname));
  let servicioSeleccionado = $state<Servicio | null>(null);
  let cargandoDetalle = $state(false);
  let errorDetalle = $state("");
  let revision = 0;

  async function sincronizarRuta() {
    const version = ++revision;
    ruta = interpretarRuta(window.location.pathname);
    menuAbierto = false;
    errorDetalle = "";

    if (ruta.detalleId === null) {
      servicioSeleccionado = null;
      cargandoDetalle = false;
      return;
    }

    const id = ruta.detalleId;
    // El listado operativo es el contrato GET disponible en v1.0.
    if (servicioSeleccionado?.id === id) {
      cargandoDetalle = false;
      return;
    }
    servicioSeleccionado = null;
    cargandoDetalle = true;
    try {
      const servicios = await obtenerServicios();
      if (version !== revision) return;
      const encontrado = servicios.find((servicio) => servicio.id === id);
      if (encontrado) servicioSeleccionado = encontrado;
      else errorDetalle = "El servicio no existe o está archivado.";
    } catch (error) {
      if (version !== revision) return;
      errorDetalle = error instanceof Error ? error.message : "No se pudo cargar el servicio.";
    } finally {
      if (version === revision) cargandoDetalle = false;
    }
  }

  function cambiarRuta(path: string) {
    if (window.location.pathname !== path) window.history.pushState(null, "", path);
    void sincronizarRuta();
  }

  function navegar(pagina: Pagina) {
    cambiarRuta(pagina === "dashboard" ? "/" : `/${pagina}`);
  }

  function abrirDetalle(servicio: Servicio) {
    // Ya disponemos del servicio validado: evitamos una consulta duplicada.
    servicioSeleccionado = servicio;
    cambiarRuta(`/servicios/${servicio.id}`);
  }

  onMount(() => {
    void sincronizarRuta();
    const manejarHistorial = () => void sincronizarRuta();
    window.addEventListener("popstate", manejarHistorial);
    return () => {
      ++revision;
      window.removeEventListener("popstate", manejarHistorial);
    };
  });
</script>

<svelte:head>
  <title>SARGAS Sentinel</title>
</svelte:head>

<div class="app-shell">
  <Sidebar {menuAbierto} paginaActual={ruta.pagina} {navegar} />
  <div class="workspace">
    <Topbar {menuAbierto} alternarMenu={() => (menuAbierto = !menuAbierto)} />
    <main class="page-content">
      {#if ruta.pagina === "dashboard"}
        <Dashboard />
      {:else if ruta.pagina === "servicios"}
        {#if ruta.detalleId !== null}
          {#if cargandoDetalle}
            <p role="status">Cargando servicio...</p>
          {:else if errorDetalle}
            <p role="alert">{errorDetalle}</p>
            <button type="button" onclick={() => navegar("servicios")}>← Volver a servicios</button>
          {:else if servicioSeleccionado}
            {#key ruta.detalleId}
              <DetalleServicio servicio={servicioSeleccionado} onVolver={() => navegar("servicios")} />
            {/key}
          {/if}
        {:else}
          <Servicios onVerDetalle={abrirDetalle} />
        {/if}
      {:else if ruta.pagina === "incidentes"}
        <Incidentes />
      {/if}
    </main>
  </div>
</div>

<style>
  .app-shell { display: flex; min-height: 100vh; }
  .workspace { flex: 1; min-width: 0; }
  .page-content { max-width: 1400px; margin: 0 auto; padding: 42px 32px; }
  @media (max-width: 760px) { .page-content { padding: 28px 16px; } }
</style>

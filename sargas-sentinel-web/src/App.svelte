<script lang="ts">
  import { onMount } from "svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import Topbar from "./components/Topbar.svelte";
  import Dashboard from "./pages/Dashboard.svelte";
  import Servicios from "./pages/Servicios.svelte";
  import Incidentes from "./pages/Incidentes.svelte";
  import DetalleServicio from "./pages/DetalleServicio.svelte";
  import { obtenerServicios, leerServiciosGuardados } from "./api/servicios";
  import type { Pagina } from "./types/navegacion";
  import type { Servicio } from "./types/servicio";
  import Footer from "./components/Footer.svelte";

  type Ruta =
    | { pagina: Pagina; detalleId: null }
    | { pagina: "servicios"; detalleId: number };

  function interpretarRuta(path: string): Ruta {
    if (path.length > 1) path = path.replace(/\/+$/, "");
    if (path === "/servicios") return { pagina: "servicios", detalleId: null };
    if (path === "/incidentes")
      return { pagina: "incidentes", detalleId: null };
    const resultado = /^\/servicios\/([1-9]\d*)\/?$/.exec(path);
    if (resultado) {
      const id = Number(resultado[1]);
      if (Number.isSafeInteger(id))
        return { pagina: "servicios", detalleId: id };
    }
    return { pagina: "dashboard", detalleId: null };
  }

  type Tema = "neon-blue" | "neon-blue-light";
  let tema = $state<Tema>(
    document.documentElement.dataset.theme === "neon-blue-light" ? "neon-blue-light" : "neon-blue",
  );

  function alternarTema() {
    tema = tema === "neon-blue" ? "neon-blue-light" : "neon-blue";
    document.documentElement.dataset.theme = tema;
    try {
      localStorage.setItem("sargas.tema", tema);
    } catch {
      // El cambio visual funciona aunque no se pueda guardar la preferencia.
    }
  }

  let menuAbierto = $state(false);
  let ruta = $state<Ruta>(interpretarRuta(window.location.pathname));
  let servicioSeleccionado = $state<Servicio | null>(null);
  let cargandoDetalle = $state(false);
  let errorDetalle = $state("");
  let avisoDetalle = $state("");
  let revision = 0;

  let tituloTopbar = $derived(
    ruta.detalleId !== null
      ? "Detalle del servicio"
      : ruta.pagina === "servicios"
        ? "Servicios"
        : ruta.pagina === "incidentes"
          ? "Incidentes"
          : "Panel general",
  );

  let descripcionTopbar = $derived(
    ruta.detalleId !== null
      ? "Información y monitoreo del servicio seleccionado."
      : ruta.pagina === "servicios"
        ? "Administración de servicios monitorizados."
        : ruta.pagina === "incidentes"
          ? "Seguimiento y gestión de incidentes."
          : "Disponibilidad y rendimiento de tus servicios en tiempo real.",
  );

  async function sincronizarRuta() {
    const version = ++revision;
    const idAnterior = ruta.detalleId;
    ruta = interpretarRuta(window.location.pathname);
    menuAbierto = false;
    errorDetalle = "";
    if (idAnterior !== ruta.detalleId) avisoDetalle = "";

    if (ruta.detalleId === null) {
      servicioSeleccionado = null;
      cargandoDetalle = false;
      return;
    }

    const id = ruta.detalleId;
    const copia = leerServiciosGuardados();
    const conocido = servicioSeleccionado?.id === id ? servicioSeleccionado : null;
    servicioSeleccionado = conocido ?? copia?.datos.find((servicio) => servicio.id === id) ?? null;
    cargandoDetalle = servicioSeleccionado === null;

    try {
      const servicios = await obtenerServicios();
      if (version !== revision) return;
      const encontrado = servicios.find((servicio) => servicio.id === id);
      servicioSeleccionado = encontrado ?? null;
      avisoDetalle = "";
      if (!encontrado) errorDetalle = "El servicio no existe o está archivado.";
    } catch {
      if (version !== revision) return;
      if (servicioSeleccionado) {
        const fecha = copia?.sincronizadoEn;
        avisoDetalle = `Datos guardados / sin conexión · última sincronización: ${fecha ? new Date(fecha).toLocaleString("es-UY") : "desconocida"}.`;
      } else {
        errorDetalle = "Sin conexión. No hay datos guardados de este servicio.";
      }
    } finally {
      if (version === revision) cargandoDetalle = false;
    }
  }

  function cambiarRuta(path: string) {
    if (window.location.pathname !== path)
      window.history.pushState(null, "", path);
    void sincronizarRuta();
  }

  function navegar(pagina: Pagina) {
    cambiarRuta(pagina === "dashboard" ? "/" : `/${pagina}`);
  }

  function abrirDetalle(servicio: Servicio) {
    servicioSeleccionado = servicio;
    cambiarRuta(`/servicios/${servicio.id}`);
  }

  onMount(() => {
    void sincronizarRuta();
    const manejarHistorial = () => void sincronizarRuta();
    let reintentando = false;
    const recuperarDetalle = async () => {
      if (reintentando || document.hidden || ruta.detalleId === null || (!avisoDetalle && !errorDetalle)) return;
      reintentando = true;
      try { await sincronizarRuta(); }
      finally { reintentando = false; }
    };
    window.addEventListener("popstate", manejarHistorial);
    window.addEventListener("online", recuperarDetalle);
    const intervalo = window.setInterval(() => void recuperarDetalle(), 15_000);
    return () => {
      ++revision;
      window.clearInterval(intervalo);
      window.removeEventListener("popstate", manejarHistorial);
      window.removeEventListener("online", recuperarDetalle);
    };
  });
</script>

<svelte:head>
  <title>SARGAS Sentinel</title>
</svelte:head>

<div class="app-shell">
  <Sidebar {menuAbierto} paginaActual={ruta.pagina} {navegar} />
  <div class="workspace">
    <Topbar
      {menuAbierto}
      alternarMenu={() => (menuAbierto = !menuAbierto)}
      titulo={tituloTopbar}
      descripcion={descripcionTopbar}
      {tema}
      {alternarTema}
    />
    <main class="page-content">
      {#if ruta.pagina === "dashboard"}
        <Dashboard onVerDetalle={abrirDetalle} />
      {:else if ruta.pagina === "servicios"}
        {#if ruta.detalleId !== null}
          {#if cargandoDetalle && !servicioSeleccionado}
            <p role="status">Cargando servicio...</p>
          {:else if errorDetalle && !servicioSeleccionado}
            <p role="alert">{errorDetalle}</p>
            <button type="button" onclick={() => navegar("servicios")}
              >← Volver a servicios</button
            >
          {:else if servicioSeleccionado}
            {#if avisoDetalle}<p class="stale-warning cache-note" role="status">{avisoDetalle}</p>{/if}
            {#key ruta.detalleId}
              <DetalleServicio
                servicio={servicioSeleccionado}
                onVolver={() => navegar("servicios")}
              />
            {/key}
          {/if}
        {:else}
          <Servicios onVerDetalle={abrirDetalle} />
        {/if}
      {:else if ruta.pagina === "incidentes"}
        <Incidentes />
      {/if}
    </main>
    <div class="footer-container">
      <Footer />
    </div>
  </div>
</div>

<style>
  .cache-note {
    margin: 0 0 12px;
  }

  .app-shell {
    display: flex;
    min-height: 100vh;
  }
  .workspace {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
  }

  .page-content {
    width: 100%;
    max-width: 1600px;
    margin: 0 auto;
    padding: 24px 24px 32px;
    flex: 1;
  }

  .footer-container {
    width: 100%;
    max-width: 1600px;
    margin: 0 auto;
    padding: 0 24px;
  }

  @media (max-width: 760px) {
    .page-content {
      padding: 20px 16px 24px;
    }

    .footer-container {
      padding: 0 16px;
    }
  }
</style>

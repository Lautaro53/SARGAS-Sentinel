<script lang="ts">
  import type { Pagina } from "../types/navegacion";

  type Props = {
    menuAbierto: boolean;
    paginaActual: Pagina;
    navegar: (pagina: Pagina) => void;
  };

  let { menuAbierto, paginaActual, navegar }: Props = $props();

  function manejarEnlace(event: MouseEvent, pagina: Pagina) {
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
    navegar(pagina);
  }
</script>

<aside id="sidebar-principal" class="sidebar" class:abierto={menuAbierto}>
  <div class="brand">
    <img src="/sargas-sentinel-logo-v1.png" alt="" width="48" height="48" />

    <div class="brand-text">
      <strong>SARGAS</strong>
      <span>SENTINEL</span>
    </div>
  </div>

  <nav class="navigation" aria-label="Navegación principal">
    <a
      href="/"
      class="nav-item"
      class:active={paginaActual === "dashboard"}
      aria-current={paginaActual === "dashboard" ? "page" : undefined}
      onclick={(event) => manejarEnlace(event, "dashboard")}
    >
      Panel general
    </a>

    <a
      href="/servicios"
      class="nav-item"
      class:active={paginaActual === "servicios"}
      aria-current={paginaActual === "servicios" ? "page" : undefined}
      onclick={(event) => manejarEnlace(event, "servicios")}
    >
      Servicios
    </a>

    <a
      href="/incidentes"
      class="nav-item"
      class:active={paginaActual === "incidentes"}
      aria-current={paginaActual === "incidentes" ? "page" : undefined}
      onclick={(event) => manejarEnlace(event, "incidentes")}
    >
      Incidentes
    </a>
  </nav>

  <div class="sidebar-footer">SARGAS Sentinel · v1</div>
</aside>

<style>
  .sidebar {
    width: 248px;
    flex-shrink: 0;
    background: var(--color-sidebar);
    border-right: 1px solid var(--color-border);

    display: flex;
    flex-direction: column;
    padding: 24px 14px;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 0 10px 28px;
  }

  .brand img {
    border-radius: 10px;
    object-fit: cover;
  }

  .brand-text {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .brand-text strong {
    font-size: 17px;
    letter-spacing: 1.5px;
  }

  .brand-text span {
    color: var(--color-primary);
    font-size: 11px;
    letter-spacing: 2px;
  }

  .navigation {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .nav-item {
    display: block;
    width: 100%;

    padding: 13px 14px;
    border: 0;
    border-left: 3px solid transparent;
    border-radius: 8px;

    background: transparent;
    color: var(--color-muted);

    text-align: left;
    font-size: 14px;
    cursor: pointer;
    text-decoration: none;
  }

  .nav-item:hover {
    background: #132843;
    color: var(--color-text);
  }

  .nav-item.active {
    background: #163968;
    border-left-color: var(--color-primary);
    color: white;
  }

  .nav-item:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  .sidebar-footer {
    margin-top: auto;
    padding: 20px 12px 0;
    color: var(--color-muted);
    font-size: 12px;
  }

  @media (max-width: 760px) {
    .sidebar {
      display: none;
      position: absolute;
      top: 64px;
      left: 0;
      right: 0;
      width: 100%;
      min-height: 0;
      z-index: 10;
    }

    .sidebar.abierto {
      display: flex;
    }
  }
</style>

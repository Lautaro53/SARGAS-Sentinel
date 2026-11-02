<script lang="ts">
  import type { Pagina } from "../types/navegacion";
  import { MonitorDot, Blocks, TriangleAlert } from "@lucide/svelte";
  import { Orbit } from "@lucide/svelte";

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
    <img
      class="brand-logo"
      src="/s3.jpg"
      alt="SARGAS Sentinel"
      width="40"
      height="40"
    />
    <div class="brand-text">
      <span>SARGAS</span>
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
      <MonitorDot size={19} strokeWidth={1.8} aria-hidden="true" />
      <span>Panel general</span>
    </a>

    <a
      href="/servicios"
      class="nav-item"
      class:active={paginaActual === "servicios"}
      aria-current={paginaActual === "servicios" ? "page" : undefined}
      onclick={(event) => manejarEnlace(event, "servicios")}
    >
      <Blocks size={19} strokeWidth={1.8} aria-hidden="true" />
      <span>Servicios</span>
    </a>

    <a
      href="/incidentes"
      class="nav-item"
      class:active={paginaActual === "incidentes"}
      aria-current={paginaActual === "incidentes" ? "page" : undefined}
      onclick={(event) => manejarEnlace(event, "incidentes")}
    >
      <TriangleAlert size={19} strokeWidth={1.8} aria-hidden="true" />
      <span>Incidentes</span>
    </a>
  </nav>

  <div class="sidebar-footer">
    <div class="sidebar-footer-content">
      <span class="version-badge">v1.1.6</span>
    </div>
  </div>
</aside>

<style>
  .sidebar {
    width: 248px;
    height: 100vh;
    height: 100dvh;
    position: sticky;
    top: 0;
    flex-shrink: 0;

    background: var(--color-sidebar);
    border-right: 1px solid var(--color-border);

    display: flex;
    flex-direction: column;
    padding: 20px 14px 0;

    overflow-y: auto;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 0 10px 28px;
  }

  .brand-logo {
    display: block;
    width: 64px;
    height: 64px;
    object-fit: contain;
    flex-shrink: 0;
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

  .brand-text {
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
    display: flex;
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

    align-items: center;
    gap: 12px;

    transition:
      background-color 0.2s ease,
      color 0.2s ease,
      border-color 0.2s ease;
  }

  .nav-item:hover:not(.active) {
    background: color-mix(in srgb, var(--color-primary) 6%, transparent);
    color: var(--color-text);
  }

  .nav-item.active {
    background: color-mix(in srgb, var(--color-primary) 10%, transparent);
    color: var(--color-primary-hover);
    font-weight: 600;
  }

  .nav-item:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 3px;
  }

  .sidebar-footer {
    margin-top: auto;
    min-height: 65px;
    padding: 0 10px;

    display: flex;
    align-items: center;
    border-top: 1px solid var(--color-border);
  }

  .sidebar-footer-content {
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--color-primary);
  }

  .version-badge {
    flex-shrink: 0;
    padding: 4px 6px;
    background: var(--color-surface-elevated);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    color: var(--color-primary-hover);
    font-size: 10px;
    font-weight: 600;
  }

  @media (max-width: 760px) {
    .sidebar {
      display: none;
      position: absolute;
      top: 64px;
      left: 0;
      right: 0;

      width: 100%;
      height: auto;
      max-height: calc(100dvh - 64px);
      min-height: 0;

      z-index: 10;
      overflow-y: auto;
    }

    .sidebar.abierto {
      display: flex;
    }
  }
</style>

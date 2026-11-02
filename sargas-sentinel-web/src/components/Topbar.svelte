<script lang="ts">
  import { onMount } from "svelte";
  import { Moon, Menu } from "@lucide/svelte";
  import { obtenerServicios } from "../api/servicios";

  type EstadoConexion = "comprobando" | "conectado" | "desconectado";

  type Props = {
    menuAbierto: boolean;
    alternarMenu: () => void;
  };

  let { menuAbierto, alternarMenu }: Props = $props();

  let estadoConexion = $state<EstadoConexion>("comprobando");
  let horaLocal = $state("");

  onMount(() => {
    let activo = true;
    let consultaEnCurso = false;

    function actualizarHora() {
      horaLocal = new Date().toLocaleTimeString("es-UY", {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
      });
    }

    async function comprobarConexion() {
      if (consultaEnCurso) return;
      consultaEnCurso = true;

      try {
        await obtenerServicios();

        if (activo) {
          estadoConexion = "conectado";
        }
      } catch {
        if (activo) {
          estadoConexion = "desconectado";
        }
      } finally {
        consultaEnCurso = false;
      }
    }

    actualizarHora();
    void comprobarConexion();

    const reloj = window.setInterval(actualizarHora, 1000);
    const conexion = window.setInterval(() => void comprobarConexion(), 30_000);

    return () => {
      activo = false;
      window.clearInterval(reloj);
      window.clearInterval(conexion);
    };
  });
</script>

<header class="topbar">
  <div class="topbar-heading">
    <button
      class="menu-button"
      type="button"
      aria-label="Abrir o cerrar menú"
      aria-controls="sidebar-principal"
      aria-expanded={menuAbierto}
      onclick={alternarMenu}
    >
      <Menu size={21} strokeWidth={1.8} />
    </button>

    <div class="page-heading">
      <h1>Panel general</h1>
      <p>Disponibilidad y rendimiento de tus servicios en tiempo real.</p>
    </div>
  </div>

  <div class="topbar-actions">
    <div class="system-indicator">
      <span
        class="system-dot"
        class:connected={estadoConexion === "conectado"}
        class:disconnected={estadoConexion === "desconectado"}
        aria-hidden="true"
      ></span>

      <div class="system-copy" role="status">
        <strong>
          {#if estadoConexion === "conectado"}
            API conectada
          {:else if estadoConexion === "desconectado"}
            Sin conexión
          {:else}
            Comprobando...
          {/if}
        </strong>

        <span>
          {#if estadoConexion === "conectado"}
            Servidor disponible
          {:else if estadoConexion === "desconectado"}
            No se pudo consultar la API
          {:else}
            Verificando servidor
          {/if}
        </span>
      </div>
    </div>

    <div class="topbar-separator"></div>

    <div class="datetime">
      <span>{horaLocal || "—"}</span>
      <small>Hora local</small>
    </div>

    <div
      class="theme-placeholder"
      title="Cambio de tema disponible próximamente"
      aria-label="Cambio de tema disponible próximamente"
    >
      <Moon size={19} strokeWidth={1.8} aria-hidden="true" />
    </div>
  </div>
</header>

<style>
  .topbar {
    min-height: 72px;
    padding: 12px 24px;

    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;

    background: var(--color-sidebar);

    border: none;
    border-bottom: 1px solid var(--color-border);
    border-radius: 0;
  }

  .topbar-heading,
  .topbar-actions,
  .system-indicator {
    display: flex;
    align-items: center;
  }

  .topbar-heading {
    gap: 12px;
    min-width: 0;
  }

  .page-heading {
    min-width: 0;
  }

  .page-heading h1 {
    margin: 0;
    color: var(--color-text);
    font-size: 20px;
    font-weight: 650;
    line-height: 1.3;
  }

  .page-heading p {
    margin: 4px 0 0;
    color: var(--color-muted);
    font-size: 13px;
    line-height: 1.5;
  }

  .topbar-actions {
    gap: 16px;
    flex-shrink: 0;
  }

  .system-indicator {
    gap: 10px;
  }

  .system-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
    background: var(--color-unknown);
  }

  .system-dot.connected {
    background: var(--color-up);
  }

  .system-dot.disconnected {
    background: var(--color-down);
  }

  .datetime > span {
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .system-copy,
  .datetime {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .system-copy strong,
  .datetime > span {
    color: var(--color-text);
    font-size: 12px;
    font-weight: 600;
  }

  .system-copy > span,
  .datetime small {
    color: var(--color-muted);
    font-size: 11px;
  }

  .topbar-separator {
    height: 32px;
    width: 1px;
    background: var(--color-border);
  }

  .theme-placeholder {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;

    color: var(--color-muted);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-control);

    cursor: default;
  }

  .menu-button {
    display: none;
    align-items: center;
    justify-content: center;

    padding: 6px;
    color: var(--color-text);
    background: transparent;
    border: 0;
    cursor: pointer;
  }

  .menu-button:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 3px;
  }

  @media (max-width: 1050px) {
    .system-copy > span,
    .datetime {
      display: none;
    }

    .topbar-separator {
      display: none;
    }
  }

  @media (max-width: 760px) {
    .topbar {
      min-height: 64px;
      padding: 12px 16px;
      gap: 12px;
    }

    .menu-button {
      display: flex;
    }

    .page-heading h1 {
      font-size: 16px;
    }

    .page-heading p,
    .system-copy {
      display: none;
    }

    .topbar-actions {
      gap: 12px;
    }
  }
</style>

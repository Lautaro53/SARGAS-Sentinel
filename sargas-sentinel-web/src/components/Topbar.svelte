<script lang="ts">
  import { Moon, Menu } from "@lucide/svelte";

  type Props = {
    menuAbierto: boolean;
    alternarMenu: () => void;
  };

  let { menuAbierto, alternarMenu }: Props = $props();
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
      <span class="system-dot"></span>

      <div class="system-copy">
        <strong>Estado no conectado</strong>
        <span>Pendiente de integración</span>
      </div>
    </div>

    <div class="topbar-separator"></div>

    <div class="datetime">
      <span>—</span>
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
    min-height: 92px;
    padding: 20px 28px;

    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 28px;

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
    gap: 14px;
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
    margin: 6px 0 0;
    color: var(--color-muted);
    font-size: 13px;
    line-height: 1.5;
  }

  .topbar-actions {
    gap: 20px;
    flex-shrink: 0;
  }

  .system-indicator {
    gap: 10px;
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
      min-height: 72px;
      padding: 14px 16px;
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

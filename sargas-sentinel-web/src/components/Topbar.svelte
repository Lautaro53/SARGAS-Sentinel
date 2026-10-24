
<script lang="ts">
  type Props = {
    menuAbierto: boolean
    alternarMenu: () => void
  }

  let { menuAbierto, alternarMenu }: Props = $props()
</script>

<header class="topbar">
  <button
    class="menu-button"
    type="button"
    aria-label="Abrir o cerrar menú"
    aria-controls="sidebar-principal"
    aria-expanded={menuAbierto}
    onclick={alternarMenu}
  >
    ☰
  </button>

  <span>Centro de monitoreo</span>

  <span class="development-label">
    Interfaz - SARGAS Sentinel · v1
  </span>
</header>

<style>
  .topbar {
    height: 64px;
    padding: 0 28px;
    border-bottom: 1px solid var(--color-border);

    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 15px;

    background: var(--color-sidebar);
  }

  .development-label {
    color: var(--color-muted);
    font-size: 12px;
  }

  .menu-button {
    display: none;
    color: var(--color-text);
    background: none;
    border: 0;
    cursor: pointer;
    font-size: 22px;
  }

  .menu-button:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 4px;
  }

  @media (max-width: 760px) {
    .menu-button {
      display: block;
    }

    .topbar {
      padding: 0 16px;
    }
  }
</style>

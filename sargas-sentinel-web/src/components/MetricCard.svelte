<script lang="ts">
  import {
    Server,
    CircleCheck,
    CircleX,
    CircleHelp,
    AlertTriangle,
  } from "@lucide/svelte";

  type IconoMetrica = "server" | "check" | "error" | "unknown" | "incident";

  const iconos = {
    server: Server,
    check: CircleCheck,
    error: CircleX,
    unknown: CircleHelp,
    incident: AlertTriangle,
  };

  type Tono = "normal" | "up" | "down" | "unknown";

  type Props = {
    titulo: string;
    valor: number;
    descripcion: string;
    tono?: Tono;
    icono?: IconoMetrica;
  };

  let { titulo, valor, descripcion, tono = "normal", icono }: Props = $props();

  const Icono = $derived(icono ? iconos[icono] : null);
</script>

<article
  class="metric-card"
  class:up={tono === "up"}
  class:down={tono === "down"}
  class:unknown={tono === "unknown"}
>
  <div class="metric-header">
    <span class="metric-title">{titulo}</span>

    {#if Icono}
      <Icono size={20} strokeWidth={1.8} aria-hidden="true" />
    {/if}
  </div>

  <strong class="metric-value">{valor}</strong>

  <span class="metric-description">
    {descripcion}
  </span>
</article>

<style>
  .metric-card {
    display: flex;
    flex-direction: column;
    gap: 14px;

    padding: 24px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-card);
    background: var(--color-surface);

    box-shadow: var(--shadow-card);

    transition:
      border-color 0.2s ease,
      background-color 0.2s ease;
  }

  .metric-title {
    color: var(--color-muted);
    font-size: 13px;
  }

  .metric-value {
    color: var(--color-text);
    font-size: 36px;
    line-height: 1;
    font-variant-numeric: tabular-nums;
  }

  .metric-description {
    color: var(--color-muted);
    font-size: 12px;
  }

  .up .metric-value {
    color: var(--color-up);
  }

  .down .metric-value {
    color: var(--color-down);
  }

  .unknown .metric-value {
    color: var(--color-unknown);
  }

  .metric-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .metric-header :global(svg) {
    flex-shrink: 0;
    color: var(--color-primary);
    opacity: 0.85;
  }
</style>

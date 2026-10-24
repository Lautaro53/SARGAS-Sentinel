
<script lang="ts">
  type Tono = 'normal' | 'up' | 'down' | 'unknown'

  type Props = {
    titulo: string
    valor: number
    descripcion: string
    tono?: Tono
  }

  let {
    titulo,
    valor,
    descripcion,
    tono = 'normal'
  }: Props = $props()
</script>

<article
  class="metric-card"
  class:up={tono === 'up'}
  class:down={tono === 'down'}
  class:unknown={tono === 'unknown'}
>
  <span class="metric-title">{titulo}</span>

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
    border-radius: 12px;
    background: var(--color-surface);
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
</style>

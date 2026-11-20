<script lang="ts">
    type Props = {
        pagina: number;
        totalPaginas: number;
        onCambiarPagina: (pagina: number) => void;
    };

    let {
        pagina,
        totalPaginas,
        onCambiarPagina
    }: Props = $props();
</script>

<nav class="pagination" aria-label="Paginación de servicios">
    <span class="pagination-info">
        Página {pagina} de {totalPaginas}
    </span>

    <div class="pagination-actions">
        <button
            type="button"
            disabled={pagina <= 1}
            onclick={() => onCambiarPagina(pagina - 1)}
        >
            Anterior
        </button>

        <button
            type="button"
            disabled={pagina >= totalPaginas}
            onclick={() => onCambiarPagina(pagina + 1)}
        >
            Siguiente
        </button>
    </div>
</nav>

<style>
    .pagination {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: 12px;
        padding: 12px 16px;
        border-top: 1px solid var(--color-border);
    }

    .pagination-info {
        color: var(--color-muted);
        font-size: 12px;
    }

    .pagination-actions {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .pagination-actions button {
        min-height: 34px;
        padding: 7px 12px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface-elevated);
        color: var(--color-text);
        font: inherit;
        font-size: 12px;
        cursor: pointer;
    }

    .pagination-actions button:hover:not(:disabled) {
        border-color: var(--color-primary);
        color: var(--color-primary-hover);
    }

    .pagination-actions button:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }

    .pagination-actions button:focus-visible {
        outline: 2px solid var(--color-focus);
        outline-offset: 2px;
    }
</style>
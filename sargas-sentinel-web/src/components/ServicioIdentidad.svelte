<script lang="ts">
    import { Globe } from "@lucide/svelte";

    type Props = {
        id: number;
        nombre: string;
        url: string;
        mostrarUrl?: boolean;
        onVerDetalle: () => void;
    };

    let {
        id,
        nombre,
        url,
        mostrarUrl = false,
        onVerDetalle
    }: Props = $props();

    function navegar(evento: MouseEvent) {
        if (
            evento.defaultPrevented ||
            evento.button !== 0 ||
            evento.ctrlKey ||
            evento.metaKey ||
            evento.shiftKey ||
            evento.altKey
        ) {
            return;
        }

        evento.preventDefault();
        onVerDetalle();
    }
</script>

<div class="service-identity">
    <div class="service-icon" aria-hidden="true">
        <Globe size={19} strokeWidth={1.8} />
    </div>

    <div class="service-info">
        <a
            class="service-name"
            href={`/servicios/${id}`}
            title={mostrarUrl ? nombre : url}
            onclick={navegar}
        >
            {nombre}
        </a>

        {#if mostrarUrl}
            <span class="service-url" title={url}>
                {url}
            </span>
        {/if}
    </div>
</div>

<style>
    .service-identity {
        display: flex;
        align-items: center;
        gap: 12px;
        min-width: 0;
    }

    .service-icon {
        display: grid;
        place-items: center;
        flex: 0 0 32px;
        width: 32px;
        height: 32px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface-elevated);
        color: var(--color-primary);
    }

    .service-info {
        min-width: 0;
    }

    .service-name {
        display: block;
        width: fit-content;
        max-width: 100%;
        color: var(--color-text);
        font-size: 13px;
        font-weight: 600;
        line-height: 1.4;
        text-decoration: none;
        overflow-wrap: anywhere;
    }

    .service-name:hover {
        color: var(--color-primary-hover);
        text-decoration: none;
    }

    .service-name:focus-visible {
        outline: 2px solid var(--color-focus);
        outline-offset: 3px;
        border-radius: 2px;
    }

    .service-url {
        display: block;
        margin-top: 3px;
        color: var(--color-muted);
        font-size: 11px;
        line-height: 1.4;
        overflow-wrap: anywhere;
    }
</style>
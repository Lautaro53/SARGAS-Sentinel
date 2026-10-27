<script lang="ts">
    import type { Estado } from "../types/servicio";

    type Props = {
        estado: Estado;
    };

    let { estado }: Props = $props();
</script>

<span
    class="status-badge"
    class:up={estado === "UP"}
    class:down={estado === "DOWN"}
    class:unknown={estado === "UNKNOWN"}
>
    <span class="status-dot" aria-hidden="true"></span>
    {estado}
</span>

<style>
    .status-badge {
        display: inline-flex;
        align-items: center;
        gap: 7px;

        width: fit-content;
        padding: 6px 11px;
        border-radius: 7px;

        font-size: 12px;
        font-weight: 600;
    }

    .status-dot {
        width: 7px;
        height: 7px;
        border-radius: 50%;
        background: currentColor;
    }

    .up {
        color: var(--color-up);
        background: color-mix(
            in srgb,
            var(--color-up) 12%,
            var(--color-surface)
        );
    }

    .down {
        color: var(--color-down);
        background: color-mix(
            in srgb,
            var(--color-down) 12%,
            var(--color-surface)
        );
    }

    .unknown {
        color: var(--color-unknown);
        background: color-mix(
            in srgb,
            var(--color-unknown) 12%,
            var(--color-surface)
        );
    }
</style>

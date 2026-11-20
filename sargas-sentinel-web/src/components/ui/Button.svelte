<script lang="ts">
    import type { Snippet } from "svelte";

    type Variante = "primary" | "secondary" | "danger" | "ghost";
    type Tamano = "sm" | "md";

    type Props = {
        children: Snippet;
        variant?: Variante;
        size?: Tamano;
        type?: "button" | "submit" | "reset";
        disabled?: boolean;
        loading?: boolean;
        title?: string;
        ariaLabel?: string;
        onClick?: (evento: MouseEvent) => void;
    };

    let {
        children,
        variant = "secondary",
        size = "md",
        type = "button",
        disabled = false,
        loading = false,
        title,
        ariaLabel,
        onClick,
    }: Props = $props();
</script>

<button
    {type}
    class="button"
    class:primary={variant === "primary"}
    class:secondary={variant === "secondary"}
    class:danger={variant === "danger"}
    class:ghost={variant === "ghost"}
    class:small={size === "sm"}
    disabled={disabled || loading}
    aria-busy={loading}
    aria-label={ariaLabel}
    {title}
    onclick={onClick}
>
    {#if loading}
        <span class="spinner" aria-hidden="true"></span>
    {/if}
    {@render children()}
</button>

<style>
    .button {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        min-height: 40px;
        padding: 0 14px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface);
        color: var(--color-text);
        font: inherit;
        font-size: 13px;
        font-weight: 500;
        cursor: pointer;
        white-space: nowrap;
        transition: background 150ms, border-color 150ms;
    }

    .button.small {
        min-height: 34px;
        padding: 0 10px;
        font-size: 12px;
    }

    .button.primary {
        border-color: var(--color-primary);
        background: var(--color-primary);
        color: var(--color-on-primary);
    }

    .button.primary:hover:not(:disabled) {
        background: var(--color-primary-hover);
        border-color: var(--color-primary-hover);
    }

    .button.secondary:hover:not(:disabled),
    .button.ghost:hover:not(:disabled) {
        background: var(--color-surface-elevated);
        border-color: var(--color-primary-hover);
    }

    .button.danger {
        color: var(--color-down);
    }

    .button.danger:hover:not(:disabled) {
        border-color: var(--color-down);
        background: color-mix(in srgb, var(--color-down) 12%, transparent);
    }

    .button.ghost {
        border-color: transparent;
        background: transparent;
    }

    .button:focus-visible {
        outline: 2px solid var(--color-focus);
        outline-offset: 2px;
    }

    .button:disabled {
        opacity: 0.45;
        cursor: not-allowed;
    }

    .spinner {
        width: 14px;
        height: 14px;
        border: 2px solid currentColor;
        border-right-color: transparent;
        border-radius: 50%;
        animation: spin 800ms linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    @media (prefers-reduced-motion: reduce) {
        .button { transition: none; }
        .spinner { animation: none; }
    }
</style>

<script lang="ts">
    import { Search } from "@lucide/svelte";

    type Props = {
        value: string;
        onChange: (value: string) => void;
        label: string;
        placeholder?: string;
        id?: string;
        disabled?: boolean;
    };

    let {
        value,
        onChange,
        label,
        placeholder = "Buscar...",
        id,
        disabled = false,
    }: Props = $props();

    const generatedId = $props.id();
</script>

<div class="search-field" class:disabled>
    <label class="visually-hidden" for={id ?? generatedId}>{label}</label>
    <Search size={17} aria-hidden="true" />
    <input
        id={id ?? generatedId}
        type="search"
        {placeholder}
        {disabled}
        autocomplete="off"
        {value}
        oninput={(evento) => onChange(evento.currentTarget.value)}
    />
</div>

<style>
    .search-field {
        display: flex;
        align-items: center;
        gap: 10px;
        min-width: 0;
        min-height: 40px;
        padding: 0 12px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface);
        color: var(--color-primary-hover);
    }

    .search-field:focus-within {
        border-color: var(--color-primary-hover);
        box-shadow: 0 0 0 1px color-mix(in srgb, var(--color-primary) 20%, transparent);
    }

    .search-field.disabled { opacity: 0.45; }

    input {
        flex: 1;
        width: 100%;
        min-width: 0;
        height: 38px;
        padding: 0;
        margin: 0;
        border: none;
        outline: none;
        box-shadow: none;
        background: transparent;
        color: var(--color-text);
        font: inherit;
        font-size: 13px;
        appearance: none;
    }

    input::placeholder { color: var(--color-muted); opacity: 1; }
    input::-webkit-search-cancel-button { cursor: pointer; }

    .visually-hidden {
        position: absolute;
        width: 1px;
        height: 1px;
        padding: 0;
        margin: -1px;
        overflow: hidden;
        clip: rect(0, 0, 0, 0);
        white-space: nowrap;
        border: 0;
    }
</style>

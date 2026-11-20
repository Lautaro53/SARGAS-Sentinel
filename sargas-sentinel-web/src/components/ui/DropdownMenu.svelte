<script lang="ts" generics="T extends string">
    import { tick } from "svelte";
    import type { Snippet } from "svelte";
    import { Check, ChevronDown } from "@lucide/svelte";

    type Opcion = { value: T; label: string };
    type Props = {
        label: string;
        value: T;
        options: readonly Opcion[];
        onChange: (value: T) => void;
        triggerText?: string;
        icon?: Snippet;
        disabled?: boolean;
        menuAlign?: "start" | "end";
        onClear?: () => void;
        showClear?: boolean;
        clearLabel?: string;
    };

    let {
        label,
        value,
        options,
        onChange,
        triggerText,
        icon,
        disabled = false,
        menuAlign = "end",
        onClear,
        showClear = false,
        clearLabel = "Limpiar filtros",
    }: Props = $props();

    let abierto = $state(false);
    let contenedor: HTMLDivElement;
    let activador: HTMLButtonElement;
    const idMenu = $props.id();

    let texto = $derived(triggerText ?? options.find((opcion) => opcion.value === value)?.label ?? label);

    async function alternar() {
        if (disabled) return;
        abierto = !abierto;
        if (abierto) {
            await tick();
            contenedor?.querySelector<HTMLButtonElement>(".option")?.focus();
        }
    }

    function seleccionar(value: T) {
        onChange(value);
        abierto = false;
        activador?.focus();
    }

    function limpiar() {
        onClear?.();
        abierto = false;
        activador?.focus();
    }

    function clickExterno(evento: MouseEvent) {
        if (abierto && evento.target instanceof Node && !contenedor?.contains(evento.target)) {
            abierto = false;
        }
    }

    function focoFuera(evento: FocusEvent) {
        if (abierto && evento.relatedTarget instanceof Node && !contenedor?.contains(evento.relatedTarget)) {
            abierto = false;
        }
    }

    function tecla(evento: KeyboardEvent) {
        if (!abierto) return;

        if (evento.key === "Escape") {
            abierto = false;
            activador?.focus();
            evento.preventDefault();
            return;
        }

        if (evento.key !== "ArrowDown" && evento.key !== "ArrowUp") return;
        if (!(evento.target instanceof Node) || !contenedor?.contains(evento.target)) return;

        const botones = Array.from(contenedor.querySelectorAll<HTMLButtonElement>(".option"));
        if (botones.length === 0) return;
        const indice = botones.indexOf(document.activeElement as HTMLButtonElement);
        const siguiente = evento.key === "ArrowDown"
            ? (indice + 1) % botones.length
            : (indice - 1 + botones.length) % botones.length;
        botones[siguiente].focus();
        evento.preventDefault();
    }
</script>

<svelte:window onclick={clickExterno} onkeydown={tecla} />

<div class="dropdown" class:align-start={menuAlign === "start"} bind:this={contenedor} onfocusout={focoFuera}>
    <button
        bind:this={activador}
        type="button"
        class="trigger"
        aria-label={`${label}: ${texto}`}
        aria-expanded={abierto}
        aria-controls={idMenu}
        disabled={disabled}
        onclick={() => void alternar()}
    >
        {#if icon}
            <span class="icon" aria-hidden="true">{@render icon()}</span>
        {/if}
        <span>{texto}</span>
        <ChevronDown size={14} aria-hidden="true" />
    </button>

    {#if abierto}
        <div id={idMenu} class="options" role="group" aria-label={label}>
            {#each options as opcion (opcion.value)}
                <button
                    type="button"
                    class="option"
                    class:seleccionado={opcion.value === value}
                    aria-pressed={opcion.value === value}
                    onclick={() => seleccionar(opcion.value)}
                >
                    <span>{opcion.label}</span>
                    {#if opcion.value === value}
                        <Check size={15} aria-hidden="true" />
                    {/if}
                </button>
            {/each}
            {#if onClear && showClear}
                <div class="divider"></div>
                <button type="button" class="option clear" onclick={limpiar}>
                    {clearLabel}
                </button>
            {/if}
        </div>
    {/if}
</div>

<style>
    .dropdown { position: relative; min-width: 0; }

    .trigger {
        display: inline-flex;
        align-items: center;
        justify-content: space-between;
        gap: 10px;
        min-height: 40px;
        padding: 0 12px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface);
        color: var(--color-text);
        font: inherit;
        font-size: 13px;
        white-space: nowrap;
        cursor: pointer;
    }

    .icon { display: inline-flex; align-items: center; color: var(--color-primary-hover); }
    .trigger:disabled { opacity: .45; cursor: not-allowed; }
    .trigger:focus-visible, .option:focus-visible { outline: 2px solid var(--color-focus); outline-offset: 2px; }

    .options {
        position: absolute;
        top: calc(100% + 6px);
        right: 0;
        z-index: 30;
        min-width: max(100%, 165px);
        padding: 5px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius-control);
        background: var(--color-surface-elevated);
        box-shadow: var(--shadow-card);
    }

    .align-start .options { right: auto; left: 0; }

    .option {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        width: 100%;
        padding: 9px 10px;
        border: none;
        border-radius: 6px;
        background: transparent;
        color: var(--color-text);
        text-align: left;
        font: inherit;
        font-size: 12px;
        white-space: nowrap;
        cursor: pointer;
    }

    .option:hover, .option.seleccionado { background: var(--color-surface); }
    .option.seleccionado, .option.clear { color: var(--color-primary-hover); }
    .divider { height: 1px; margin: 5px 4px; background: var(--color-border); }
</style>

<script lang="ts">
    import type { Comprobacion } from "../types/comprobacion";

    type Props = {
        comprobaciones: Comprobacion[];
    };

    let { comprobaciones }: Props = $props();

    const ultimasDiez = $derived(
        [...comprobaciones]
            .sort(
                (a, b) =>
                    new Date(a.comprobado_en).getTime() -
                    new Date(b.comprobado_en).getTime(),
            )
            .slice(-10),
    );

    const segmentos = $derived([
        ...Array(Math.max(0, 10 - ultimasDiez.length)).fill(null),
        ...ultimasDiez,
    ] as (Comprobacion | null)[]);

    function exitosa(comprobacion: Comprobacion): boolean {
        return (
            comprobacion.error === null &&
            comprobacion.codigo_http !== null &&
            comprobacion.codigo_http >= 200 &&
            comprobacion.codigo_http <= 299
        );
    }

    function descripcion(comprobacion: Comprobacion | null): string {
        if (comprobacion === null) {
            return "Sin comprobación";
        }

        const fecha = new Date(comprobacion.comprobado_en).toLocaleString(
            "es-UY",
        );

        const resultado = exitosa(comprobacion) ? "Exitosa" : "Fallida";

        const detalle = comprobacion.error
            ? comprobacion.error
            : comprobacion.codigo_http !== null
              ? `HTTP ${comprobacion.codigo_http}`
              : "Sin respuesta HTTP";

        return `${fecha} · ${resultado} · ${detalle}`;
    }
</script>

<div class="check-history" aria-label="Últimas cinco comprobaciones">
    {#each segmentos as comprobacion, indice (indice)}
        <span
            class="check-segment"
            class:success={comprobacion !== null && exitosa(comprobacion)}
            class:failure={comprobacion !== null && !exitosa(comprobacion)}
            class:empty={comprobacion === null}
            title={descripcion(comprobacion)}
        ></span>
    {/each}
</div>

<style>
    .check-history {
        display: flex;
        align-items: center;
        gap: 3px;
        width: max-content;
    }

    .check-segment {
        display: block;
        width: 10px;
        height: 18px;
        flex-shrink: 0;
        border-radius: 2px;
    }

    .success {
        background: var(--color-up);
    }

    .failure {
        background: var(--color-down);
    }

    .empty {
        background: var(--color-border);
    }
</style>

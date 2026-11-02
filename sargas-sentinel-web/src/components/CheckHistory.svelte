<script lang="ts">
    import type { Comprobacion } from "../types/comprobacion";

    type Props = {
        comprobaciones: Comprobacion[];
    };

    let { comprobaciones }: Props = $props();

    const ultimasVeinte = $derived(
        [...comprobaciones]
            .sort(
                (a, b) =>
                    new Date(a.comprobado_en).getTime() -
                    new Date(b.comprobado_en).getTime(),
            )
            .slice(-20),
    );

    const segmentos = $derived([
        ...Array(Math.max(0, 20 - ultimasVeinte.length)).fill(null),
        ...ultimasVeinte,
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

        const fecha = new Date(comprobacion.comprobado_en);

        const resultado = exitosa(comprobacion)
            ? "Comprobación exitosa"
            : "Comprobación fallida";

        const detalle = comprobacion.error
            ? comprobacion.error
            : comprobacion.codigo_http !== null
              ? `HTTP ${comprobacion.codigo_http}`
              : "Sin respuesta HTTP";

        const latencia = comprobacion.latencia_ms.toLocaleString("es-UY", {
            minimumFractionDigits: 0,
            maximumFractionDigits: 1,
        });

        return [
            `${resultado} · ${detalle}`,
            `Latencia: ${latencia} ms`,
            `Fecha: ${fecha.toLocaleString("es-UY")}`,
        ].join("\n");
    }
</script>

<div
    class="check-history"
    role="img"
    aria-label="Últimas veinte comprobaciones, ordenadas de la más antigua a la más reciente"
>
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
        gap: 2px;
        width: max-content;
    }

    .check-segment {
        display: block;
        width: 6px;
        height: 14px;
        flex-shrink: 0;
        border-radius: 3px;
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

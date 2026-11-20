<script lang="ts">
    import type { Comprobacion } from "../types/comprobacion";

    type Cantidad = 20 | 50 | 100;

    type Props = {
        comprobaciones: Comprobacion[];
        cantidad: Cantidad;
        ultimaSincronizacion: string | null;
        onCambiarCantidad: (cantidad: Cantidad) => void;
    };

    let {
        comprobaciones,
        cantidad,
        ultimaSincronizacion,
        onCambiarCantidad,
    }: Props = $props();

    let puntoActivo = $state<number | null>(null);

    const IZQUIERDA = 64;
    const DERECHA = 708;
    const ARRIBA = 22;
    const ABAJO = 204;
    const ANCHO = DERECHA - IZQUIERDA;
    const ALTO = ABAJO - ARRIBA;

    function esFallida(c: Comprobacion): boolean {
        return c.error !== null || c.codigo_http === null ||
            c.codigo_http < 200 || c.codigo_http >= 300;
    }

    function pasoLegible(valor: number): number {
        const potencia = 10 ** Math.floor(Math.log10(Math.max(valor, 0.001)));
        const normalizado = valor / potencia;
        const factor = normalizado <= 1 ? 1 : normalizado <= 2 ? 2 :
            normalizado <= 5 ? 5 : 10;
        return factor * potencia;
    }

    function milisegundos(valor: number): string {
        return valor.toLocaleString("es-UY", { maximumFractionDigits: 1 });
    }

    function fechaBreve(valor: string): string {
        return new Date(valor).toLocaleString("es-UY", {
            day: "2-digit", month: "2-digit",
            hour: "2-digit", minute: "2-digit", hour12: false,
        });
    }

    let grafico = $derived.by(() => {
        const ordenadas = [...comprobaciones].sort(
            (a, b) => Date.parse(a.comprobado_en) - Date.parse(b.comprobado_en) || a.id - b.id,
        );
        const latencias = ordenadas.map((c) => c.latencia_ms);
        const maxima = Math.max(0, ...latencias);
        const minima = latencias.length > 0 ? Math.min(...latencias) : 0;
        const promedio = latencias.length > 0
            ? latencias.reduce((suma, valor) => suma + valor, 0) / latencias.length
            : 0;
        const paso = pasoLegible(Math.max(1, maxima) / 4);
        const escalaMaxima = paso * 4;
        const primero = ordenadas.length > 0 ? Date.parse(ordenadas[0].comprobado_en) : 0;
        const ultimo = ordenadas.length > 0 ? Date.parse(ordenadas[ordenadas.length - 1].comprobado_en) : 0;
        const rango = ultimo - primero;

        const puntos = ordenadas.map((c, indice) => ({
            ...c,
            x: IZQUIERDA + ANCHO * (rango > 0
                ? (Date.parse(c.comprobado_en) - primero) / rango
                : ordenadas.length > 1 ? indice / (ordenadas.length - 1) : 0.5),
            y: ABAJO - (c.latencia_ms / escalaMaxima) * ALTO,
            fallida: esFallida(c),
        }));

        const cantidadMarcas = Math.min(5, puntos.length);
        const marcas = Array.from({ length: cantidadMarcas }, (_, i) => {
            const indice = cantidadMarcas === 1 ? 0 : Math.round(i * (puntos.length - 1) / (cantidadMarcas - 1));
            return puntos[indice];
        }).filter((p, i, lista) => lista.findIndex((o) => o.id === p.id) === i);

        const linea = puntos.map((p) => `${p.x},${p.y}`).join(" ");
        const area = puntos.length > 1
            ? `M ${puntos[0].x} ${ABAJO} L ${puntos.map((p) => `${p.x} ${p.y}`).join(" L ")} L ${puntos[puntos.length - 1].x} ${ABAJO} Z`
            : "";

        return { puntos, marcas, linea, area, paso, escalaMaxima, minima, maxima, promedio };
    });

    let destacado = $derived(grafico.puntos.find((p) => p.id === puntoActivo) ?? null);
</script>

<div class="graph-layout">
    <header class="graph-header">
        <div>
            <h2>Evolución de latencia</h2>
            <p>Tiempo de respuesta HTTP · datos reales</p>
        </div>
        <label class="range-picker">
            <span>Mostrar</span>
            <select
                value={cantidad}
                onchange={(evento) => {
                    puntoActivo = null;
                    onCambiarCantidad(Number(evento.currentTarget.value) as Cantidad);
                }}
                aria-label="Cantidad de comprobaciones a graficar"
            >
                <option value={20}>Últimas 20</option>
                <option value={50}>Últimas 50</option>
                <option value={100}>Últimas 100</option>
            </select>
        </label>
    </header>

    <div class="graph-legend">
        <span><i class="legend-line"></i> Latencia (ms)</span>
        <span><i class="legend-error"></i> Comprobación fallida</span>
    </div>

    <div class="graph-scroller">
        <div class="graph-stage">
            <svg
                class="latency-svg"
                viewBox="0 0 732 256"
                role="img"
                aria-label="Gráfica de latencia de {grafico.puntos.length} comprobaciones, ordenadas por fecha"
            >
                <title>Latencia de las comprobaciones del servicio</title>
                <defs>
                    <linearGradient id="sargas-latency-gradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stop-color="var(--color-primary)" stop-opacity="0.30" />
                        <stop offset="100%" stop-color="var(--color-primary)" stop-opacity="0.01" />
                    </linearGradient>
                </defs>

                {#each [0, 1, 2, 3, 4] as indice}
                    {@const y = ARRIBA + indice * ALTO / 4}
                    <line x1={IZQUIERDA} x2={DERECHA} y1={y} y2={y} class="grid-line" />
                    <text x={IZQUIERDA - 12} y={y + 4} text-anchor="end" class="axis-text">
                        {milisegundos(grafico.escalaMaxima - indice * grafico.paso)}
                    </text>
                {/each}
                {#each grafico.marcas as marca, indice}
                    <line x1={marca.x} x2={marca.x} y1={ARRIBA} y2={ABAJO} class="grid-line" />
                    <text
                        x={marca.x}
                        y="233"
                        text-anchor={indice === 0 ? "start" : indice === grafico.marcas.length - 1 ? "end" : "middle"}
                        class="axis-text"
                    >{fechaBreve(marca.comprobado_en)}</text>
                {/each}

                {#if grafico.area}
                    <path d={grafico.area} fill="url(#sargas-latency-gradient)" />
                {/if}
                <polyline
                    points={grafico.linea}
                    fill="none"
                    stroke="var(--color-primary-hover)"
                    stroke-width="2.5"
                    stroke-linejoin="round"
                    stroke-linecap="round"
                />
                {#each grafico.puntos as punto (punto.id)}
                    <circle
                        cx={punto.x}
                        cy={punto.y}
                        r={puntoActivo === punto.id ? 5.5 : cantidad === 100 ? 2.6 : 3.5}
                        fill={punto.fallida ? "var(--color-down)" : "var(--color-primary-hover)"}
                        stroke="var(--color-surface)"
                        stroke-width="1.25"
                        role="button"
                        tabindex="0"
                        aria-label={`Comprobación del ${new Date(punto.comprobado_en).toLocaleString("es-UY")}, ${milisegundos(punto.latencia_ms)} milisegundos, ${punto.fallida ? "fallida" : "exitosa"}`}
                        onpointerenter={() => (puntoActivo = punto.id)}
                        onpointerleave={() => (puntoActivo = null)}
                        onfocus={() => (puntoActivo = punto.id)}
                        onblur={() => (puntoActivo = null)}
                        onclick={() => (puntoActivo = punto.id)}
                        onkeydown={(evento) => {
                            if (evento.key === "Enter" || evento.key === " ") {
                                evento.preventDefault();
                                puntoActivo = punto.id;
                            } else if (evento.key === "Escape") {
                                puntoActivo = null;
                            }
                        }}
                    >
                        <title>{new Date(punto.comprobado_en).toLocaleString("es-UY")} · {milisegundos(punto.latencia_ms)} ms · {punto.error ?? (punto.codigo_http === null ? "Sin respuesta HTTP" : `HTTP ${punto.codigo_http}`)}</title>
                    </circle>
                {/each}
            </svg>
            {#if destacado}
                <div class="graph-tooltip" role="status">
                    <strong>{milisegundos(destacado.latencia_ms)} ms</strong>
                    <span>{new Date(destacado.comprobado_en).toLocaleString("es-UY")}</span>
                    <span class:failed={destacado.fallida}>
                        {destacado.error ?? (destacado.codigo_http === null ? "Sin respuesta HTTP" : `HTTP ${destacado.codigo_http}`)}
                    </span>
                </div>
            {/if}
        </div>
    </div>

    <div class="graph-footer">
        <div class="stats" aria-label="Estadísticas del rango seleccionado">
            <div><small>Mínima</small><strong>{milisegundos(grafico.minima)} ms</strong></div>
            <div><small>Promedio</small><strong>{milisegundos(grafico.promedio)} ms</strong></div>
            <div><small>Máxima</small><strong>{milisegundos(grafico.maxima)} ms</strong></div>
        </div>
        <span class="sync-status">
            {grafico.puntos.length} comprobaciones
            {#if ultimaSincronizacion}
                · actualizado {new Date(ultimaSincronizacion).toLocaleTimeString("es-UY")}
            {/if}
        </span>
    </div>
</div>

<style>
    .graph-layout { min-width: 0; }
    .graph-header { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; }
    .graph-header h2 { margin: 0 0 4px; font-size: 16px; }
    .graph-header p { margin: 0; color: var(--color-muted); font-size: 12px; }
    .range-picker { display: flex; align-items: center; gap: 8px; color: var(--color-muted); font-size: 12px; }
    .range-picker select { min-height: 36px; padding: 7px 10px; border: 1px solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface-elevated); color: var(--color-text); font: inherit; font-size: 12px; cursor: pointer; }
    .range-picker select:focus-visible { outline: 2px solid var(--color-focus); outline-offset: 2px; }
    .graph-legend { display: flex; flex-wrap: wrap; gap: 16px; margin: 16px 0 6px; color: var(--color-muted); font-size: 12px; }
    .graph-legend span { display: inline-flex; align-items: center; gap: 7px; }
    .legend-line { width: 17px; height: 3px; background: var(--color-primary-hover); border-radius: 3px; }
    .legend-error { width: 8px; height: 8px; border-radius: 50%; background: var(--color-down); }
    .graph-scroller { overflow-x: auto; }
    .graph-stage { position: relative; min-width: 590px; }
    .latency-svg { display: block; width: 100%; height: auto; }
    .grid-line { stroke: var(--color-border); stroke-width: 0.8; stroke-dasharray: 3 5; }
    .axis-text { fill: var(--color-muted); font-size: 11px; }
    .graph-tooltip { position: absolute; top: 12px; right: 12px; display: flex; flex-direction: column; gap: 4px; max-width: 240px; padding: 10px 12px; pointer-events: none; border: 1px solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface-elevated); color: var(--color-text); font-size: 12px; box-shadow: var(--shadow-card); }
    .graph-tooltip strong { font-size: 14px; }
    .graph-tooltip span { color: var(--color-muted); overflow-wrap: anywhere; }
    .graph-tooltip .failed { color: var(--color-down); }
    .graph-footer { display: flex; align-items: flex-end; justify-content: space-between; flex-wrap: wrap; gap: 12px; margin-top: 8px; }
    .stats { display: flex; gap: 26px; flex-wrap: wrap; }
    .stats div { display: flex; flex-direction: column; gap: 3px; }
    .stats small, .sync-status { color: var(--color-muted); font-size: 11px; }
    .stats strong { color: var(--color-text); font-size: 14px; font-variant-numeric: tabular-nums; }
    @media (max-width: 600px) { .stats { gap: 16px; } }
</style>

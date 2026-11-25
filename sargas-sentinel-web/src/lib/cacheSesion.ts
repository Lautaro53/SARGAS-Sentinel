// Caché de sesión versionada. 
const PREFIJO = "sargas.cache.v2.";
const VERSION = 2;
const EDAD_MAXIMA_MS = 24 * 60 * 60 * 1000;

export type Copia<T> = { datos: T; sincronizadoEn: string };

export function leerCopia<T>(clave: string, valida: (valor: unknown) => valor is T): Copia<T> | null {
  try {
    const bruto = sessionStorage.getItem(PREFIJO + clave);
    if (!bruto) return null;
    const objeto: unknown = JSON.parse(bruto);
    if (!objeto || typeof objeto !== "object" || Array.isArray(objeto)) return null;
    const c = objeto as Record<string, unknown>;
    if (c.version !== VERSION || typeof c.sincronizadoEn !== "string" || !valida(c.datos)) return null;
    const fecha = Date.parse(c.sincronizadoEn);
    if (!Number.isFinite(fecha) || fecha > Date.now() + 60_000 || Date.now() - fecha > EDAD_MAXIMA_MS) return null;
    return { datos: c.datos, sincronizadoEn: c.sincronizadoEn };
  } catch { return null; }
}

export function guardarCopia<T>(clave: string, datos: T, sincronizadoEn = new Date().toISOString()): string {
  try {
    sessionStorage.setItem(PREFIJO + clave, JSON.stringify({ version: VERSION, sincronizadoEn, datos }));
  } catch { /* Sin almacenamiento disponible, se conservan los datos de la vista. */ }
  return sincronizadoEn;
}

export function invalidarCopias(): void {
  try {
    for (let i = sessionStorage.length - 1; i >= 0; i--) {
      const clave = sessionStorage.key(i);
      if (clave?.startsWith(PREFIJO) || clave === "sargas.dashboard.v1" || clave === "sargas.dashboard.incidentes.v1") {
        sessionStorage.removeItem(clave);
      }
    }
  } catch { /* La falta de almacenamiento no bloquea acciones confirmadas por la API. */ }
}

export function esLista<T>(valor: unknown, valida: (v: unknown) => v is T): valor is T[] {
  return Array.isArray(valor) && valor.every(valida);
}

export function esPagina<T>(valor: unknown, valida: (v: unknown) => v is T, servicioId?: number): valor is { registros: T[]; total: number } {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)) return false;
  const p = valor as Record<string, unknown>;
  return Number.isSafeInteger(p.total) && (p.total as number) >= 0 &&
    esLista(p.registros, valida) && p.registros.length <= 10 &&
    (servicioId === undefined || p.registros.every((v) => (v as { servicio_id?: number }).servicio_id === servicioId));
}

export function mensajeErrorConsulta(error: unknown, mensaje: string): string {
  if (error instanceof TypeError) return "No se pudo conectar con la API.";
  return error instanceof Error ? error.message : mensaje;
}

import type { Comprobacion } from '../types/comprobacion'
import type { Incidente } from '../types/incidente'
import { esComprobacion } from './comprobaciones'
import { validarIncidentes } from './incidentes'

export type CodigoHTTP = 'todos' | 'fallidas' | '2xx' | '3xx' | '4xx' | '5xx' | 'sin_respuesta'
export type OrdenComprobaciones = 'fecha_desc' | 'fecha_asc' | 'latencia_desc' | 'latencia_asc'
export type EstadoIncidentes = 'todos' | 'abiertos' | 'cerrados'
export type OrdenIncidentes = 'fecha_desc' | 'fecha_asc'

export type FiltrosComprobaciones = {
  codigo: CodigoHTTP
  orden: OrdenComprobaciones
  latenciaMin: number | null
  latenciaMax: number | null
}
export type FiltrosIncidentes = { estado: EstadoIncidentes; orden: OrdenIncidentes; desde?: string; hasta?: string }
export type PaginaHistorial<T> = { registros: T[]; total: number }

function validarPagina<T>(datos: unknown, validador: (valor: unknown) => valor is T, limite: number): PaginaHistorial<T> {
  if (typeof datos !== 'object' || datos === null || Array.isArray(datos)) {
    throw new Error('La API no devolvió una página de historial')
  }
  const respuesta = datos as Record<string, unknown>
  if (!Number.isSafeInteger(respuesta.total) || (respuesta.total as number) < 0 ||
      !Array.isArray(respuesta.registros) || respuesta.registros.length > limite ||
      !respuesta.registros.every(validador)) {
    throw new Error('La API devolvió una página de historial inválida')
  }
  return { registros: respuesta.registros as T[], total: respuesta.total as number }
}

async function consultar<T>(ruta: string, params: URLSearchParams, limite: number,
  validador: (dato: unknown) => dato is T, signal?: AbortSignal): Promise<PaginaHistorial<T>> {
  const respuesta = await fetch(`${ruta}?${params.toString()}`, { signal })
  if (!respuesta.ok) throw new Error(`No se pudo consultar el historial: HTTP ${respuesta.status}`)
  return validarPagina(await respuesta.json(), validador, limite)
}

function parametrosPagina(pagina: number, limite: number): URLSearchParams {
  if (!Number.isSafeInteger(pagina) || pagina < 1 || !Number.isSafeInteger(limite) || limite < 1 || limite > 100) {
    throw new Error('Página o tamaño de página inválido')
  }
  const offset = (pagina - 1) * limite
  if (!Number.isSafeInteger(offset) || offset > 2147483647) throw new Error('Offset fuera de rango')
  return new URLSearchParams({ limite: String(limite), offset: String(offset) })
}

export async function consultarHistorialComprobaciones(servicioId: number, pagina: number, limite: number,
  filtros: FiltrosComprobaciones, signal?: AbortSignal): Promise<PaginaHistorial<Comprobacion>> {
  const params = parametrosPagina(pagina, limite)
  params.set('codigo', filtros.codigo)
  params.set('orden', filtros.orden)
  for (const [nombre, valor] of [['latencia_min', filtros.latenciaMin], ['latencia_max', filtros.latenciaMax]] as const) {
    if (valor === null) continue
    if (!Number.isFinite(valor) || valor < 0) throw new Error('Latencia inválida')
    params.set(nombre, String(valor))
  }
  if (filtros.latenciaMin !== null && filtros.latenciaMax !== null && filtros.latenciaMin > filtros.latenciaMax) {
    throw new Error('La latencia mínima no puede superar la máxima')
  }
  return consultar(`/api/servicios/${servicioId}/comprobaciones/consulta`, params, limite,
    (dato): dato is Comprobacion => esComprobacion(dato) && dato.servicio_id === servicioId, signal)
}

export async function consultarHistorialIncidentes(servicioId: number | null, pagina: number, limite: number,
  filtros: FiltrosIncidentes, signal?: AbortSignal): Promise<PaginaHistorial<Incidente>> {
  const params = parametrosPagina(pagina, limite)
  params.set('estado', filtros.estado)
  params.set('orden', filtros.orden)
  const fechaInstantanea = (valor: string, diaSiguiente: boolean): string => {
    const partes = /^(\d{4})-(\d{2})-(\d{2})$/.exec(valor)
    if (!partes) throw new Error('Fecha inválida')
    const fecha = new Date(Number(partes[1]), Number(partes[2]) - 1, Number(partes[3]) + (diaSiguiente ? 1 : 0))
    const original = new Date(Number(partes[1]), Number(partes[2]) - 1, Number(partes[3]))
    if (original.getFullYear() !== Number(partes[1]) ||
        original.getMonth() !== Number(partes[2]) - 1 || original.getDate() !== Number(partes[3])) {
      throw new Error('Fecha inválida')
    }
    return fecha.toISOString()
  }
  if (filtros.desde) params.set('desde', fechaInstantanea(filtros.desde, false))
  if (filtros.hasta) params.set('hasta', fechaInstantanea(filtros.hasta, true))
  if (filtros.desde && filtros.hasta && filtros.desde > filtros.hasta) throw new Error('La fecha inicial supera la final')
  const ruta = servicioId === null ? '/api/incidentes/consulta' : `/api/servicios/${servicioId}/incidentes/consulta`
  const resultado = await consultar(ruta, params, limite,
    (dato): dato is Incidente => {
      try { return validarIncidentes([dato], servicioId === null ? undefined : servicioId).length === 1 }
      catch { return false }
    },
    signal)
  return resultado
}

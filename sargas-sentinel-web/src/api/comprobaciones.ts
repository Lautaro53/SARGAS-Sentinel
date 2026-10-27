
import type { Comprobacion } from '../types/comprobacion'


function esComprobacion(
  valor: unknown
): valor is Comprobacion {
  if (
    typeof valor !== 'object' ||
    valor === null ||
    Array.isArray(valor)
  ) {
    return false
  }

  const c = valor as Record<string, unknown>

  return (
    typeof c.id === 'number' &&
    Number.isSafeInteger(c.id) &&
    c.id > 0 &&

    typeof c.servicio_id === 'number' &&
    Number.isSafeInteger(c.servicio_id) &&
    c.servicio_id > 0 &&

    (
      c.codigo_http === null ||
      (
        typeof c.codigo_http === 'number' &&
        Number.isInteger(c.codigo_http) &&
        c.codigo_http >= 100 &&
        c.codigo_http <= 599
      )
    ) &&

    typeof c.latencia_ms === 'number' &&
    Number.isFinite(c.latencia_ms) &&
    c.latencia_ms >= 0 &&

    (
      c.error === null ||
      typeof c.error === 'string'
    ) &&

    typeof c.comprobado_en === 'string' &&
    !Number.isNaN(Date.parse(c.comprobado_en))
  )
}


export async function obtenerComprobaciones(
  servicioId: number,
  limite: number = 20
): Promise<Comprobacion[]> {

  if (!Number.isInteger(limite) || limite < 1 || limite > 100) {
    throw new Error('El límite debe estar entre 1 y 100')
  }

  const respuesta = await fetch(
    `/api/servicios/${servicioId}/comprobaciones?limite=${limite}&offset=0`
  )

  if (!respuesta.ok) {
    throw new Error(
      `No se pudieron consultar las comprobaciones: HTTP ${respuesta.status}`
    )
  }

  const datos: unknown = await respuesta.json()

  if (!Array.isArray(datos)) {
    throw new Error(
      'La API no devolvió una lista de comprobaciones'
    )
  }


  if (
    !datos.every(
      (dato: unknown) =>
        esComprobacion(dato) &&
        dato.servicio_id === servicioId
    )
  ) {
    throw new Error(
      'La API devolvió comprobaciones con datos inválidos'
    )
  }

  return datos

}

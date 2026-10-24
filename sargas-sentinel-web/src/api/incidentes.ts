import type { Incidente } from "../types/incidente";


function esIncidente(valor: unknown): valor is Incidente {
  if (
    typeof valor !== 'object' ||
    valor === null ||
    Array.isArray(valor)
  ) {
    return false
  }

  const i = valor as Record<string, unknown>

  const inicio =
    typeof i.iniciado_en === 'string'
      ? Date.parse(i.iniciado_en)
      : NaN

  const cierre =
    typeof i.cerrado_en === 'string'
      ? Date.parse(i.cerrado_en)
      : NaN

  return (
    typeof i.id === 'number' &&
    Number.isSafeInteger(i.id) &&
    i.id > 0 &&

    typeof i.servicio_id === 'number' &&
    Number.isSafeInteger(i.servicio_id) &&
    i.servicio_id > 0 &&

    Number.isFinite(inicio) &&

    (
      i.cerrado_en === null ||
      (
        typeof i.cerrado_en === 'string' &&
        Number.isFinite(cierre) &&
        cierre >= inicio
      )
    ) &&

    typeof i.evidencia_inicio === 'string' &&

    (
      i.evidencia_cierre === null ||
      typeof i.evidencia_cierre === 'string'
    ) &&

    // Fecha y evidencia de cierre deben
    // existir juntas o ser ambas null.
    (
      (i.cerrado_en === null) ===
      (i.evidencia_cierre === null)
    ) &&

    typeof i.nombre_servicio === 'string' &&
    typeof i.servicio_archivado === 'boolean'
  )
}

function validarIncidentes(
  datos: unknown,
  servicioId?: number
): Incidente[] {
  if (!Array.isArray(datos)) {
    throw new Error(
      'La API no devolvió una lista de incidentes'
    )
  }

  const incidentes: unknown[] = datos

  if (!incidentes.every(esIncidente)) {
    throw new Error(
      'La API devolvió incidentes con datos inválidos'
    )
  }

  if (
    servicioId !== undefined &&
    incidentes.some(i => i.servicio_id !== servicioId)
  ) {
    throw new Error(
      'La API devolvió incidentes de otro servicio'
    )
  }

  return incidentes
}


export async function obtenerIncidentes(
  servicioId: number
): Promise<Incidente[]> {
  const respuesta = await fetch(
    `/api/servicios/${servicioId}/incidentes?limite=10&offset=0`
  );

  if (!respuesta.ok) {
    throw new Error(
      `No se pudieron consultar los incidentes: HTTP ${respuesta.status}`
    );
  }

  const datos: unknown = await respuesta.json();

  return validarIncidentes(datos, servicioId)
}

export async function obtenerTodosIncidentes(): Promise<Incidente[]> {
  const respuesta = await fetch(
    "/api/incidentes?limite=50&offset=0"
  );

  if (!respuesta.ok) {
    throw new Error(
      `No se pudieron consultar los incidentes: HTTP ${respuesta.status}`
    );
  }

  const datos: unknown = await respuesta.json();

  return validarIncidentes(datos)
}
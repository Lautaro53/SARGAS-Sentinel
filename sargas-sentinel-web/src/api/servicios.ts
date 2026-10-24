
import type { Servicio, EstadoServicio } from '../types/servicio'


export async function obtenerServicios(): Promise<Servicio[]> {
  const respuesta = await fetch('/api/servicios')

  if (!respuesta.ok) {
    throw new Error(
      `Error al consultar servicios: HTTP ${respuesta.status}`
    )
  }

  const datos: unknown = await respuesta.json()

  if (!Array.isArray(datos)) {
    throw new Error('La API no devolvió una lista de servicios')
  }

  if (!datos.every(esServicio)) {
    throw new Error(
      'La API devolvió servicios con datos inválidos'
    )
  }

  return datos
}


export async function obtenerEstadoServicio(
  id: number
): Promise<EstadoServicio> {
  const respuesta = await fetch(`/api/servicios/${id}/estado`)

  if (!respuesta.ok) {
    throw new Error(
      `No se pudo consultar el estado del servicio ${id}: HTTP ${respuesta.status}`
    )
  }


  const datos: unknown = await respuesta.json()

  if (
    !esEstadoServicio(datos) ||
    datos.servicio_id !== id
  ) {
    throw new Error(
      `La API devolvió un estado inválido para el servicio ${id}`
    )
  }

  return datos

}


type DatosServicio = Pick<
  Servicio,
  'nombre' | 'url' | 'intervalo_segundos' | 'timeout_segundos'
>

export async function crearServicio(
  datos: DatosServicio
): Promise<Servicio> {
  const respuesta = await fetch('/api/servicios', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(datos)
  })

  if (!respuesta.ok) {
    const detalle = (await respuesta.text()).trim()

    if (respuesta.status === 400) {
      throw new Error(detalle || 'Datos del servicio inválidos')
    }

    throw new Error(
      `No se pudo registrar el servicio: HTTP ${respuesta.status}`
    )
  }

  const recibido: unknown = await respuesta.json()

  if (!esServicio(recibido)) {
    throw new Error(
      'La API devolvió un servicio creado con datos inválidos'
    )
  }

  return recibido
}


export async function actualizarServicio(
  id: number,
  datos: DatosServicio
): Promise<Servicio> {
  const respuesta = await fetch(`/api/servicios/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(datos)
  })

  if (!respuesta.ok) {
    const detalle = (await respuesta.text()).trim()

    if (respuesta.status === 400) {
      throw new Error(detalle || 'Datos del servicio inválidos')
    }

    if (respuesta.status === 404) {
      throw new Error('El servicio ya no existe')
    }

    throw new Error(
      `No se pudo actualizar el servicio: HTTP ${respuesta.status}`
    )
  }

  const recibido: unknown = await respuesta.json()

  if (!esServicio(recibido) || recibido.id !== id) {
    throw new Error(
      'La API devolvió un servicio actualizado con datos inválidos'
    )
  }

  return recibido
}


export async function cambiarHabilitacion(
  id: number,
  habilitado: boolean
): Promise<void> {
  const respuesta = await fetch(
    `/api/servicios/${id}/habilitacion`,
    {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ habilitado })
    }
  )

  if (!respuesta.ok) {
    if (respuesta.status === 404) {
      throw new Error('El servicio no existe')
    }

    const detalle = (await respuesta.text()).trim()

    throw new Error(
      detalle ||
      `No se pudo cambiar el monitoreo: HTTP ${respuesta.status}`
    )
  }
}

export async function archivarServicio(
  id: number
): Promise<void> {
  const respuesta = await fetch(
    `/api/servicios/${id}/archivar`,
    {
      method: "PATCH"
    }
  );

  if (!respuesta.ok) {
    if (respuesta.status === 404) {
      throw new Error(
        "El servicio no existe o ya fue archivado"
      );
    }

    if (respuesta.status === 503) {
      throw new Error(
        "El servicio fue archivado, pero Go no pudo " +
        "sincronizar el monitoreo. Revisá el backend."
      );
    }

    throw new Error(
      `No se pudo archivar el servicio: HTTP ${respuesta.status}`
    );
  }
}


export function esServicio(valor: unknown): valor is Servicio {
  if (
    typeof valor !== 'object' ||
    valor === null ||
    Array.isArray(valor)
  ) {
    return false
  }

  const s = valor as Record<string, unknown>

  return (
    typeof s.id === 'number' &&
    Number.isSafeInteger(s.id) &&
    s.id > 0 &&
    typeof s.nombre === 'string' &&
    s.nombre.trim().length > 0 &&
    typeof s.url === 'string' &&
    s.url.trim().length > 0 &&
    s.tipo === 'HTTP' &&
    typeof s.intervalo_segundos === 'number' &&
    Number.isFinite(s.intervalo_segundos) &&
    s.intervalo_segundos > 0 &&
    typeof s.timeout_segundos === 'number' &&
    Number.isFinite(s.timeout_segundos) &&
    s.timeout_segundos > 0 &&
    typeof s.habilitado === 'boolean'
  )
}


export function esEstadoServicio(
  valor: unknown
): valor is EstadoServicio {
  if (
    typeof valor !== 'object' ||
    valor === null ||
    Array.isArray(valor)
  ) {
    return false
  }

  const e = valor as Record<string, unknown>

  return (
    typeof e.servicio_id === 'number' &&
    Number.isSafeInteger(e.servicio_id) &&
    e.servicio_id > 0 &&

    (e.estado === 'UP' ||
      e.estado === 'DOWN' ||
      e.estado === 'UNKNOWN') &&

    typeof e.habilitado === 'boolean' &&

    typeof e.exitos_consecutivos === 'number' &&
    Number.isSafeInteger(e.exitos_consecutivos) &&
    e.exitos_consecutivos >= 0 &&

    typeof e.fallos_consecutivos === 'number' &&
    Number.isSafeInteger(e.fallos_consecutivos) &&
    e.fallos_consecutivos >= 0 &&

    (
      e.ultima_comprobacion_en === null ||
      (
        typeof e.ultima_comprobacion_en === 'string' &&
        !Number.isNaN(Date.parse(e.ultima_comprobacion_en))
      )
    )
  )
}

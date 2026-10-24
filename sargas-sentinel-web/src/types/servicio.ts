
export type Estado = 'UNKNOWN' | 'UP' | 'DOWN'

export interface Servicio {
  id: number
  nombre: string
  url: string
  tipo: 'HTTP'
  intervalo_segundos: number
  timeout_segundos: number
  habilitado: boolean
}


export interface EstadoServicio {
  servicio_id: number
  estado: Estado
  habilitado: boolean
  exitos_consecutivos: number
  fallos_consecutivos: number
  ultima_comprobacion_en: string | null
}


export interface Comprobacion {
  id: number
  servicio_id: number
  codigo_http: number | null
  latencia_ms: number
  error: string | null
  comprobado_en: string
}

export interface Incidente {
  id: number;
  servicio_id: number;
  iniciado_en: string;
  cerrado_en: string | null;
  evidencia_inicio: string;
  evidencia_cierre: string | null;
  nombre_servicio: string;
  servicio_archivado: boolean;
}
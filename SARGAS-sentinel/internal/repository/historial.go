package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type FiltroHistorialComprobaciones struct {
	Codigo      string
	Orden       string
	LatenciaMin *float64
	LatenciaMax *float64
}

type FiltroHistorialIncidentes struct {
	Estado string
	Orden  string
	Desde  *time.Time
	Hasta  *time.Time
}

// ConsultarHistorial lista y cuenta los registros del mismo conjunto filtrado.
func (r *ComprobacionRepository) ConsultarHistorial(
	ctx context.Context, servicioID int64, limite, offset int32, filtro FiltroHistorialComprobaciones,
) ([]ComprobacionRegistro, int64, error) {
	var existe bool
	if err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM servicios WHERE id = $1)", servicioID).Scan(&existe); err != nil {
		return nil, 0, fmt.Errorf("verificar servicio: %w", err)
	}
	if !existe {
		return nil, 0, pgx.ErrNoRows
	}

	condicion := `FROM comprobaciones c
        WHERE c.servicio_id = $1
          AND (
            $2::text = 'todos'
            OR ($2::text = 'fallidas' AND (c.error IS NOT NULL OR c.codigo_http IS NULL OR c.codigo_http < 200 OR c.codigo_http >= 300))
            OR ($2::text = 'sin_respuesta' AND c.codigo_http IS NULL)
            OR ($2::text = '2xx' AND c.codigo_http BETWEEN 200 AND 299)
            OR ($2::text = '3xx' AND c.codigo_http BETWEEN 300 AND 399)
            OR ($2::text = '4xx' AND c.codigo_http BETWEEN 400 AND 499)
            OR ($2::text = '5xx' AND c.codigo_http BETWEEN 500 AND 599)
          )
          AND ($3::double precision IS NULL OR c.latencia_ms >= $3)
          AND ($4::double precision IS NULL OR c.latencia_ms <= $4)`
	args := []any{servicioID, filtro.Codigo, filtro.LatenciaMin, filtro.LatenciaMax}
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) "+condicion, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("contar comprobaciones: %w", err)
	}
	ordenes := map[string]string{
		"fecha_desc":    "c.comprobado_en DESC, c.id DESC",
		"fecha_asc":     "c.comprobado_en ASC, c.id ASC",
		"latencia_desc": "c.latencia_ms DESC, c.comprobado_en DESC, c.id DESC",
		"latencia_asc":  "c.latencia_ms ASC, c.comprobado_en DESC, c.id DESC",
	}
	orden, ok := ordenes[filtro.Orden]
	if !ok {
		return nil, 0, fmt.Errorf("orden de comprobaciones no permitido")
	}
	query := `SELECT c.id, c.servicio_id, c.codigo_http, c.latencia_ms, c.error, c.comprobado_en ` + condicion +
		" ORDER BY " + orden + " LIMIT $5 OFFSET $6"
	rows, err := r.pool.Query(ctx, query, append(args, limite, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("consultar historial de comprobaciones: %w", err)
	}
	defer rows.Close()
	registros := make([]ComprobacionRegistro, 0)
	for rows.Next() {
		var c ComprobacionRegistro
		if err := rows.Scan(&c.ID, &c.ServicioID, &c.CodigoHTTP, &c.LatenciaMS, &c.Error, &c.ComprobadoEn); err != nil {
			return nil, 0, fmt.Errorf("leer comprobacion: %w", err)
		}
		registros = append(registros, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("recorrer comprobaciones: %w", err)
	}
	return registros, total, nil
}

func (r *ConsultaRepository) ConsultarHistorialIncidentes(
	ctx context.Context, servicioID int64, limite, offset int32, filtro FiltroHistorialIncidentes,
) ([]IncidenteRegistro, int64, error) {
	if servicioID != 0 {
		if _, err := r.ObtenerEstado(ctx, servicioID); err != nil {
			return nil, 0, fmt.Errorf("verificar servicio: %w", err)
		}
	}
	condicion := `FROM incidentes i INNER JOIN servicios s ON s.id = i.servicio_id
        WHERE ($1::bigint = 0 OR i.servicio_id = $1)
          AND ($2::text = 'todos'
            OR ($2::text = 'abiertos' AND i.cerrado_en IS NULL)
            OR ($2::text = 'cerrados' AND i.cerrado_en IS NOT NULL))
          AND ($3::timestamptz IS NULL OR i.iniciado_en >= $3::timestamptz)
          AND ($4::timestamptz IS NULL OR i.iniciado_en < $4::timestamptz)`
	args := []any{servicioID, filtro.Estado, filtro.Desde, filtro.Hasta}
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) "+condicion, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("contar incidentes: %w", err)
	}
	ordenes := map[string]string{
		"fecha_desc": "i.iniciado_en DESC, i.id DESC",
		"fecha_asc":  "i.iniciado_en ASC, i.id ASC",
	}
	orden, ok := ordenes[filtro.Orden]
	if !ok {
		return nil, 0, fmt.Errorf("orden de incidentes no permitido")
	}
	query := `SELECT i.id, i.servicio_id, i.iniciado_en, i.cerrado_en,
        i.evidencia_inicio, i.evidencia_cierre, s.nombre,
        (s.archivado_en IS NOT NULL) ` + condicion + " ORDER BY " + orden + " LIMIT $5 OFFSET $6"
	rows, err := r.pool.Query(ctx, query, append(args, limite, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("consultar historial de incidentes: %w", err)
	}
	defer rows.Close()
	registros := make([]IncidenteRegistro, 0)
	for rows.Next() {
		var i IncidenteRegistro
		if err := rows.Scan(&i.ID, &i.ServicioID, &i.IniciadoEn, &i.CerradoEn,
			&i.EvidenciaInicio, &i.EvidenciaCierre, &i.NombreServicio, &i.ServicioArchivado); err != nil {
			return nil, 0, fmt.Errorf("leer incidente: %w", err)
		}
		registros = append(registros, i)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("recorrer incidentes: %w", err)
	}
	return registros, total, nil
}

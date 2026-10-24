package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EstadoActual struct {
	ServicioID         int64
	Estado             string
	Habilitado         bool
	ExitosConsecutivos int
	FallosConsecutivos int
	UltimaComprobacion *time.Time
}

type ConsultaRepository struct {
	pool *pgxpool.Pool
}

type IncidenteRegistro struct {
	ID                int64
	ServicioID        int64
	IniciadoEn        time.Time
	CerradoEn         *time.Time
	EvidenciaInicio   string
	EvidenciaCierre   *string
	NombreServicio    string
	ServicioArchivado bool
}

func (r *ConsultaRepository) ListarIncidentes(
	ctx context.Context,
	servicioID int64,
	limite int32,
	offset int32,
) ([]IncidenteRegistro, error) {

	// Verificar que el servicio exista.
	_, err := r.ObtenerEstado(ctx, servicioID)
	if err != nil {
		return nil, fmt.Errorf(
			"verificar servicio: %w", err,
		)
	}

	query := `
			SELECT
				i.id,
				i.servicio_id,
				i.iniciado_en,
				i.cerrado_en,
				i.evidencia_inicio,
				i.evidencia_cierre,
				s.nombre,
				(s.archivado_en IS NOT NULL)
			FROM incidentes i
			INNER JOIN servicios s
				ON s.id = i.servicio_id
			WHERE i.servicio_id = $1
			ORDER BY i.iniciado_en DESC, i.id DESC
			LIMIT $2 OFFSET $3
		`

	rows, err := r.pool.Query(
		ctx,
		query,
		servicioID,
		limite,
		offset,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"consultar incidentes: %w", err,
		)
	}
	defer rows.Close()

	incidentes := make([]IncidenteRegistro, 0)

	for rows.Next() {
		var incidente IncidenteRegistro

		err := rows.Scan(
			&incidente.ID,
			&incidente.ServicioID,
			&incidente.IniciadoEn,
			&incidente.CerradoEn,
			&incidente.EvidenciaInicio,
			&incidente.EvidenciaCierre,
			&incidente.NombreServicio,
			&incidente.ServicioArchivado,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"leer incidente: %w", err,
			)
		}

		incidentes = append(incidentes, incidente)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"recorrer incidentes: %w", err,
		)
	}

	return incidentes, nil
}

func NuevoConsultaRepository(
	pool *pgxpool.Pool,
) *ConsultaRepository {
	return &ConsultaRepository{pool: pool}
}

func (r *ConsultaRepository) ObtenerEstado(
	ctx context.Context,
	servicioID int64,
) (EstadoActual, error) {

	var estado EstadoActual

	query := `
		SELECT
			s.id,
			COALESCE(e.estado, 'UNKNOWN'),
			s.habilitado,
			COALESCE(e.exitos_consecutivos, 0),
			COALESCE(e.fallos_consecutivos, 0),
			e.ultima_comprobacion_en
		FROM servicios s
		LEFT JOIN estados_servicio e
			ON e.servicio_id = s.id
		WHERE s.id = $1
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		servicioID,
	).Scan(
		&estado.ServicioID,
		&estado.Estado,
		&estado.Habilitado,
		&estado.ExitosConsecutivos,
		&estado.FallosConsecutivos,
		&estado.UltimaComprobacion,
	)

	if err != nil {
		return estado, fmt.Errorf(
			"consultar estado del servicio: %w",
			err,
		)
	}

	return estado, nil
}

func (r *ConsultaRepository) ListarTodosIncidentes(
	ctx context.Context,
	limite int32,
	offset int32,
) ([]IncidenteRegistro, error) {

	query := `
		SELECT
			i.id,
			i.servicio_id,
			i.iniciado_en,
			i.cerrado_en,
			i.evidencia_inicio,
			i.evidencia_cierre,
			s.nombre,
			(s.archivado_en IS NOT NULL)
		FROM incidentes i
		INNER JOIN servicios s
			ON s.id = i.servicio_id
		ORDER BY i.iniciado_en DESC, i.id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(
		ctx,
		query,
		limite,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"consultar incidentes globales: %w", err,
		)
	}
	defer rows.Close()

	incidentes := make([]IncidenteRegistro, 0)

	for rows.Next() {
		var incidente IncidenteRegistro

		err := rows.Scan(
			&incidente.ID,
			&incidente.ServicioID,
			&incidente.IniciadoEn,
			&incidente.CerradoEn,
			&incidente.EvidenciaInicio,
			&incidente.EvidenciaCierre,
			&incidente.NombreServicio,
			&incidente.ServicioArchivado,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"leer incidente global: %w", err,
			)
		}

		incidentes = append(incidentes, incidente)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"recorrer incidentes globales: %w", err,
		)
	}

	return incidentes, nil
}

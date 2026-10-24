package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Comprobacion struct {
	ServicioID int64
	CodigoHTTP *int32
	Latencia   time.Duration
	Error      *string
}

type ComprobacionRepository struct {
	pool *pgxpool.Pool
}

func NuevoComprobacionRepository(
	pool *pgxpool.Pool,
) *ComprobacionRepository {
	return &ComprobacionRepository{pool: pool}
}

type ComprobacionRegistro struct {
	ID           int64
	ServicioID   int64
	CodigoHTTP   *int32
	LatenciaMS   float64
	Error        *string
	ComprobadoEn time.Time
}

func (r *ComprobacionRepository) Guardar(
	ctx context.Context,
	c Comprobacion,
) error {

	query := `
        INSERT INTO comprobaciones (
            servicio_id,
            codigo_http,
            latencia_ms,
            error
        )
        VALUES ($1, $2, $3, $4)
    `

	latenciaMS := float64(c.Latencia) / float64(time.Millisecond)

	_, err := r.pool.Exec(
		ctx,
		query,
		c.ServicioID,
		c.CodigoHTTP,
		latenciaMS,
		c.Error,
	)

	if err != nil {
		return fmt.Errorf("guardar comprobacion: %w", err)
	}

	return nil
}

func (r *ComprobacionRepository) ListarPorServicio(
	ctx context.Context,
	servicioID int64,
	limite int32,
	offset int32,
) ([]ComprobacionRegistro, error) {

	var existe bool

	err := r.pool.QueryRow(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM servicios WHERE id = $1)",
		servicioID,
	).Scan(&existe)

	if err != nil {
		return nil, fmt.Errorf("verificar servicio: %w", err)
	}

	if !existe {
		return nil, pgx.ErrNoRows
	}

	query := `
        SELECT
            id,
            servicio_id,
            codigo_http,
            latencia_ms,
            error,
            comprobado_en
        FROM comprobaciones
        WHERE servicio_id = $1
        ORDER BY comprobado_en DESC, id DESC
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
		return nil, fmt.Errorf("consultar comprobaciones: %w", err)
	}
	defer rows.Close()

	registros := make([]ComprobacionRegistro, 0)

	for rows.Next() {
		var c ComprobacionRegistro

		err := rows.Scan(
			&c.ID,
			&c.ServicioID,
			&c.CodigoHTTP,
			&c.LatenciaMS,
			&c.Error,
			&c.ComprobadoEn,
		)

		if err != nil {
			return nil, fmt.Errorf("leer comprobacion: %w", err)
		}

		registros = append(registros, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer comprobaciones: %w", err)
	}

	return registros, nil
}

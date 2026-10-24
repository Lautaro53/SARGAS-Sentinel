package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sargas-sentinel/internal/service"
)

type ServicioRepository struct {
	pool *pgxpool.Pool
}

func NuevoServicioRepository(
	pool *pgxpool.Pool,
) *ServicioRepository {
	return &ServicioRepository{pool: pool}
}

func (r *ServicioRepository) Listar(
	ctx context.Context,
) ([]service.Servicio, error) {

	query := `
        SELECT
            id,
            nombre,
            url,
            tipo,
            intervalo_segundos,
            timeout_segundos,
            habilitado
			FROM servicios
			WHERE archivado_en IS NULL
			ORDER BY id
			`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("consultar servicios: %w", err)
	}
	defer rows.Close()

	servicios := make([]service.Servicio, 0)

	for rows.Next() {
		var s service.Servicio
		var intervalo, timeout int32

		err := rows.Scan(
			&s.ID,
			&s.Nombre,
			&s.URL,
			&s.Tipo,
			&intervalo,
			&timeout,
			&s.Habilitado,
		)

		if err != nil {
			return nil, fmt.Errorf("leer servicio: %w", err)
		}

		s.Intervalo = time.Duration(intervalo) * time.Second
		s.Timeout = time.Duration(timeout) * time.Second

		servicios = append(servicios, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer servicios: %w", err)
	}

	return servicios, nil
}

func (r *ServicioRepository) Crear(
	ctx context.Context,
	s service.Servicio,
) (service.Servicio, error) {

	query := `
        INSERT INTO servicios (
            nombre,
            url,
            tipo,
            intervalo_segundos,
            timeout_segundos,
            habilitado
        )
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `

	err := r.pool.QueryRow(
		ctx,
		query,
		s.Nombre,
		s.URL,
		s.Tipo,
		int32(s.Intervalo/time.Second),
		int32(s.Timeout/time.Second),
		s.Habilitado,
	).Scan(&s.ID)

	if err != nil {
		return s, fmt.Errorf("crear servicio: %w", err)
	}

	return s, nil
}

func (r *ServicioRepository) Actualizar(
	ctx context.Context,
	s service.Servicio,
) (service.Servicio, error) {

	query := `
        UPDATE servicios
        SET nombre = $2,
            url = $3,
            intervalo_segundos = $4,
            timeout_segundos = $5
        WHERE id = $1 AND archivado_en IS NULL
        RETURNING tipo, habilitado
    `

	err := r.pool.QueryRow(
		ctx,
		query,
		s.ID,
		s.Nombre,
		s.URL,
		int32(s.Intervalo/time.Second),
		int32(s.Timeout/time.Second),
	).Scan(&s.Tipo, &s.Habilitado)

	if err != nil {
		return s, fmt.Errorf("actualizar servicio: %w", err)
	}

	return s, nil
}

func (r *ServicioRepository) CambiarHabilitacion(
	ctx context.Context,
	id int64,
	habilitado bool,
) (service.Servicio, error) {

	query := `
        UPDATE servicios
        SET habilitado = $2
        WHERE id = $1 AND archivado_en IS NULL
        RETURNING
            id, nombre, url, tipo,
            intervalo_segundos, timeout_segundos,
            habilitado
    `

	var s service.Servicio
	var intervalo, timeout int32

	err := r.pool.QueryRow(
		ctx, query, id, habilitado,
	).Scan(
		&s.ID,
		&s.Nombre,
		&s.URL,
		&s.Tipo,
		&intervalo,
		&timeout,
		&s.Habilitado,
	)

	if err != nil {
		return s, fmt.Errorf("cambiar habilitacion: %w", err)
	}

	s.Intervalo = time.Duration(intervalo) * time.Second
	s.Timeout = time.Duration(timeout) * time.Second

	return s, nil
}

func (r *ServicioRepository) Archivar(
	ctx context.Context,
	id int64,
) (service.Servicio, error) {

	query := `
        UPDATE servicios
        SET
            archivado_en = NOW(),
            habilitado = FALSE
        WHERE id = $1
          AND archivado_en IS NULL
        RETURNING
            id, nombre, url, tipo,
            intervalo_segundos, timeout_segundos,
            habilitado
    `

	var s service.Servicio
	var intervalo, timeout int32

	err := r.pool.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&s.ID,
		&s.Nombre,
		&s.URL,
		&s.Tipo,
		&intervalo,
		&timeout,
		&s.Habilitado,
	)

	if err != nil {
		return s, fmt.Errorf(
			"archivar servicio: %w", err,
		)
	}

	s.Intervalo = time.Duration(intervalo) * time.Second
	s.Timeout = time.Duration(timeout) * time.Second

	return s, nil
}

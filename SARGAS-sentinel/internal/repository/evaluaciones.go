package repository

import (
	"context"
	"fmt"
	"time"

	"sargas-sentinel/internal/monitor"
)

type ResultadoEvaluacion struct {
	Anterior          monitor.Estado
	Actual            monitor.Estado
	Cambio            bool
	Evento            monitor.TipoEventoIncidente
	IncidenteID       int64
	DuracionIncidente time.Duration
}

func (r *ComprobacionRepository) RegistrarEvaluacion(
	ctx context.Context,
	c Comprobacion,
) (ResultadoEvaluacion, error) {

	var resultado ResultadoEvaluacion

	// Iniciar transacción.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return resultado, fmt.Errorf("iniciar transaccion: %w", err)
	}
	defer tx.Rollback(context.Background())

	// Crear el estado inicial si todavía no existe.
	_, err = tx.Exec(ctx, `
		INSERT INTO estados_servicio (servicio_id)
		VALUES ($1)
		ON CONFLICT (servicio_id) DO NOTHING
	`, c.ServicioID)

	if err != nil {
		return resultado, fmt.Errorf("inicializar estado: %w", err)
	}

	// Recuperar y bloquear el estado del servicio.
	var estado string
	var exitos, fallos int

	err = tx.QueryRow(ctx, `
		SELECT estado, exitos_consecutivos, fallos_consecutivos
		FROM estados_servicio
		WHERE servicio_id = $1
		FOR UPDATE
	`, c.ServicioID).Scan(&estado, &exitos, &fallos)

	if err != nil {
		return resultado, fmt.Errorf("cargar estado: %w", err)
	}

	evaluador := monitor.EvaluadorEstado{
		Actual:             monitor.Estado(estado),
		ExitosConsecutivos: exitos,
		FallosConsecutivos: fallos,
	}

	exito := c.Error == nil &&
		c.CodigoHTTP != nil &&
		*c.CodigoHTTP >= 200 &&
		*c.CodigoHTTP < 300

	resultado.Anterior, resultado.Actual, resultado.Cambio =
		evaluador.Registrar(exito)

	momento := time.Now().UTC()
	latenciaMS := float64(c.Latencia) / float64(time.Millisecond)

	// Registrar comprobación.
	_, err = tx.Exec(ctx, `
		INSERT INTO comprobaciones (
			servicio_id,
			codigo_http,
			latencia_ms,
			error,
			comprobado_en
		)
		VALUES ($1, $2, $3, $4, $5)
	`,
		c.ServicioID,
		c.CodigoHTTP,
		latenciaMS,
		c.Error,
		momento,
	)

	if err != nil {
		return resultado, fmt.Errorf("guardar comprobacion: %w", err)
	}

	// Persistir el estado y sus contadores.
	_, err = tx.Exec(ctx, `
		UPDATE estados_servicio
		SET estado = $2,
			exitos_consecutivos = $3,
			fallos_consecutivos = $4,
			ultima_comprobacion_en = $5
		WHERE servicio_id = $1
	`,
		c.ServicioID,
		string(evaluador.Actual),
		evaluador.ExitosConsecutivos,
		evaluador.FallosConsecutivos,
		momento,
	)

	if err != nil {
		return resultado, fmt.Errorf("guardar estado: %w", err)
	}

	// Preparar evidencia observable.
	evidencia := "Sin respuesta HTTP"

	if c.CodigoHTTP != nil {
		evidencia = fmt.Sprintf("HTTP %d", *c.CodigoHTTP)
	}

	if c.Error != nil {
		evidencia = *c.Error
	}

	// Gestionar incidentes solamente ante transiciones.
	if resultado.Cambio {
		switch resultado.Actual {

		case monitor.EstadoDown:
			err = tx.QueryRow(ctx, `
				INSERT INTO incidentes (
					servicio_id,
					iniciado_en,
					evidencia_inicio
				)
				VALUES ($1, $2, $3)
				RETURNING id
			`,
				c.ServicioID,
				momento,
				evidencia,
			).Scan(&resultado.IncidenteID)

			if err != nil {
				return resultado, fmt.Errorf("abrir incidente: %w", err)
			}

			resultado.Evento = monitor.IncidenteAbierto

		case monitor.EstadoUp:
			if resultado.Anterior == monitor.EstadoDown {
				var iniciadoEn time.Time

				err = tx.QueryRow(ctx, `
					UPDATE incidentes
					SET cerrado_en = $2,
						evidencia_cierre = $3
					WHERE servicio_id = $1
						AND cerrado_en IS NULL
					RETURNING id, iniciado_en
				`,
					c.ServicioID,
					momento,
					evidencia,
				).Scan(&resultado.IncidenteID, &iniciadoEn)

				if err != nil {
					return resultado, fmt.Errorf("cerrar incidente: %w", err)
				}

				resultado.Evento = monitor.IncidenteCerrado
				resultado.DuracionIncidente = momento.Sub(iniciadoEn)
			}
		}
	}

	// Confirmar todos los cambios.
	if err := tx.Commit(ctx); err != nil {
		return ResultadoEvaluacion{}, fmt.Errorf(
			"confirmar transaccion: %w", err,
		)
	}

	return resultado, nil
}

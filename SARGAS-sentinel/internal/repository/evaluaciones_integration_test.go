package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sargas-sentinel/internal/monitor"
	"sargas-sentinel/internal/service"
)

func TestRegistrarEvaluacionAbreYCierraIncidenteIntegracion(t *testing.T) {
	dsn := os.Getenv("SARGAS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SARGAS_TEST_DATABASE_URL no configurada")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(), 20*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// Proteger la base de produccion.
	var base string
	if err := pool.QueryRow(ctx,
		"SELECT current_database()",
	).Scan(&base); err != nil {
		t.Fatal(err)
	}

	if base != "sargas_sentinel_test" {
		t.Fatalf("Base no permitida: %s", base)
	}

	serviciosRepo := NuevoServicioRepository(pool)
	comprobacionesRepo := NuevoComprobacionRepository(pool)

	// Crear un servicio exclusivo para la prueba.
	s, err := serviciosRepo.Crear(ctx, service.Servicio{
		Nombre:     "Integracion evaluaciones",
		URL:        "https://example.com",
		Tipo:       "HTTP",
		Intervalo:  time.Minute,
		Timeout:    10 * time.Second,
		Habilitado: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Eliminar solamente los registros de este servicio.
	defer func() {
		cleanupCtx, cancelar := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancelar()

		consultas := []string{
			"DELETE FROM incidentes WHERE servicio_id = $1",
			"DELETE FROM comprobaciones WHERE servicio_id = $1",
			"DELETE FROM estados_servicio WHERE servicio_id = $1",
			"DELETE FROM servicios WHERE id = $1",
		}

		for _, consulta := range consultas {
			if _, err := pool.Exec(
				cleanupCtx, consulta, s.ID,
			); err != nil {
				t.Errorf("Error de limpieza: %v", err)
			}
		}
	}()

	codigoFallo := int32(503)
	codigoExito := int32(200)

	casos := []struct {
		nombre   string
		codigo   *int32
		estado   monitor.Estado
		evento   monitor.TipoEventoIncidente
		exitos   int
		fallos   int
		abiertos int
	}{
		{"Primer fallo", &codigoFallo,
			monitor.EstadoUnknown, "", 0, 1, 0},

		{"Segundo fallo", &codigoFallo,
			monitor.EstadoDown, monitor.IncidenteAbierto, 0, 2, 1},

		{"Primer exito", &codigoExito,
			monitor.EstadoDown, "", 1, 0, 1},

		{"Segundo exito", &codigoExito,
			monitor.EstadoUp, monitor.IncidenteCerrado, 2, 0, 0},
	}

	var incidenteID int64

	for i, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resultado, err := comprobacionesRepo.RegistrarEvaluacion(
				ctx,
				Comprobacion{
					ServicioID: s.ID,
					CodigoHTTP: caso.codigo,
					Latencia:   20 * time.Millisecond,
				},
			)
			if err != nil {
				t.Fatalf("RegistrarEvaluacion: %v", err)
			}

			if resultado.Actual != caso.estado ||
				resultado.Evento != caso.evento {
				t.Fatalf(
					"Estado/evento: %s/%s; esperado %s/%s",
					resultado.Actual, resultado.Evento,
					caso.estado, caso.evento,
				)
			}

			if caso.evento == monitor.IncidenteAbierto {
				incidenteID = resultado.IncidenteID
			}

			if caso.evento == monitor.IncidenteCerrado &&
				resultado.IncidenteID != incidenteID {
				t.Fatal("Se cerro un incidente diferente")
			}

			// Consultar los valores realmente persistidos.
			var estado string
			var exitos, fallos int

			err = pool.QueryRow(ctx, `
				SELECT estado, exitos_consecutivos, fallos_consecutivos
				FROM estados_servicio
				WHERE servicio_id = $1
			`, s.ID).Scan(&estado, &exitos, &fallos)

			if err != nil {
				t.Fatal(err)
			}

			if estado != string(caso.estado) ||
				exitos != caso.exitos ||
				fallos != caso.fallos {
				t.Fatalf(
					"Estado persistido incorrecto: %s, %d, %d",
					estado, exitos, fallos,
				)
			}

			var comprobaciones, abiertos int

			err = pool.QueryRow(ctx, `
				SELECT COUNT(*) FROM comprobaciones
				WHERE servicio_id = $1
			`, s.ID).Scan(&comprobaciones)

			if err != nil {
				t.Fatal(err)
			}

			err = pool.QueryRow(ctx, `
				SELECT COUNT(*) FROM incidentes
				WHERE servicio_id = $1 AND cerrado_en IS NULL
			`, s.ID).Scan(&abiertos)

			if err != nil {
				t.Fatal(err)
			}

			if comprobaciones != i+1 || abiertos != caso.abiertos {
				t.Fatalf(
					"Persistencia incorrecta: comprobaciones=%d, abiertos=%d",
					comprobaciones, abiertos,
				)
			}
		})

		if t.Failed() {
			return
		}
	}

	// Verificar el incidente historico cerrado.
	var id int64
	var inicio, cierre time.Time
	var evidenciaInicio, evidenciaCierre string

	err = pool.QueryRow(ctx, `
		SELECT id, iniciado_en, cerrado_en,
		       evidencia_inicio, evidencia_cierre
		FROM incidentes
		WHERE servicio_id = $1
	`, s.ID).Scan(
		&id, &inicio, &cierre,
		&evidenciaInicio, &evidenciaCierre,
	)

	if err != nil {
		t.Fatal(err)
	}

	if id != incidenteID ||
		cierre.Before(inicio) ||
		evidenciaInicio != "HTTP 503" ||
		evidenciaCierre != "HTTP 200" {
		t.Fatal("El incidente historico no se guardo correctamente")
	}

	var total int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM incidentes WHERE servicio_id = $1
	`, s.ID).Scan(&total); err != nil {
		t.Fatal(err)
	}

	if total != 1 {
		t.Fatalf("Se esperaba 1 incidente, hay %d", total)
	}
}

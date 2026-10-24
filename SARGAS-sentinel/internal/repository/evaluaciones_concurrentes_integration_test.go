package repository

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sargas-sentinel/internal/monitor"
	"sargas-sentinel/internal/service"
)

func TestRegistrarEvaluacionConcurrenteIntegracion(t *testing.T) {
	dsn := os.Getenv("SARGAS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SARGAS_TEST_DATABASE_URL no configurada")
	}

	ctx, cancelar := context.WithTimeout(
		context.Background(), 30*time.Second,
	)
	defer cancelar()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// Proteccion de la base principal.
	var base string
	if err := pool.QueryRow(
		ctx, "SELECT current_database()",
	).Scan(&base); err != nil {
		t.Fatal(err)
	}

	if base != "sargas_sentinel_test" {
		t.Fatalf("Base no permitida: %s", base)
	}

	serviciosRepo := NuevoServicioRepository(pool)
	comprobacionesRepo := NuevoComprobacionRepository(pool)

	s, err := serviciosRepo.Crear(ctx, service.Servicio{
		Nombre:     "Prueba concurrencia PostgreSQL",
		URL:        "https://example.com",
		Tipo:       "HTTP",
		Intervalo:  time.Minute,
		Timeout:    10 * time.Second,
		Habilitado: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Limpiar unicamente el servicio de prueba.
	defer func() {
		cleanupCtx, fin := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer fin()

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
				t.Errorf("Limpieza: %v", err)
			}
		}
	}()

	const total = 12
	codigo := int32(503)

	inicio := make(chan struct{})

	type resultado struct {
		evento monitor.TipoEventoIncidente
		err    error
	}

	salidas := make(chan resultado, total)

	var wg sync.WaitGroup

	// Ejecutar doce evaluaciones concurrentes.
	for i := 0; i < total; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-inicio

			r, err := comprobacionesRepo.RegistrarEvaluacion(
				ctx,
				Comprobacion{
					ServicioID: s.ID,
					CodigoHTTP: &codigo,
					Latencia:   20 * time.Millisecond,
				},
			)

			salidas <- resultado{
				evento: r.Evento,
				err:    err,
			}
		}()
	}

	// Liberar las doce goroutines.
	close(inicio)

	wg.Wait()
	close(salidas)

	// Debe producirse una sola apertura.
	aperturas := 0

	for r := range salidas {
		if r.err != nil {
			t.Fatalf("Evaluacion concurrente: %v", r.err)
		}

		if r.evento == monitor.IncidenteAbierto {
			aperturas++
		} else if r.evento != "" {
			t.Fatalf("Evento inesperado: %s", r.evento)
		}
	}

	if aperturas != 1 {
		t.Fatalf("Esperada 1 apertura, obtenidas %d", aperturas)
	}

	// Comprobar el estado persistido.
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

	if estado != "DOWN" || exitos != 0 || fallos != 2 {
		t.Fatalf(
			"Estado incorrecto: %s, exitos=%d, fallos=%d",
			estado, exitos, fallos,
		)
	}

	// Verificar todas las comprobaciones e incidentes.
	var comprobaciones, incidentes, abiertos int

	err = pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM comprobaciones
			 WHERE servicio_id = $1),
			(SELECT COUNT(*) FROM incidentes
			 WHERE servicio_id = $1),
			(SELECT COUNT(*) FROM incidentes
			 WHERE servicio_id = $1 AND cerrado_en IS NULL)
	`, s.ID).Scan(
		&comprobaciones, &incidentes, &abiertos,
	)

	if err != nil {
		t.Fatal(err)
	}

	if comprobaciones != total ||
		incidentes != 1 ||
		abiertos != 1 {
		t.Fatalf(
			"Persistencia incorrecta: comprobaciones=%d, incidentes=%d, abiertos=%d",
			comprobaciones, incidentes, abiertos,
		)
	}
}

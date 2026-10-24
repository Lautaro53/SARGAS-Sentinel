package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sargas-sentinel/internal/service"
)

func TestRegistrarEvaluacionRollbackSinIncidenteIntegracion(t *testing.T) {
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

	// Impedir escrituras en la base real.
	var base string
	if err := pool.QueryRow(
		ctx, "SELECT current_database()",
	).Scan(&base); err != nil {
		t.Fatal(err)
	}

	if base != "sargas_sentinel_test" {
		t.Fatalf("Base no permitida: %s", base)
	}

	// Crear un servicio exclusivo para la prueba.
	repo := NuevoServicioRepository(pool)

	s, err := repo.Crear(ctx, service.Servicio{
		Nombre:     "Prueba rollback",
		URL:        "https://example.com",
		Tipo:       "HTTP",
		Intervalo:  time.Minute,
		Timeout:    10 * time.Second,
		Habilitado: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Limpiar solamente los datos de este servicio.
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

	// Preparar intencionalmente un estado inconsistente:
	// DOWN con un exito, pero sin incidente abierto.
	_, err = pool.Exec(ctx, `
		INSERT INTO estados_servicio (
			servicio_id,
			estado,
			exitos_consecutivos,
			fallos_consecutivos
		)
		VALUES ($1, 'DOWN', 1, 0)
	`, s.ID)

	if err != nil {
		t.Fatalf("Preparar estado: %v", err)
	}

	// El segundo exito intentara cerrar un incidente
	// que no existe, provocando un error tardio.
	codigo := int32(200)

	_, err = NuevoComprobacionRepository(pool).
		RegistrarEvaluacion(ctx, Comprobacion{
			ServicioID: s.ID,
			CodigoHTTP: &codigo,
			Latencia:   20 * time.Millisecond,
		})

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"Se esperaba ErrNoRows al cerrar incidente: %v",
			err,
		)
	}

	// Verificar que la transaccion no altero el estado.
	var estado string
	var exitos, fallos int
	var fechaNula bool

	err = pool.QueryRow(ctx, `
		SELECT estado,
		       exitos_consecutivos,
		       fallos_consecutivos,
		       ultima_comprobacion_en IS NULL
		FROM estados_servicio
		WHERE servicio_id = $1
	`, s.ID).Scan(
		&estado, &exitos, &fallos, &fechaNula,
	)

	if err != nil {
		t.Fatal(err)
	}

	if estado != "DOWN" ||
		exitos != 1 ||
		fallos != 0 ||
		!fechaNula {
		t.Fatalf(
			"Rollback incorrecto: estado=%s exitos=%d fallos=%d fechaNula=%v",
			estado, exitos, fallos, fechaNula,
		)
	}

	// No deben existir comprobaciones ni incidentes.
	var comprobaciones, incidentes int

	err = pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM comprobaciones
			 WHERE servicio_id = $1),
			(SELECT COUNT(*) FROM incidentes
			 WHERE servicio_id = $1)
	`, s.ID).Scan(&comprobaciones, &incidentes)

	if err != nil {
		t.Fatal(err)
	}

	if comprobaciones != 0 || incidentes != 0 {
		t.Fatalf(
			"Quedaron datos parciales: comprobaciones=%d incidentes=%d",
			comprobaciones, incidentes,
		)
	}
}

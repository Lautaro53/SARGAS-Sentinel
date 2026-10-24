package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sargas-sentinel/internal/service"

	"github.com/jackc/pgx/v5"
)

func TestServicioRepositoryCrearListarIntegracion(t *testing.T) {
	dsn := os.Getenv("SARGAS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SARGAS_TEST_DATABASE_URL no configurada")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(), 15*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// Proteccion: nunca escribir en la base real.
	var base string
	err = pool.QueryRow(ctx,
		"SELECT current_database()",
	).Scan(&base)

	if err != nil {
		t.Fatal(err)
	}
	if base != "sargas_sentinel_test" {
		t.Fatalf("Base no permitida: %s", base)
	}

	repo := NuevoServicioRepository(pool)

	original := service.Servicio{
		Nombre:     "Test integracion Go",
		URL:        "https://example.com",
		Tipo:       "HTTP",
		Intervalo:  60 * time.Second,
		Timeout:    10 * time.Second,
		Habilitado: false,
	}

	creado, err := repo.Crear(ctx, original)
	if err != nil {
		t.Fatalf("Crear fallo: %v", err)
	}

	// Limpiar solamente el registro de esta prueba.
	defer func() {
		cleanupCtx, cancelar := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancelar()

		_, err := pool.Exec(cleanupCtx,
			"DELETE FROM servicios WHERE id = $1",
			creado.ID,
		)
		if err != nil {
			t.Errorf("Error limpiando prueba: %v", err)
		}
	}()

	if creado.ID <= 0 {
		t.Fatal("El ID generado no es valido")
	}

	servicios, err := repo.Listar(ctx)
	if err != nil {
		t.Fatalf("Listar fallo: %v", err)
	}

	encontrado := false

	for _, s := range servicios {
		if s.ID != creado.ID {
			continue
		}

		encontrado = true

		if s.Nombre != original.Nombre ||
			s.URL != original.URL ||
			s.Tipo != original.Tipo ||
			s.Intervalo != original.Intervalo ||
			s.Timeout != original.Timeout ||
			s.Habilitado != original.Habilitado {
			t.Fatal("Los datos recuperados no coinciden")
		}
	}

	if !encontrado {
		t.Fatal("El servicio creado no aparece en el listado")
	}
}

func TestServicioRepositoryActualizarHabilitacionIntegracion(t *testing.T) {
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

	// Impedir modificaciones en la base principal.
	var base string
	err = pool.QueryRow(ctx,
		"SELECT current_database()",
	).Scan(&base)

	if err != nil {
		t.Fatal(err)
	}
	if base != "sargas_sentinel_test" {
		t.Fatalf("Base no permitida: %s", base)
	}

	repo := NuevoServicioRepository(pool)

	// 1. Crear un servicio inicialmente pausado.
	original := service.Servicio{
		Nombre:     "Prueba edicion",
		URL:        "https://example.com",
		Tipo:       "HTTP",
		Intervalo:  60 * time.Second,
		Timeout:    10 * time.Second,
		Habilitado: false,
	}

	creado, err := repo.Crear(ctx, original)
	if err != nil {
		t.Fatalf("Crear: %v", err)
	}

	// Limpiar el registro incluso si una asercion falla.
	defer func() {
		cleanupCtx, cancelar := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancelar()

		_, err := pool.Exec(cleanupCtx,
			"DELETE FROM servicios WHERE id = $1",
			creado.ID,
		)
		if err != nil {
			t.Errorf("Limpieza: %v", err)
		}
	}()

	// 2. Modificar la configuracion.
	editado := creado
	editado.Nombre = "Prueba editada"
	editado.URL = "https://example.org"
	editado.Intervalo = 120 * time.Second
	editado.Timeout = 15 * time.Second

	actualizado, err := repo.Actualizar(ctx, editado)
	if err != nil {
		t.Fatalf("Actualizar: %v", err)
	}

	if actualizado != editado {
		t.Fatalf(
			"Actualizacion incorrecta:\nesperado: %+v\nrecibido: %+v",
			editado, actualizado,
		)
	}

	// 3. Habilitar el servicio.
	habilitado, err := repo.CambiarHabilitacion(
		ctx, creado.ID, true,
	)
	if err != nil {
		t.Fatalf("Habilitar: %v", err)
	}

	if !habilitado.Habilitado {
		t.Fatal("El servicio no quedo habilitado")
	}

	// 4. Pausar nuevamente.
	pausado, err := repo.CambiarHabilitacion(
		ctx, creado.ID, false,
	)
	if err != nil {
		t.Fatalf("Pausar: %v", err)
	}

	if pausado.Habilitado {
		t.Fatal("El servicio sigue habilitado")
	}

	// 5. Recuperar los datos desde PostgreSQL.
	servicios, err := repo.Listar(ctx)
	if err != nil {
		t.Fatalf("Listar: %v", err)
	}

	encontrado := false

	for _, s := range servicios {
		if s.ID != creado.ID {
			continue
		}

		encontrado = true

		if s != editado {
			t.Fatalf(
				"Datos persistidos incorrectos:\nesperado: %+v\nrecibido: %+v",
				editado, s,
			)
		}
	}

	if !encontrado {
		t.Fatal("El servicio no aparece en PostgreSQL")
	}
}

func TestServicioRepositoryArchivarConservaHistorialIntegracion(t *testing.T) {
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

	// Proteccion contra uso accidental de produccion.
	var base string
	err = pool.QueryRow(
		ctx, "SELECT current_database()",
	).Scan(&base)
	if err != nil {
		t.Fatal(err)
	}
	if base != "sargas_sentinel_test" {
		t.Fatalf("Base no permitida: %s", base)
	}

	repo := NuevoServicioRepository(pool)
	comprobacionesRepo := NuevoComprobacionRepository(pool)
	consultasRepo := NuevoConsultaRepository(pool)

	// 1. Crear un servicio habilitado.
	s, err := repo.Crear(ctx, service.Servicio{
		Nombre:     "Prueba archivado",
		URL:        "https://example.com",
		Tipo:       "HTTP",
		Intervalo:  time.Minute,
		Timeout:    10 * time.Second,
		Habilitado: true,
	})
	if err != nil {
		t.Fatalf("Crear servicio: %v", err)
	}

	// Limpiar exclusivamente los registros creados.
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
				t.Errorf("Limpieza: %v", err)
			}
		}
	}()

	// 2. Crear historial previo al archivado.
	codigo := int32(503)

	err = comprobacionesRepo.Guardar(ctx, Comprobacion{
		ServicioID: s.ID,
		CodigoHTTP: &codigo,
		Latencia:   100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Guardar comprobacion: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO incidentes (
			servicio_id, iniciado_en, evidencia_inicio
		)
		VALUES ($1, NOW(), 'HTTP 503')
	`, s.ID)
	if err != nil {
		t.Fatalf("Crear incidente de prueba: %v", err)
	}

	// 3. Archivar el servicio.
	archivado, err := repo.Archivar(ctx, s.ID)
	if err != nil {
		t.Fatalf("Archivar: %v", err)
	}
	if archivado.Habilitado {
		t.Fatal("El servicio archivado sigue habilitado")
	}

	var fechaArchivado time.Time
	var habilitado bool

	err = pool.QueryRow(ctx, `
		SELECT archivado_en, habilitado
		FROM servicios WHERE id = $1
	`, s.ID).Scan(&fechaArchivado, &habilitado)
	if err != nil {
		t.Fatalf("Consultar archivado: %v", err)
	}
	if fechaArchivado.IsZero() || habilitado {
		t.Fatal("El archivado no se persistio correctamente")
	}

	// 4. No debe aparecer en el listado operativo.
	listado, err := repo.Listar(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range listado {
		if item.ID == s.ID {
			t.Fatal("El servicio archivado aparece en Listar")
		}
	}

	// 5. No puede volver a habilitarse.
	_, err = repo.CambiarHabilitacion(ctx, s.ID, true)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Se esperaba ErrNoRows al habilitar: %v", err)
	}

	// 6. No puede editarse.
	_, err = repo.Actualizar(ctx, s)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Se esperaba ErrNoRows al editar: %v", err)
	}

	// 7. Debe conservar sus comprobaciones.
	historial, err := comprobacionesRepo.ListarPorServicio(
		ctx, s.ID, 10, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(historial) != 1 ||
		historial[0].CodigoHTTP == nil ||
		*historial[0].CodigoHTTP != 503 {
		t.Fatal("No se conservo la comprobacion historica")
	}

	// 8. Debe conservar sus incidentes.
	incidentes, err := consultasRepo.ListarIncidentes(
		ctx, s.ID, 10, 0,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(incidentes) != 1 {
		t.Fatalf("Se esperaba 1 incidente, hay %d", len(incidentes))
	}

	incidente := incidentes[0]

	if !incidente.ServicioArchivado {
		t.Fatal("El incidente no identifica al servicio archivado")
	}
	if incidente.NombreServicio != s.Nombre {
		t.Fatal("Se perdio el nombre del servicio")
	}
	if incidente.CerradoEn != nil ||
		incidente.EvidenciaCierre != nil {
		t.Fatal("El archivado cerro artificialmente el incidente")
	}
}

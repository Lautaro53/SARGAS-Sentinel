package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sargas-sentinel/internal/monitor"
	"sargas-sentinel/internal/repository"
	"sargas-sentinel/internal/service"
)

func TestAPIEdicionesConcurrentesIntegracion(t *testing.T) {
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

	repo := repository.NuevoServicioRepository(pool)

	s, err := repo.Crear(ctx, service.Servicio{
		Nombre:     "Original",
		URL:        "https://example.com",
		Tipo:       "HTTP",
		Intervalo:  time.Hour,
		Timeout:    10 * time.Second,
		Habilitado: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Eliminar exclusivamente el servicio creado.
	defer func() {
		cleanupCtx, fin := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer fin()

		_, err := pool.Exec(cleanupCtx,
			"DELETE FROM servicios WHERE id = $1", s.ID,
		)
		if err != nil {
			t.Errorf("Limpieza: %v", err)
		}
	}()

	const total = 32
	inicios := make(chan string, total+1)

	// Trabajador simulado, sin peticiones externas.
	manager := monitor.NewManager(
		context.Background(),
		func(ctx context.Context, s service.Servicio) {
			inicios <- s.Nombre
			<-ctx.Done()
		},
	)
	defer manager.Cerrar()

	if err := manager.Aplicar(s); err != nil {
		t.Fatal(err)
	}

	// Esperar el primer trabajador.
	select {
	case <-inicios:
	case <-time.After(3 * time.Second):
		t.Fatal("No inicio el trabajador original")
	}

	handler := NewHandler(repo, nil, manager, nil)

	var wg sync.WaitGroup
	salida := make(chan struct{})
	errores := make(chan string, total)

	// Ejecutar 32 ediciones simultaneas.
	for i := 0; i < total; i++ {
		wg.Add(1)

		go func(numero int) {
			defer wg.Done()
			<-salida

			cuerpo := fmt.Sprintf(
				`{"nombre":"Edicion-%02d","url":"https://example.com","intervalo_segundos":3600,"timeout_segundos":10}`,
				numero,
			)

			req := httptest.NewRequest(
				http.MethodPut,
				fmt.Sprintf("/servicios/%d", s.ID),
				strings.NewReader(cuerpo),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				errores <- fmt.Sprintf(
					"Edicion %d: HTTP %d: %s",
					numero, rec.Code, rec.Body.String(),
				)
			}
		}(i)
	}

	close(salida)
	wg.Wait()
	close(errores)

	for mensaje := range errores {
		t.Error(mensaje)
	}
	if t.Failed() {
		return
	}

	// Cada edicion debio iniciar un trabajador.
	// El ultimo inicio representa la configuracion
	// que quedo aplicada en el Manager.
	var ultimoNombre string

	for i := 0; i < total; i++ {
		select {
		case ultimoNombre = <-inicios:
		case <-time.After(5 * time.Second):
			t.Fatal("Faltan inicios de trabajadores")
		}
	}

	// Consultar la configuracion final en PostgreSQL.
	var nombreDB string
	err = pool.QueryRow(ctx,
		"SELECT nombre FROM servicios WHERE id = $1",
		s.ID,
	).Scan(&nombreDB)

	if err != nil {
		t.Fatal(err)
	}

	if ultimoNombre != nombreDB {
		t.Fatalf(
			"Inconsistencia: PostgreSQL=%q, Manager=%q",
			nombreDB, ultimoNombre,
		)
	}
}

func TestAPIArchivadoConcurrenteIntegracion(t *testing.T) {
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

	var base string
	if err := pool.QueryRow(
		ctx, "SELECT current_database()",
	).Scan(&base); err != nil {
		t.Fatal(err)
	}

	if base != "sargas_sentinel_test" {
		t.Fatalf("Base no permitida: %s", base)
	}

	repo := repository.NuevoServicioRepository(pool)

	s, err := repo.Crear(ctx, service.Servicio{
		Nombre:     "Prueba archivado concurrente",
		URL:        "http://127.0.0.1:1",
		Tipo:       "HTTP",
		Intervalo:  time.Hour,
		Timeout:    5 * time.Second,
		Habilitado: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		limpiezaCtx, fin := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer fin()

		if _, err := pool.Exec(
			limpiezaCtx,
			"DELETE FROM servicios WHERE id = $1",
			s.ID,
		); err != nil {
			t.Errorf("Limpieza: %v", err)
		}
	}()

	var trabajadoresActivos atomic.Int32

	manager := monitor.NewManager(
		context.Background(),
		func(ctx context.Context, _ service.Servicio) {
			trabajadoresActivos.Add(1)
			defer trabajadoresActivos.Add(-1)
			<-ctx.Done()
		},
	)
	defer manager.Cerrar()

	handler := NewHandler(repo, nil, manager, nil)

	operaciones := []struct {
		metodo string
		ruta   string
		cuerpo string
		valido func(int) bool
	}{
		{
			http.MethodPut,
			fmt.Sprintf("/servicios/%d", s.ID),
			`{"nombre":"Editado","url":"http://127.0.0.1:1","intervalo_segundos":3600,"timeout_segundos":5}`,
			func(c int) bool {
				return c == 200 || c == 404
			},
		},
		{
			http.MethodPatch,
			fmt.Sprintf("/servicios/%d/habilitacion", s.ID),
			`{"habilitado":true}`,
			func(c int) bool {
				return c == 204 || c == 404
			},
		},
		{
			http.MethodPatch,
			fmt.Sprintf("/servicios/%d/archivar", s.ID),
			"",
			func(c int) bool {
				return c == 204
			},
		},
	}

	var wg sync.WaitGroup
	inicio := make(chan struct{})
	errores := make(chan string, len(operaciones))

	for _, op := range operaciones {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-inicio

			req := httptest.NewRequest(
				op.metodo,
				op.ruta,
				strings.NewReader(op.cuerpo),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if !op.valido(rec.Code) {
				errores <- fmt.Sprintf(
					"%s %s: HTTP %d",
					op.metodo, op.ruta, rec.Code,
				)
			}
		}()
	}

	close(inicio)
	wg.Wait()
	close(errores)

	for mensaje := range errores {
		t.Error(mensaje)
	}

	var archivado, habilitado bool

	err = pool.QueryRow(ctx, `
        SELECT archivado_en IS NOT NULL, habilitado
        FROM servicios WHERE id = $1
    `, s.ID).Scan(&archivado, &habilitado)

	if err != nil {
		t.Fatal(err)
	}

	if !archivado || habilitado {
		t.Fatal("El servicio no quedó archivado y deshabilitado")
	}

	if trabajadoresActivos.Load() != 0 {
		t.Fatal("Quedó un trabajador activo tras el archivado")
	}
}

package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sargas-sentinel/internal/monitor"
	"sargas-sentinel/internal/repository"
	"sargas-sentinel/internal/service"
)

func TestAPIManagerCerradoIntegracion(t *testing.T) {
	dsn := os.Getenv("SARGAS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SARGAS_TEST_DATABASE_URL no configurada")
	}

	ctx, cancelar := context.WithTimeout(
		context.Background(), 15*time.Second,
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
		Nombre:     "Prueba Manager cerrado",
		URL:        "https://example.com",
		Tipo:       "HTTP",
		Intervalo:  time.Minute,
		Timeout:    10 * time.Second,
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

		_, err := pool.Exec(limpiezaCtx,
			"DELETE FROM servicios WHERE id = $1",
			s.ID,
		)
		if err != nil {
			t.Errorf("Limpieza: %v", err)
		}
	}()

	manager := monitor.NewManager(
		context.Background(),
		func(context.Context, service.Servicio) {
			t.Error("No deberia iniciar un trabajador")
		},
	)

	manager.Cerrar()

	handler := NewHandler(repo, nil, manager, nil)

	req := httptest.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("/servicios/%d/habilitacion", s.ID),
		strings.NewReader(`{"habilitado":true}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"Esperado HTTP 503, recibido %d: %s",
			rec.Code, rec.Body.String(),
		)
	}

	var habilitado bool

	err = pool.QueryRow(ctx,
		"SELECT habilitado FROM servicios WHERE id = $1",
		s.ID,
	).Scan(&habilitado)

	if err != nil {
		t.Fatal(err)
	}

	if !habilitado {
		t.Fatal("El comportamiento de persistencia cambio")
	}

	t.Log("HTTP 503: cambio persistido, Manager no sincronizado")

	iniciados := make(chan service.Servicio, 1)

	nuevoManager := monitor.NewManager(
		context.Background(),
		func(ctx context.Context, s service.Servicio) {
			iniciados <- s
			<-ctx.Done()
		},
	)
	defer nuevoManager.Cerrar()

	servicios, err := repo.Listar(ctx)
	if err != nil {
		t.Fatalf("Recuperar servicios: %v", err)
	}

	encontrado := false

	for _, actual := range servicios {
		if actual.ID != s.ID {
			continue
		}

		encontrado = true

		if !actual.Habilitado {
			t.Fatal("La configuracion recuperada no esta habilitada")
		}

		if err := nuevoManager.Aplicar(actual); err != nil {
			t.Fatalf("Restaurar monitoreo: %v", err)
		}
	}

	if !encontrado {
		t.Fatal("El servicio no se recupero desde PostgreSQL")
	}

	select {
	case activo := <-iniciados:
		if activo.ID != s.ID || !activo.Habilitado {
			t.Fatal("El trabajador recuperado es incorrecto")
		}

	case <-time.After(3 * time.Second):
		t.Fatal("El monitoreo no se reanudo")
	}

	t.Log("Monitoreo recuperado desde PostgreSQL")

}

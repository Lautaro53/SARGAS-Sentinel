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

func TestAPICicloVidaServicioIntegracion(t *testing.T) {
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

	repo := repository.NuevoServicioRepository(pool)

	s, err := repo.Crear(ctx, service.Servicio{
		Nombre:     "Integracion API Manager",
		URL:        "https://example.com",
		Tipo:       "HTTP",
		Intervalo:  time.Hour,
		Timeout:    10 * time.Second,
		Habilitado: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		cleanupCtx, cancelar := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancelar()

		_, err := pool.Exec(cleanupCtx,
			"DELETE FROM servicios WHERE id = $1", s.ID,
		)
		if err != nil {
			t.Errorf("Limpieza: %v", err)
		}
	}()

	iniciados := make(chan int64, 4)
	terminados := make(chan int64, 4)

	manager := monitor.NewManager(
		context.Background(),
		func(ctx context.Context, s service.Servicio) {
			iniciados <- s.ID
			<-ctx.Done()
			terminados <- s.ID
		},
	)
	defer manager.Cerrar()

	handler := NewHandler(repo, nil, manager, nil)

	pasos := []struct {
		nombre     string
		metodo     string
		ruta       string
		cuerpo     string
		evento     string
		habilitado bool
		archivado  bool
	}{
		{
			"Habilitar", http.MethodPatch,
			"/servicios/%d/habilitacion",
			`{"habilitado":true}`,
			"inicio", true, false,
		},
		{
			"Pausar", http.MethodPatch,
			"/servicios/%d/habilitacion",
			`{"habilitado":false}`,
			"fin", false, false,
		},
		{
			"Reanudar", http.MethodPatch,
			"/servicios/%d/habilitacion",
			`{"habilitado":true}`,
			"inicio", true, false,
		},
		{
			"Archivar", http.MethodPatch,
			"/servicios/%d/archivar",
			"",
			"fin", false, true,
		},
	}

	for _, paso := range pasos {
		t.Run(paso.nombre, func(t *testing.T) {
			req := httptest.NewRequest(
				paso.metodo,
				fmt.Sprintf(paso.ruta, s.ID),
				strings.NewReader(paso.cuerpo),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Fatalf(
					"Esperado 204, recibido %d: %s",
					rec.Code, rec.Body.String(),
				)
			}

			var habilitado, archivado bool

			err := pool.QueryRow(ctx, `
				SELECT habilitado, archivado_en IS NOT NULL
				FROM servicios WHERE id = $1
			`, s.ID).Scan(&habilitado, &archivado)

			if err != nil {
				t.Fatal(err)
			}

			if habilitado != paso.habilitado ||
				archivado != paso.archivado {
				t.Fatalf(
					"PostgreSQL: habilitado=%v archivado=%v",
					habilitado, archivado,
				)
			}

			var canal <-chan int64

			if paso.evento == "inicio" {
				canal = iniciados
			} else {
				canal = terminados
			}

			select {
			case id := <-canal:
				if id != s.ID {
					t.Fatalf("Trabajador inesperado: %d", id)
				}
			case <-time.After(3 * time.Second):
				t.Fatalf(
					"No se produjo el evento %s",
					paso.evento,
				)
			}
		})
	}
}

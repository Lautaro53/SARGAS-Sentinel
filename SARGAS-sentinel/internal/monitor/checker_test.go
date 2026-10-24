package monitor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCheckHTTPRespuestas(t *testing.T) {
	casos := []struct {
		nombre string
		codigo int
	}{
		{"HTTP 200", http.StatusOK},
		{"HTTP 204", http.StatusNoContent},
		{"HTTP 503", http.StatusServiceUnavailable},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servidor := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodGet {
						t.Errorf("Metodo inesperado: %s", r.Method)
					}

					w.WriteHeader(caso.codigo)
				}),
			)
			defer servidor.Close()

			codigo, latencia, err := CheckHTTP(
				context.Background(),
				servidor.URL,
				2*time.Second,
			)

			if err != nil {
				t.Fatalf("Error inesperado: %v", err)
			}

			if codigo != caso.codigo {
				t.Fatalf(
					"Esperado HTTP %d, recibido HTTP %d",
					caso.codigo, codigo,
				)
			}

			if latencia < 0 {
				t.Fatal("La latencia no puede ser negativa")
			}
		})
	}
}

func TestCheckHTTPTimeout(t *testing.T) {
	servidor := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simular un servicio que no responde.
			<-r.Context().Done()
		}),
	)
	defer servidor.Close()

	codigo, _, err := CheckHTTP(
		context.Background(),
		servidor.URL,
		150*time.Millisecond,
	)

	if err == nil {
		t.Fatal("Se esperaba un error por timeout")
	}

	if codigo != 0 {
		t.Fatalf("Esperado codigo 0, recibido %d", codigo)
	}
}

func TestCheckHTTPCancelacion(t *testing.T) {
	recibida := make(chan struct{})

	servidor := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(recibida)

			// Esperar hasta que el cliente cancele.
			<-r.Context().Done()
		}),
	)
	defer servidor.Close()

	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()

	type resultado struct {
		codigo int
		err    error
	}

	terminado := make(chan resultado, 1)

	// Ejecutar la comprobacion en otra goroutine.
	go func() {
		codigo, _, err := CheckHTTP(
			ctx,
			servidor.URL,
			5*time.Second,
		)

		terminado <- resultado{codigo, err}
	}()

	// Confirmar que el servidor recibio la peticion.
	select {
	case <-recibida:
	case <-time.After(3 * time.Second):
		t.Fatal("La peticion HTTP no llego al servidor")
	}

	// Cancelar antes de que venza el timeout.
	cancelar()

	select {
	case r := <-terminado:
		if r.codigo != 0 {
			t.Fatalf("Esperado codigo 0, recibido %d", r.codigo)
		}

		if !errors.Is(r.err, context.Canceled) {
			t.Fatalf(
				"Esperado context.Canceled, recibido %v",
				r.err,
			)
		}

	case <-time.After(3 * time.Second):
		t.Fatal("La peticion no termino tras cancelar")
	}
}

func TestCheckHTTPNoSigueRedirecciones(t *testing.T) {
	var visitasDestino atomic.Int32

	// Servidor al que apunta la redirección.
	destino := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			visitasDestino.Add(1)
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer destino.Close()

	// Servidor que responde con HTTP 302.
	origen := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, destino.URL, http.StatusFound)
		}),
	)
	defer origen.Close()

	codigo, _, err := CheckHTTP(
		context.Background(),
		origen.URL,
		2*time.Second,
	)

	if err != nil {
		t.Fatalf("Error inesperado: %v", err)
	}

	if codigo != http.StatusFound {
		t.Fatalf("Esperado HTTP 302, recibido %d", codigo)
	}

	if visitasDestino.Load() != 0 {
		t.Fatal("SARGAS siguió la redirección HTTP")
	}
}

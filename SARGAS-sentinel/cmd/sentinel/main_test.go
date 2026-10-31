package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestEjecutarRechazaDatabaseURLVacia(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	err := ejecutar()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL no configurada") {
		t.Fatalf("Se esperaba error al iniciar sin DATABASE_URL; recibido: %v", err)
	}
}

func TestProtegerHostLocal(t *testing.T) {
	casos := []struct {
		nombre   string
		host     string
		esperado int
	}{
		{"IP local", "127.0.0.1:8080", 204},
		{"localhost", "localhost:8080", 204},
		{"Mayusculas", "LOCALHOST:8080", 204},
		{"Host externo", "sitio-externo.example", 403},
		{"Puerto incorrecto", "localhost:9090", 403},
		{"Host vacio", "", 403},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alcanzado := false

			siguiente := http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					alcanzado = true
					w.WriteHeader(http.StatusNoContent)
				},
			)

			handler := protegerHostLocal(siguiente)

			req := httptest.NewRequest(
				http.MethodGet,
				"/servicios",
				nil,
			)
			req.Host = caso.host

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != caso.esperado {
				t.Fatalf(
					"Esperado HTTP %d, recibido %d",
					caso.esperado, rec.Code,
				)
			}

			if alcanzado != (caso.esperado == 204) {
				t.Fatal("El middleware no controló correctamente el acceso")
			}
		})
	}
}

func TestMensajeSeguroMonitoreo(t *testing.T) {
	const secreto = "clave-super-secreta"

	casos := []struct {
		nombre   string
		err      error
		esperado string
	}{
		{
			nombre: "DNS no encontrado",
			err: &url.Error{
				Op:  "Get",
				URL: "https://usuario:" + secreto + "@example.test/",
				Err: &net.DNSError{
					Err:        "no such host",
					Name:       "example.test",
					IsNotFound: true,
				},
			},
			esperado: "DNS: dominio no encontrado",
		},
		{
			nombre:   "Timeout",
			err:      context.DeadlineExceeded,
			esperado: "Tiempo de espera agotado",
		},
		{
			nombre:   "Cancelacion",
			err:      context.Canceled,
			esperado: "Comprobación cancelada",
		},
		{
			nombre: "Error generico con URL sensible",
			err: &url.Error{
				Op:  "Get",
				URL: "https://example.test/?token=" + secreto,
				Err: errors.New("connection refused"),
			},
			esperado: "Error de conexión HTTP",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			obtenido := mensajeSeguroMonitoreo(caso.err)

			if obtenido != caso.esperado {
				t.Fatalf(
					"Esperado %q, obtenido %q",
					caso.esperado,
					obtenido,
				)
			}

			if strings.Contains(obtenido, secreto) ||
				strings.Contains(obtenido, "https://") {
				t.Fatalf(
					"El mensaje contiene información sensible: %q",
					obtenido,
				)
			}
		})
	}
}

func TestProtegerHostLocalPuertoAlternativo(t *testing.T) {
	t.Setenv("SARGAS_PUBLIC_PORT", "8081")

	handler := protegerHostLocal(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	))

	casos := []struct {
		host     string
		esperado int
	}{
		{"127.0.0.1:8081", 204},
		{"localhost:8081", 204},
		{"127.0.0.1:8080", 403},
		{"localhost:8082", 403},
		{"ejemplo.com:8081", 403},
	}

	for _, caso := range casos {
		t.Run(caso.host, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet, "/", nil,
			)
			req.Host = caso.host

			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			if resp.Code != caso.esperado {
				t.Errorf(
					"Host %s: esperado %d, obtenido %d",
					caso.host,
					caso.esperado,
					resp.Code,
				)
			}
		})
	}
}

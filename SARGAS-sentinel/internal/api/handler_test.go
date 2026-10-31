package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCrearServicioRechazaSolicitudesInvalidas(t *testing.T) {
	casos := []struct {
		nombre string
		body   string
	}{
		{
			nombre: "JSON mal formado",
			body:   `{"nombre":`,
		},
		{
			nombre: "Nombre vacio",
			body:   `{"nombre":"","url":"https://example.com","intervalo_segundos":60,"timeout_segundos":10}`,
		},
		{
			nombre: "URL no permitida",
			body:   `{"nombre":"Demo","url":"ftp://example.com","intervalo_segundos":60,"timeout_segundos":10}`,
		},
		{
			nombre: "Intervalo invalido",
			body:   `{"nombre":"Demo","url":"https://example.com","intervalo_segundos":0,"timeout_segundos":10}`,
		},
		{
			nombre: "Campo desconocido",
			body:   `{"nombre":"Demo","url":"https://example.com","intervalo_segundos":60,"timeout_segundos":10,"extra":true}`,
		},
		{
			nombre: "Multiples objetos JSON",
			body:   `{"nombre":"Demo"} {"nombre":"Otro"}`,
		},
	}

	handler := NewHandler(nil, nil, nil, nil)

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/servicios",
				strings.NewReader(caso.body),
			)

			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf(
					"Esperado HTTP 400, recibido HTTP %d: %s",
					rec.Code,
					rec.Body.String(),
				)
			}
		})
	}
}

func TestAPIRutasYParametrosInvalidos(t *testing.T) {
	casos := []struct {
		nombre   string
		metodo   string
		ruta     string
		esperado int
	}{
		{
			"Ruta inexistente",
			http.MethodGet,
			"/ruta-inexistente",
			http.StatusNotFound,
		},
		{
			"Metodo no permitido",
			http.MethodDelete,
			"/servicios",
			http.StatusMethodNotAllowed,
		},
		{
			"ID no numerico",
			http.MethodPut,
			"/servicios/abc",
			http.StatusBadRequest,
		},
		{
			"ID igual a cero",
			http.MethodGet,
			"/servicios/0/estado",
			http.StatusBadRequest,
		},
		{
			"ID negativo al archivar",
			http.MethodPatch,
			"/servicios/-1/archivar",
			http.StatusBadRequest,
		},
		{
			"Limite igual a cero",
			http.MethodGet,
			"/incidentes?limite=0",
			http.StatusBadRequest,
		},
		{
			"Limite superior al permitido",
			http.MethodGet,
			"/incidentes?limite=101",
			http.StatusBadRequest,
		},
		{
			"Offset negativo",
			http.MethodGet,
			"/incidentes?offset=-1",
			http.StatusBadRequest,
		},
	}

	handler := NewHandler(nil, nil, nil, nil)

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			req := httptest.NewRequest(
				caso.metodo,
				caso.ruta,
				nil,
			)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != caso.esperado {
				t.Fatalf(
					"Esperado HTTP %d, recibido HTTP %d: %s",
					caso.esperado,
					rec.Code,
					rec.Body.String(),
				)
			}
		})
	}
}

func TestAPIRechazaJSONExcesivo(t *testing.T) {
	const limite = 16 * 1024

	casos := []struct {
		nombre string
		metodo string
		ruta   string
		cuerpo string
	}{
		{
			nombre: "Crear servicio",
			metodo: http.MethodPost,
			ruta:   "/servicios",
			cuerpo: `{"nombre":"Demo","url":"https://example.com","intervalo_segundos":60,"timeout_segundos":10}`,
		},
		{
			nombre: "Editar servicio",
			metodo: http.MethodPut,
			ruta:   "/servicios/1",
			cuerpo: `{"nombre":"Demo","url":"https://example.com","intervalo_segundos":60,"timeout_segundos":10}`,
		},
		{
			nombre: "Cambiar habilitacion",
			metodo: http.MethodPatch,
			ruta:   "/servicios/1/habilitacion",
			cuerpo: `{"habilitado":true}`,
		},
	}

	handler := NewHandler(nil, nil, nil, nil)

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			cuerpo := caso.cuerpo +
				strings.Repeat(" ", limite)

			if len(cuerpo) <= limite {
				t.Fatal("El cuerpo no supera el limite")
			}

			req := httptest.NewRequest(
				caso.metodo,
				caso.ruta,
				strings.NewReader(cuerpo),
			)

			req.Header.Set(
				"Content-Type",
				"application/json",
			)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf(
					"Esperado HTTP 400, recibido HTTP %d: %s",
					rec.Code,
					rec.Body.String(),
				)
			}
		})
	}
}

func TestHabilitacionRechazaJSONInvalido(t *testing.T) {
	casos := []struct {
		nombre string
		body   string
	}{
		{
			nombre: "Campo obligatorio ausente",
			body:   `{}`,
		},
		{
			nombre: "Habilitado nulo",
			body:   `{"habilitado":null}`,
		},
		{
			nombre: "Habilitado como texto",
			body:   `{"habilitado":"false"}`,
		},
		{
			nombre: "Habilitado como numero",
			body:   `{"habilitado":0}`,
		},
		{
			nombre: "Campo desconocido",
			body:   `{"habilitado":true,"extra":1}`,
		},
		{
			nombre: "Multiples objetos JSON",
			body:   `{"habilitado":true} {"habilitado":false}`,
		},
		{
			nombre: "JSON incompleto",
			body:   `{"habilitado":`,
		},
	}

	handler := NewHandler(nil, nil, nil, nil)

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPatch,
				"/servicios/1/habilitacion",
				strings.NewReader(caso.body),
			)

			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf(
					"Esperado HTTP 400, recibido HTTP %d: %s",
					rec.Code,
					rec.Body.String(),
				)
			}
		})
	}
}

func TestContratoJSONIncidentes(t *testing.T) {
	inicio := time.Date(
		2026, time.October, 16, 12, 0, 0, 0, time.UTC,
	)
	cierre := inicio.Add(5 * time.Minute)
	evidenciaCierre := "HTTP 200"

	casos := []struct {
		nombre    string
		incidente incidenteResponse
		cerrado   bool
	}{
		{
			nombre: "Incidente abierto",
			incidente: incidenteResponse{
				ID:                1,
				ServicioID:        2,
				IniciadoEn:        inicio,
				EvidenciaInicio:   "HTTP 503",
				NombreServicio:    "API principal",
				ServicioArchivado: false,
			},
			cerrado: false,
		},
		{
			nombre: "Incidente cerrado de servicio archivado",
			incidente: incidenteResponse{
				ID:                2,
				ServicioID:        3,
				IniciadoEn:        inicio,
				CerradoEn:         &cierre,
				EvidenciaInicio:   "Timeout",
				EvidenciaCierre:   &evidenciaCierre,
				NombreServicio:    "Servicio archivado",
				ServicioArchivado: true,
			},
			cerrado: true,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			rec := httptest.NewRecorder()

			responderJSON(
				rec,
				http.StatusOK,
				caso.incidente,
			)

			if rec.Code != http.StatusOK {
				t.Fatalf("HTTP inesperado: %d", rec.Code)
			}

			if rec.Header().Get("Content-Type") != "application/json" {
				t.Fatal("Content-Type incorrecto")
			}

			var datos map[string]json.RawMessage

			if err := json.Unmarshal(rec.Body.Bytes(), &datos); err != nil {
				t.Fatalf("JSON invalido: %v", err)
			}

			campos := []string{
				"id",
				"servicio_id",
				"iniciado_en",
				"cerrado_en",
				"evidencia_inicio",
				"evidencia_cierre",
				"nombre_servicio",
				"servicio_archivado",
			}

			if len(datos) != len(campos) {
				t.Fatalf("Cantidad de campos inesperada: %d", len(datos))
			}

			for _, campo := range campos {
				if _, existe := datos[campo]; !existe {
					t.Errorf("Falta el campo %q", campo)
				}
			}

			var nombre string
			if err := json.Unmarshal(datos["nombre_servicio"], &nombre); err != nil {
				t.Fatalf("Nombre invalido: %v", err)
			}
			if nombre != caso.incidente.NombreServicio {
				t.Fatal("Nombre de servicio incorrecto")
			}

			var archivado bool
			if err := json.Unmarshal(datos["servicio_archivado"], &archivado); err != nil {
				t.Fatalf("Archivado invalido: %v", err)
			}
			if archivado != caso.incidente.ServicioArchivado {
				t.Fatal("Estado de archivado incorrecto")
			}

			for _, campo := range []string{"cerrado_en", "evidencia_cierre"} {
				esNulo := string(datos[campo]) == "null"

				if caso.cerrado == esNulo {
					t.Errorf("Valor incorrecto para %s", campo)
				}
			}
		})
	}
}

func TestOrigenAdministrativo(t *testing.T) {
	casos := []struct {
		nombre, metodo, ruta, origen string
		esperado                     int
	}{
		{"POST externo", "POST", "/servicios",
			"https://externo.example", 403},

		{"PUT externo", "PUT", "/servicios/abc",
			"https://externo.example", 403},

		{"PATCH externo", "PATCH", "/servicios/abc/archivar",
			"https://externo.example", 403},

		{"Origen null", "POST", "/servicios",
			"null", 403},

		{"POST Vite localhost", "POST", "/servicios",
			"http://localhost:5173", 400},

		{"POST Vite 127", "POST", "/servicios",
			"http://127.0.0.1:5173", 400},

		{"POST sin Origin", "POST", "/servicios",
			"", 400},

		{"GET externo", "GET", "/incidentes?limite=0",
			"https://externo.example", 400},
	}

	handler := NewHandler(nil, nil, nil, nil)

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			req := httptest.NewRequest(
				caso.metodo,
				caso.ruta,
				strings.NewReader("{"),
			)

			if caso.origen != "" {
				req.Header.Set("Origin", caso.origen)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != caso.esperado {
				t.Fatalf(
					"Esperado HTTP %d, recibido HTTP %d",
					caso.esperado,
					rec.Code,
				)
			}
		})
	}
}

func TestAPIRechazaURLConCredenciales(t *testing.T) {
	casos := []struct {
		nombre string
		metodo string
		ruta   string
		url    string
	}{
		{
			"Crear con usuario y contraseña",
			http.MethodPost,
			"/servicios",
			"https://usuario:clave-ficticia@example.com",
		},
		{
			"Editar con usuario y contraseña",
			http.MethodPut,
			"/servicios/1",
			"https://usuario:clave-ficticia@example.com",
		},
		{
			"Crear con usuario",
			http.MethodPost,
			"/servicios",
			"https://usuario@example.com",
		},
		{
			"Editar con usuario",
			http.MethodPut,
			"/servicios/1",
			"https://usuario@example.com",
		},
	}

	handler := NewHandler(nil, nil, nil, nil)

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			cuerpo := fmt.Sprintf(
				`{"nombre":"Demo","url":%q,"intervalo_segundos":60,"timeout_segundos":10}`,
				caso.url,
			)

			req := httptest.NewRequest(
				caso.metodo,
				caso.ruta,
				strings.NewReader(cuerpo),
			)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf(
					"Esperado HTTP 400, recibido %d",
					rec.Code,
				)
			}

			if strings.Contains(rec.Body.String(), "clave-ficticia") {
				t.Fatal("La respuesta expuso credenciales")
			}
		})
	}
}

func TestOrigenAdministrativoPuertoAlternativo(t *testing.T) {
	t.Setenv("SARGAS_PUBLIC_PORT", "8081")

	handler := protegerOrigenAdministrativo(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	casos := []struct {
		nombre   string
		origen   string
		esperado int
	}{
		{"localhost permitido", "http://localhost:8081", 204},
		{"127 permitido", "http://127.0.0.1:8081", 204},
		{"puerto anterior", "http://localhost:8080", 403},
		{"puerto distinto", "http://localhost:8082", 403},
		{"origen externo", "https://ejemplo.com", 403},
		{"desarrollo Vite", "http://localhost:5173", 204},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/servicios", nil)
			req.Header.Set("Origin", caso.origen)

			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			if resp.Code != caso.esperado {
				t.Errorf(
					"Origen %s: esperado %d, obtenido %d",
					caso.origen,
					caso.esperado,
					resp.Code,
				)
			}
		})
	}
}

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHistorialValidaFiltrosSinConsultarBase(t *testing.T) {
	casos := []string{
		"/servicios/1/comprobaciones/consulta?codigo=DROP",
		"/servicios/1/comprobaciones/consulta?orden=otra",
		"/servicios/1/comprobaciones/consulta?latencia_min=-1",
		"/servicios/1/comprobaciones/consulta?latencia_min=NaN",
		"/servicios/1/comprobaciones/consulta?latencia_min=800&latencia_max=200",
		"/servicios/1/comprobaciones/consulta?limite=101",
		"/servicios/1/comprobaciones/consulta?offset=-1",
		"/servicios/0/comprobaciones/consulta",
		"/servicios/1/incidentes/consulta?estado=no-existe",
		"/servicios/1/incidentes/consulta?orden=latencia_desc",
		"/incidentes/consulta?limite=0",
		"/incidentes/consulta?desde=ayer",
		"/incidentes/consulta?desde=2026-11-20T10:00:00Z&hasta=2026-11-19T10:00:00Z",
		"/incidentes/consulta?offset=999999999999",
	}
	handler := NewHandler(nil, nil, nil, nil)
	for _, ruta := range casos {
		t.Run(ruta, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("se esperaba 400, recibido %d", w.Code)
			}
		})
	}
}

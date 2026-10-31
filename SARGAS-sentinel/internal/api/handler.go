package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"sargas-sentinel/internal/monitor"
	"sargas-sentinel/internal/repository"
	"sargas-sentinel/internal/service"

	"github.com/jackc/pgx/v5"
)

type servicioResponse struct {
	ID                int64   `json:"id"`
	Nombre            string  `json:"nombre"`
	URL               string  `json:"url"`
	Tipo              string  `json:"tipo"`
	IntervaloSegundos float64 `json:"intervalo_segundos"`
	TimeoutSegundos   float64 `json:"timeout_segundos"`
	Habilitado        bool    `json:"habilitado"`
}

type crearServicioRequest struct {
	Nombre            string `json:"nombre"`
	URL               string `json:"url"`
	IntervaloSegundos int32  `json:"intervalo_segundos"`
	TimeoutSegundos   int32  `json:"timeout_segundos"`
}

type estadoResponse struct {
	ServicioID         int64      `json:"servicio_id"`
	Estado             string     `json:"estado"`
	Habilitado         bool       `json:"habilitado"`
	ExitosConsecutivos int        `json:"exitos_consecutivos"`
	FallosConsecutivos int        `json:"fallos_consecutivos"`
	UltimaComprobacion *time.Time `json:"ultima_comprobacion_en"`
}

type incidenteResponse struct {
	ID                int64      `json:"id"`
	ServicioID        int64      `json:"servicio_id"`
	IniciadoEn        time.Time  `json:"iniciado_en"`
	CerradoEn         *time.Time `json:"cerrado_en"`
	EvidenciaInicio   string     `json:"evidencia_inicio"`
	EvidenciaCierre   *string    `json:"evidencia_cierre"`
	NombreServicio    string     `json:"nombre_servicio"`
	ServicioArchivado bool       `json:"servicio_archivado"`
}

func protegerOrigenAdministrativo(siguiente http.Handler) http.Handler {
	puerto := strings.TrimSpace(os.Getenv("SARGAS_PUBLIC_PORT"))

	if puerto == "" {
		puerto = "8080"
	}

	origenesPermitidos := map[string]bool{
		"http://localhost:5173":      true,
		"http://127.0.0.1:5173":      true,
		"http://localhost:" + puerto: true,
		"http://127.0.0.1:" + puerto: true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost ||
			r.Method == http.MethodPut ||
			r.Method == http.MethodPatch {

			origen := r.Header.Get("Origin")

			if origen != "" && !origenesPermitidos[origen] {
				http.Error(w, "Origen no permitido", http.StatusForbidden)
				return
			}
		}

		siguiente.ServeHTTP(w, r)
	})
}

func NewHandler(
	repo *repository.ServicioRepository,
	comprobacionRepo *repository.ComprobacionRepository,
	manager *monitor.Manager,
	consultaRepo *repository.ConsultaRepository,
) http.Handler {
	mux := http.NewServeMux()

	var administracionMu sync.Mutex

	mux.HandleFunc("/servicios", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		switch r.Method {
		case http.MethodGet:
			listarServicios(w, r, repo)

		case http.MethodPost:
			administracionMu.Lock()
			defer administracionMu.Unlock()

			crearServicio(w, r, repo, manager)

		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Método no permitido",
				http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("PUT /servicios/{id}", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		administracionMu.Lock()
		defer administracionMu.Unlock()
		actualizarServicio(w, r, repo, manager)

	})

	mux.HandleFunc("PATCH /servicios/{id}/habilitacion", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		administracionMu.Lock()
		defer administracionMu.Unlock()
		cambiarHabilitacion(w, r, repo, manager)
	})

	mux.HandleFunc("PATCH /servicios/{id}/archivar", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		administracionMu.Lock()
		defer administracionMu.Unlock()

		archivarServicio(w, r, repo, manager)
	})

	mux.HandleFunc("GET /servicios/{id}/comprobaciones", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		listarComprobaciones(w, r, comprobacionRepo)
	})

	mux.HandleFunc("GET /servicios/{id}/estado", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		consultarEstado(w, r, consultaRepo)
	})

	mux.HandleFunc("GET /servicios/{id}/incidentes", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		listarIncidentes(w, r, consultaRepo)
	})

	mux.HandleFunc("GET /incidentes", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		listarTodosIncidentes(w, r, consultaRepo)
	})

	return protegerOrigenAdministrativo(mux)
}

func listarServicios(
	w http.ResponseWriter,
	r *http.Request,
	repo *repository.ServicioRepository,
) {
	servicios, err := repo.Listar(r.Context())

	if err != nil {
		log.Printf("Error consultando servicios: %v", err)
		http.Error(w, "Error interno",
			http.StatusInternalServerError)
		return
	}

	respuesta := make([]servicioResponse, 0, len(servicios))

	for _, s := range servicios {
		respuesta = append(respuesta, convertirResponse(s))
	}

	responderJSON(w, http.StatusOK, respuesta)
}

func crearServicio(
	w http.ResponseWriter,
	r *http.Request,
	repo *repository.ServicioRepository,
	manager *monitor.Manager,
) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req crearServicioRequest

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "Se admite un único objeto JSON",
			http.StatusBadRequest)
		return
	}

	req.Nombre = strings.TrimSpace(req.Nombre)
	req.URL = strings.TrimSpace(req.URL)

	if req.Nombre == "" ||
		utf8.RuneCountInString(req.Nombre) > 120 {
		http.Error(w, "Nombre inválido",
			http.StatusBadRequest)
		return
	}

	destino, err := url.ParseRequestURI(req.URL)

	if err != nil ||
		(destino.Scheme != "http" && destino.Scheme != "https") ||
		destino.Hostname() == "" ||
		destino.User != nil {
		http.Error(w, "URL HTTP/HTTPS inválida",
			http.StatusBadRequest)
		return
	}

	if req.IntervaloSegundos <= 0 || req.TimeoutSegundos <= 0 {
		http.Error(w, "Intervalo y timeout deben ser positivos",
			http.StatusBadRequest)
		return
	}

	s := service.Servicio{
		Nombre:     req.Nombre,
		URL:        req.URL,
		Tipo:       "HTTP",
		Intervalo:  time.Duration(req.IntervaloSegundos) * time.Second,
		Timeout:    time.Duration(req.TimeoutSegundos) * time.Second,
		Habilitado: true,
	}

	creado, err := repo.Crear(r.Context(), s)

	if err != nil {
		log.Printf("Error creando servicio: %v", err)
		http.Error(w, "Error interno",
			http.StatusInternalServerError)
		return
	}

	if err := manager.Aplicar(creado); err != nil {
		log.Printf(
			"Servicio %d guardado, pero no sincronizado: %v",
			creado.ID, err,
		)

		http.Error(w, "Servicio guardado, monitoreo no sincronizado",
			http.StatusServiceUnavailable)
		return
	}

	responderJSON(w, http.StatusCreated, convertirResponse(creado))
}

func convertirResponse(s service.Servicio) servicioResponse {
	return servicioResponse{
		ID:                s.ID,
		Nombre:            s.Nombre,
		URL:               s.URL,
		Tipo:              s.Tipo,
		IntervaloSegundos: s.Intervalo.Seconds(),
		TimeoutSegundos:   s.Timeout.Seconds(),
		Habilitado:        s.Habilitado,
	}
}

func responderJSON(w http.ResponseWriter, status int, datos any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(datos); err != nil {
		log.Printf("Error generando JSON: %v", err)
	}
}

func actualizarServicio(
	w http.ResponseWriter,
	r *http.Request,
	repo *repository.ServicioRepository,
	manager *monitor.Manager,
) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)

	if err != nil || id <= 0 {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req crearServicioRequest

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "Se admite un único objeto JSON",
			http.StatusBadRequest)
		return
	}

	req.Nombre = strings.TrimSpace(req.Nombre)
	req.URL = strings.TrimSpace(req.URL)

	if req.Nombre == "" ||
		utf8.RuneCountInString(req.Nombre) > 120 {
		http.Error(w, "Nombre inválido", http.StatusBadRequest)
		return
	}

	destino, err := url.ParseRequestURI(req.URL)

	if err != nil ||
		(destino.Scheme != "http" && destino.Scheme != "https") ||
		destino.Hostname() == "" ||
		destino.User != nil {
		http.Error(w, "URL HTTP/HTTPS inválida",
			http.StatusBadRequest)
		return
	}

	if req.IntervaloSegundos <= 0 || req.TimeoutSegundos <= 0 {
		http.Error(w, "Intervalo y timeout deben ser positivos",
			http.StatusBadRequest)
		return
	}

	s := service.Servicio{
		ID:        id,
		Nombre:    req.Nombre,
		URL:       req.URL,
		Intervalo: time.Duration(req.IntervaloSegundos) * time.Second,
		Timeout:   time.Duration(req.TimeoutSegundos) * time.Second,
	}

	actualizado, err := repo.Actualizar(r.Context(), s)

	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Servicio no encontrado",
			http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("Error actualizando servicio: %v", err)
		http.Error(w, "Error interno",
			http.StatusInternalServerError)
		return
	}

	if err := manager.Aplicar(actualizado); err != nil {
		log.Printf(
			"Servicio %d actualizado, pero no sincronizado: %v",
			actualizado.ID, err,
		)

		http.Error(w, "Servicio actualizado, monitoreo no sincronizado",
			http.StatusServiceUnavailable)
		return
	}

	responderJSON(w, http.StatusOK, convertirResponse(actualizado))
}

type habilitacionRequest struct {
	Habilitado *bool `json:"habilitado"`
}

type comprobacionResponse struct {
	ID           int64     `json:"id"`
	ServicioID   int64     `json:"servicio_id"`
	CodigoHTTP   *int32    `json:"codigo_http"`
	LatenciaMS   float64   `json:"latencia_ms"`
	Error        *string   `json:"error"`
	ComprobadoEn time.Time `json:"comprobado_en"`
}

func cambiarHabilitacion(
	w http.ResponseWriter,
	r *http.Request,
	repo *repository.ServicioRepository,
	manager *monitor.Manager,
) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)

	if err != nil || id <= 0 {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req habilitacionRequest

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "Se admite un único objeto JSON",
			http.StatusBadRequest)
		return
	}

	if req.Habilitado == nil {
		http.Error(w, "El campo habilitado es obligatorio",
			http.StatusBadRequest)
		return
	}

	actualizado, err := repo.CambiarHabilitacion(
		r.Context(),
		id,
		*req.Habilitado,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Servicio no encontrado",
			http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("Error cambiando habilitación: %v", err)
		http.Error(w, "Error interno",
			http.StatusInternalServerError)
		return
	}

	if err := manager.Aplicar(actualizado); err != nil {
		log.Printf(
			"Habilitación del servicio %d no sincronizada: %v",
			actualizado.ID, err,
		)

		http.Error(w, "Cambio guardado, monitoreo no sincronizado",
			http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func listarComprobaciones(
	w http.ResponseWriter,
	r *http.Request,
	repo *repository.ComprobacionRepository,
) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)

	if err != nil || id <= 0 {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	limite := int32(50)
	offset := int32(0)

	if valor := r.URL.Query().Get("limite"); valor != "" {
		numero, err := strconv.ParseInt(valor, 10, 32)

		if err != nil || numero < 1 || numero > 100 {
			http.Error(w, "Límite inválido (1-100)",
				http.StatusBadRequest)
			return
		}

		limite = int32(numero)
	}

	if valor := r.URL.Query().Get("offset"); valor != "" {
		numero, err := strconv.ParseInt(valor, 10, 32)

		if err != nil || numero < 0 {
			http.Error(w, "Offset inválido",
				http.StatusBadRequest)
			return
		}

		offset = int32(numero)
	}

	registros, err := repo.ListarPorServicio(
		r.Context(),
		id,
		limite,
		offset,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Servicio no encontrado",
			http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("Error consultando historial: %v", err)
		http.Error(w, "Error interno",
			http.StatusInternalServerError)
		return
	}

	respuesta := make([]comprobacionResponse, 0, len(registros))

	for _, c := range registros {
		respuesta = append(respuesta, comprobacionResponse{
			ID:           c.ID,
			ServicioID:   c.ServicioID,
			CodigoHTTP:   c.CodigoHTTP,
			LatenciaMS:   c.LatenciaMS,
			Error:        c.Error,
			ComprobadoEn: c.ComprobadoEn,
		})
	}

	responderJSON(w, http.StatusOK, respuesta)
}

func consultarEstado(
	w http.ResponseWriter,
	r *http.Request,
	repo *repository.ConsultaRepository,
) {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)

	if err != nil || id <= 0 {
		http.Error(
			w,
			"ID inválido",
			http.StatusBadRequest,
		)
		return
	}

	estado, err := repo.ObtenerEstado(
		r.Context(),
		id,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(
			w,
			"Servicio no encontrado",
			http.StatusNotFound,
		)
		return
	}

	if err != nil {
		log.Printf("Error consultando estado: %v", err)

		http.Error(
			w,
			"Error interno",
			http.StatusInternalServerError,
		)
		return
	}

	respuesta := estadoResponse{
		ServicioID:         estado.ServicioID,
		Estado:             estado.Estado,
		Habilitado:         estado.Habilitado,
		ExitosConsecutivos: estado.ExitosConsecutivos,
		FallosConsecutivos: estado.FallosConsecutivos,
		UltimaComprobacion: estado.UltimaComprobacion,
	}

	responderJSON(w, http.StatusOK, respuesta)
}

func listarIncidentes(
	w http.ResponseWriter,
	r *http.Request,
	repo *repository.ConsultaRepository,
) {
	id, err := strconv.ParseInt(
		r.PathValue("id"), 10, 64,
	)

	if err != nil || id <= 0 {
		http.Error(
			w, "ID inválido",
			http.StatusBadRequest,
		)
		return
	}

	limite := int32(50)
	offset := int32(0)

	if valor := r.URL.Query().Get("limite"); valor != "" {
		numero, err := strconv.ParseInt(valor, 10, 32)

		if err != nil || numero < 1 || numero > 100 {
			http.Error(
				w, "Límite inválido (1-100)",
				http.StatusBadRequest,
			)
			return
		}

		limite = int32(numero)
	}

	if valor := r.URL.Query().Get("offset"); valor != "" {
		numero, err := strconv.ParseInt(valor, 10, 32)

		if err != nil || numero < 0 {
			http.Error(
				w, "Offset inválido",
				http.StatusBadRequest,
			)
			return
		}

		offset = int32(numero)
	}

	incidentes, err := repo.ListarIncidentes(
		r.Context(), id, limite, offset,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(
			w, "Servicio no encontrado",
			http.StatusNotFound,
		)
		return
	}

	if err != nil {
		log.Printf("Error consultando incidentes: %v", err)
		http.Error(
			w, "Error interno",
			http.StatusInternalServerError,
		)
		return
	}

	respuesta := make([]incidenteResponse, 0, len(incidentes))

	for _, i := range incidentes {
		respuesta = append(respuesta, incidenteResponse{
			ID:                i.ID,
			ServicioID:        i.ServicioID,
			IniciadoEn:        i.IniciadoEn,
			CerradoEn:         i.CerradoEn,
			EvidenciaInicio:   i.EvidenciaInicio,
			EvidenciaCierre:   i.EvidenciaCierre,
			NombreServicio:    i.NombreServicio,
			ServicioArchivado: i.ServicioArchivado,
		})
	}

	responderJSON(w, http.StatusOK, respuesta)
}

func listarTodosIncidentes(
	w http.ResponseWriter,
	r *http.Request,
	repo *repository.ConsultaRepository,
) {
	limite := int32(50)
	offset := int32(0)

	if valor := r.URL.Query().Get("limite"); valor != "" {
		numero, err := strconv.ParseInt(valor, 10, 32)

		if err != nil || numero < 1 || numero > 100 {
			http.Error(
				w, "Límite inválido (1-100)",
				http.StatusBadRequest,
			)
			return
		}

		limite = int32(numero)
	}

	if valor := r.URL.Query().Get("offset"); valor != "" {
		numero, err := strconv.ParseInt(valor, 10, 32)

		if err != nil || numero < 0 {
			http.Error(
				w, "Offset inválido",
				http.StatusBadRequest,
			)
			return
		}

		offset = int32(numero)
	}

	incidentes, err := repo.ListarTodosIncidentes(
		r.Context(), limite, offset,
	)

	if err != nil {
		log.Printf("Error consultando incidentes globales: %v", err)
		http.Error(
			w, "Error interno",
			http.StatusInternalServerError,
		)
		return
	}

	respuesta := make([]incidenteResponse, 0, len(incidentes))

	for _, i := range incidentes {
		respuesta = append(respuesta, incidenteResponse{
			ID:                i.ID,
			ServicioID:        i.ServicioID,
			IniciadoEn:        i.IniciadoEn,
			CerradoEn:         i.CerradoEn,
			EvidenciaInicio:   i.EvidenciaInicio,
			EvidenciaCierre:   i.EvidenciaCierre,
			NombreServicio:    i.NombreServicio,
			ServicioArchivado: i.ServicioArchivado,
		})
	}

	responderJSON(w, http.StatusOK, respuesta)
}

func archivarServicio(
	w http.ResponseWriter,
	r *http.Request,
	repo *repository.ServicioRepository,
	manager *monitor.Manager,
) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)

	if err != nil || id <= 0 {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	archivado, err := repo.Archivar(r.Context(), id)

	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(
			w,
			"Servicio no encontrado o ya archivado",
			http.StatusNotFound,
		)
		return
	}

	if err != nil {
		log.Printf("Error archivando servicio %d: %v", id, err)
		http.Error(
			w,
			"Error interno",
			http.StatusInternalServerError,
		)
		return
	}

	if err := manager.Aplicar(archivado); err != nil {
		log.Printf(
			"Servicio %d archivado, pero monitoreo no sincronizado: %v",
			id, err,
		)

		http.Error(
			w,
			"Servicio archivado, monitoreo no sincronizado",
			http.StatusServiceUnavailable,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

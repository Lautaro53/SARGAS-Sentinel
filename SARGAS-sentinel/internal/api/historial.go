package api

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"sargas-sentinel/internal/repository"

	"github.com/jackc/pgx/v5"
)

type paginaHistorial[T any] struct {
	Registros []T   `json:"registros"`
	Total     int64 `json:"total"`
}

func parametroEntero(r *http.Request, nombre string, defecto int32, minimo, maximo int64) (int32, error) {
	valor := r.URL.Query().Get(nombre)
	if valor == "" {
		return defecto, nil
	}
	numero, err := strconv.ParseInt(valor, 10, 32)
	if err != nil || numero < minimo || numero > maximo {
		return 0, fmt.Errorf("%s inválido", nombre)
	}
	return int32(numero), nil
}

func parametroLista(r *http.Request, nombre, defecto string, permitidos ...string) (string, error) {
	valor := r.URL.Query().Get(nombre)
	if valor == "" {
		valor = defecto
	}
	for _, permitido := range permitidos {
		if valor == permitido {
			return valor, nil
		}
	}
	return "", fmt.Errorf("%s inválido", nombre)
}

func parametroLatencia(r *http.Request, nombre string) (*float64, error) {
	valor := r.URL.Query().Get(nombre)
	if valor == "" {
		return nil, nil
	}
	numero, err := strconv.ParseFloat(valor, 64)
	if err != nil || math.IsNaN(numero) || math.IsInf(numero, 0) || numero < 0 {
		return nil, fmt.Errorf("%s inválida", nombre)
	}
	return &numero, nil
}

func parametroFecha(r *http.Request, nombre string) (*time.Time, error) {
	valor := r.URL.Query().Get(nombre)
	if valor == "" {
		return nil, nil
	}
	fecha, err := time.Parse(time.RFC3339Nano, valor)
	if err != nil {
		return nil, fmt.Errorf("%s inválida", nombre)
	}
	return &fecha, nil
}

func parametrosPagina(r *http.Request) (int32, int32, error) {
	limite, err := parametroEntero(r, "limite", 10, 1, 100)
	if err != nil {
		return 0, 0, err
	}
	offset, err := parametroEntero(r, "offset", 0, 0, math.MaxInt32)
	return limite, offset, err
}

func idServicioHistorial(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("ID inválido")
	}
	return id, nil
}

func consultarHistorialComprobaciones(w http.ResponseWriter, r *http.Request, repo *repository.ComprobacionRepository) {
	id, err := idServicioHistorial(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	limite, offset, err := parametrosPagina(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	codigo, err := parametroLista(r, "codigo", "todos", "todos", "fallidas", "2xx", "3xx", "4xx", "5xx", "sin_respuesta")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	orden, err := parametroLista(r, "orden", "fecha_desc", "fecha_desc", "fecha_asc", "latencia_desc", "latencia_asc")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	minimo, err := parametroLatencia(r, "latencia_min")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	maximo, err := parametroLatencia(r, "latencia_max")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if minimo != nil && maximo != nil && *minimo > *maximo {
		http.Error(w, "La latencia mínima supera la máxima", http.StatusBadRequest)
		return
	}
	datos, total, err := repo.ConsultarHistorial(r.Context(), id, limite, offset,
		repository.FiltroHistorialComprobaciones{Codigo: codigo, Orden: orden, LatenciaMin: minimo, LatenciaMax: maximo})
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Servicio no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error consultando historial filtrado: %v", err)
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}
	registros := make([]comprobacionResponse, 0, len(datos))
	for _, c := range datos {
		registros = append(registros, comprobacionResponse{ID: c.ID, ServicioID: c.ServicioID, CodigoHTTP: c.CodigoHTTP,
			LatenciaMS: c.LatenciaMS, Error: c.Error, ComprobadoEn: c.ComprobadoEn})
	}
	responderJSON(w, http.StatusOK, paginaHistorial[comprobacionResponse]{Registros: registros, Total: total})
}

func consultarHistorialIncidentes(w http.ResponseWriter, r *http.Request, repo *repository.ConsultaRepository, global bool) {
	var id int64
	if !global {
		var err error
		id, err = idServicioHistorial(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	limite, offset, err := parametrosPagina(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	estado, err := parametroLista(r, "estado", "todos", "todos", "abiertos", "cerrados")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	orden, err := parametroLista(r, "orden", "fecha_desc", "fecha_desc", "fecha_asc")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	desde, err := parametroFecha(r, "desde")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hasta, err := parametroFecha(r, "hasta")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if desde != nil && hasta != nil && !desde.Before(*hasta) {
		http.Error(w, "La fecha inicial debe ser anterior a la final", http.StatusBadRequest)
		return
	}
	datos, total, err := repo.ConsultarHistorialIncidentes(r.Context(), id, limite, offset,
		repository.FiltroHistorialIncidentes{Estado: estado, Orden: orden, Desde: desde, Hasta: hasta})
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Servicio no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error consultando incidentes filtrados: %v", err)
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}
	registros := make([]incidenteResponse, 0, len(datos))
	for _, i := range datos {
		registros = append(registros, incidenteResponse{ID: i.ID, ServicioID: i.ServicioID, IniciadoEn: i.IniciadoEn,
			CerradoEn: i.CerradoEn, EvidenciaInicio: i.EvidenciaInicio, EvidenciaCierre: i.EvidenciaCierre,
			NombreServicio: i.NombreServicio, ServicioArchivado: i.ServicioArchivado})
	}
	responderJSON(w, http.StatusOK, paginaHistorial[incidenteResponse]{Registros: registros, Total: total})
}

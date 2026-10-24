package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sargas-sentinel/internal/api"
	"sargas-sentinel/internal/database"
	"sargas-sentinel/internal/monitor"
	"sargas-sentinel/internal/repository"
	"sargas-sentinel/internal/service"
)

const (
	reset    = "\x1b[0m"
	verde    = "\x1b[32m"
	rojo     = "\x1b[31m"
	amarillo = "\x1b[33m"
	cyan     = "\x1b[36m"
)

func main() {
	if err := ejecutar(); err != nil {
		fmt.Fprintln(os.Stderr, "SARGAS Sentinel:", err)
		os.Exit(1)
	}
}

func ejecutar() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Conexión a PostgreSQL
	pool, err := database.Connect(ctx)
	if err != nil {
		return fmt.Errorf("conectar PostgreSQL: %w", err)
	}
	defer pool.Close()

	fmt.Println("PostgreSQL conectado correctamente")

	// Repositorios
	servicioRepo := repository.NuevoServicioRepository(pool)
	comprobacionRepo := repository.NuevoComprobacionRepository(pool)
	consultaRepo := repository.NuevoConsultaRepository(pool)

	// Cargar servicios registrados
	servicios, err := servicioRepo.Listar(ctx)
	if err != nil {
		return fmt.Errorf("cargar servicios: %w", err)
	}

	// Administrador de monitoreo

	manager := monitor.NewManager(
		context.Background(),
		func(ctx context.Context, s service.Servicio) {
			comprobar(ctx, s, comprobacionRepo)
		},
	)

	// Iniciar monitoreo de servicios habilitados
	activos := 0

	for _, s := range servicios {
		if err := manager.Aplicar(s); err != nil {
			manager.Cerrar()
			return fmt.Errorf("iniciar monitoreo de %q: %w", s.Nombre, err)
		}

		if s.Habilitado {
			activos++
		}
	}

	fmt.Printf("Servicios en monitoreo: %d\n", activos)

	// Servidor HTTP

	// API HTTP
	apiHandler := api.NewHandler(
		servicioRepo,
		comprobacionRepo,
		manager,
		consultaRepo,
	)

	// Frontend compilado + API
	webHandler, err := nuevoHandlerWeb(apiHandler)
	if err != nil {
		manager.Cerrar()
		return fmt.Errorf("preparar frontend: %w", err)
	}

	direccionHTTP := os.Getenv("SARGAS_HTTP_ADDR")
	if direccionHTTP == "" {
		direccionHTTP = "127.0.0.1:8080"
	}

	// Servidor HTTP
	servidor := &http.Server{
		Addr:              direccionHTTP,
		Handler:           protegerHostLocal(webHandler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	erroresServidor := make(chan error, 1)

	go func() {
		if err := servidor.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			erroresServidor <- err
		}
	}()

	fmt.Println("SARGAS Sentinel iniciado")
	fmt.Println("Web: http://127.0.0.1:8080/")
	fmt.Println("API: http://127.0.0.1:8080/api/servicios")
	fmt.Println("Presiona Ctrl+C para detener")

	// Esperar una señal de apagado o un fallo real del servidor.
	var errorServidor error
	select {
	case <-ctx.Done():
		fmt.Println("Deteniendo Sentinel...")

	case err := <-erroresServidor:
		errorServidor = fmt.Errorf("servidor HTTP: %w", err)
		stop()
	}

	// Cierre ordenado del servidor HTTP
	cierreCtx, cancelar := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancelar()

	errorCierre := servidor.Shutdown(cierreCtx)

	// Finalizar trabajadores antes de cerrar la conexión a PostgreSQL.
	manager.Cerrar()

	fmt.Println("SARGAS Sentinel finalizado")
	if errorCierre != nil {
		errorCierre = fmt.Errorf("cerrar API: %w", errorCierre)
	}
	return errors.Join(errorServidor, errorCierre)
}

func protegerHostLocal(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.ToLower(r.Host) {
		case "127.0.0.1:8080", "localhost:8080":
			siguiente.ServeHTTP(w, r)

		default:
			http.Error(w, "Host no permitido", http.StatusForbidden)
		}
	})
}

func mensajeSeguroMonitoreo(err error) string {
	var errorDNS *net.DNSError
	var errorRed net.Error

	switch {
	case errors.Is(err, context.Canceled):
		return "Comprobación cancelada"

	case errors.Is(err, context.DeadlineExceeded):
		return "Tiempo de espera agotado"

	case errors.As(err, &errorDNS):
		if errorDNS.IsNotFound {
			return "DNS: dominio no encontrado"
		}
		return "Error de resolución DNS"

	case errors.As(err, &errorRed) && errorRed.Timeout():
		return "Tiempo de espera agotado"

	default:
		return "Error de conexión HTTP"
	}
}

func comprobar(
	ctx context.Context,
	s service.Servicio,
	repo *repository.ComprobacionRepository,
) {
	if ctx.Err() != nil {
		return
	}

	// Ejecutar comprobación HTTP.
	codigo, duracion, err := monitor.CheckHTTP(
		ctx,
		s.URL,
		s.Timeout,
	)

	if ctx.Err() != nil {
		return
	}

	// Preparar resultado.
	registro := repository.Comprobacion{
		ServicioID: s.ID,
		Latencia:   duracion,
	}

	if err != nil {
		mensaje := mensajeSeguroMonitoreo(err)
		registro.Error = &mensaje
	} else {
		codigoHTTP := int32(codigo)
		registro.CodigoHTTP = &codigoHTTP
	}

	// Persistir comprobación, estado e incidente.
	resultado, errGuardar := repo.RegistrarEvaluacion(
		ctx,
		registro,
	)

	if errGuardar != nil {
		fmt.Fprintf(
			os.Stderr,
			"%sERROR DE PERSISTENCIA%s | %s | %v\n",
			rojo, reset,
			s.Nombre,
			errGuardar,
		)
		return
	}

	// Informar transiciones confirmadas.
	if resultado.Cambio {
		fmt.Printf(
			"%s CAMBIO DE ESTADO: %s -> %s\n",
			s.Nombre,
			resultado.Anterior,
			resultado.Actual,
		)
	}

	// Informar eventos de incidentes.
	switch resultado.Evento {

	case monitor.IncidenteAbierto:
		fmt.Printf(
			"%sINCIDENTE ABIERTO%s | %s | ID: %d\n",
			rojo, reset,
			s.Nombre,
			resultado.IncidenteID,
		)

	case monitor.IncidenteCerrado:
		fmt.Printf(
			"%sINCIDENTE CERRADO%s | %s | ID: %d | Duración: %s\n",
			verde, reset,
			s.Nombre,
			resultado.IncidenteID,
			resultado.DuracionIncidente.Round(time.Second),
		)
	}

	// Preparar salida de consola.
	hora := time.Now().Format("15:04:05")
	latencia := duracion.Round(time.Microsecond)

	encabezado := fmt.Sprintf(
		"%s[%s]%s  %-20s",
		cyan, hora, reset, s.Nombre,
	)

	colorEstado := amarillo

	switch resultado.Actual {
	case monitor.EstadoUp:
		colorEstado = verde

	case monitor.EstadoDown:
		colorEstado = rojo
	}

	// Mostrar errores de conexión.
	if err != nil {
		fmt.Printf(
			"%s  %sERROR%s  %10s  ESTADO %s%s%s\n    %v\n",
			encabezado,
			rojo, reset,
			latencia,
			colorEstado, resultado.Actual, reset,
			*registro.Error,
		)
		return
	}

	// Mostrar respuestas HTTP.
	colorHTTP := amarillo

	if codigo >= 200 && codigo < 300 {
		colorHTTP = verde
	}

	fmt.Printf(
		"%s  %sHTTP %-3d%s  %10s  ESTADO %s%s%s\n",
		encabezado,
		colorHTTP, codigo, reset,
		latencia,
		colorEstado, resultado.Actual, reset,
	)
}

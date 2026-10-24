package monitor

import (
	"testing"
	"time"
)

func TestCicloDeIncidente(t *testing.T) {
	gestor := NuevoGestorIncidentes()

	inicio := time.Date(
		2026, 10, 11, 12, 0, 0, 0, time.UTC,
	)

	// Un servicio UP no genera un incidente.
	tipo, incidente := gestor.ProcesarTransicion(
		1, EstadoUnknown, EstadoUp,
		inicio, "HTTP 200",
	)

	if tipo != "" || incidente != nil {
		t.Fatal("No debe abrir incidente al pasar a UP")
	}

	// Se confirma una caida.
	tipo, incidente = gestor.ProcesarTransicion(
		1, EstadoUp, EstadoDown,
		inicio.Add(time.Minute), "HTTP 500",
	)

	if tipo != IncidenteAbierto || incidente == nil {
		t.Fatal("Debe abrir un incidente")
	}

	if incidente.CerradoEn != nil {
		t.Fatal("El incidente debe permanecer abierto")
	}

	// Permanecer DOWN no debe abrir otro incidente.
	tipo, incidente = gestor.ProcesarTransicion(
		1, EstadoDown, EstadoDown,
		inicio.Add(2*time.Minute), "HTTP 500",
	)

	if tipo != "" || incidente != nil {
		t.Fatal("No debe duplicar incidentes")
	}

	// Se confirma la recuperacion.
	tipo, incidente = gestor.ProcesarTransicion(
		1, EstadoDown, EstadoUp,
		inicio.Add(3*time.Minute), "HTTP 200",
	)

	if tipo != IncidenteCerrado || incidente == nil {
		t.Fatal("Debe cerrar el incidente")
	}

	if incidente.CerradoEn == nil {
		t.Fatal("Falta la fecha de cierre")
	}

	if incidente.CerradoEn.Sub(incidente.IniciadoEn) != 2*time.Minute {
		t.Fatal("Duracion incorrecta")
	}
}

func TestIncidenteDesdeUnknown(t *testing.T) {
	gestor := NuevoGestorIncidentes()

	tipo, incidente := gestor.ProcesarTransicion(
		2,
		EstadoUnknown,
		EstadoDown,
		time.Now(),
		"Conexion rechazada",
	)

	if tipo != IncidenteAbierto || incidente == nil {
		t.Fatal("Debe abrir incidente desde UNKNOWN")
	}
}

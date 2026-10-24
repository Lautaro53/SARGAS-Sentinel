package monitor

import "time"

type TipoEventoIncidente string

const (
	IncidenteAbierto TipoEventoIncidente = "ABIERTO"
	IncidenteCerrado TipoEventoIncidente = "CERRADO"
)

type Incidente struct {
	ServicioID      int64
	IniciadoEn      time.Time
	CerradoEn       *time.Time
	EvidenciaInicio string
	EvidenciaCierre string
}

type GestorIncidentes struct {
	activo *Incidente
}

func NuevoGestorIncidentes() *GestorIncidentes {
	return &GestorIncidentes{}
}

func (g *GestorIncidentes) ProcesarTransicion(
	servicioID int64,
	anterior Estado,
	actual Estado,
	momento time.Time,
	evidencia string,
) (TipoEventoIncidente, *Incidente) {

	switch {
	case anterior != EstadoDown && actual == EstadoDown:
		if g.activo != nil {
			return "", nil
		}

		g.activo = &Incidente{
			ServicioID:      servicioID,
			IniciadoEn:      momento,
			EvidenciaInicio: evidencia,
		}

		return IncidenteAbierto, g.activo

	case anterior == EstadoDown && actual == EstadoUp:
		if g.activo == nil {
			return "", nil
		}

		incidente := g.activo
		incidente.CerradoEn = &momento
		incidente.EvidenciaCierre = evidencia

		g.activo = nil

		return IncidenteCerrado, incidente
	}

	return "", nil
}

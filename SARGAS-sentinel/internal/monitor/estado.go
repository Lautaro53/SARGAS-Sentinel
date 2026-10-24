package monitor

type Estado string

const (
	EstadoUnknown Estado = "UNKNOWN"
	EstadoUp      Estado = "UP"
	EstadoDown    Estado = "DOWN"
)

const (
	ExitosNecesarios = 2
	FallosNecesarios = 2
)

type EvaluadorEstado struct {
	Actual             Estado
	ExitosConsecutivos int
	FallosConsecutivos int
}

func NuevoEvaluadorEstado() *EvaluadorEstado {
	return &EvaluadorEstado{
		Actual: EstadoUnknown,
	}
}

func (e *EvaluadorEstado) Registrar(
	exito bool,
) (Estado, Estado, bool) {

	anterior := e.Actual

	if exito {
		e.FallosConsecutivos = 0

		if e.ExitosConsecutivos < ExitosNecesarios {
			e.ExitosConsecutivos++
		}

		if e.ExitosConsecutivos >= ExitosNecesarios {
			e.Actual = EstadoUp
		}

	} else {
		e.ExitosConsecutivos = 0

		if e.FallosConsecutivos < FallosNecesarios {
			e.FallosConsecutivos++
		}

		if e.FallosConsecutivos >= FallosNecesarios {
			e.Actual = EstadoDown
		}
	}

	return anterior, e.Actual, anterior != e.Actual
}

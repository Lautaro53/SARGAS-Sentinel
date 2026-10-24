package monitor

import "testing"

func TestEvaluadorEstado(t *testing.T) {
	casos := []struct {
		nombre     string
		resultados []bool
		esperados  []Estado
	}{
		{
			nombre:     "Dos exitos consecutivos",
			resultados: []bool{true, true},
			esperados:  []Estado{EstadoUnknown, EstadoUp},
		},
		{
			nombre:     "Dos fallos consecutivos",
			resultados: []bool{false, false},
			esperados:  []Estado{EstadoUnknown, EstadoDown},
		},
		{
			nombre:     "Fallo aislado no provoca caida",
			resultados: []bool{true, true, false, true},
			esperados:  []Estado{EstadoUnknown, EstadoUp, EstadoUp, EstadoUp},
		},
		{
			nombre:     "Caida y recuperacion",
			resultados: []bool{true, true, false, false, true, true},
			esperados: []Estado{
				EstadoUnknown,
				EstadoUp,
				EstadoUp,
				EstadoDown,
				EstadoDown,
				EstadoUp,
			},
		},
		{
			nombre:     "Resultados alternados",
			resultados: []bool{true, false, true, false},
			esperados: []Estado{
				EstadoUnknown,
				EstadoUnknown,
				EstadoUnknown,
				EstadoUnknown,
			},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			evaluador := NuevoEvaluadorEstado()

			for i, exito := range caso.resultados {
				_, actual, _ := evaluador.Registrar(exito)

				if actual != caso.esperados[i] {
					t.Fatalf(
						"Comprobacion %d: esperado %s, obtenido %s",
						i+1,
						caso.esperados[i],
						actual,
					)
				}
			}
		})
	}
}

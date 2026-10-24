package monitor

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"sargas-sentinel/internal/service"
)

func TestManagerPausaTrabajador(t *testing.T) {
	iniciado := make(chan struct{})
	cancelado := make(chan struct{})

	manager := NewManager(
		context.Background(),
		func(ctx context.Context, s service.Servicio) {
			close(iniciado)
			<-ctx.Done()
			close(cancelado)
		},
	)
	defer manager.Cerrar()

	s := service.Servicio{
		ID:         1,
		Intervalo:  time.Hour,
		Habilitado: true,
	}

	// Iniciar el monitoreo.
	if err := manager.Aplicar(s); err != nil {
		t.Fatal(err)
	}

	select {
	case <-iniciado:
	case <-time.After(3 * time.Second):
		t.Fatal("El trabajador no inicio")
	}

	// Pausar el servicio.
	s.Habilitado = false

	if err := manager.Aplicar(s); err != nil {
		t.Fatal(err)
	}

	// La cancelacion debe haberse completado.
	select {
	case <-cancelado:
	default:
		t.Fatal("El trabajador no fue cancelado")
	}

	// No debe quedar ningun trabajador activo.
	manager.mu.Lock()
	restantes := len(manager.trabajadores)
	manager.mu.Unlock()

	if restantes != 0 {
		t.Fatalf(
			"Se esperaban 0 trabajadores, hay %d",
			restantes,
		)
	}
}

func TestManagerReemplazaTrabajador(t *testing.T) {
	eventos := make(chan string, 4)

	manager := NewManager(
		context.Background(),
		func(ctx context.Context, s service.Servicio) {
			eventos <- "inicio:" + s.Nombre
			<-ctx.Done()
			eventos <- "fin:" + s.Nombre
		},
	)
	defer manager.Cerrar()

	esperarEvento := func(esperado string) {
		t.Helper()

		select {
		case recibido := <-eventos:
			if recibido != esperado {
				t.Fatalf("Esperado %q, recibido %q",
					esperado, recibido)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("No se recibio el evento %q", esperado)
		}
	}

	s := service.Servicio{
		ID:         1,
		Nombre:     "Original",
		Intervalo:  time.Hour,
		Habilitado: true,
	}

	// Iniciar el primer trabajador.
	if err := manager.Aplicar(s); err != nil {
		t.Fatal(err)
	}

	esperarEvento("inicio:Original")

	// Simular una edicion del servicio.
	s.Nombre = "Actualizado"

	if err := manager.Aplicar(s); err != nil {
		t.Fatal(err)
	}

	// El anterior debe finalizar antes de iniciar el nuevo.
	esperarEvento("fin:Original")
	esperarEvento("inicio:Actualizado")

	manager.mu.Lock()
	restantes := len(manager.trabajadores)
	manager.mu.Unlock()

	if restantes != 1 {
		t.Fatalf(
			"Se esperaba 1 trabajador, hay %d",
			restantes,
		)
	}
}

func TestManagerCerrarDetieneTrabajadores(t *testing.T) {
	iniciados := make(chan int64, 2)
	terminados := make(chan int64, 2)

	manager := NewManager(
		context.Background(),
		func(ctx context.Context, s service.Servicio) {
			iniciados <- s.ID

			<-ctx.Done()

			terminados <- s.ID
		},
	)

	// Iniciar dos trabajadores independientes.
	for id := int64(1); id <= 2; id++ {
		s := service.Servicio{
			ID:         id,
			Intervalo:  time.Hour,
			Habilitado: true,
		}

		if err := manager.Aplicar(s); err != nil {
			t.Fatal(err)
		}
	}

	// Esperar hasta que ambos esten ejecutandose.
	for i := 0; i < 2; i++ {
		select {
		case <-iniciados:
		case <-time.After(3 * time.Second):
			t.Fatal("No iniciaron todos los trabajadores")
		}
	}

	// Ejecutar el cierre sin bloquear indefinidamente el test.
	cierreCompleto := make(chan struct{})

	go func() {
		manager.Cerrar()
		close(cierreCompleto)
	}()

	select {
	case <-cierreCompleto:
	case <-time.After(3 * time.Second):
		t.Fatal("El Manager no termino de cerrar")
	}

	// Comprobar que ambos trabajadores finalizaron.
	finalizados := make(map[int64]bool)

	for i := 0; i < 2; i++ {
		select {
		case id := <-terminados:
			finalizados[id] = true
		default:
			t.Fatal("El cierre retorno antes de terminar los trabajadores")
		}
	}

	if !finalizados[1] || !finalizados[2] {
		t.Fatal("No finalizaron los dos trabajadores")
	}

	// Verificar el estado interno del Manager.
	manager.mu.Lock()
	restantes := len(manager.trabajadores)
	cerrado := manager.cerrado
	manager.mu.Unlock()

	if restantes != 0 || !cerrado {
		t.Fatalf(
			"Estado incorrecto: restantes=%d, cerrado=%v",
			restantes, cerrado,
		)
	}

	// Cerrar nuevamente debe ser seguro.
	manager.Cerrar()

	// No deben aceptarse nuevos trabajadores.
	err := manager.Aplicar(service.Servicio{
		ID:         3,
		Intervalo:  time.Hour,
		Habilitado: true,
	})

	if err == nil {
		t.Fatal("El Manager acepto un trabajador despues de cerrar")
	}
}

func TestManagerAplicarConcurrenteMismoServicio(t *testing.T) {
	var activos atomic.Int32
	var huboSolapamiento atomic.Bool
	iniciado := make(chan struct{}, 1)

	manager := NewManager(context.Background(), func(ctx context.Context, s service.Servicio) {
		if activos.Add(1) > 1 {
			huboSolapamiento.Store(true)
		}

		select {
		case iniciado <- struct{}{}:
		default:
		}

		<-ctx.Done()
		activos.Add(-1)
	})
	defer manager.Cerrar()

	s := service.Servicio{
		ID:         1,
		Intervalo:  time.Hour,
		Habilitado: true,
	}

	if err := manager.Aplicar(s); err != nil {
		t.Fatal(err)
	}

	select {
	case <-iniciado:
	case <-time.After(3 * time.Second):
		t.Fatal("No inicio el primer trabajador")
	}

	const llamadas = 12
	var wg sync.WaitGroup
	errores := make(chan error, llamadas)
	salida := make(chan struct{})

	// Lanzar 12 modificaciones concurrentes.
	for i := 0; i < llamadas; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-salida

			if err := manager.Aplicar(s); err != nil {
				errores <- err
			}
		}()
	}

	close(salida)

	terminado := make(chan struct{})
	go func() {
		wg.Wait()
		close(terminado)
	}()

	select {
	case <-terminado:
	case <-time.After(5 * time.Second):
		t.Fatal("Las actualizaciones concurrentes no terminaron")
	}

	close(errores)

	for err := range errores {
		t.Fatalf("Aplicar fallo: %v", err)
	}

	manager.mu.Lock()
	restantes := len(manager.trabajadores)
	manager.mu.Unlock()

	if restantes != 1 {
		t.Fatalf("Se esperaba 1 trabajador, hay %d", restantes)
	}

	manager.Cerrar()

	if huboSolapamiento.Load() {
		t.Fatal("Se ejecutaron dos trabajadores del mismo servicio a la vez")
	}

	if n := activos.Load(); n != 0 {
		t.Fatalf("Quedaron %d trabajadores activos", n)
	}
}

func TestManagerAislamientoTrabajadores(t *testing.T) {
	inicioA := make(chan struct{}, 1)
	inicioB := make(chan struct{}, 1)

	manager := NewManager(
		context.Background(),
		func(ctx context.Context, s service.Servicio) {
			switch s.ID {
			case 1:
				inicioA <- struct{}{}
			case 2:
				inicioB <- struct{}{}
			}

			// Simular una comprobacion bloqueada.
			<-ctx.Done()
		},
	)
	defer manager.Cerrar()

	activar := func(id int64) {
		t.Helper()

		err := manager.Aplicar(service.Servicio{
			ID:         id,
			Intervalo:  time.Hour,
			Habilitado: true,
		})

		if err != nil {
			t.Fatal(err)
		}
	}

	// Iniciar el primer servicio.
	activar(1)

	select {
	case <-inicioA:
	case <-time.After(3 * time.Second):
		t.Fatal("El servicio A no inicio")
	}

	// Iniciar el segundo sin liberar el primero.
	activar(2)

	select {
	case <-inicioB:
		// El servicio B inicio correctamente.
	case <-time.After(3 * time.Second):
		t.Fatal("El servicio B quedo bloqueado por A")
	}
}

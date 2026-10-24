package monitor

import (
	"context"
	"errors"
	"sync"
	"time"

	"sargas-sentinel/internal/service"
)

type trabajador struct {
	cancelar  context.CancelFunc
	terminado chan struct{}
}

type Manager struct {
	ctx          context.Context
	comprobar    func(context.Context, service.Servicio)
	mu           sync.Mutex
	trabajadores map[int64]trabajador
	cerrado      bool
}

func NewManager(
	ctx context.Context,
	comprobar func(context.Context, service.Servicio),
) *Manager {
	return &Manager{
		ctx:          ctx,
		comprobar:    comprobar,
		trabajadores: make(map[int64]trabajador),
	}
}

func (m *Manager) Aplicar(s service.Servicio) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cerrado {
		return errors.New("manager cerrado")
	}

	if err := m.ctx.Err(); err != nil {
		return err
	}

	if actual, existe := m.trabajadores[s.ID]; existe {
		actual.cancelar()
		<-actual.terminado
		delete(m.trabajadores, s.ID)
	}

	if !s.Habilitado {
		return nil
	}

	ctx, cancelar := context.WithCancel(m.ctx)
	terminado := make(chan struct{})

	m.trabajadores[s.ID] = trabajador{
		cancelar:  cancelar,
		terminado: terminado,
	}

	go func() {
		defer close(terminado)

		ticker := time.NewTicker(s.Intervalo)
		defer ticker.Stop()

		m.comprobar(ctx, s)

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				if ctx.Err() != nil {
					return
				}

				m.comprobar(ctx, s)
			}
		}
	}()

	return nil
}

func (m *Manager) Cerrar() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cerrado {
		return
	}

	m.cerrado = true

	for _, t := range m.trabajadores {
		t.cancelar()
	}

	for _, t := range m.trabajadores {
		<-t.terminado
	}

	m.trabajadores = make(map[int64]trabajador)
}

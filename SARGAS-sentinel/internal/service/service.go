package service

import "time"

type Servicio struct {
	ID         int64
	Nombre     string
	URL        string
	Tipo       string
	Intervalo  time.Duration
	Timeout    time.Duration
	Habilitado bool
}

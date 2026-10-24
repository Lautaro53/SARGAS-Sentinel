package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
)

// Incluye la compilación de Svelte en el ejecutable Go.
//
//go:embed web/dist
var archivosWeb embed.FS

var rutaDetalleWeb = regexp.MustCompile(
	`^/servicios/[1-9][0-9]*/?$`,
)

func esRutaWeb(ruta string) bool {
	switch ruta {
	case "/", "/servicios", "/servicios/",
		"/incidentes", "/incidentes/":
		return true
	}

	return rutaDetalleWeb.MatchString(ruta)
}

func nuevoHandlerWeb(apiHandler http.Handler) (http.Handler, error) {
	dist, err := fs.Sub(archivosWeb, "web/dist")
	if err != nil {
		return nil, fmt.Errorf("abrir frontend: %w", err)
	}

	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, fmt.Errorf("leer index.html: %w", err)
	}

	archivos := http.FileServer(http.FS(dist))
	mux := http.NewServeMux()

	// La API conserva sus rutas internas originales.
	// Solo eliminamos /api antes de delegar.
	mux.Handle(
		"/api/",
		http.StripPrefix("/api", apiHandler),
	)

	// No devolver index.html para una ruta API inválida.
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet &&
			r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		rutaArchivo := strings.TrimPrefix(
			path.Clean(r.URL.Path), "/",
		)

		// Archivos compilados: JavaScript, CSS, imágenes, etc.
		if rutaArchivo != "" && rutaArchivo != "." {
			info, err := fs.Stat(dist, rutaArchivo)

			if err == nil && !info.IsDir() {
				archivos.ServeHTTP(w, r)
				return
			}
		}

		// Navegación Svelte: soporta F5 y enlaces directos.
		if !esRutaWeb(r.URL.Path) {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")

		if r.Method == http.MethodGet {
			_, _ = w.Write(index)
		}
	})

	return mux, nil
}

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed all:frontend/dist/*
var frontendFS embed.FS

func main() {
	// 1. Endpoint de la API (Temporal)
	http.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message": "¡Hola Mundo desde Go! Comunicación exitosa."}`)
	})

	// 2. Servir los archivos estáticos del Frontend (Vue)
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Fatal("Error al crear el sub-sistema de archivos: ", err)
	}
	fileServer := http.FileServer(http.FS(distFS))
	http.Handle("/", fileServer)

	// 3. Configuración del Puerto para Render
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Servidor corriendo exitosamente en http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

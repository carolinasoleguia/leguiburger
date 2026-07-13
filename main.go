package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"leguiburger/internal/db"

	"github.com/joho/godotenv"
)

//go:embed all:frontend/dist/*
var frontendFS embed.FS

func main() {
	// Usamos la librería inmediatamente para que VS Code no la borre al guardar
	_ = godotenv.Load()

	db.Connect()
	defer db.Close()

	http.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message": "¡Hola Mundo desde Go! Base de datos conectada con éxito."}`)
	})

	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Fatal("Error al crear el sub-sistema de archivos: ", err)
	}
	fileServer := http.FileServer(http.FS(distFS))
	http.Handle("/", fileServer)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Servidor corriendo exitosamente en http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"leguiburger/internal/db"
	"leguiburger/internal/tenants"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	// Conectamos a Supabase con GORM
	db.Connect()

	// Inicializamos las capas de Tenants (Inyección de Dependencias)
	tenantRepo := tenants.NewRepository()
	tenantService := tenants.NewService(tenantRepo)
	tenantHandler := tenants.NewHandler(tenantService)

	// Registramos las rutas

	//TENANTS
	http.HandleFunc("/api/tenants/", tenantHandler.HandleTenantRoutes)
	http.HandleFunc("/api/tenants", tenantHandler.HandleTenantRoutes)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Servidor corriendo exitosamente en http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

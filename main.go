package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"leguiburger/internal/customers"
	"leguiburger/internal/db"
	"leguiburger/internal/shipping"
	"leguiburger/internal/tenants"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	db.Connect()

	//----------------------------------------------------------------//
	tenantRepo := tenants.NewRepository()
	tenantService := tenants.NewService(tenantRepo)
	tenantHandler := tenants.NewHandler(tenantService)

	http.HandleFunc("/api/tenants/", tenantHandler.HandleTenantRoutes)
	http.HandleFunc("/api/tenants", tenantHandler.HandleTenantRoutes)

	//----------------------------------------------------------------//

	shippingRepo := shipping.NewRepository()
	shippingService := shipping.NewService(shippingRepo, tenantRepo)
	shippingHandler := shipping.NewHandler(shippingService)

	http.HandleFunc("/api/shipping-methods/", shippingHandler.HandleShippingRoutes)
	http.HandleFunc("/api/shipping-methods", shippingHandler.HandleShippingRoutes)

	//----------------------------------------------------------------//

	customerRepo := customers.NewRepository()
	customerService := customers.NewService(customerRepo, tenantRepo)
	customerHandler := customers.NewHandler(customerService)

	http.HandleFunc("/api/customers/", customerHandler.HandleCustomerRoutes)
	http.HandleFunc("/api/customers", customerHandler.HandleCustomerRoutes)

	//----------------------------------------------------------------//

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Servidor corriendo exitosamente en http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

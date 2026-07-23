package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"leguiburger/internal/auth"
	"leguiburger/internal/customers"
	"leguiburger/internal/db"
	"leguiburger/internal/employees"
	"leguiburger/internal/extras"
	"leguiburger/internal/products"
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

	extraRepo := extras.NewRepository()
	extraService := extras.NewService(extraRepo, tenantRepo)
	extraHandler := extras.NewHandler(extraService)

	http.HandleFunc("/api/extras/", extraHandler.HandleExtraRoutes)
	http.HandleFunc("/api/extras", extraHandler.HandleExtraRoutes)

	//----------------------------------------------------------------//

	productRepo := products.NewRepository()
	productService := products.NewService(productRepo, tenantRepo)
	productHandler := products.NewHandler(productService)

	http.HandleFunc("/api/products/", productHandler.HandleProductRoutes)
	http.HandleFunc("/api/products", productHandler.HandleProductRoutes)

	//----------------------------------------------------------------//

	employeeRepo := employees.NewRepository()
	employeeService := employees.NewService(employeeRepo, tenantRepo)
	employeeHandler := employees.NewHandler(employeeService)

	http.HandleFunc("/api/employees/", employeeHandler.HandleEmployeeRoutes)

	//----------------------------------------------------------------//

	authRepo := auth.NewRepository()
	authSvc := auth.NewService(authRepo, tenantRepo)
	authHandler := auth.NewHandler(authSvc)

	http.HandleFunc("/api/auth/", authHandler.HandleAuthRoutes)

	http.HandleFunc("/api/employees", auth.AuthMiddleware(employeeHandler.CreateEmployee))

	//----------------------------------------------------------------//
	// 📂 SERVIR EL FRONTEND ESTÁTICO EN LA RAIZ (/)
	//----------------------------------------------------------------//
	fs := http.FileServer(http.Dir("./frontend/dist"))
	http.Handle("/", fs)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Servidor corriendo exitosamente en http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

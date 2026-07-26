package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"leguiburger/internal/auth"
	"leguiburger/internal/brands"
	"leguiburger/internal/customers"
	"leguiburger/internal/db"
	"leguiburger/internal/employees"
	"leguiburger/internal/extras"
	"leguiburger/internal/products"
	"leguiburger/internal/recipes"
	"leguiburger/internal/shipping"
	"leguiburger/internal/production"
	"leguiburger/internal/supplies"
	"leguiburger/internal/tenants"
	"leguiburger/internal/users"

	"github.com/joho/godotenv"
)

func registerRoute(path string, handler http.HandlerFunc) {
	http.HandleFunc(path, handler)
	http.HandleFunc(path+"/", handler)
}

func main() {
	_ = godotenv.Load()

	db.Connect()

	//----------------------------------------------------------------//

	brandRepo := brands.NewRepository()
	brandService := brands.NewService(brandRepo)
	brandHandler := brands.NewHandler(brandService)

	brandHandlerWithOwner := auth.AuthMiddleware(auth.RequireOwnerMiddleware(brandHandler.HandleBrandRoutes))
	http.HandleFunc("/api/brands", brandHandlerWithOwner)
	http.HandleFunc("/api/brands/", brandHandlerWithOwner)

	//----------------------------------------------------------------//

	tenantRepo := tenants.NewRepository()
	tenantService := tenants.NewService(tenantRepo, brandRepo)
	tenantHandler := tenants.NewHandler(tenantService)

	tenantHandlerWithAuth := auth.AuthMiddleware(tenantHandler.HandleTenantRoutes)
	http.HandleFunc("/api/tenants", tenantHandlerWithAuth)
	http.HandleFunc("/api/tenants/", tenantHandlerWithAuth)

	//----------------------------------------------------------------//

	customerRepo := customers.NewRepository()
	customerService := customers.NewService(customerRepo, tenantRepo)
	customerHandler := customers.NewHandler(customerService)
	customerHandlerWithAuth := auth.AuthMiddleware(customerHandler.HandleCustomerRoutes)

	registerRoute("/api/customers", customerHandlerWithAuth)

	//----------------------------------------------------------------//

	extraRepo := extras.NewRepository()
	extraService := extras.NewService(extraRepo, tenantRepo)
	extraHandler := extras.NewHandler(extraService)
	extraHandlerWithAuth := auth.AuthMiddleware(extraHandler.HandleExtraRoutes)

	registerRoute("/api/extras", extraHandlerWithAuth)

	//----------------------------------------------------------------//

	supplyRepo := supplies.NewRepository()
	supplyService := supplies.NewService(supplyRepo, tenantRepo)
	supplyHandler := supplies.NewHandler(supplyService)
	supplyHandlerWithAuth := auth.AuthMiddleware(supplyHandler.HandleSupplyRoutes)

	registerRoute("/api/supplies", supplyHandlerWithAuth)

	//----------------------------------------------------------------//

	shippingRepo := shipping.NewRepository()
	shippingService := shipping.NewService(shippingRepo, tenantRepo)
	shippingHandler := shipping.NewHandler(shippingService)
	shippingHandlerWithAuth := auth.AuthMiddleware(shippingHandler.HandleShippingRoutes)

	registerRoute("/api/shipping-methods", shippingHandlerWithAuth)

	//----------------------------------------------------------------//

	productRepo := products.NewRepository()
	productService := products.NewService(productRepo, tenantRepo)
	productHandler := products.NewHandler(productService)
	productHandlerWithAuth := auth.AuthMiddleware(productHandler.HandleProductRoutes)

	registerRoute("/api/products", productHandlerWithAuth)

	//----------------------------------------------------------------//

	recipeRepo := recipes.NewRepository()
	recipeService := recipes.NewService(recipeRepo, tenantRepo)
	recipeHandler := recipes.NewHandler(recipeService)
	recipeHandlerWithAuth := auth.AuthMiddleware(recipeHandler.HandleRecipeRoutes)

	registerRoute("/api/recipes", recipeHandlerWithAuth)

	//----------------------------------------------------------------//

	employeeRepo := employees.NewRepository()
	employeeService := employees.NewService(employeeRepo, tenantRepo)
	employeeHandler := employees.NewHandler(employeeService)

	employeeHandlerWithAuth := auth.AuthMiddleware(employeeHandler.HandleEmployeeRoutes)
	registerRoute("/api/employees", employeeHandlerWithAuth)

	//----------------------------------------------------------------//

	userRepo := users.NewRepository()
	userService := users.NewService(userRepo, brandRepo)
	userHandler := users.NewHandler(userService)
	userHandlerWithAuth := auth.AuthMiddleware(userHandler.HandleUserRoutes)
	registerRoute("/api/users", userHandlerWithAuth)

	//----------------------------------------------------------------//

	productionRepo := production.NewRepository()
	productionService := production.NewService(productionRepo, tenantRepo)
	productionHandler := production.NewHandler(productionService)
	productionHandlerWithAuth := auth.AuthMiddleware(productionHandler.HandleProductionRoutes)
	registerRoute("/api/production", productionHandlerWithAuth)

	//----------------------------------------------------------------//

	authRepo := auth.NewRepository()
	authSvc, err := auth.NewService(authRepo, tenantRepo)
	if err != nil {
		log.Fatalf("Error al configurar autenticacion: %v", err)
	}
	authHandler := auth.NewHandler(authSvc)

	registerRoute("/api/auth", authHandler.HandleAuthRoutes)

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

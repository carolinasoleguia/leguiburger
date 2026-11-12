package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"leguiburger/internal/auth"
	"leguiburger/internal/brands"
	"leguiburger/internal/businesscare"
	"leguiburger/internal/customers"
	"leguiburger/internal/db"
	"leguiburger/internal/employees"
	"leguiburger/internal/extras"
	"leguiburger/internal/mercadopago"
	"leguiburger/internal/observability"
	"leguiburger/internal/operations"
	"leguiburger/internal/orderstream"
	"leguiburger/internal/production"
	"leguiburger/internal/products"
	"leguiburger/internal/recipes"
	"leguiburger/internal/shipping"
	"leguiburger/internal/supplies"
	"leguiburger/internal/tenantproducts"
	"leguiburger/internal/tenants"
	"leguiburger/internal/users"
	"leguiburger/internal/workers"

	"github.com/joho/godotenv"
)

func registerRoute(path string, handler http.HandlerFunc) {
	http.HandleFunc(path, handler)
	http.HandleFunc(path+"/", handler)
}

func main() {
	_ = godotenv.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db.Connect()
	if err := operations.VerifyDatabase(db.DB, os.Getenv("DATABASE_URL")); err != nil {
		log.Fatal(err)
	}

	//----------------------------------------------------------------//

	brandRepo := brands.NewRepository()
	brandService := brands.NewService(brandRepo)
	brandHandler := brands.NewHandler(brandService)

	brandHandlerWithOwner := auth.AuthMiddleware(auth.AuthorizeLegacy(auth.RequireOwnerMiddleware(brandHandler.HandleBrandRoutes)))
	http.HandleFunc("/api/brands", brandHandlerWithOwner)
	http.HandleFunc("/api/brands/", brandHandlerWithOwner)

	//----------------------------------------------------------------//

	tenantRepo := tenants.NewRepository()
	tenantService := tenants.NewService(tenantRepo, brandRepo)
	tenantHandler := tenants.NewHandler(tenantService)

	tenantHandlerWithAuth := auth.AuthMiddleware(auth.AuthorizeLegacy(tenantHandler.HandleTenantRoutes))
	http.HandleFunc("/api/tenants", tenantHandlerWithAuth)
	http.HandleFunc("/api/tenants/", tenantHandlerWithAuth)

	//----------------------------------------------------------------//

	customerRepo := customers.NewRepository()
	customerService := customers.NewService(customerRepo, tenantRepo)
	customerHandler := customers.NewHandler(customerService)
	customerHandlerWithAuth := auth.AuthMiddleware(auth.AuthorizeLegacy(customerHandler.HandleCustomerRoutes))

	registerRoute("/api/customers", customerHandlerWithAuth)

	//----------------------------------------------------------------//

	extraRepo := extras.NewRepository()
	extraService := extras.NewService(extraRepo, tenantRepo)
	extraHandler := extras.NewHandler(extraService)
	extraHandlerWithAuth := auth.AuthMiddleware(auth.AuthorizeLegacy(extraHandler.HandleExtraRoutes))

	registerRoute("/api/extras", extraHandlerWithAuth)

	//----------------------------------------------------------------//

	supplyRepo := supplies.NewRepository()
	supplyService := supplies.NewService(supplyRepo, tenantRepo)
	supplyHandler := supplies.NewHandler(supplyService)
	supplyHandlerWithAuth := auth.AuthMiddleware(auth.AuthorizeLegacy(supplyHandler.HandleSupplyRoutes))

	registerRoute("/api/supplies", supplyHandlerWithAuth)

	//----------------------------------------------------------------//

	shippingRepo := shipping.NewRepository()
	shippingService := shipping.NewService(shippingRepo, tenantRepo)
	shippingHandler := shipping.NewHandler(shippingService)
	shippingHandlerWithAuth := auth.AuthMiddleware(auth.AuthorizeLegacy(shippingHandler.HandleShippingRoutes))

	registerRoute("/api/shipping-methods", shippingHandlerWithAuth)

	//----------------------------------------------------------------//

	productRepo := products.NewRepository()
	productService := products.NewService(productRepo, brandRepo)
	productHandler := products.NewHandler(productService)
	productHandlerWithAuth := auth.AuthMiddleware(products.LimitImageUploads(auth.AuthorizeLegacy(productHandler.HandleProductRoutes)))

	registerRoute("/api/products", productHandlerWithAuth)

	//----------------------------------------------------------------//

	tenantProductRepo := tenantproducts.NewRepository()
	tenantProductService := tenantproducts.NewService(tenantProductRepo, tenantRepo)
	tenantProductHandler := tenantproducts.NewHandler(tenantProductService)
	tenantProductHandlerWithAuth := auth.AuthMiddleware(auth.AuthorizeLegacy(tenantProductHandler.HandleTenantProductRoutes))

	registerRoute("/api/tenant-products", tenantProductHandlerWithAuth)

	//----------------------------------------------------------------//

	recipeRepo := recipes.NewRepository()
	recipeService := recipes.NewService(recipeRepo, tenantRepo)
	recipeHandler := recipes.NewHandler(recipeService)
	recipeHandlerWithAuth := auth.AuthMiddleware(auth.AuthorizeLegacy(recipeHandler.HandleRecipeRoutes))

	registerRoute("/api/recipes", recipeHandlerWithAuth)

	//----------------------------------------------------------------//

	employeeRepo := employees.NewRepository()
	employeeService := employees.NewService(employeeRepo, tenantRepo)
	employeeHandler := employees.NewHandler(employeeService)

	employeeHandlerWithAuth := auth.AuthMiddleware(auth.AuthorizeLegacy(employeeHandler.HandleEmployeeRoutes))
	registerRoute("/api/employees", employeeHandlerWithAuth)

	//----------------------------------------------------------------//

	userRepo := users.NewRepository()
	userService := users.NewService(userRepo, brandRepo)
	userHandler := users.NewHandler(userService)
	userHandlerWithAuth := auth.AuthMiddleware(auth.AuthorizeLegacy(userHandler.HandleUserRoutes))
	registerRoute("/api/users", userHandlerWithAuth)

	//----------------------------------------------------------------//

	productionRepo := production.NewRepository()
	productionService := production.NewService(productionRepo, tenantRepo)
	productionHandler := production.NewHandler(productionService)
	productionHandlerWithAuth := auth.AuthMiddleware(auth.AuthorizeLegacy(productionHandler.HandleProductionRoutes))
	registerRoute("/api/production", productionHandlerWithAuth)

	//----------------------------------------------------------------//

	authRepo := auth.NewRepository()
	authSvc, err := auth.NewService(authRepo, tenantRepo)
	if err != nil {
		log.Fatalf("Error al configurar autenticacion: %v", err)
	}
	auth.RegisterSecurityRoutes(http.DefaultServeMux, db.DB, authSvc)

	stream := orderstream.New()
	streamDSN := os.Getenv("SSE_DATABASE_URL")
	if streamDSN == "" {
		streamDSN = os.Getenv("DATABASE_URL")
	}
	go stream.Run(ctx, streamDSN)
	mp := mercadopago.NewHandler(db.DB, mercadopago.LoadConfig(), nil)
	operations.RegisterRoutes(http.DefaultServeMux, db.DB, operations.BuyerOptions{Stream: stream, OnlinePaymentAvailable: mp.BuyerPaymentAvailable})
	businesscare.Register(http.DefaultServeMux, db.DB)
	observability.Report = func(ctx context.Context, code, id string) { _ = businesscare.Record(ctx, db.DB, "api", code, id) }
	mp.Register(http.DefaultServeMux)
	if os.Getenv("WORKER_MODE") != "external" {
		go workers.Run(ctx, db.DB)
	}
	http.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		check, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		pool, err := db.DB.DB()
		if err != nil || pool.PingContext(check) != nil {
			http.Error(w, "unavailable", 503)
			return
		}
		w.WriteHeader(204)
	})

	//----------------------------------------------------------------//
	// 📂 SERVIR EL FRONTEND ESTÁTICO EN LA RAIZ (/)
	//----------------------------------------------------------------//
	http.HandleFunc("/", spaHandler("./frontend/dist"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Servidor corriendo exitosamente en http://localhost:%s\n", port)
	server := &http.Server{Addr: ":" + port, Handler: observability.HTTP(auth.SecurityHeaders(http.DefaultServeMux)), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func spaHandler(distDir string) http.HandlerFunc {
	fileServer := http.FileServer(http.Dir(distDir))
	indexPath := filepath.Join(distDir, "index.html")

	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" {
			http.NotFound(w, r)
			return
		}

		requestPath := filepath.Clean(r.URL.Path)
		if requestPath == "." || requestPath == string(filepath.Separator) {
			http.ServeFile(w, r, indexPath)
			return
		}

		filePath := filepath.Join(distDir, requestPath)
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, indexPath)
	}
}

package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB guarda el pool de conexiones global
var DB *pgxpool.Pool

// Connect inicializa la conexión a Supabase usando variables de entorno
func Connect() {
	// 1. Buscamos la variable de entorno DATABASE_URL (se configura en Render y .env)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("Error: La variable de entorno DATABASE_URL no está configurada")
	}

	// 2. Cargamos la configuración por defecto del pool
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("No se pudo parsear la configuración del DSN: %v", err)
	}

	// 3. Ajustes de rendimiento recomendados para producción
	config.MaxConns = 10                      // Máximo de conexiones simultáneas
	config.MinConns = 2                       // Conexiones mínimas activas en espera
	config.MaxConnIdleTime = 15 * time.Minute // Tiempo de vida de conexiones inactivas

	// 4. Crear el Pool
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("No se pudo crear el pool de conexiones a la DB: %v", err)
	}

	// 5. Verificar que la DB responda (Ping)
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Error al hacer ping a la base de datos de Supabase: %v", err)
	}

	DB = pool
	fmt.Println("⚡ Conexión exitosa a la base de datos de Supabase (PostgreSQL)!")
}

// Close cierra todas las conexiones del pool de forma segura al apagar el servidor
func Close() {
	if DB != nil {
		DB.Close()
		fmt.Println("💤 Pool de conexiones a la base de datos cerrado de forma segura.")
	}
}

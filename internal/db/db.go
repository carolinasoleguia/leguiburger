package db

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB es ahora el puntero global a la conexión de GORM
var DB *gorm.DB

func Connect() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("Error: DATABASE_URL no configurada en las variables de entorno")
	}

	// Abrimos la conexión con GORM apuntando a Supabase
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Info),
		PrepareStmt: false,
	})
	if err != nil {
		log.Fatalf("Error al conectar con GORM a Supabase: %v", err)
	}

	// Configuramos el pool de conexiones básico bajo el capó
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetConnMaxLifetime(15 * time.Minute)
	}

	DB = db
	log.Println("⚡ Conexión exitosa a Supabase usando GORM!")
}

package main

import (
	"log"
	"os"
	"strings"

	"leguiburger/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "postgresql://postgres.ljdebnvzdrlqdxnligvh:laganadora2026@aws-1-sa-east-1.pooler.supabase.com:6543/postgres"

	if envURL := os.Getenv("DATABASE_URL"); envURL != "" {
		dsn = envURL
	}

	// 🔑 Configuramos GORM para NO usar Prepared Statements con el Pooler de Supabase
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // 👈 Desactiva Prepared Statements a nivel driver
	}), &gorm.Config{
		PrepareStmt: false, // 👈 Desactiva el cache de prepared statements en GORM
	})
	if err != nil {
		log.Fatalf("Error al conectar a Supabase: %v", err)
	}

	email := "admin@admin.com"
	rawPassword := "admin123"

	// Verificar si ya existe el owner inicial
	var count int64
	db.Model(&models.Employee{}).Where("email = ?", strings.ToLower(email)).Count(&count)
	if count > 0 {
		log.Println("⚠️ El usuario Owner inicial ya existe en la base de datos.")
		return
	}

	// Hashear contraseña
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Error al hashear contraseña: %v", err)
	}

	// Insertar el primer Owner
	owner := models.Employee{
		TenantID:     nil,
		FirstName:    "Carolina",
		LastName:     "Eguia",
		Email:        strings.ToLower(email),
		PasswordHash: string(hashedBytes),
		Phone:        "2214347305",
		Role:         "owner",
		IsActive:     true,
	}

	if err := db.Create(&owner).Error; err != nil {
		log.Fatalf("Error al crear el owner inicial: %v", err)
	}

	log.Println("✅ ¡Owner inicial creado con éxito en Supabase!")
	log.Printf("Email: %s | Password: %s\n", email, rawPassword)
}

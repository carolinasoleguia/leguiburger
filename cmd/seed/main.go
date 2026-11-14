// seed creates the initial platform administrator from explicit local environment values.
package main

import (
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"leguiburger/internal/models"
	"log"
	"net/mail"
	"os"
	"strings"
)

func main() {
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	email := strings.ToLower(strings.TrimSpace(os.Getenv("SEED_OWNER_EMAIL")))
	password := os.Getenv("SEED_OWNER_PASSWORD")
	if dsn == "" || email == "" || len(password) < 12 {
		log.Fatal("Configurá DATABASE_URL, SEED_OWNER_EMAIL y SEED_OWNER_PASSWORD (mínimo 12 caracteres)")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		log.Fatal("SEED_OWNER_EMAIL no es válido")
	}
	database, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true}), &gorm.Config{PrepareStmt: false, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Fatal("No se pudo conectar a la base configurada")
	}
	var count int64
	if err = database.Model(&models.User{}).Where("lower(email)=?", email).Count(&count).Error; err != nil {
		log.Fatal("No se pudo consultar la cuenta")
	}
	if count > 0 {
		log.Println("La cuenta ya existe; no se modificó su contraseña ni sus permisos")
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("No se pudo preparar la contraseña")
	}
	owner := models.User{FirstName: strings.TrimSpace(os.Getenv("SEED_OWNER_FIRST_NAME")), LastName: strings.TrimSpace(os.Getenv("SEED_OWNER_LAST_NAME")), Email: email, PasswordHash: string(hashed), Role: "owner", IsActive: true}
	if err = database.Create(&owner).Error; err != nil {
		log.Fatal("No se pudo crear el administrador inicial")
	}
	log.Println("Administrador de plataforma creado; la contraseña no se escribe en los logs")
}

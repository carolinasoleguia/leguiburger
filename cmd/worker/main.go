package main

import (
	"context"
	"github.com/joho/godotenv"
	"leguiburger/internal/auth"
	"leguiburger/internal/db"
	"leguiburger/internal/operations"
	"leguiburger/internal/workers"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	_ = godotenv.Load()
	db.Connect()
	if err := operations.VerifyDatabase(db.DB, os.Getenv("DATABASE_URL")); err != nil {
		log.Fatal(err)
	}
	// Mail encryption and persisted authentication actions share the API's key.
	if err := auth.ConfigureJWTSecret(os.Getenv("JWT_SECRET")); err != nil {
		log.Fatal("JWT_SECRET configuration invalid")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	workers.Run(ctx, db.DB)
}

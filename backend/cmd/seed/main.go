package main

import (
	"context"
	"log"
	"time"

	"stylemind/internal/config"
	"stylemind/internal/database"
	"stylemind/internal/seed"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(ctx, db, "migrations"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	if err := seed.Run(ctx, db); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	log.Println("seed completed successfully")
}

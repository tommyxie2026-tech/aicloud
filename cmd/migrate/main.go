package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/tommyxie2026-tech/aicloud/db/migrations"
)

func main() {
	dsn := os.Getenv("AICLOUD_MIGRATION_DATABASE_URL")
	if dsn == "" {
		log.Fatal("AICLOUD_MIGRATION_DATABASE_URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open PostgreSQL: %v", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping PostgreSQL: %v", err)
	}
	if err := migrations.Run(ctx, db); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	log.Print("database migrations completed")
}

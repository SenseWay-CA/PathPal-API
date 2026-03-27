package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func ConnectDB() error {
	var err error
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		return fmt.Errorf("DATABASE_URL environment variable not set")
	}

	DB, err = pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := DB.Ping(context.Background()); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	if err := runMigrations(); err != nil {
		return fmt.Errorf("migrations failed: %w", err)
	}

	fmt.Println("Successfully connected to the database!")
	return nil
}

func runMigrations() error {
	migrations := []string{
		`ALTER TABLE Users ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT ''`,
	}
	for _, m := range migrations {
		if _, err := DB.Exec(context.Background(), m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

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
		`ALTER TABLE Fences ADD COLUMN IF NOT EXISTS starts_at TIMESTAMPTZ NULL`,
		`ALTER TABLE Fences ADD COLUMN IF NOT EXISTS ends_at TIMESTAMPTZ NULL`,
		`ALTER TABLE Fences ADD COLUMN IF NOT EXISTS timed_title TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS Appointments (
			id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES Users(user_id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			location TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			start_at TIMESTAMPTZ NOT NULL,
			end_at TIMESTAMPTZ NULL,
			fence_id INTEGER NULL REFERENCES Fences(id) ON DELETE SET NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_appointments_user_id ON Appointments(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_appointments_start_at ON Appointments(start_at ASC)`,
	}
	for _, m := range migrations {
		if _, err := DB.Exec(context.Background(), m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

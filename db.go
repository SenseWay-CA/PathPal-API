package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is a global variable to hold the database connection pool
var DB *pgxpool.Pool

// ConnectDB initializes the database connection pool
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

	// Ping the database to verify the connection
	if err := DB.Ping(context.Background()); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	fmt.Println("Successfully connected to the database!")
	return nil
}

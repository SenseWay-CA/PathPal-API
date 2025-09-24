package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Connect to the database
	if err := ConnectDB(); err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}
	defer DB.Close() // Close the connection pool when the application exits

	// Create a new Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Simple route for testing
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, SenseWay API is running!")
	})

	// TODO: We will add registration and login routes here in the next step

	// Start the server
	e.Logger.Fatal(e.Start(":1323"))
}

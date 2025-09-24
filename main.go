package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Create a new Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())  // Log HTTP requests
	e.Use(middleware.Recover()) // Recover from panics to prevent crashes

	// Simple route for testing
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, SenseWay API is running!")
	})

	// Start the server
	e.Logger.Fatal(e.Start(":1323"))
}

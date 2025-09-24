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
	defer DB.Close()

	// Create a new Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	// CORS middleware to allow requests from your Vue frontend
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // For development. For production, restrict this to your frontend's domain.
		AllowMethods: []string{http.MethodGet, http.MethodPost},
	}))

	// --- API Routes ---
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, SenseWay API is running!")
	})

	// Add the new user registration route
	e.POST("/register", registerUser)

	// TODO: Add login route next

	// Start the server
	e.Logger.Fatal(e.Start(":1323"))
}

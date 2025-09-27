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

	// Updated CORS middleware to allow necessary headers
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"}, // Changed for easier testing
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
	}))

	// --- API Routes ---
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "PathPal API is running!")
	})

	e.POST("/register", registerUser)
	e.POST("/login", loginUser)
	e.POST("/logout", logoutUser)
	e.GET("/check-auth", checkAuth)

	// --- Example Protected Route ---
	e.GET("/api/user-profile", checkAuth)

	// Start the server
	e.Logger.Fatal(e.Start(":1323"))
}

package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	if err := ConnectDB(); err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}
	defer DB.Close()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost},
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
	e.GET("/api/user-profile", checkAuth) // Reuse checkAuth as a simple middleware

	e.Logger.Fatal(e.Start(":1323"))
}

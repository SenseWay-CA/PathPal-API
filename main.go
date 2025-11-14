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
		AllowOrigins:     []string{"https://senseway.ca", "http://localhost:5173", "*"},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
	}))

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "PathPal API is running!")
	})

	// Auth Routes

	e.POST("/register", registerUser)
	e.POST("/login", loginUser)
	e.GET("/session", getSession)
	e.DELETE("/session", logoutUser)

	// Events Routes
	e.POST("/events", createEvent)

	e.Logger.Fatal(e.Start(":1323"))
}

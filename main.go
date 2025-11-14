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

	// ROUTES

	// Auth Routes
	e.POST("/register", registerUser)
	e.POST("/login", loginUser)
	e.GET("/session", getSession)
	e.DELETE("/session", logoutUser)

	// Events Routes
	e.POST("/events", createEvent)
	e.GET("/events", getEvents)
	e.GET("/EventsByType", getEventsByType)

	// Invite Routes
	e.POST("/invites", createInvite)
	e.GET("/invites/:code", getInvite)
	e.DELETE("/invites", deleteInvite)

	// Fence Routes
	e.GET("/fences", listFences)
	e.POST("/fences", createFence)
	e.PUT("/fences", updateFence)
	e.DELETE("/fences", deleteFence)

	//User Routes
	e.GET("/user", getUser)
	e.PUT("/user", putUser)
	e.DELETE("/user", deleteUser)

	// Stats Routes
	e.GET("/Location", getLocation)
	e.GET("/LocationByTime", getLocationByTime)
	e.GET("/Battery", getBattery)
	e.GET("/HeartRate", getHeartRate)
	e.GET("/HeartRateByTime", getHeartRateByTime)
	e.POST("/Status", postStatus)

	// Guardian Routes
	e.POST("/guardians", createGuardian)
	e.DELETE("/guardians", deleteGuardian)
	e.GET("/caregivers", getCaregivers)
	e.GET("/caneusers", getCaneUsers)

	e.Logger.Fatal(e.Start(":1323"))
}

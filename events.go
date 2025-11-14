package main

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type Event struct {
	EventID     int       `json:"id"`
	UserID      string    `json:"user_id"`
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type EventCreateRequest struct {
	UserID      string `json:"user_id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func createEvent(c echo.Context) error {
	var req EventCreateRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request format"})
	}

	sql := `
		INSERT INTO Events (user_id, type, name, description)
		VALUES ($1, $2, $3, $4) 
		RETURNING id, user_id, type, name, description, created_at
	`

	var newEvent Event
	err := DB.QueryRow(context.Background(), sql, req.UserID, req.Type, req.Name, req.Description).Scan(
		&newEvent.EventID,
		&newEvent.UserID,
		&newEvent.Type,
		&newEvent.Name,
		&newEvent.Description,
		&newEvent.CreatedAt,
	)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create event"})
	}

	return c.JSON(http.StatusCreated, newEvent)
}

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

type GetEventsRequest struct {
	UserID  string `json:"user_id"`
	Quanity int    `json:"quantity"`
}

type GetEventsByTypeRequest struct {
	UserID   string `json:"user_id"`
	Quantity int    `json:"quantity"`
	Type     string `json:"type"`
}

type EventResponse struct {
	EventID     int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
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

func getEvents(c echo.Context) error {
	var req GetEventsRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request format"})
	}

	sql := `
		SELECT id, user_id, type, name, description, created_at
		FROM Events
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := DB.Query(context.Background(), sql, req.UserID, req.Quanity)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to retrieve events"})
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.EventID, &event.UserID, &event.Type, &event.Name, &event.Description, &event.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to scan event"})
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error iterating over events"})
	}

	return c.JSON(http.StatusOK, events)
}

func getEventsByType(c echo.Context) error {
	var req GetEventsByTypeRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request format"})
	}

	sql := `
		SELECT id, name, description, created_at
		FROM Events
		WHERE user_id = $1 AND type = $2
		ORDER BY created_at DESC
		LIMIT $3
	`
	rows, err := DB.Query(context.Background(), sql, req.UserID, req.Type, req.Quantity)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to retrieve events"})
	}
	defer rows.Close()

	var events []EventResponse
	for rows.Next() {
		var event EventResponse
		if err := rows.Scan(&event.EventID, &event.Name, &event.Description, &event.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to scan event"})
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error iterating over events"})
	}

	return c.JSON(http.StatusOK, events)
}

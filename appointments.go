package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

type Appointment struct {
	ID          int        `json:"id"`
	UserID      string     `json:"user_id"`
	Title       string     `json:"title"`
	Location    string     `json:"location"`
	Description string     `json:"description"`
	StartAt     time.Time  `json:"start_at"`
	EndAt       *time.Time `json:"end_at"`
	FenceID     *int       `json:"fence_id"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateAppointmentRequest struct {
	UserID      string     `json:"user_id"`
	Title       string     `json:"title"`
	Location    string     `json:"location"`
	Description string     `json:"description"`
	StartAt     time.Time  `json:"start_at"`
	EndAt       *time.Time `json:"end_at"`
	FenceID     *int       `json:"fence_id"`
}

type UpdateAppointmentRequest struct {
	ID          int        `json:"id"`
	UserID      string     `json:"user_id"`
	Title       *string    `json:"title"`
	Location    *string    `json:"location"`
	Description *string    `json:"description"`
	StartAt     *time.Time `json:"start_at"`
	EndAt       *time.Time `json:"end_at"`
	FenceID     *int       `json:"fence_id"`
}

func createAppointment(c echo.Context) error {
	var req CreateAppointmentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request payload"})
	}
	req.UserID = strings.TrimSpace(req.UserID)
	req.Title = strings.TrimSpace(req.Title)
	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id is required"})
	}
	if req.Title == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "title is required"})
	}
	if req.StartAt.IsZero() {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "start_at is required"})
	}

	ctx := c.Request().Context()
	sql := `
		INSERT INTO Appointments (user_id, title, location, description, start_at, end_at, fence_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, title, location, description, start_at, end_at, fence_id, created_at
	`
	var appt Appointment
	err := DB.QueryRow(ctx, sql,
		req.UserID, req.Title, req.Location, req.Description,
		req.StartAt, req.EndAt, req.FenceID,
	).Scan(
		&appt.ID, &appt.UserID, &appt.Title, &appt.Location, &appt.Description,
		&appt.StartAt, &appt.EndAt, &appt.FenceID, &appt.CreatedAt,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create appointment"})
	}
	return c.JSON(http.StatusCreated, appt)
}

func listAppointments(c echo.Context) error {
	userID := strings.TrimSpace(c.QueryParam("user_id"))
	if userID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id is required"})
	}
	ctx := c.Request().Context()
	sql := `
		SELECT id, user_id, title, location, description, start_at, end_at, fence_id, created_at
		FROM Appointments
		WHERE user_id = $1
		ORDER BY start_at ASC
	`
	rows, err := DB.Query(ctx, sql, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to fetch appointments"})
	}
	defer rows.Close()

	appts := []Appointment{}
	for rows.Next() {
		var a Appointment
		if err := rows.Scan(&a.ID, &a.UserID, &a.Title, &a.Location, &a.Description,
			&a.StartAt, &a.EndAt, &a.FenceID, &a.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to parse appointment"})
		}
		appts = append(appts, a)
	}
	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to read appointments"})
	}
	return c.JSON(http.StatusOK, appts)
}

func updateAppointment(c echo.Context) error {
	var req UpdateAppointmentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request payload"})
	}
	req.UserID = strings.TrimSpace(req.UserID)
	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id is required"})
	}
	if req.ID <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "valid id is required"})
	}

	ctx := c.Request().Context()
	var current Appointment
	err := DB.QueryRow(ctx,
		`SELECT id, user_id, title, location, description, start_at, end_at, fence_id, created_at FROM Appointments WHERE user_id = $1 AND id = $2`,
		req.UserID, req.ID,
	).Scan(&current.ID, &current.UserID, &current.Title, &current.Location, &current.Description,
		&current.StartAt, &current.EndAt, &current.FenceID, &current.CreatedAt)
	if err == pgx.ErrNoRows {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "appointment not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to fetch appointment"})
	}

	if req.Title != nil {
		current.Title = strings.TrimSpace(*req.Title)
	}
	if req.Location != nil {
		current.Location = *req.Location
	}
	if req.Description != nil {
		current.Description = *req.Description
	}
	if req.StartAt != nil {
		current.StartAt = *req.StartAt
	}
	current.EndAt = req.EndAt
	current.FenceID = req.FenceID

	err = DB.QueryRow(ctx,
		`UPDATE Appointments SET title=$1, location=$2, description=$3, start_at=$4, end_at=$5, fence_id=$6
		 WHERE user_id=$7 AND id=$8
		 RETURNING id, user_id, title, location, description, start_at, end_at, fence_id, created_at`,
		current.Title, current.Location, current.Description, current.StartAt, current.EndAt, current.FenceID,
		req.UserID, req.ID,
	).Scan(&current.ID, &current.UserID, &current.Title, &current.Location, &current.Description,
		&current.StartAt, &current.EndAt, &current.FenceID, &current.CreatedAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update appointment"})
	}
	return c.JSON(http.StatusOK, current)
}

func deleteAppointment(c echo.Context) error {
	var req struct {
		UserID string `json:"user_id"`
		ID     int    `json:"id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request payload"})
	}
	req.UserID = strings.TrimSpace(req.UserID)
	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id is required"})
	}
	if req.ID <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "valid id is required"})
	}

	ctx := c.Request().Context()
	tag, err := DB.Exec(ctx, `DELETE FROM Appointments WHERE user_id = $1 AND id = $2`, req.UserID, req.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to delete appointment"})
	}
	if tag.RowsAffected() == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "appointment not found"})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "appointment deleted"})
}

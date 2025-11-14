package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

type Fence struct {
	FenceID   int       `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	Longitude float64   `json:"longitude"`
	Latitude  float64   `json:"latitude"`
	Radius    float32   `json:"radius"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateFenceRequest struct {
	UserID    string  `json:"user_id"`
	Name      string  `json:"name"`
	Enabled   *bool   `json:"enabled"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Radius    float32 `json:"radius"`
}

type UpdateFenceRequest struct {
	ID        *int     `json:"id"`
	UserID    string   `json:"user_id"`
	Name      *string  `json:"name"`
	Enabled   *bool    `json:"enabled"`
	Longitude *float64 `json:"longitude"`
	Latitude  *float64 `json:"latitude"`
	Radius    *float32 `json:"radius"`
}

type ListFencesRequest struct {
	UserID string `json:"user_id"`
	ID     *int   `json:"id"`
}

func createFence(c echo.Context) error {
	var req CreateFenceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request payload"})
	}

	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	ctx := c.Request().Context()
	sql := `
		INSERT INTO Fences (user_id, name, enabled, longitude, latitude, radius)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, name, enabled, longitude, latitude, radius, created_at
	`

	var fence Fence
	err := DB.QueryRow(ctx, sql, req.UserID, req.Name, enabled, req.Longitude, req.Latitude, req.Radius).Scan(
		&fence.FenceID,
		&fence.UserID,
		&fence.Name,
		&fence.Enabled,
		&fence.Longitude,
		&fence.Latitude,
		&fence.Radius,
		&fence.CreatedAt,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create fence"})
	}

	return c.JSON(http.StatusCreated, fence)
}

func listFences(c echo.Context) error {
	userID := strings.TrimSpace(c.QueryParam("user_id"))
	var body ListFencesRequest
	if userID == "" {
		if err := c.Bind(&body); err == nil {
			userID = strings.TrimSpace(body.UserID)
		}
	}
	if userID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user id is required (use ?user_id=... or JSON body)"})
	}

	// Optional id filter from query or body
	var idFilter *int
	if idStr := strings.TrimSpace(c.QueryParam("id")); idStr != "" {
		parsed, err := strconv.Atoi(idStr)
		if err != nil || parsed <= 0 {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
		}
		idFilter = &parsed
	} else if body.ID == nil {
		if err := c.Bind(&body); err == nil && body.ID != nil {
			if *body.ID <= 0 {
				return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
			}
			idFilter = body.ID
		}
	} else {
		if *body.ID <= 0 {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
		}
		idFilter = body.ID
	}

	ctx := c.Request().Context()
	baseSQL := `SELECT id, user_id, name, enabled, longitude, latitude, radius, created_at FROM Fences WHERE user_id = $1`
	var rows pgx.Rows
	var err error
	if idFilter != nil {
		rows, err = DB.Query(ctx, baseSQL+" AND id = $2 ORDER BY created_at DESC", userID, *idFilter)
	} else {
		rows, err = DB.Query(ctx, baseSQL+" ORDER BY created_at DESC", userID)
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to fetch fences"})
	}
	defer rows.Close()

	var fences []Fence
	for rows.Next() {
		var f Fence
		if err := rows.Scan(&f.FenceID, &f.UserID, &f.Name, &f.Enabled, &f.Longitude, &f.Latitude, &f.Radius, &f.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to parse fence data"})
		}
		fences = append(fences, f)
	}

	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to read fences"})
	}

	return c.JSON(http.StatusOK, fences)
}

func updateFence(c echo.Context) error {

	var req UpdateFenceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request payload"})
	}

	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user id is required"})
	}
	if req.ID == nil || *req.ID <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "valid fence id is required"})
	}
	fenceID := *req.ID

	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	ctx := c.Request().Context()
	selectSQL := `SELECT id, user_id, name, enabled, longitude, latitude, radius, created_at FROM Fences WHERE user_id = $1 AND id = $2`

	var current Fence
	var err error
	err = DB.QueryRow(ctx, selectSQL, userID, fenceID).Scan(
		&current.FenceID,
		&current.UserID,
		&current.Name,
		&current.Enabled,
		&current.Longitude,
		&current.Latitude,
		&current.Radius,
		&current.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "fence not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to fetch fence"})
	}

	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}
	if req.Longitude != nil {
		current.Longitude = *req.Longitude
	}
	if req.Latitude != nil {
		current.Latitude = *req.Latitude
	}
	if req.Radius != nil {
		current.Radius = *req.Radius
	}

	updateSQL := `
		UPDATE Fences
		SET name = $1, enabled = $2, longitude = $3, latitude = $4, radius = $5
		WHERE user_id = $6 AND id = $7
		RETURNING id, user_id, name, enabled, longitude, latitude, radius, created_at
	`

	err = DB.QueryRow(ctx, updateSQL,
		current.Name,
		current.Enabled,
		current.Longitude,
		current.Latitude,
		current.Radius,
		userID,
		fenceID,
	).Scan(
		&current.FenceID,
		&current.UserID,
		&current.Name,
		&current.Enabled,
		&current.Longitude,
		&current.Latitude,
		&current.Radius,
		&current.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "fence not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update fence"})
	}

	return c.JSON(http.StatusOK, current)
}

func deleteFence(c echo.Context) error {
	var req struct {
		UserID string `json:"user_id"`
		ID     int    `json:"id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request payload"})
	}
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user id is required"})
	}
	if req.ID <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "valid fence id is required"})
	}
	fenceID := req.ID

	ctx := c.Request().Context()
	sql := `DELETE FROM Fences WHERE user_id = $1 AND id = $2`

	cmdTag, err := DB.Exec(ctx, sql, userID, fenceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to delete fence"})
	}

	if cmdTag.RowsAffected() == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "fence not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "fence deleted"})
}

func parseFenceID(value string) (int, error) {
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid fence id")
	}
	return id, nil
}

func (r *CreateFenceRequest) Validate() error {
	r.UserID = strings.TrimSpace(r.UserID)
	if r.UserID == "" {
		return errors.New("user_id is required")
	}
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("name is required")
	}
	if !isValidLongitude(r.Longitude) {
		return errors.New("longitude must be between -180 and 180")
	}
	if !isValidLatitude(r.Latitude) {
		return errors.New("latitude must be between -90 and 90")
	}
	if r.Radius <= 0 {
		return errors.New("radius must be greater than zero")
	}
	return nil
}

func (r *UpdateFenceRequest) Validate() error {
	r.UserID = strings.TrimSpace(r.UserID)
	if r.UserID == "" {
		return errors.New("user_id is required")
	}
	if r.Name == nil && r.Enabled == nil && r.Longitude == nil && r.Latitude == nil && r.Radius == nil {
		return errors.New("at least one field must be provided")
	}

	if r.Name != nil {
		trimmed := strings.TrimSpace(*r.Name)
		if trimmed == "" {
			return errors.New("name cannot be empty")
		}
		*r.Name = trimmed
	}
	if r.Longitude != nil && !isValidLongitude(*r.Longitude) {
		return errors.New("longitude must be between -180 and 180")
	}
	if r.Latitude != nil && !isValidLatitude(*r.Latitude) {
		return errors.New("latitude must be between -90 and 90")
	}
	if r.Radius != nil && *r.Radius <= 0 {
		return errors.New("radius must be greater than zero")
	}

	return nil
}

func isValidLongitude(value float64) bool {
	return value >= -180 && value <= 180
}

func isValidLatitude(value float64) bool {
	return value >= -90 && value <= 90
}

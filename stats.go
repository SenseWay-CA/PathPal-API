package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

type LocationResponse struct {
	ID        int       `json:"id"`
	Longitude float64   `json:"longitude"`
	Latitude  float64   `json:"latitude"`
	CreatedAt time.Time `json:"created_at"`
}

type GetLocationRequest struct {
	UserID   string `json:"user_id"`
	Quantity int    `json:"quantity"`
}

// GET /Location - Get recent locations for a user
func getLocation(c echo.Context) error {
	var req GetLocationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user_id is required",
		})
	}

	if req.Quantity <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "quantity must be a positive integer",
		})
	}

	query := `
		SELECT id, longitude, latitude, created_at
		FROM stats
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := DB.Query(context.Background(), query, req.UserID, req.Quantity)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch locations",
		})
	}
	defer rows.Close()

	var locations []LocationResponse
	for rows.Next() {
		var loc LocationResponse
		if err := rows.Scan(&loc.ID, &loc.Longitude, &loc.Latitude, &loc.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to scan location data",
			})
		}
		locations = append(locations, loc)
	}

	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "error reading rows",
		})
	}

	return c.JSON(http.StatusOK, locations)
}

type GetByTimeRequest struct {
	UserID    string `json:"user_id"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// GET /LocationByTime - Get locations for a user within a time range
func getLocationByTime(c echo.Context) error {
	var req GetByTimeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if req.UserID == "" || req.StartTime == "" || req.EndTime == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user_id, start_time, and end_time are required",
		})
	}

	// Parse time strings
	start, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid start_time format, use RFC3339 (e.g., 2025-11-14T00:00:00Z)",
		})
	}

	end, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid end_time format, use RFC3339 (e.g., 2025-11-14T23:59:59Z)",
		})
	}

	if end.Before(start) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "end_time must be after start_time",
		})
	}

	query := `
		SELECT id, longitude, latitude, created_at
		FROM stats
		WHERE user_id = $1
		  AND created_at >= $2
		  AND created_at <= $3
		ORDER BY created_at DESC
	`

	rows, err := DB.Query(context.Background(), query, req.UserID, start, end)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch locations",
		})
	}
	defer rows.Close()

	var locations []LocationResponse
	for rows.Next() {
		var loc LocationResponse
		if err := rows.Scan(&loc.ID, &loc.Longitude, &loc.Latitude, &loc.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to scan location data",
			})
		}
		locations = append(locations, loc)
	}

	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "error reading rows",
		})
	}

	return c.JSON(http.StatusOK, locations)
}

type BatteryResponse struct {
	ID        int       `json:"id"`
	Battery   int       `json:"battery"`
	CreatedAt time.Time `json:"created_at"`
}

type GetBatteryRequest struct {
	UserID string `json:"user_id"`
}

// GET /Battery - Get the most recent battery record for a user
func getBattery(c echo.Context) error {
	var req GetBatteryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user_id is required",
		})
	}

	query := `
		SELECT id, battery, created_at
		FROM stats
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var batteryRecord BatteryResponse
	err := DB.QueryRow(context.Background(), query, req.UserID).Scan(
		&batteryRecord.ID,
		&batteryRecord.Battery,
		&batteryRecord.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "no battery data found for this user",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch battery data",
		})
	}

	return c.JSON(http.StatusOK, batteryRecord)
}

type HeartRateResponse struct {
	ID        int       `json:"id"`
	HeartRate *int      `json:"heart_rate"`
	CreatedAt time.Time `json:"created_at"`
}

type GetHeartRateRequest struct {
	UserID   string `json:"user_id"`
	Quantity int    `json:"quantity"`
}

// GET /HeartRate - Get recent heart rate records for a user
func getHeartRate(c echo.Context) error {
	var req GetHeartRateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user_id is required",
		})
	}

	if req.Quantity <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "quantity must be a positive integer",
		})
	}

	query := `
		SELECT id, heart_rate, created_at
		FROM stats
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := DB.Query(context.Background(), query, req.UserID, req.Quantity)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch heart rate data",
		})
	}
	defer rows.Close()

	var heartRates []HeartRateResponse
	for rows.Next() {
		var hr HeartRateResponse
		if err := rows.Scan(&hr.ID, &hr.HeartRate, &hr.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to scan heart rate data",
			})
		}
		heartRates = append(heartRates, hr)
	}

	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "error reading rows",
		})
	}

	return c.JSON(http.StatusOK, heartRates)
}

// GET /HeartRateByTime - Get heart rate records for a user within a time range
func getHeartRateByTime(c echo.Context) error {
	var req GetByTimeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.UserID == "" || req.StartTime == "" || req.EndTime == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user_id, start_time, and end_time are required",
		})
	}

	// Parse time strings
	start, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid start_time format, use RFC3339 (e.g., 2025-11-14T00:00:00Z)",
		})
	}

	end, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid end_time format, use RFC3339 (e.g., 2025-11-14T23:59:59Z)",
		})
	}

	if end.Before(start) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "end_time must be after start_time",
		})
	}

	query := `
		SELECT id, heart_rate, created_at
		FROM stats
		WHERE user_id = $1
		  AND created_at >= $2
		  AND created_at <= $3
		ORDER BY created_at DESC
	`

	rows, err := DB.Query(context.Background(), query, req.UserID, start, end)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch heart rate data",
		})
	}
	defer rows.Close()

	var heartRates []HeartRateResponse
	for rows.Next() {
		var hr HeartRateResponse
		if err := rows.Scan(&hr.ID, &hr.HeartRate, &hr.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to scan heart rate data",
			})
		}
		heartRates = append(heartRates, hr)
	}

	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "error reading rows",
		})
	}

	return c.JSON(http.StatusOK, heartRates)
}

type StatusRequest struct {
	UserID    string  `json:"user_id"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Battery   int     `json:"battery"`
	HeartRate *int    `json:"heart_rate,omitempty"`
}

// POST /Status - Create a new stats record
// Body: user_id, longitude, latitude, battery, heart_rate (optional)
func postStatus(c echo.Context) error {
	var req StatusRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user_id is required",
		})
	}

	// Optional basic validation
	if req.Longitude < -180 || req.Longitude > 180 ||
		req.Latitude < -90 || req.Latitude > 90 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid latitude/longitude range",
		})
	}

	if req.Battery < 0 || req.Battery > 100 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "battery must be between 0 and 100",
		})
	}

	query := `
		INSERT INTO stats (user_id, longitude, latitude, battery, heart_rate)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := DB.Exec(
		context.Background(),
		query,
		req.UserID,
		req.Longitude,
		req.Latitude,
		req.Battery,
		req.HeartRate,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to create status record",
		})
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message": "status created successfully",
	})
}

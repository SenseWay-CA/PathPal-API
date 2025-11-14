package main

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type GuardianRequest struct {
	CaneUserID      string `json:"cane_user_id"`
	CaregiverUserID string `json:"caregiver_user_id"`
}

type DeleteGuardianRequest struct {
	ID int `json:"id"`
}

type GetCaregiversRequest struct {
	CaneUserID string `json:"cane_user_id"`
}

type GetCaneUsersRequest struct {
	CaregiverUserID string `json:"caregiver_user_id"`
}

type GuardianResponse struct {
	ID              int       `json:"id"`
	CaneUserID      string    `json:"cane_user_id"`
	CaregiverUserID string    `json:"caregiver_user_id"`
	CreatedAt       time.Time `json:"created_at"`
}

func createGuardian(c echo.Context) error {
	var req GuardianRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.CaneUserID == "" || req.CaregiverUserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "cane_user_id and caregiver_user_id are required",
		})
	}

	query := `
		INSERT INTO guardians (cane_user_id, caregiver_user_id)
		VALUES ($1, $2)
		RETURNING id, cane_user_id, caregiver_user_id, created_at
	`

	var guardian GuardianResponse
	err := DB.QueryRow(context.Background(), query, req.CaneUserID, req.CaregiverUserID).Scan(
		&guardian.ID,
		&guardian.CaneUserID,
		&guardian.CaregiverUserID,
		&guardian.CreatedAt,
	)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create guardian relationship",
		})
	}

	return c.JSON(http.StatusOK, guardian)
}

func deleteGuardian(c echo.Context) error {
	var req DeleteGuardianRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.ID <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid guardian ID provided",
		})
	}

	query := `DELETE FROM guardians WHERE id = $1`
	cmdTag, err := DB.Exec(context.Background(), query, req.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete guardian relationship",
		})
	}

	if cmdTag.RowsAffected() == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Guardian relationship not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Guardian relationship deleted successfully"})
}

// GET /caregivers - Get all caregivers for a cane user
func getCaregivers(c echo.Context) error {
	var req GetCaregiversRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.CaneUserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cane_user_id is required"})
	}

	query := `SELECT caregiver_user_id FROM guardians WHERE cane_user_id = $1`
	rows, err := DB.Query(context.Background(), query, req.CaneUserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve caregivers"})
	}
	defer rows.Close()

	var caregiverIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to scan caregiver ID"})
		}
		caregiverIDs = append(caregiverIDs, id)
	}

	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error iterating over caregivers"})
	}

	return c.JSON(http.StatusOK, caregiverIDs)
}

// GET /caneusers - Get all cane users for a caregiver
func getCaneUsers(c echo.Context) error {
	var req GetCaneUsersRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.CaregiverUserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "caregiver_user_id is required"})
	}

	query := `SELECT cane_user_id FROM guardians WHERE caregiver_user_id = $1`
	rows, err := DB.Query(context.Background(), query, req.CaregiverUserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve cane users"})
	}
	defer rows.Close()

	var caneUserIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to scan cane user ID"})
		}
		caneUserIDs = append(caneUserIDs, id)
	}

	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error iterating over cane users"})
	}

	return c.JSON(http.StatusOK, caneUserIDs)
}

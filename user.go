package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type UserGET struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	BirthDate time.Time `json:"birth_date"`
	HomeLong  float64   `json:"home_long"`
	HomeLat   float64   `json:"home_lat"`
	CreatedAt time.Time `json:"created_at"`
	Password  string    `json:"password"`
}

func getUser(c echo.Context) error {
	var req UserGET
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request"})
	}

	query := `
		SELECT user_id, email, name, type, birth_date, home_long, home_lat, created_at
		FROM users
		WHERE user_id = $1`

	var user UserGET
	err := DB.QueryRow(context.Background(), query, req.UserID).
		Scan(
			&user.UserID,
			&user.Email,
			&user.Name,
			&user.Type,
			&user.BirthDate,
			&user.HomeLong,
			&user.HomeLat,
			&user.CreatedAt,
		)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error": "User not found",
			})
		}
		fmt.Println("DB error:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Internal server error",
		})
	}

	return c.JSON(http.StatusOK, user)
}

func deleteUser(c echo.Context) error {
	var req UserGET
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request",
		})
	}

	if strings.TrimSpace(req.UserID) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "user_id is required",
		})
	}

	query := `
		DELETE FROM users
		WHERE user_id = $1
		RETURNING user_id
	`

	var deletedUserID string
	err := DB.QueryRow(context.Background(), query, req.UserID).Scan(&deletedUserID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error": "User not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to delete user",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "User deleted",
		"user_id": deletedUserID,
	})

}

func putUser(c echo.Context) error {
	var req UserGET
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request",
		})
	}

	if strings.TrimSpace(req.UserID) == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "user_id is required",
		})
	}

	var hashedPassword []byte
	var err error

	if req.Password != "" {
		hashedPassword, err = bcrypt.GenerateFromPassword([]byte(req.Password), 12)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "Password hashing failed",
			})
		}
	}

	query := `
		UPDATE users
		SET email=$1, password=$2, name=$3, type=$4, birth_date=$5,
		    home_long=$6, home_lat=$7
		WHERE user_id=$8
		RETURNING user_id, email, name, type, birth_date, home_long, home_lat, created_at
	`

	var user UserGET

	err = DB.QueryRow(context.Background(), query,
		req.Email,
		hashedPassword,
		req.Name,
		req.Type,
		req.BirthDate,
		req.HomeLong,
		req.HomeLat,
		req.UserID,
	).Scan(
		&user.UserID,
		&user.Email,
		&user.Name,
		&user.Type,
		&user.BirthDate,
		&user.HomeLong,
		&user.HomeLat,
		&user.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error": "User not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to update user",
		})
	}

	return c.JSON(http.StatusOK, user)
}

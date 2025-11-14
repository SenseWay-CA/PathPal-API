package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

type Invite struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CreateInviteRequest struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type DeleteInviteRequest struct {
	Code string `json:"code"`
}

func generateInviteCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func createInvite(c echo.Context) error {
	var req CreateInviteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request format"})
	}

	inviteCode, err := generateInviteCode(16)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to generate invite code"})
	}

	expiresAt := time.Now().Add(48 * time.Hour)

	sql := `
		INSERT INTO Invites (code, user_id, email, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, code, user_id, email, created_at, expires_at
	`

	var newInvite Invite
	err = DB.QueryRow(context.Background(), sql, inviteCode, req.UserID, req.Email, expiresAt).Scan(
		&newInvite.ID,
		&newInvite.Code,
		&newInvite.UserID,
		&newInvite.Email,
		&newInvite.CreatedAt,
		&newInvite.ExpiresAt,
	)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create invite"})
	}

	return c.JSON(http.StatusCreated, newInvite)
}

func getInvite(c echo.Context) error {
	code := c.Param("code")

	sql := `
		SELECT id, code, user_id, email, created_at, expires_at
		FROM Invites
		WHERE code = $1
	`

	var invite Invite
	err := DB.QueryRow(context.Background(), sql, code).Scan(
		&invite.ID,
		&invite.Code,
		&invite.UserID,
		&invite.Email,
		&invite.CreatedAt,
		&invite.ExpiresAt,
	)

	if err == pgx.ErrNoRows {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Invite not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to retrieve invite"})
	}

	if time.Now().After(invite.ExpiresAt) {
		return c.JSON(http.StatusGone, echo.Map{"error": "Invite is no longer valid"})
	}

	return c.JSON(http.StatusOK, invite)
}

func deleteInvite(c echo.Context) error {
	var req DeleteInviteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request format"})
	}

	if req.Code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invite code is required"})
	}

	sql := `DELETE FROM Invites WHERE code = $1`

	cmdTag, err := DB.Exec(context.Background(), sql, req.Code)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete invite"})
	}

	if cmdTag.RowsAffected() == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Invite not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Invite deleted successfully"})
}

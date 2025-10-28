package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	BirthDate    time.Time `json:"birth_date"`
	HomeLong     float64   `json:"home_long"`
	HomeLat      float64   `json:"home_lat"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash string    `json:"-"`
}

type RegisterRequest struct {
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	BirthDate time.Time `json:"birth_date"`
	HomeLong  float64   `json:"home_long"`
	HomeLat   float64   `json:"home_lat"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	BirthDate time.Time `json:"birth_date"`
	HomeLong  float64   `json:"home_long"`
	HomeLat   float64   `json:"home_lat"`
	CreatedAt time.Time `json:"created_at"`
}

const sessionCookieName = "pathpal_session"
const sessionDuration = 7 * 24 * time.Hour

func registerUser(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request format"})
	}

	if req.Email == "" || req.Password == "" || req.Name == "" || req.Type == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "All fields are required"})
	}
	if req.Type != "Cane_User" && req.Type != "Caregiver" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid user type"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to hash password"})
	}

	sql := `
		INSERT INTO Users (email, password_hash, name, type, birth_date, home_long, home_lat)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING user_id, email, name, type, birth_date, home_long, home_lat, created_at
	`
	var user UserResponse
	err = DB.QueryRow(context.Background(), sql,
		req.Email, string(hashedPassword), req.Name, req.Type, req.BirthDate, req.HomeLong, req.HomeLat,
	).Scan(
		&user.UserID, &user.Email, &user.Name, &user.Type, &user.BirthDate, &user.HomeLong, &user.HomeLat, &user.CreatedAt,
	)

	if err != nil {

		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return c.JSON(http.StatusConflict, echo.Map{"error": "Email already exists"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": fmt.Sprintf("Failed to create user: %v", err)})
	}

	return c.JSON(http.StatusCreated, user)
}

func loginUser(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request format"})
	}

	sql := `SELECT user_id, email, password_hash, name, type, birth_date, home_long, home_lat, created_at FROM Users WHERE email = $1`
	var user User
	err := DB.QueryRow(context.Background(), sql, req.Email).Scan(
		&user.UserID, &user.Email, &user.PasswordHash, &user.Name, &user.Type, &user.BirthDate, &user.HomeLong, &user.HomeLat, &user.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid credentials"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid credentials"})
	}

	sessionToken, err := generateSecureToken(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to generate session token"})
	}

	tokenHash, err := bcrypt.GenerateFromPassword([]byte(sessionToken), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to hash token"})
	}

	expiresAt := time.Now().Add(sessionDuration)

	sql = `INSERT INTO Sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`
	_, err = DB.Exec(context.Background(), sql, user.UserID, string(tokenHash), expiresAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create session"})
	}

	setSessionCookie(c, sessionToken, expiresAt)

	return c.JSON(http.StatusOK, UserResponse{
		UserID:    user.UserID,
		Email:     user.Email,
		Name:      user.Name,
		Type:      user.Type,
		BirthDate: user.BirthDate,
		HomeLong:  user.HomeLong,
		HomeLat:   user.HomeLat,
		CreatedAt: user.CreatedAt,
	})
}

func getSession(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookieName)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "No session cookie"})
	}
	sessionToken := cookie.Value

	sql := `SELECT user_id, token_hash FROM Sessions WHERE expires_at > $1`
	rows, err := DB.Query(context.Background(), sql, time.Now())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}
	defer rows.Close()

	var userID string
	var tokenHash string
	var found = false

	for rows.Next() {
		if err := rows.Scan(&userID, &tokenHash); err != nil {
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(tokenHash), []byte(sessionToken)); err == nil {
			found = true
			break
		}
	}

	if !found {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Invalid session"})
	}

	sql = `SELECT user_id, email, name, type, birth_date, home_long, home_lat, created_at FROM Users WHERE user_id = $1`
	var user UserResponse
	err = DB.QueryRow(context.Background(), sql, userID).Scan(
		&user.UserID, &user.Email, &user.Name, &user.Type, &user.BirthDate, &user.HomeLong, &user.HomeLat, &user.CreatedAt,
	)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}

	return c.JSON(http.StatusOK, user)
}

func logoutUser(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookieName)
	if err != nil {
		return c.JSON(http.StatusOK, echo.Map{"message": "Not logged in"})
	}
	sessionToken := cookie.Value

	sql := `SELECT id, token_hash FROM Sessions WHERE expires_at > $1`
	rows, err := DB.Query(context.Background(), sql, time.Now())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}
	defer rows.Close()

	var sessionID int
	var tokenHash string
	var found = false

	for rows.Next() {
		if err := rows.Scan(&sessionID, &tokenHash); err != nil {
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(tokenHash), []byte(sessionToken)); err == nil {
			found = true
			break
		}
	}

	if found {
		_, err := DB.Exec(context.Background(), "DELETE FROM Sessions WHERE id = $1", sessionID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete session"})
		}
	}

	clearSessionCookie(c)

	return c.JSON(http.StatusOK, echo.Map{"message": "Logged out successfully"})
}

func setSessionCookie(c echo.Context, token string, expires time.Time) {
	cookie := new(http.Cookie)
	cookie.Name = sessionCookieName
	cookie.Value = token
	cookie.Expires = expires
	cookie.HttpOnly = false
	cookie.Path = "/"
	cookie.Secure = true
	cookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(cookie)
}

func clearSessionCookie(c echo.Context) {
	cookie := new(http.Cookie)
	cookie.Name = sessionCookieName
	cookie.Value = ""
	cookie.Expires = time.Unix(0, 0)
	cookie.HttpOnly = false
	cookie.Path = "/"
	cookie.Secure = true
	cookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(cookie)
}

func generateSecureToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

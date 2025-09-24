package main

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// RegisterRequest defines the structure for a user registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// registerUser is the handler for the POST /register endpoint
func registerUser(c echo.Context) error {
	// 1. Bind the incoming JSON to our RegisterRequest struct
	req := new(RegisterRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// 2. Validate the input
	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email and password are required"})
	}

	// 3. Hash the password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"})
	}

	// 4. Insert the new user into the database
	user := &User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	sql := `INSERT INTO users (email, password_hash, created_at, updated_at) VALUES ($1, $2, $3, $4) RETURNING id`
	err = DB.QueryRow(context.Background(), sql, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt).Scan(&user.ID)
	if err != nil {
		// This could be a unique constraint violation (email already exists) or another DB error
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not create user"})
	}

	// 5. Send a success response
	// We create a new user object to return, ensuring the password hash is not included.
	responseUser := &User{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	return c.JSON(http.StatusCreated, responseUser)
}

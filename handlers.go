package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func registerUser(c echo.Context) error {
	// 1. Bind and validate the request
	req := new(RegisterRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}
	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email and password are required"})
	}

	// 2. Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"})
	}

	// 3. Insert the new user into the database
	user := &User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}
	sql := `INSERT INTO users (email, password_hash, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id, created_at`
	err = DB.QueryRow(context.Background(), sql, user.Email, user.PasswordHash).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Email may already be in use"})
	}

	// --- Create Session and Log In ---

	// 4. Generate a secure, random session token
	sessionToken := uuid.New().String()
	hash := sha256.Sum256([]byte(sessionToken))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	// 5. Store the session in the database
	expiresAt := time.Now().Add(72 * time.Hour)
	insertSQL := `INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`
	_, err = DB.Exec(context.Background(), insertSQL, user.ID, tokenHash, expiresAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not create session after registration"})
	}

	// 6. Set the secure cookie
	cookie := new(http.Cookie)
	cookie.Name = "session_token"
	cookie.Value = sessionToken
	cookie.Expires = expiresAt
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.Secure = true
	cookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(cookie)

	// 7. Return the new user object
	return c.JSON(http.StatusCreated, user)
}

func loginUser(c echo.Context) error {
	req := new(LoginRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	user := new(User)
	sql := `SELECT id, email, password_hash FROM users WHERE email=$1`
	err := DB.QueryRow(context.Background(), sql, req.Email).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	// 1. Generate a secure, random session token
	sessionToken := uuid.New().String()
	// 2. Hash the token for database storage
	hash := sha256.Sum256([]byte(sessionToken))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	// 3. Store the session in the database
	expiresAt := time.Now().Add(72 * time.Hour) // 3-day session
	insertSQL := `INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`
	_, err = DB.Exec(context.Background(), insertSQL, user.ID, tokenHash, expiresAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not create session"})
	}

	// 4. Set the secure cookie
	cookie := new(http.Cookie)
	cookie.Name = "session_token"
	cookie.Value = sessionToken
	cookie.Expires = expiresAt
	cookie.Path = "/"
	cookie.HttpOnly = true                 // Prevent JavaScript access
	cookie.Secure = true                   // Only send over HTTPS
	cookie.SameSite = http.SameSiteLaxMode // CSRF protection

	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, &User{ID: user.ID, Email: user.Email})
}

func logoutUser(c echo.Context) error {
	cookie, err := c.Cookie("session_token")
	if err == nil {
		// Hash the token from the cookie to find it in the database
		hash := sha256.Sum256([]byte(cookie.Value))
		tokenHash := base64.URLEncoding.EncodeToString(hash[:])
		// Delete the session from the database
		DB.Exec(context.Background(), "DELETE FROM sessions WHERE token_hash = $1", tokenHash)
	}

	// Expire the cookie on the client
	expireCookie := new(http.Cookie)
	expireCookie.Name = "session_token"
	expireCookie.Value = ""
	expireCookie.Expires = time.Unix(0, 0)
	expireCookie.Path = "/"
	expireCookie.HttpOnly = true
	expireCookie.Secure = true
	expireCookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(expireCookie)

	return c.JSON(http.StatusOK, map[string]string{"message": "Logged out"})
}

func checkAuth(c echo.Context) error {
	cookie, err := c.Cookie("session_token")
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Not authenticated"})
	}

	// Hash the token from the cookie
	hash := sha256.Sum256([]byte(cookie.Value))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	// Look up the session and user info from the database
	var user User
	sql := `SELECT u.id, u.email FROM users u
           JOIN sessions s ON u.id = s.user_id
           WHERE s.token_hash = $1 AND s.expires_at > NOW()`

	err = DB.QueryRow(context.Background(), sql, tokenHash).Scan(&user.ID, &user.Email)
	if err != nil {
		// If no rows are returned, the session is invalid or expired
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid session"})
	}

	return c.JSON(http.StatusOK, &user)
}

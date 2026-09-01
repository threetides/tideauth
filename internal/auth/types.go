package auth

import (
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

type claims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Sub           string `json:"sub"`
}

type signUp struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type signIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GoogleCFG struct {
	Config          oauth2.Config
	IDTokenVerifier *oidc.IDTokenVerifier
	DB              *pgxpool.Pool
}

type AuthHandler struct {
	DB *pgxpool.Pool
}

type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	EmailVerified bool      `json:"emailVerified"`
	CreatedAt     time.Time `json:"createdAt"`
}

type fieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

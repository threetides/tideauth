package auth

import (
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

type claims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type GoogleCFG struct {
	Config          oauth2.Config
	IDTokenVerifier *oidc.IDTokenVerifier
	DB              *pgxpool.Pool
}

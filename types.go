package tideauth

import "github.com/jackc/pgx/v5/pgxpool"

type Config struct {
	DB *pgxpool.Pool

	// Secure controls the Secure attribute on the HttpOnly cookies tideauth
	// issues. When true, browsers only send the cookies over HTTPS. Set it to
	// true in production and false for local development over plain HTTP.
	Secure bool
}

type Auth struct {
	Config Config
}

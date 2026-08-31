package tideauth

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/threetides/tideauth/internal/auth"
)

type Config struct {
	DatabaseURL  string
	CookieDomain string
}

type Auth struct {
	db  *pgxpool.Pool
	cfg Config
}

func New(cfg Config) (*Auth, error) {
	// Connect to db
	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to db: %e", err)
	}

	defer db.Close()

	err = db.Ping(context.Background())

	if err != nil {
		return nil, fmt.Errorf("Error pinging db: %e", err)
	}

	log.Println("Connected to db")

	return &Auth{db: db, cfg: cfg}, nil
}

func (a *Auth) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /google/sign-in", auth.GoogleSignIn())
	mux.HandleFunc("POST /sign-out", auth.SignOut())
	return mux
}

func (a *Auth) Migrate(ctx context.Context) error {
	log.Println("Migrations completed")
	return nil
}

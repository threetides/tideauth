package tideauth

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/threetides/tideauth/internal/auth"
)

type Google struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type Config struct {
	DB           *pgxpool.Pool
	CookieDomain string
	Google       Google
}

type Auth struct {
	db  *pgxpool.Pool
	cfg Config
}

func New(cfg Config) *Auth {
	return &Auth{db: cfg.DB, cfg: cfg}
}

func (a *Auth) Migrate(ctx context.Context) error {
	db, err := sql.Open("postgres", a.cfg.DB.Config().ConnString())
	if err != nil {
		return fmt.Errorf("error opening db connection: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println("Error closing db connection:", err)
		}
	}()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("error creating postgres driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/migrations",
		"postgres", driver)

	if err != nil {
		return fmt.Errorf("error running migrate.NewWithDatabaseInstance(): %v", err)
	}
	_ = m.Up()

	log.Println("Migrations completed")
	return nil
}

func (a *Auth) Protected(next http.HandlerFunc) http.HandlerFunc {
	authWithDB := auth.AuthHandler{DB: a.db}
	return authWithDB.Protected(next)
}

func (a *Auth) Routes() (http.Handler, error) {
	mux := http.NewServeMux()
	googleCFG, err := auth.GoogleConfig(a.cfg.Google.ClientID, a.cfg.Google.ClientSecret, a.cfg.Google.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("error configuring Google: %v", err)
	}

	googleHandler := &auth.GoogleCFG{Config: googleCFG.Config, IDTokenVerifier: googleCFG.IDTokenVerifier, DB: a.db}
	authHandler := &auth.AuthHandler{DB: a.db}

	mux.HandleFunc("GET /google/sign-in", googleHandler.GoogleSignIn())
	mux.HandleFunc("GET /google/callback", googleHandler.GoogleCallback())
	mux.HandleFunc("GET /session", authHandler.GetSession())
	mux.HandleFunc("POST /sign-out", authHandler.SignOut())
	return mux, nil
}

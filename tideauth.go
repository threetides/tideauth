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
	DatabaseURL  string
	CookieDomain string
	Google       Google
}

type Auth struct {
	db  *pgxpool.Pool
	cfg Config
}

func New(cfg Config) (*Auth, error) {
	// Connect to db
	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("error connecting to db: %e", err)
	}

	defer db.Close()

	err = db.Ping(context.Background())

	if err != nil {
		return nil, fmt.Errorf("error pinging db: %e", err)
	}

	log.Println("Connected to db")

	return &Auth{db: db, cfg: cfg}, nil
}

func (a *Auth) Migrate(ctx context.Context) error {
	db, err := sql.Open("postgres", a.cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("error opening db connection: %e", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("error creating postgres driver: %e", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/migrations",
		"postgres", driver)

	if err != nil {
		return fmt.Errorf("error running migrate.NewWithDatabaseInstance(): %e", err)
	}
	_ = m.Up()

	log.Println("Migrations completed")
	return nil
}

func (a *Auth) Routes() (http.Handler, error) {
	mux := http.NewServeMux()
	googleCFG, err := auth.GoogleConfig(a.cfg.Google.ClientID, a.cfg.Google.ClientSecret, a.cfg.Google.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("error configuring Google: %e", err)
	}

	authHandler := &auth.GoogleCFG{Config: googleCFG.Config, IDTokenVerifier: googleCFG.IDTokenVerifier, DB: a.db}

	mux.HandleFunc("GET /google/sign-in", authHandler.GoogleSignIn())
	mux.HandleFunc("GET /google/callback", authHandler.GoogleCallback())
	mux.HandleFunc("POST /sign-out", auth.SignOut())
	return mux, nil
}

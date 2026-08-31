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

func (a *Auth) Migrate(ctx context.Context) error {
	db, err := sql.Open("postgres", a.cfg.DatabaseURL)
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/migrations",
		"postgres", driver)
	m.Up()

	if err != nil {
		return fmt.Errorf("Error running migrations: %e", err)
	}

	log.Println("Migrations completed")
	return nil
}

func (a *Auth) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /google/sign-in", auth.GoogleSignIn())
	mux.HandleFunc("POST /sign-out", auth.SignOut())
	return mux
}

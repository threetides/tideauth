package tideauth

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
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

//go:embed internal/migrations/*.sql
var migrationFiles embed.FS

func New(cfg Config) *Auth {
	return &Auth{db: cfg.DB, cfg: cfg}
}

func (a *Auth) Migrate(ctx context.Context) error {
	db, err := sql.Open("pgx", a.cfg.DB.Config().ConnString())
	if err != nil {
		return fmt.Errorf("error opening db connection: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println("Error closing db connection:", err)
		}
	}()

	driver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return fmt.Errorf("error creating pgx driver: %v", err)
	}

	// 2. Wrap the embedded folder using iofs
	d, err := iofs.New(migrationFiles, "internal/migrations")
	if err != nil {
		return fmt.Errorf("error getting migration files: %v", err)
	}

	m, err := migrate.NewWithInstance("iofs", d, "pgx", driver)
	if err != nil {
		return fmt.Errorf("error creating m *migrate.Migrate: %v", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("error migrating up: %v", err)
	}

	log.Println("Migrations completed")
	return nil
}

func (a *Auth) Protected(next http.HandlerFunc) http.HandlerFunc {
	authWithDB := auth.AuthHandler{DB: a.db}
	return authWithDB.Protected(next)
}

func (a *Auth) UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(auth.UserIDKey).(string)
	return userID, ok
}

func (a *Auth) Routes() (http.Handler, error) {
	mux := http.NewServeMux()
	googleCFG, err := auth.GoogleConfig(a.cfg.Google.ClientID, a.cfg.Google.ClientSecret, a.cfg.Google.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("error configuring Google: %v", err)
	}

	googleHandler := &auth.GoogleCFG{Config: googleCFG.Config, IDTokenVerifier: googleCFG.IDTokenVerifier, DB: a.db}
	authHandler := &auth.AuthHandler{DB: a.db}

	mux.HandleFunc("POST /sign-up", authHandler.SignUp())
	mux.HandleFunc("POST /sign-in", authHandler.SignIn())
	mux.HandleFunc("GET /google/sign-in", googleHandler.GoogleSignIn())
	mux.HandleFunc("GET /google/callback", googleHandler.GoogleCallback())
	mux.HandleFunc("GET /session", authHandler.GetSession())
	mux.HandleFunc("POST /sign-out", authHandler.SignOut())
	return mux, nil
}

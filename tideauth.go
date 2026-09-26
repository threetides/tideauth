package tideauth

import (
	"embed"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/threetides/tideauth/internal/auth"
)

type Config struct {
	DB *pgxpool.Pool
}

type Auth struct {
	Config Config
}

//go:embed internal/migrations/*.sql
var migrationFS embed.FS

func (a *Auth) Migrate() error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Prefix = time.Now().Format("2006/01/02 15:04:05") + " "
	s.Suffix = " running migrations"
	s.Color("blue")
	s.Start()
	defer s.Stop()

	// * Create a source driver from the embedded filesystem
	sourceDriver, err := iofs.New(migrationFS, "internal/migrations")
	if err != nil {
		return fmt.Errorf("failed to create source driver: %w", err)
	}

	// * Create a new Migrate instance
	m, err := migrate.NewWithSourceInstance(
		"file://internal/migrations",
		sourceDriver,
		a.Config.DB.Config().ConnString(),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migration: %w", err)
	}

	// * Run all up migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}

	s.Stop()
	log.Println("migrations ran successfully")

	return nil
}

func New(cfg Config) Auth {
	return Auth{
		Config: cfg,
	}
}

func (a *Auth) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", auth.RegisterHandler(a.Config.DB))
	return mux
}

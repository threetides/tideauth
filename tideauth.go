package tideauth

import (
	"embed"
	_ "embed"
	"fmt"
	"log"
	"time"

	"github.com/briandowns/spinner"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	DBurl string
}

type Auth struct {
	Config Config
}

//go:embed internal/migrations/*.sql
var migrationFS embed.FS

func (a *Auth) Migrate() error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = "running migrations"
	s.Color("blue")
	s.Start()

	// * Create a source driver from the embedded filesystem
	sourceDriver, err := iofs.New(migrationFS, "internal/migrations")
	if err != nil {
		return fmt.Errorf("failed to create source driver: %w", err)
	}

	// * Create a new Migrate instance
	m, err := migrate.NewWithSourceInstance(
		"file://internal/migrations",
		sourceDriver,
		a.Config.DBurl,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migration: %w", err)
	}

	// * Run all up migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}

	s.Stop()
	log.Println("migrations run successfully")

	return nil
}

func New(cfg Config) Auth {
	return Auth{
		Config: cfg,
	}
}

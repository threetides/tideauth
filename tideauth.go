package tideauth

import (
	"database/sql"
	"embed"
	_ "embed"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"
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
	// Convert *pgxpool.Pool to *sql.DB
	db, err := sql.Open("postgres", a.Config.DBurl)
	if err != nil {
		return fmt.Errorf("error opening sql connection: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println("error closing db connection")
		}
	}()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("error setting dialect: %w", err)
	}

	goose.SetBaseFS(migrationFS)

	// Run up migrations from your embedded files or directory
	err = goose.Up(db, "internal/migrations")
	if err != nil {
		return fmt.Errorf("error running migrations: %w", err)
	}
	return nil
}

func New(cfg Config) Auth {
	return Auth{
		Config: cfg,
	}
}

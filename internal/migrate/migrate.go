package migrator

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	sqlmigrations "desafio-backend/migrations"
)

// Up aplica todas as migrações pendentes. Idempotente quando já está na versão mais recente.
func Up(ctx context.Context, databaseURL string) error {
	_ = ctx
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return err
	}
	return nil
}

// Down desfaz uma migração. Passos=1 desfaz a versão corrente.
func Down(ctx context.Context, databaseURL string, steps int) error {
	_ = ctx
	if steps < 1 {
		return errors.New("steps must be >= 1")
	}
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Steps(-steps); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return err
	}
	return nil
}

// Version devolve a versão actual e se está "dirty" (candidata a reparação).
func Version(ctx context.Context, databaseURL string) (version uint, dirty bool, err error) {
	_ = ctx
	m, err := newMigrate(databaseURL)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()
	return m.Version()
}

func newMigrate(databaseURL string) (*migrate.Migrate, error) {
	dbURL, err := toPgx5URL(databaseURL)
	if err != nil {
		return nil, err
	}
	d, err := iofs.New(sqlmigrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("iofs: %w", err)
	}
	return migrate.NewWithSourceInstance("iofs", d, dbURL)
}

func toPgx5URL(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse database url: %w", err)
	}
	switch u.Scheme {
	case "postgres", "postgresql":
		u.Scheme = "pgx5"
	case "pgx5":
	default:
		return "", fmt.Errorf("unsupported database url scheme %q (use postgres:// or pgx5://)", u.Scheme)
	}
	return u.String(), nil
}

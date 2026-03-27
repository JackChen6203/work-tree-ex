package database

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/config"
)

func RunMigrations(_ context.Context, cfg config.DatabaseConfig) error {
	migrationsPath, err := resolveMigrationsPath()
	if err != nil {
		return err
	}

	sourceURL := "file://" + migrationsPath
	m, err := migrate.New(sourceURL, cfg.DSN())
	if err != nil {
		return err
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func resolveMigrationsPath() (string, error) {
	candidates := []string{
		"backend/migrations",
		"./migrations",
	}
	for _, candidate := range candidates {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		info, err := os.Stat(absPath)
		if err == nil && info.IsDir() {
			return absPath, nil
		}
	}
	return "", os.ErrNotExist
}

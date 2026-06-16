package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pressly/goose/v3"
)

func (s *MySQLStore) runMigrations(ctx context.Context) error {
	migrationDir, err := resolveMigrationDir()
	if err != nil {
		return err
	}
	if err := goose.SetDialect("mysql"); err != nil {
		return err
	}
	return goose.UpContext(ctx, s.db, migrationDir)
}

func resolveMigrationDir() (string, error) {
	candidates := []string{
		filepath.Join("apps", "panel", "api", "migrations"),
		"migrations",
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		base := filepath.Dir(file)
		candidates = append(candidates, filepath.Join(base, "..", "..", "migrations"))
	}
	for _, candidate := range candidates {
		cleaned := filepath.Clean(candidate)
		if stat, err := os.Stat(cleaned); err == nil && stat.IsDir() {
			return cleaned, nil
		}
	}
	return "", fmt.Errorf("migration directory not found")
}

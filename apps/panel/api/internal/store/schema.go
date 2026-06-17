package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func (s *MySQLStore) initSchema(ctx context.Context) error {
	empty, err := s.databaseIsEmpty(ctx)
	if err != nil {
		return err
	}
	if !empty {
		return nil
	}
	schemaFile, err := resolveSchemaFile()
	if err != nil {
		return err
	}
	content, err := os.ReadFile(schemaFile)
	if err != nil {
		return err
	}
	for _, statement := range strings.Split(string(content), ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *MySQLStore) databaseIsEmpty(ctx context.Context) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()").Scan(&count); err != nil {
		return false, err
	}
	return count == 0, nil
}

func resolveSchemaFile() (string, error) {
	candidates := []string{
		filepath.Join("apps", "panel", "api", "schema", "final.sql"),
		filepath.Join("schema", "final.sql"),
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		base := filepath.Dir(file)
		candidates = append(candidates, filepath.Join(base, "..", "..", "schema", "final.sql"))
	}
	for _, candidate := range candidates {
		cleaned := filepath.Clean(candidate)
		if stat, err := os.Stat(cleaned); err == nil && !stat.IsDir() {
			return cleaned, nil
		}
	}
	return "", fmt.Errorf("schema file not found")
}

package store

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func resolveSchemaFiles() ([]string, error) {
	schemaDir, err := resolveSchemaDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, filepath.Join(schemaDir, entry.Name()))
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no schema files found")
	}
	return files, nil
}

func resolveSchemaDir() (string, error) {
	candidates := []string{
		filepath.Join("apps", "one-panel-api", "schema"),
		"schema",
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		base := filepath.Dir(file)
		candidates = append(candidates,
			filepath.Join(base, "..", "..", "schema"),
		)
	}
	for _, candidate := range candidates {
		cleaned := filepath.Clean(candidate)
		if stat, err := os.Stat(cleaned); err == nil && stat.IsDir() {
			return cleaned, nil
		}
	}
	return "", fmt.Errorf("schema directory not found")
}

func resolveSchemaPath() (string, error) {
	candidates := []string{
		filepath.Join("apps", "one-panel-api", "schema", "001_init.sql"),
		filepath.Join("schema", "001_init.sql"),
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		base := filepath.Dir(file)
		candidates = append(candidates,
			filepath.Join(base, "..", "..", "schema", "001_init.sql"),
		)
	}
	for _, candidate := range candidates {
		cleaned := filepath.Clean(candidate)
		if _, err := os.Stat(cleaned); err == nil {
			return cleaned, nil
		}
	}
	return "", fmt.Errorf("schema file not found")
}

func splitSQLStatements(schema string) []string {
	statements := make([]string, 0)
	current := ""
	for _, line := range strings.Split(schema, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		current += line + "\n"
		if strings.HasSuffix(trimmed, ";") {
			statement := strings.TrimSpace(strings.TrimSuffix(current, ";"))
			if statement != "" {
				statements = append(statements, statement)
			}
			current = ""
		}
	}
	if strings.TrimSpace(current) != "" {
		statements = append(statements, strings.TrimSpace(current))
	}
	return statements
}

package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	mysqldriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLStore struct {
	gormDB                 *gorm.DB
	db                     *sql.DB
	bootstrapAdminPassword string
}

func NewMySQLStore(dsn string) (*MySQLStore, error) {
	if err := ensureDatabaseExists(dsn); err != nil {
		return nil, err
	}
	gormDB, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	store := &MySQLStore{
		gormDB: gormDB,
		db:     db,
	}
	if err := store.init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *MySQLStore) BootstrapAdminPassword() string {
	return s.bootstrapAdminPassword
}

func ensureDatabaseExists(dsn string) error {
	config, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return err
	}
	databaseName := config.DBName
	if databaseName == "" {
		return nil
	}
	config.DBName = ""
	rootDB, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return err
	}
	defer rootDB.Close()
	if err := rootDB.Ping(); err != nil {
		return err
	}
	quotedName := "`" + strings.ReplaceAll(databaseName, "`", "``") + "`"
	_, err = rootDB.Exec(
		"CREATE DATABASE IF NOT EXISTS " + quotedName + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
	)
	return err
}

func (s *MySQLStore) init(ctx context.Context) error {
	schemaFiles, err := resolveSchemaFiles()
	if err != nil {
		return err
	}
	for _, schemaPath := range schemaFiles {
		schemaBytes, err := os.ReadFile(schemaPath)
		if err != nil {
			return err
		}
		statements := splitSQLStatements(string(schemaBytes))
		for _, statement := range statements {
			if _, err := s.db.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
	}
	if err := s.gormDB.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		return err
	}
	if err := s.bootstrapAdmin(ctx); err != nil {
		return err
	}
	if err := s.cleanupLegacyDemoTopology(ctx); err != nil {
		return err
	}
	if err := s.repairLegacyUnreportedNodeStatus(ctx); err != nil {
		return err
	}
	if err := s.bootstrapConfig(ctx); err != nil {
		return err
	}
	return nil
}

func (s *MySQLStore) bootstrapAdmin(ctx context.Context) error {
	now := nowRFC3339()
	if err := s.ensureRole(ctx, "role-super-admin", domain.AccountRoleSuperAdmin, now); err != nil {
		return err
	}
	exists, err := s.exists(ctx, "SELECT 1 FROM accounts WHERE account = ?", "admin")
	if err != nil || exists {
		return err
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO accounts
		 (id, account, password_hash, role_id, status, must_rotate_password, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"acct-admin", "admin", hash, "role-super-admin", domain.AccountStatusActive, 0, now, now,
	)
	if err == nil {
		s.bootstrapAdminPassword = password
	}
	return err
}

func (s *MySQLStore) ensureRole(ctx context.Context, id string, name string, now string) error {
	existingID, ok, err := s.roleIDByName(ctx, name)
	if err != nil {
		return err
	}
	if ok {
		if existingID != id {
			return nil
		}
	}
	exists, err := s.exists(ctx, "SELECT 1 FROM roles WHERE id = ?", id)
	if err != nil || exists {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO roles (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)",
		id, name, now, now,
	)
	return err
}

func (s *MySQLStore) roleIDByName(ctx context.Context, name string) (string, bool, error) {
	var id string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM roles WHERE name = ?", name).Scan(&id)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

func (s *MySQLStore) bootstrapConfig(ctx context.Context) error {
	exists, err := s.exists(ctx, "SELECT 1 FROM config WHERE name = ?", "jwt_signing_key")
	if err != nil || exists {
		return err
	}
	key := os.Getenv("JWT_SIGNING_KEY")
	if key == "" || key == "change-me" {
		return nil
	}
	now := nowRFC3339()
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO config (name, value, updated_at) VALUES (?, ?, ?)",
		"jwt_signing_key", key, now,
	)
	return err
}

func (s *MySQLStore) cleanupLegacyDemoTopology(ctx context.Context) error {
	var nodeCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM nodes").Scan(&nodeCount); err != nil {
		return err
	}
	if nodeCount == 0 {
		return nil
	}
	var pathCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM node_access_paths").Scan(&pathCount); err != nil {
		return err
	}
	if pathCount > 0 {
		return nil
	}
	var taskCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM node_onboarding_tasks").Scan(&taskCount); err != nil {
		return err
	}
	if taskCount > 0 {
		return nil
	}
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM nodes ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()
	nodeIDs := make([]string, 0, 4)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		nodeIDs = append(nodeIDs, id)
	}
	if len(nodeIDs) != 4 ||
		nodeIDs[0] != "edge-a" ||
		nodeIDs[1] != "relay-b" ||
		nodeIDs[2] != "relay-c" ||
		nodeIDs[3] != "relay-d" {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return err
	}
	statements := []string{
		"DELETE FROM node_onboarding_tasks",
		"DELETE FROM node_access_paths",
		"DELETE FROM node_policy_assignments",
		"DELETE FROM node_health_snapshots",
		"DELETE FROM node_api_tokens",
		"DELETE FROM node_trust_materials",
		"DELETE FROM bootstrap_tokens",
		"DELETE FROM certificates",
		"DELETE FROM policy_revisions",
		"DELETE FROM route_rules",
		"DELETE FROM chain_hops",
		"DELETE FROM chains",
		"DELETE FROM node_links",
		"DELETE FROM nodes",
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MySQLStore) repairLegacyUnreportedNodeStatus(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes
		 SET status = 'healthy', updated_at = ?
		 WHERE status = 'degraded'
		   AND id NOT IN (SELECT node_id FROM node_health_snapshots)`,
		nowRFC3339(),
	)
	return err
}

func (s *MySQLStore) IsInitialized() bool {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&count); err != nil {
		return false
	}
	return count > 0
}

func (s *MySQLStore) ReinitializeStore(adminPassword string) error {
	s.bootstrapAdminPassword = adminPassword
	if err := os.Setenv("ADMIN_PASSWORD", adminPassword); err != nil {
		return err
	}
	ctx := context.Background()
	if err := s.init(ctx); err != nil {
		return err
	}
	if adminPassword != "" {
		hash, err := auth.HashPassword(adminPassword)
		if err != nil {
			return err
		}
		_, _ = s.db.ExecContext(ctx,
			"UPDATE accounts SET password_hash = ?, must_rotate_password = 0, updated_at = ? WHERE account = ?",
			hash, nowRFC3339(), "admin",
		)
	}
	return nil
}

func (s *MySQLStore) GetOverview() domain.Overview {
	nodes := s.ListNodes()
	health := s.ListNodeHealth()
	healthy := 0
	degraded := 0
	for _, node := range nodes {
		if node.Status == domain.NodeStatusHealthy {
			healthy++
		} else {
			degraded++
		}
	}
	renewSoon := 0
	for _, item := range health {
		for _, state := range item.CertStatus {
			if state == domain.CertStatusRenewSoon || state == "rotate" {
				renewSoon++
				break
			}
		}
	}
	latest := domain.OverviewPolicies{}
	_ = s.db.QueryRow(
		"SELECT version, created_at FROM policy_revisions ORDER BY created_at DESC LIMIT 1",
	).Scan(&latest.ActiveRevision, &latest.PublishedAt)
	return domain.Overview{
		Nodes:        domain.OverviewNodes{Healthy: healthy, Degraded: degraded},
		Policies:     latest,
		Certificates: domain.OverviewCertificates{RenewSoon: renewSoon},
	}
}

func (s *MySQLStore) ListCertificates() []domain.Certificate {
	rows, err := s.db.Query(
		`SELECT id, owner_type, owner_id, cert_type, provider, status, COALESCE(not_before, ''), COALESCE(not_after, '')
		 FROM certificates
		 ORDER BY owner_id, cert_type`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.Certificate, 0)
	for rows.Next() {
		var item domain.Certificate
		if err := rows.Scan(&item.ID, &item.OwnerType, &item.OwnerID, &item.CertType, &item.Provider, &item.Status, &item.NotBefore, &item.NotAfter); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) exists(ctx context.Context, query string, args ...any) (bool, error) {
	var value int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&value)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

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

func (s *MySQLStore) nextNodeID() (string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	now := nowRFC3339()
	_, err = tx.Exec(
		`INSERT INTO id_sequences (name, current_value, updated_at)
		 VALUES ('node_id', 1, ?)
		 ON DUPLICATE KEY UPDATE current_value = current_value + 1, updated_at = ?`,
		now, now,
	)
	if err != nil {
		return "", err
	}

	var nextID int64
	err = tx.QueryRow(`SELECT current_value FROM id_sequences WHERE name = 'node_id'`).Scan(&nextID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", nextID), nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

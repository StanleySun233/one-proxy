package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/policy"
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

func (s *MySQLStore) BootstrapAdminPassword() string {
	return s.bootstrapAdminPassword
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

func (s *MySQLStore) bootstrapAdmin(ctx context.Context) error {
	now := nowRFC3339()
	if err := s.ensureRole(ctx, "role-super-admin", "super_admin", now); err != nil {
		return err
	}
	exists, err := s.exists(ctx, "SELECT 1 FROM accounts WHERE account = ?", "admin")
	if err != nil || exists {
		return err
	}
	password := "admin"
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO accounts
		 (id, account, password_hash, role_id, status, must_rotate_password, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"acct-admin", "admin", hash, "role-super-admin", "active", 1, now, now,
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

func (s *MySQLStore) GetOverview() domain.Overview {
	nodes := s.ListNodes()
	health := s.ListNodeHealth()
	healthy := 0
	degraded := 0
	for _, node := range nodes {
		if node.Status == "healthy" {
			healthy++
		} else {
			degraded++
		}
	}
	renewSoon := 0
	for _, item := range health {
		for _, state := range item.CertStatus {
			if state == "renew-soon" || state == "rotate" {
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

func (s *MySQLStore) ListAccounts() []domain.Account {
	rows, err := s.db.Query(
		`SELECT a.id, a.account, r.name, a.status, a.must_rotate_password
		 FROM accounts a
		 JOIN roles r ON r.id = a.role_id
		 ORDER BY a.account`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	accounts := make([]domain.Account, 0)
	for rows.Next() {
		var item domain.Account
		var mustRotate int
		if err := rows.Scan(&item.ID, &item.Account, &item.Role, &item.Status, &mustRotate); err != nil {
			continue
		}
		item.MustRotatePassword = mustRotate == 1
		accounts = append(accounts, item)
	}
	return accounts
}

func (s *MySQLStore) CreateAccount(input domain.CreateAccountInput) (domain.Account, error) {
	roleID := roleIDForName(input.Role)
	now := nowRFC3339()
	if err := s.ensureRole(context.Background(), roleID, input.Role, now); err != nil {
		return domain.Account{}, err
	}
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return domain.Account{}, err
	}
	item := domain.Account{
		ID:                 newID("acct"),
		Account:            input.Account,
		Role:               input.Role,
		Status:             "active",
		MustRotatePassword: false,
	}
	_, err = s.db.Exec(
		`INSERT INTO accounts (id, account, password_hash, role_id, status, must_rotate_password, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Account, hash, roleID, item.Status, 0, now, now,
	)
	return item, err
}

func (s *MySQLStore) ListNodeLinks() []domain.NodeLink {
	rows, err := s.db.Query(
		`SELECT id, source_node_id, target_node_id, link_type, trust_state FROM node_links ORDER BY source_node_id, target_node_id`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.NodeLink, 0)
	for rows.Next() {
		var item domain.NodeLink
		if err := rows.Scan(&item.ID, &item.SourceNodeID, &item.TargetNodeID, &item.LinkType, &item.TrustState); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateNodeLink(input domain.CreateNodeLinkInput) (domain.NodeLink, error) {
	item := domain.NodeLink{
		ID:           newID("link"),
		SourceNodeID: input.SourceNodeID,
		TargetNodeID: input.TargetNodeID,
		LinkType:     input.LinkType,
		TrustState:   input.TrustState,
	}
	now := nowRFC3339()
	_, err := s.db.Exec(
		`INSERT INTO node_links (id, source_node_id, target_node_id, link_type, trust_state, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.SourceNodeID, item.TargetNodeID, item.LinkType, item.TrustState, now, now,
	)
	return item, err
}

func (s *MySQLStore) ListNodeAccessPaths() []domain.NodeAccessPath {
	rows, err := s.db.Query(
		`SELECT id, name, mode, COALESCE(target_node_id, ''), COALESCE(entry_node_id, ''), relay_node_ids_json, COALESCE(target_host, ''), COALESCE(target_port, 0), enabled
		 FROM node_access_paths
		 ORDER BY name`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.NodeAccessPath, 0)
	for rows.Next() {
		var item domain.NodeAccessPath
		var relayJSON string
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.Mode, &item.TargetNodeID, &item.EntryNodeID, &relayJSON, &item.TargetHost, &item.TargetPort, &enabled); err != nil {
			continue
		}
		item.RelayNodeIDs = decodeJSONStringSlice(relayJSON)
		item.Enabled = enabled == 1
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateNodeAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	item := domain.NodeAccessPath{
		ID:           newID("path"),
		Name:         input.Name,
		Mode:         input.Mode,
		TargetNodeID: input.TargetNodeID,
		EntryNodeID:  input.EntryNodeID,
		RelayNodeIDs: normalizeStringSlice(input.RelayNodeIDs),
		TargetHost:   input.TargetHost,
		TargetPort:   input.TargetPort,
		Enabled:      true,
	}
	now := nowRFC3339()
	_, err := s.db.Exec(
		`INSERT INTO node_access_paths (id, name, mode, target_node_id, entry_node_id, relay_node_ids_json, target_host, target_port, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, ?, ?)`,
		item.ID, item.Name, item.Mode, item.TargetNodeID, item.EntryNodeID, encodeJSONStringSlice(item.RelayNodeIDs), item.TargetHost, item.TargetPort, 1, now, now,
	)
	return item, err
}

func (s *MySQLStore) UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE node_access_paths
		 SET name = ?, mode = ?, target_node_id = NULLIF(?, ''), entry_node_id = NULLIF(?, ''), relay_node_ids_json = ?, target_host = NULLIF(?, ''), target_port = ?, enabled = ?, updated_at = ?
		 WHERE id = ?`,
		input.Name, input.Mode, input.TargetNodeID, input.EntryNodeID, encodeJSONStringSlice(input.RelayNodeIDs), input.TargetHost, input.TargetPort, boolToInt(input.Enabled), now, pathID,
	)
	if err != nil {
		return domain.NodeAccessPath{}, err
	}
	for _, item := range s.ListNodeAccessPaths() {
		if item.ID == pathID {
			return item, nil
		}
	}
	return domain.NodeAccessPath{}, sql.ErrNoRows
}

func (s *MySQLStore) DeleteNodeAccessPath(pathID string) error {
	_, err := s.db.Exec("DELETE FROM node_access_paths WHERE id = ?", pathID)
	return err
}

func (s *MySQLStore) ListNodeOnboardingTasks() []domain.NodeOnboardingTask {
	rows, err := s.db.Query(
		`SELECT id, mode, COALESCE(path_id, ''), COALESCE(target_node_id, ''), COALESCE(target_host, ''), COALESCE(target_port, 0), status, status_message, requested_by_account_id, created_at, updated_at
		 FROM node_onboarding_tasks
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.NodeOnboardingTask, 0)
	for rows.Next() {
		var item domain.NodeOnboardingTask
		if err := rows.Scan(&item.ID, &item.Mode, &item.PathID, &item.TargetNodeID, &item.TargetHost, &item.TargetPort, &item.Status, &item.StatusMessage, &item.RequestedByAccountID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateNodeOnboardingTask(accountID string, input domain.CreateNodeOnboardingTaskInput) (domain.NodeOnboardingTask, error) {
	now := nowRFC3339()
	item := domain.NodeOnboardingTask{
		ID:                   newID("task"),
		Mode:                 input.Mode,
		PathID:               input.PathID,
		TargetNodeID:         input.TargetNodeID,
		TargetHost:           input.TargetHost,
		TargetPort:           input.TargetPort,
		Status:               "planned",
		StatusMessage:        "task created",
		RequestedByAccountID: accountID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	_, err := s.db.Exec(
		`INSERT INTO node_onboarding_tasks (id, mode, path_id, target_node_id, target_host, target_port, status, status_message, requested_by_account_id, created_at, updated_at)
		 VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Mode, item.PathID, item.TargetNodeID, item.TargetHost, item.TargetPort, item.Status, item.StatusMessage, item.RequestedByAccountID, item.CreatedAt, item.UpdatedAt,
	)
	return item, err
}

func (s *MySQLStore) UpdateNodeOnboardingTaskStatus(taskID string, status string, statusMessage string) (domain.NodeOnboardingTask, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE node_onboarding_tasks SET status = ?, status_message = ?, updated_at = ? WHERE id = ?`,
		status, statusMessage, now, taskID,
	)
	if err != nil {
		return domain.NodeOnboardingTask{}, err
	}
	for _, item := range s.ListNodeOnboardingTasks() {
		if item.ID == taskID {
			return item, nil
		}
	}
	return domain.NodeOnboardingTask{}, sql.ErrNoRows
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

func (s *MySQLStore) UpdateAccount(accountID string, input domain.UpdateAccountInput) (domain.Account, error) {
	current, ok := s.getAccountByID(accountID)
	if !ok {
		return domain.Account{}, sql.ErrNoRows
	}
	role := current.Role
	if input.Role != "" {
		role = input.Role
	}
	status := current.Status
	if input.Status != "" {
		status = input.Status
	}
	roleID := roleIDForName(role)
	now := nowRFC3339()
	if err := s.ensureRole(context.Background(), roleID, role, now); err != nil {
		return domain.Account{}, err
	}
	if input.Password != "" {
		hash, err := auth.HashPassword(input.Password)
		if err != nil {
			return domain.Account{}, err
		}
		if _, err := s.db.Exec(
			`UPDATE accounts SET password_hash = ?, role_id = ?, status = ?, must_rotate_password = 0, updated_at = ? WHERE id = ?`,
			hash, roleID, status, now, accountID,
		); err != nil {
			return domain.Account{}, err
		}
	} else {
		if _, err := s.db.Exec(
			`UPDATE accounts SET role_id = ?, status = ?, updated_at = ? WHERE id = ?`,
			roleID, status, now, accountID,
		); err != nil {
			return domain.Account{}, err
		}
	}
	item, _ := s.getAccountByID(accountID)
	return item, nil
}

func (s *MySQLStore) DeleteAccount(accountID string) error {
	account, ok := s.getAccountByID(accountID)
	if !ok {
		return sql.ErrNoRows
	}
	if account.Account == "admin" {
		return fmt.Errorf("cannot_delete_admin")
	}
	_, err := s.db.Exec("DELETE FROM accounts WHERE id = ?", accountID)
	return err
}

func roleIDForName(name string) string {
	replacer := strings.NewReplacer("_", "-", " ", "-")
	return "role-" + replacer.Replace(name)
}

func (s *MySQLStore) getAccountByID(accountID string) (domain.Account, bool) {
	var item domain.Account
	var mustRotate int
	err := s.db.QueryRow(
		`SELECT a.id, a.account, r.name, a.status, a.must_rotate_password
		 FROM accounts a
		 JOIN roles r ON r.id = a.role_id
		 WHERE a.id = ?`,
		accountID,
	).Scan(&item.ID, &item.Account, &item.Role, &item.Status, &mustRotate)
	if err != nil {
		return domain.Account{}, false
	}
	item.MustRotatePassword = mustRotate == 1
	return item, true
}

func (s *MySQLStore) Authenticate(account string, password string) (domain.LoginResult, bool) {
	var (
		id         string
		name       string
		role       string
		status     string
		hash       string
		mustRotate int
	)
	err := s.db.QueryRow(
		`SELECT a.id, a.account, r.name, a.status, a.password_hash, a.must_rotate_password
		 FROM accounts a
		 JOIN roles r ON r.id = a.role_id
		 WHERE a.account = ?`,
		account,
	).Scan(&id, &name, &role, &status, &hash, &mustRotate)
	if err != nil || status != "active" || !auth.CheckPassword(hash, password) {
		return domain.LoginResult{}, false
	}
	return s.createSession(id, name, role, status, mustRotate == 1)
}

func (s *MySQLStore) AuthenticateAccessToken(accessToken string) (domain.Account, bool) {
	var (
		accountID string
		expiresAt string
	)
	err := s.db.QueryRow(
		"SELECT account_id, expires_at FROM sessions WHERE access_token_hash = ?",
		accessToken,
	).Scan(&accountID, &expiresAt)
	if err != nil {
		return domain.Account{}, false
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expiry) {
		return domain.Account{}, false
	}
	item, ok := s.getAccountByID(accountID)
	if !ok || item.Status != "active" {
		return domain.Account{}, false
	}
	return item, true
}

func (s *MySQLStore) RefreshSession(refreshToken string) (domain.LoginResult, bool) {
	var (
		accountID string
		expiresAt string
	)
	err := s.db.QueryRow(
		"SELECT account_id, expires_at FROM sessions WHERE refresh_token_hash = ?",
		refreshToken,
	).Scan(&accountID, &expiresAt)
	if err != nil {
		return domain.LoginResult{}, false
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expiry) {
		return domain.LoginResult{}, false
	}
	item, ok := s.getAccountByID(accountID)
	if !ok || item.Status != "active" {
		return domain.LoginResult{}, false
	}
	return s.createSession(item.ID, item.Account, item.Role, item.Status, item.MustRotatePassword)
}

func (s *MySQLStore) createSession(accountID string, account string, role string, status string, mustRotate bool) (domain.LoginResult, bool) {
	accessToken, err := auth.RandomToken()
	if err != nil {
		return domain.LoginResult{}, false
	}
	refreshToken, err := auth.RandomToken()
	if err != nil {
		return domain.LoginResult{}, false
	}
	now := time.Now().UTC()
	expiresAt := now.Add(12 * time.Hour).Format(time.RFC3339)
	_, _ = s.db.Exec(
		`INSERT INTO sessions (id, account_id, access_token_hash, refresh_token_hash, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		newID("sess"), accountID, accessToken, refreshToken, expiresAt, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	return domain.LoginResult{
		Account:            domain.Account{ID: accountID, Account: account, Role: role, Status: status, MustRotatePassword: mustRotate},
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		ExpiresAt:          expiresAt,
		MustRotatePassword: mustRotate,
	}, true
}

func (s *MySQLStore) Logout(accessToken string) bool {
	result, err := s.db.Exec("DELETE FROM sessions WHERE access_token_hash = ?", accessToken)
	if err != nil {
		return false
	}
	affected, err := result.RowsAffected()
	return err == nil && affected > 0
}

func (s *MySQLStore) ListNodes() []domain.Node {
	rows, err := s.db.Query(
		`SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0)
		 FROM nodes ORDER BY name`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.Node, 0)
	for rows.Next() {
		var item domain.Node
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.Mode, &item.ScopeKey, &item.ParentNodeID, &enabled, &item.Status, &item.PublicHost, &item.PublicPort); err != nil {
			continue
		}
		item.Enabled = enabled == 1
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ListNodeTransports() []domain.NodeTransport {
	rows, err := s.db.Query(
		`SELECT id, node_id, transport_type, direction, address, status, COALESCE(parent_node_id, ''), COALESCE(connected_at, ''), COALESCE(last_heartbeat_at, ''), latency_ms, details_json
		 FROM node_transports ORDER BY node_id, transport_type, address`,
	)
	if err != nil {
		return s.syntheticPublicTransports(nil)
	}
	defer rows.Close()
	items := make([]domain.NodeTransport, 0)
	for rows.Next() {
		var item domain.NodeTransport
		var detailsJSON string
		if err := rows.Scan(&item.ID, &item.NodeID, &item.TransportType, &item.Direction, &item.Address, &item.Status, &item.ParentNodeID, &item.ConnectedAt, &item.LastHeartbeatAt, &item.LatencyMs, &detailsJSON); err != nil {
			continue
		}
		item.Details = decodeJSONMap(detailsJSON)
		items = append(items, item)
	}
	return s.syntheticPublicTransports(items)
}

func (s *MySQLStore) syntheticPublicTransports(items []domain.NodeTransport) []domain.NodeTransport {
	nodes := s.ListNodes()
	healthByNodeID := make(map[string]domain.NodeHealth)
	for _, health := range s.ListNodeHealth() {
		healthByNodeID[health.NodeID] = health
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		seen[item.NodeID+"|"+item.TransportType+"|"+item.Address] = struct{}{}
	}
	for _, node := range nodes {
		if node.PublicHost == "" || node.PublicPort <= 0 {
			continue
		}
		address := fmt.Sprintf("http://%s:%d", node.PublicHost, node.PublicPort)
		key := node.ID + "|public_http|" + address
		if _, ok := seen[key]; ok {
			continue
		}
		status := "available"
		lastHeartbeat := ""
		if health, ok := healthByNodeID[node.ID]; ok {
			lastHeartbeat = health.HeartbeatAt
			if node.Status == "healthy" {
				status = "connected"
			} else {
				status = node.Status
			}
		}
		items = append(items, domain.NodeTransport{
			ID:              "derived-public-" + node.ID,
			NodeID:          node.ID,
			TransportType:   "public_http",
			Direction:       "inbound",
			Address:         address,
			Status:          status,
			ConnectedAt:     lastHeartbeat,
			LastHeartbeatAt: lastHeartbeat,
			LatencyMs:       0,
			Details:         map[string]string{"source": "derived_public_endpoint"},
		})
	}
	return items
}

func (s *MySQLStore) UpsertNodeTransport(input domain.UpsertNodeTransportInput) (domain.NodeTransport, error) {
	id := newID("transport")
	now := nowRFC3339()
	detailsJSON := encodeJSONMap(input.Details)
	existingID := ""
	_ = s.db.QueryRow(
		`SELECT id FROM node_transports WHERE node_id = ? AND transport_type = ? AND address = ? LIMIT 1`,
		input.NodeID, input.TransportType, input.Address,
	).Scan(&existingID)
	if existingID != "" {
		id = existingID
	}
	_, err := s.db.Exec(
		`INSERT INTO node_transports (id, node_id, transport_type, direction, address, status, parent_node_id, connected_at, last_heartbeat_at, latency_ms, details_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   direction = VALUES(direction),
		   status = VALUES(status),
		   parent_node_id = VALUES(parent_node_id),
		   connected_at = VALUES(connected_at),
		   last_heartbeat_at = VALUES(last_heartbeat_at),
		   latency_ms = VALUES(latency_ms),
		   details_json = VALUES(details_json),
		   updated_at = VALUES(updated_at)`,
		id, input.NodeID, input.TransportType, input.Direction, input.Address, input.Status, input.ParentNodeID, input.ConnectedAt, input.LastHeartbeatAt, input.LatencyMs, detailsJSON, now, now,
	)
	if err != nil {
		return domain.NodeTransport{}, err
	}
	return domain.NodeTransport{
		ID:              id,
		NodeID:          input.NodeID,
		TransportType:   input.TransportType,
		Direction:       input.Direction,
		Address:         input.Address,
		Status:          input.Status,
		ParentNodeID:    input.ParentNodeID,
		ConnectedAt:     input.ConnectedAt,
		LastHeartbeatAt: input.LastHeartbeatAt,
		LatencyMs:       input.LatencyMs,
		Details:         input.Details,
	}, nil
}

func (s *MySQLStore) CreateNode(input domain.CreateNodeInput) (domain.Node, error) {
	nodeID, err := s.nextNodeID()
	if err != nil {
		return domain.Node{}, err
	}
	item := domain.Node{
		ID:           nodeID,
		Name:         input.Name,
		Mode:         input.Mode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		Enabled:      true,
		Status:       "healthy",
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
	}
	now := nowRFC3339()
	_, err = s.db.Exec(
		`INSERT INTO nodes (id, name, mode, public_host, public_port, scope_key, parent_node_id, enabled, status, created_at, updated_at)
		 VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?, ?, ?)`,
		item.ID, item.Name, item.Mode, item.PublicHost, item.PublicPort, item.ScopeKey, item.ParentNodeID, 1, item.Status, now, now,
	)
	return item, err
}

func (s *MySQLStore) ProvisionNodeAccess(nodeID string) (domain.ApproveNodeEnrollmentResult, error) {
	var (
		node    domain.Node
		enabled int
	)
	err := s.db.QueryRow(
		`SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0)
		 FROM nodes WHERE id = ?`,
		nodeID,
	).Scan(&node.ID, &node.Name, &node.Mode, &node.ScopeKey, &node.ParentNodeID, &enabled, &node.Status, &node.PublicHost, &node.PublicPort)
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	node.Enabled = enabled == 1
	trustMaterial, err := auth.RandomToken()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	accessToken, err := auth.RandomToken()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	now := nowRFC3339()
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM node_api_tokens WHERE node_id = ?", nodeID); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`UPDATE node_trust_materials SET status = ?, updated_at = ? WHERE node_id = ? AND status = 'active'`,
		"rotated", now, nodeID,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec("UPDATE nodes SET status = ?, updated_at = ? WHERE id = ?", "healthy", now, nodeID); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_trust_materials (id, node_id, material_type, material_value, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		newID("trust"), nodeID, "shared_secret", trustMaterial, "active", now, now,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_api_tokens (id, node_id, token_hash, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		newID("node-token"), nodeID, accessToken, expiresAt, now, now,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if err := s.assignLatestPolicyTx(tx, nodeID, now); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	node.Status = "healthy"
	return domain.ApproveNodeEnrollmentResult{
		Node:          node,
		AccessToken:   accessToken,
		TrustMaterial: trustMaterial,
		ExpiresAt:     expiresAt,
	}, nil
}

func (s *MySQLStore) assignLatestPolicyTx(tx *sql.Tx, nodeID string, assignedAt string) error {
	var latestRevisionID string
	err := tx.QueryRow(
		`SELECT id FROM policy_revisions ORDER BY created_at DESC LIMIT 1`,
	).Scan(&latestRevisionID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	snapshotJSON, err := policy.CompileForNode(nodeID, s.policyNodes(), s.ListNodeLinks(), s.ListChains(), s.ListRouteRules())
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM node_policy_assignments WHERE node_id = ?`, nodeID); err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO node_policy_assignments (node_id, policy_revision_id, snapshot_json, assigned_at) VALUES (?, ?, ?, ?)`,
		nodeID, latestRevisionID, snapshotJSON, assignedAt,
	)
	return err
}

func (s *MySQLStore) UpdateNode(nodeID string, input domain.UpdateNodeInput) (domain.Node, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE nodes
		 SET name = ?, mode = ?, public_host = NULLIF(?, ''), public_port = ?, scope_key = ?, parent_node_id = NULLIF(?, ''), enabled = ?, status = ?, updated_at = ?
		 WHERE id = ?`,
		input.Name, input.Mode, input.PublicHost, input.PublicPort, input.ScopeKey, input.ParentNodeID, boolToInt(input.Enabled), input.Status, now, nodeID,
	)
	if err != nil {
		return domain.Node{}, err
	}
	for _, item := range s.ListNodes() {
		if item.ID == nodeID {
			return item, nil
		}
	}
	return domain.Node{}, sql.ErrNoRows
}

func (s *MySQLStore) DeleteNode(nodeID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statements := []string{
		"DELETE FROM chain_hops WHERE node_id = ?",
		"DELETE FROM node_links WHERE source_node_id = ? OR target_node_id = ?",
		"DELETE FROM node_onboarding_tasks WHERE target_node_id = ?",
		"DELETE FROM node_access_paths WHERE target_node_id = ? OR entry_node_id = ?",
		"DELETE FROM node_policy_assignments WHERE node_id = ?",
		"DELETE FROM node_health_snapshots WHERE node_id = ?",
		"DELETE FROM node_api_tokens WHERE node_id = ?",
		"DELETE FROM node_trust_materials WHERE node_id = ?",
		"UPDATE nodes SET parent_node_id = NULL WHERE parent_node_id = ?",
	}
	for _, statement := range statements {
		if strings.Count(statement, "?") == 2 {
			if _, err := tx.Exec(statement, nodeID, nodeID); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.Exec(statement, nodeID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("DELETE FROM nodes WHERE id = ?", nodeID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MySQLStore) ListChains() []domain.Chain {
	rows, err := s.db.Query("SELECT id, name, destination_scope, enabled FROM chains ORDER BY name")
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.Chain, 0)
	for rows.Next() {
		var item domain.Chain
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.DestinationScope, &enabled); err != nil {
			continue
		}
		item.Enabled = enabled == 1
		item.Hops = s.loadChainHops(item.ID)
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) GetChainProbeResult(chainID string) (domain.ChainProbeResult, bool) {
	var item domain.ChainProbeResult
	var hopsJSON string
	err := s.db.QueryRow(
		`SELECT chain_id, status, message, resolved_hops_json, COALESCE(blocking_node_id, ''), COALESCE(blocking_reason, ''), COALESCE(target_host, ''), target_port, probed_at
		 FROM chain_probe_results WHERE chain_id = ?`,
		chainID,
	).Scan(&item.ChainID, &item.Status, &item.Message, &hopsJSON, &item.BlockingNodeID, &item.BlockingReason, &item.TargetHost, &item.TargetPort, &item.ProbedAt)
	if err != nil {
		return domain.ChainProbeResult{}, false
	}
	_ = json.Unmarshal([]byte(hopsJSON), &item.ResolvedHops)
	return item, true
}

func (s *MySQLStore) SaveChainProbeResult(input domain.SaveChainProbeResultInput) (domain.ChainProbeResult, error) {
	hopsJSON, err := json.Marshal(input.ResolvedHops)
	if err != nil {
		return domain.ChainProbeResult{}, err
	}
	_, err = s.db.Exec(
		`INSERT INTO chain_probe_results (chain_id, status, message, resolved_hops_json, blocking_node_id, blocking_reason, target_host, target_port, probed_at)
		 VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?)
		 ON DUPLICATE KEY UPDATE
		   status = VALUES(status),
		   message = VALUES(message),
		   resolved_hops_json = VALUES(resolved_hops_json),
		   blocking_node_id = VALUES(blocking_node_id),
		   blocking_reason = VALUES(blocking_reason),
		   target_host = VALUES(target_host),
		   target_port = VALUES(target_port),
		   probed_at = VALUES(probed_at)`,
		input.ChainID, input.Status, input.Message, string(hopsJSON), input.BlockingNodeID, input.BlockingReason, input.TargetHost, input.TargetPort, input.ProbedAt,
	)
	if err != nil {
		return domain.ChainProbeResult{}, err
	}
	return domain.ChainProbeResult{
		ChainID:        input.ChainID,
		Status:         input.Status,
		Message:        input.Message,
		ResolvedHops:   input.ResolvedHops,
		BlockingNodeID: input.BlockingNodeID,
		BlockingReason: input.BlockingReason,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		ProbedAt:       input.ProbedAt,
	}, nil
}

func (s *MySQLStore) loadChainHops(chainID string) []string {
	rows, err := s.db.Query("SELECT node_id FROM chain_hops WHERE chain_id = ? ORDER BY hop_index", chainID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	hops := make([]string, 0)
	for rows.Next() {
		var nodeID string
		if err := rows.Scan(&nodeID); err != nil {
			continue
		}
		hops = append(hops, nodeID)
	}
	return hops
}

func (s *MySQLStore) CreateChain(input domain.CreateChainInput) (domain.Chain, error) {
	item := domain.Chain{ID: newID("chain"), Name: input.Name, DestinationScope: input.DestinationScope, Enabled: true, Hops: input.Hops}
	now := nowRFC3339()
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Chain{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO chains (id, name, destination_scope, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, item.Name, item.DestinationScope, 1, now, now,
	); err != nil {
		return domain.Chain{}, err
	}
	for index, hop := range item.Hops {
		if _, err := tx.Exec("INSERT INTO chain_hops (chain_id, hop_index, node_id) VALUES (?, ?, ?)", item.ID, index, hop); err != nil {
			return domain.Chain{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.Chain{}, err
	}
	return item, nil
}

func (s *MySQLStore) UpdateChain(chainID string, input domain.UpdateChainInput) (domain.Chain, error) {
	now := nowRFC3339()
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Chain{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`UPDATE chains SET name = ?, destination_scope = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		input.Name, input.DestinationScope, boolToInt(input.Enabled), now, chainID,
	); err != nil {
		return domain.Chain{}, err
	}
	if _, err := tx.Exec("DELETE FROM chain_hops WHERE chain_id = ?", chainID); err != nil {
		return domain.Chain{}, err
	}
	for index, hop := range input.Hops {
		if _, err := tx.Exec("INSERT INTO chain_hops (chain_id, hop_index, node_id) VALUES (?, ?, ?)", chainID, index, hop); err != nil {
			return domain.Chain{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.Chain{}, err
	}
	for _, item := range s.ListChains() {
		if item.ID == chainID {
			return item, nil
		}
	}
	return domain.Chain{}, sql.ErrNoRows
}

func (s *MySQLStore) DeleteChain(chainID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM chain_hops WHERE chain_id = ?", chainID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM chains WHERE id = ?", chainID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MySQLStore) ListRouteRules() []domain.RouteRule {
	rows, err := s.db.Query(
		`SELECT id, priority, match_type, match_value, action_type, COALESCE(chain_id, ''), COALESCE(destination_scope, ''), enabled
		 FROM route_rules ORDER BY priority ASC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.RouteRule, 0)
	for rows.Next() {
		var item domain.RouteRule
		var enabled int
		if err := rows.Scan(&item.ID, &item.Priority, &item.MatchType, &item.MatchValue, &item.ActionType, &item.ChainID, &item.DestinationScope, &enabled); err != nil {
			continue
		}
		item.Enabled = enabled == 1
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateRouteRule(input domain.CreateRouteRuleInput) (domain.RouteRule, error) {
	item := domain.RouteRule{
		ID:               newID("rule"),
		Priority:         input.Priority,
		MatchType:        input.MatchType,
		MatchValue:       input.MatchValue,
		ActionType:       input.ActionType,
		ChainID:          input.ChainID,
		DestinationScope: input.DestinationScope,
		Enabled:          true,
	}
	now := nowRFC3339()
	_, err := s.db.Exec(
		`INSERT INTO route_rules (id, priority, match_type, match_value, action_type, chain_id, destination_scope, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)`,
		item.ID, item.Priority, item.MatchType, item.MatchValue, item.ActionType, item.ChainID, item.DestinationScope, 1, now, now,
	)
	return item, err
}

func (s *MySQLStore) UpdateRouteRule(ruleID string, input domain.UpdateRouteRuleInput) (domain.RouteRule, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE route_rules
		 SET priority = ?, match_type = ?, match_value = ?, action_type = ?, chain_id = NULLIF(?, ''), destination_scope = NULLIF(?, ''), enabled = ?, updated_at = ?
		 WHERE id = ?`,
		input.Priority, input.MatchType, input.MatchValue, input.ActionType, input.ChainID, input.DestinationScope, boolToInt(input.Enabled), now, ruleID,
	)
	if err != nil {
		return domain.RouteRule{}, err
	}
	for _, item := range s.ListRouteRules() {
		if item.ID == ruleID {
			return item, nil
		}
	}
	return domain.RouteRule{}, sql.ErrNoRows
}

func (s *MySQLStore) DeleteRouteRule(ruleID string) error {
	_, err := s.db.Exec("DELETE FROM route_rules WHERE id = ?", ruleID)
	return err
}

func (s *MySQLStore) ListNodeHealth() []domain.NodeHealth {
	rows, err := s.db.Query(
		`SELECT node_id, heartbeat_at, COALESCE(policy_revision_id, ''), listener_status_json, cert_status_json
		 FROM node_health_snapshots ORDER BY node_id`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.NodeHealth, 0)
	for rows.Next() {
		var item domain.NodeHealth
		var listenerJSON string
		var certJSON string
		if err := rows.Scan(&item.NodeID, &item.HeartbeatAt, &item.PolicyRevisionID, &listenerJSON, &certJSON); err != nil {
			continue
		}
		item.ListenerStatus = decodeJSONMap(listenerJSON)
		item.CertStatus = decodeJSONMap(certJSON)
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ListNodeHealthHistory(nodeID string, window time.Duration) ([]domain.NodeHealth, error) {
	since := time.Now().Add(-window).UTC().Format(time.RFC3339)
	rows, err := s.db.Query(
		`SELECT node_id, heartbeat_at, policy_revision_id, listener_status_json, cert_status_json
		 FROM node_health_history
		 WHERE node_id = ? AND heartbeat_at >= ?
		 ORDER BY heartbeat_at ASC`, nodeID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.NodeHealth, 0)
	for rows.Next() {
		var item domain.NodeHealth
		var listenerJSON, certJSON string
		if err := rows.Scan(&item.NodeID, &item.HeartbeatAt, &item.PolicyRevisionID, &listenerJSON, &certJSON); err != nil {
			continue
		}
		item.ListenerStatus = decodeJSONMap(listenerJSON)
		item.CertStatus = decodeJSONMap(certJSON)
		items = append(items, item)
	}
	return items, nil
}

func (s *MySQLStore) CreateBootstrapToken(input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error) {
	token, err := auth.RandomToken()
	if err != nil {
		return domain.BootstrapToken{}, err
	}
	item := domain.BootstrapToken{
		ID:         newID("bootstrap"),
		Token:      token,
		TargetType: input.TargetType,
		TargetID:   input.TargetID,
		ExpiresAt:  time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
	}
	_, err = s.db.Exec(
		`INSERT INTO bootstrap_tokens (id, token_hash, target_type, target_id, expires_at, consumed_at, created_at)
		 VALUES (?, ?, ?, NULLIF(?, ''), ?, NULL, ?)`,
		item.ID, token, item.TargetType, item.TargetID, item.ExpiresAt, nowRFC3339(),
	)
	return item, err
}

func (s *MySQLStore) EnrollNode(input domain.EnrollNodeInput) (domain.EnrollNodeResult, error) {
	var (
		tokenID    string
		targetID   sql.NullString
		expiresAt  string
		consumedAt sql.NullString
	)
	err := s.db.QueryRow(
		`SELECT id, target_id, expires_at, consumed_at FROM bootstrap_tokens WHERE token_hash = ?`,
		input.Token,
	).Scan(&tokenID, &targetID, &expiresAt, &consumedAt)
	if err != nil {
		return domain.EnrollNodeResult{}, err
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expiry) || consumedAt.Valid {
		return domain.EnrollNodeResult{}, fmt.Errorf("invalid bootstrap token")
	}
	now := nowRFC3339()
	enrollmentSecret, err := auth.RandomToken()
	if err != nil {
		return domain.EnrollNodeResult{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.EnrollNodeResult{}, err
	}
	defer tx.Rollback()
	var node domain.Node
	if targetID.Valid && targetID.String != "" {
		var enabled int
		err = tx.QueryRow(
			`SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0)
			 FROM nodes WHERE id = ?`,
			targetID.String,
		).Scan(&node.ID, &node.Name, &node.Mode, &node.ScopeKey, &node.ParentNodeID, &enabled, &node.Status, &node.PublicHost, &node.PublicPort)
		if err != nil {
			return domain.EnrollNodeResult{}, err
		}
		if _, err := tx.Exec(
			`UPDATE nodes
			 SET name = ?, mode = ?, public_host = NULLIF(?, ''), public_port = ?, scope_key = ?, parent_node_id = NULLIF(?, ''), enabled = ?, status = ?, updated_at = ?
			 WHERE id = ?`,
			input.Name, input.Mode, input.PublicHost, input.PublicPort, input.ScopeKey, input.ParentNodeID, 1, "pending", now, node.ID,
		); err != nil {
			return domain.EnrollNodeResult{}, err
		}
		node.Name = input.Name
		node.Mode = input.Mode
		node.ScopeKey = input.ScopeKey
		node.ParentNodeID = input.ParentNodeID
		node.PublicHost = input.PublicHost
		node.PublicPort = input.PublicPort
		node.Enabled = true
		node.Status = "pending"
	} else {
		nodeID, err := s.nextNodeID()
		if err != nil {
			return domain.EnrollNodeResult{}, err
		}
		node = domain.Node{
			ID:           nodeID,
			Name:         input.Name,
			Mode:         input.Mode,
			ScopeKey:     input.ScopeKey,
			ParentNodeID: input.ParentNodeID,
			Enabled:      true,
			Status:       "pending",
			PublicHost:   input.PublicHost,
			PublicPort:   input.PublicPort,
		}
		if _, err := tx.Exec(
			`INSERT INTO nodes (id, name, mode, public_host, public_port, scope_key, parent_node_id, enabled, status, created_at, updated_at)
			 VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?, ?, ?)`,
			node.ID, node.Name, node.Mode, node.PublicHost, node.PublicPort, node.ScopeKey, node.ParentNodeID, 1, node.Status, now, now,
		); err != nil {
			return domain.EnrollNodeResult{}, err
		}
	}
	if _, err := tx.Exec("UPDATE bootstrap_tokens SET consumed_at = ? WHERE id = ?", now, tokenID); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	if _, err := tx.Exec("DELETE FROM node_api_tokens WHERE node_id = ?", node.ID); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	if _, err := tx.Exec(
		`UPDATE node_trust_materials SET status = ?, updated_at = ? WHERE node_id = ?`,
		"rotated", now, node.ID,
	); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_trust_materials (id, node_id, material_type, material_value, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		newID("trust"), node.ID, "enrollment_secret", enrollmentSecret, "pending", now, now,
	); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	return domain.EnrollNodeResult{
		Node:             domain.Node{ID: node.ID, Name: node.Name, Mode: node.Mode, ScopeKey: node.ScopeKey, ParentNodeID: node.ParentNodeID, Enabled: node.Enabled, Status: "pending", PublicHost: node.PublicHost, PublicPort: node.PublicPort},
		EnrollmentSecret: enrollmentSecret,
		ApprovalState:    "pending",
	}, nil
}

func (s *MySQLStore) ApproveNodeEnrollment(nodeID string) (domain.ApproveNodeEnrollmentResult, error) {
	var (
		node    domain.Node
		enabled int
	)
	err := s.db.QueryRow(
		`SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0)
		 FROM nodes WHERE id = ?`,
		nodeID,
	).Scan(&node.ID, &node.Name, &node.Mode, &node.ScopeKey, &node.ParentNodeID, &enabled, &node.Status, &node.PublicHost, &node.PublicPort)
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	node.Enabled = enabled == 1
	if node.Status != "pending" {
		return domain.ApproveNodeEnrollmentResult{}, fmt.Errorf("node_not_pending")
	}
	trustMaterial, err := auth.RandomToken()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	accessToken, err := auth.RandomToken()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	now := nowRFC3339()
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM node_api_tokens WHERE node_id = ?", nodeID); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec("UPDATE nodes SET status = ?, updated_at = ? WHERE id = ?", "healthy", now, nodeID); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`UPDATE node_trust_materials SET status = ?, updated_at = ? WHERE node_id = ? AND material_type = 'shared_secret' AND status = 'active'`,
		"rotated", now, nodeID,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_trust_materials (id, node_id, material_type, material_value, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		newID("trust"), nodeID, "shared_secret", trustMaterial, "active", now, now,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_api_tokens (id, node_id, token_hash, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		newID("node-token"), nodeID, accessToken, expiresAt, now, now,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	node.Status = "healthy"
	return domain.ApproveNodeEnrollmentResult{
		Node:          node,
		AccessToken:   accessToken,
		TrustMaterial: trustMaterial,
		ExpiresAt:     expiresAt,
	}, nil
}

func (s *MySQLStore) ExchangeNodeEnrollment(input domain.ExchangeNodeEnrollmentInput) (domain.ApproveNodeEnrollmentResult, error) {
	var (
		node        domain.Node
		enabled     int
		accessToken string
		expiresAt   string
		trustValue  string
	)
	err := s.db.QueryRow(
		`SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0)
		 FROM nodes WHERE id = ?`,
		input.NodeID,
	).Scan(&node.ID, &node.Name, &node.Mode, &node.ScopeKey, &node.ParentNodeID, &enabled, &node.Status, &node.PublicHost, &node.PublicPort)
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	node.Enabled = enabled == 1
	var enrollmentSecretCount int
	if err := s.db.QueryRow(
		`SELECT COUNT(1) FROM node_trust_materials
		 WHERE node_id = ? AND material_type = 'enrollment_secret' AND material_value = ? AND status = 'pending'`,
		input.NodeID, input.EnrollmentSecret,
	).Scan(&enrollmentSecretCount); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if enrollmentSecretCount == 0 {
		return domain.ApproveNodeEnrollmentResult{}, fmt.Errorf("invalid_enrollment_secret")
	}
	if node.Status == "pending" {
		return domain.ApproveNodeEnrollmentResult{}, fmt.Errorf("node_enrollment_pending")
	}
	err = s.db.QueryRow(
		`SELECT token_hash, expires_at FROM node_api_tokens WHERE node_id = ? ORDER BY created_at DESC LIMIT 1`,
		input.NodeID,
	).Scan(&accessToken, &expiresAt)
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	err = s.db.QueryRow(
		`SELECT material_value FROM node_trust_materials
		 WHERE node_id = ? AND material_type = 'shared_secret' AND status = 'active'
		 ORDER BY created_at DESC LIMIT 1`,
		input.NodeID,
	).Scan(&trustValue)
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := s.db.Exec(
		`UPDATE node_trust_materials
		 SET status = ?, updated_at = ?
		 WHERE node_id = ? AND material_type = 'enrollment_secret' AND material_value = ? AND status = 'pending'`,
		"consumed", nowRFC3339(), input.NodeID, input.EnrollmentSecret,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	return domain.ApproveNodeEnrollmentResult{
		Node:          node,
		AccessToken:   accessToken,
		TrustMaterial: trustValue,
		ExpiresAt:     expiresAt,
	}, nil
}

func (s *MySQLStore) ListNodeEnrollmentApprovals() []domain.NodeEnrollmentApproval {
	rows, err := s.db.Query(
		`SELECT id, bootstrap_token_id, node_name, node_mode, scope_key, parent_node_id,
		        public_host, public_port, status, reviewed_by, reviewed_at, reject_reason,
		        created_at, updated_at
		 FROM node_enrollment_approvals
		 WHERE status = 'pending'
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.NodeEnrollmentApproval, 0)
	for rows.Next() {
		var item domain.NodeEnrollmentApproval
		var reviewedBy, reviewedAt, rejectReason sql.NullString
		if err := rows.Scan(
			&item.ID, &item.BootstrapTokenID, &item.NodeName, &item.NodeMode, &item.ScopeKey,
			&item.ParentNodeID, &item.PublicHost, &item.PublicPort, &item.Status,
			&reviewedBy, &reviewedAt, &rejectReason, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			continue
		}
		if reviewedBy.Valid {
			item.ReviewedBy = reviewedBy.String
		}
		if reviewedAt.Valid {
			item.ReviewedAt = reviewedAt.String
		}
		if rejectReason.Valid {
			item.RejectReason = rejectReason.String
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ApproveNodeEnrollmentApproval(approvalID string, accountID string, input domain.ApproveEnrollmentInput) (domain.NodeEnrollmentApproval, error) {
	var approval domain.NodeEnrollmentApproval
	var reviewedBy, reviewedAt, rejectReason sql.NullString
	err := s.db.QueryRow(
		`SELECT id, bootstrap_token_id, node_name, node_mode, scope_key, parent_node_id,
		        public_host, public_port, status, reviewed_by, reviewed_at, reject_reason,
		        created_at, updated_at
		 FROM node_enrollment_approvals WHERE id = ?`,
		approvalID,
	).Scan(
		&approval.ID, &approval.BootstrapTokenID, &approval.NodeName, &approval.NodeMode,
		&approval.ScopeKey, &approval.ParentNodeID, &approval.PublicHost, &approval.PublicPort,
		&approval.Status, &reviewedBy, &reviewedAt, &rejectReason, &approval.CreatedAt, &approval.UpdatedAt,
	)
	if err != nil {
		return domain.NodeEnrollmentApproval{}, err
	}

	if approval.Status != "pending" {
		return domain.NodeEnrollmentApproval{}, fmt.Errorf("approval already processed")
	}

	now := nowRFC3339()
	_, err = s.db.Exec(
		`UPDATE node_enrollment_approvals
		 SET status = 'approved', reviewed_by = ?, reviewed_at = ?, updated_at = ?
		 WHERE id = ?`,
		accountID, now, now, approvalID,
	)
	if err != nil {
		return domain.NodeEnrollmentApproval{}, err
	}

	approval.Status = "approved"
	approval.ReviewedBy = accountID
	approval.ReviewedAt = now
	approval.UpdatedAt = now

	return approval, nil
}

func (s *MySQLStore) RejectNodeEnrollmentApproval(approvalID string, accountID string, input domain.RejectEnrollmentInput) (domain.NodeEnrollmentApproval, error) {
	var approval domain.NodeEnrollmentApproval
	var reviewedBy, reviewedAt, rejectReason sql.NullString
	err := s.db.QueryRow(
		`SELECT id, bootstrap_token_id, node_name, node_mode, scope_key, parent_node_id,
		        public_host, public_port, status, reviewed_by, reviewed_at, reject_reason,
		        created_at, updated_at
		 FROM node_enrollment_approvals WHERE id = ?`,
		approvalID,
	).Scan(
		&approval.ID, &approval.BootstrapTokenID, &approval.NodeName, &approval.NodeMode,
		&approval.ScopeKey, &approval.ParentNodeID, &approval.PublicHost, &approval.PublicPort,
		&approval.Status, &reviewedBy, &reviewedAt, &rejectReason, &approval.CreatedAt, &approval.UpdatedAt,
	)
	if err != nil {
		return domain.NodeEnrollmentApproval{}, err
	}

	if approval.Status != "pending" {
		return domain.NodeEnrollmentApproval{}, fmt.Errorf("approval already processed")
	}

	now := nowRFC3339()
	_, err = s.db.Exec(
		`UPDATE node_enrollment_approvals
		 SET status = 'rejected', reviewed_by = ?, reviewed_at = ?, reject_reason = ?, updated_at = ?
		 WHERE id = ?`,
		accountID, now, input.RejectReason, now, approvalID,
	)
	if err != nil {
		return domain.NodeEnrollmentApproval{}, err
	}

	approval.Status = "rejected"
	approval.ReviewedBy = accountID
	approval.ReviewedAt = now
	approval.RejectReason = input.RejectReason
	approval.UpdatedAt = now

	return approval, nil
}

func (s *MySQLStore) ListPolicyRevisions() []domain.PolicyRevision {
	rows, err := s.db.Query(
		`SELECT p.id, p.version, p.status, p.created_at, COUNT(a.node_id)
		 FROM policy_revisions p
		 LEFT JOIN node_policy_assignments a ON a.policy_revision_id = p.id
		 GROUP BY p.id, p.version, p.status, p.created_at
		 ORDER BY p.created_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.PolicyRevision, 0)
	for rows.Next() {
		var item domain.PolicyRevision
		if err := rows.Scan(&item.ID, &item.Version, &item.Status, &item.CreatedAt, &item.AssignedNodes); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) PublishPolicy(accountID string) (domain.PolicyRevision, error) {
	nodes := s.policyNodes()
	links := s.ListNodeLinks()
	chains := s.ListChains()
	rules := s.ListRouteRules()
	raw, err := policy.Compile(nodes, links, chains, rules)
	if err != nil {
		return domain.PolicyRevision{}, err
	}
	item := domain.PolicyRevision{
		ID:            newID("policy"),
		Version:       fmt.Sprintf("rev-%d", time.Now().Unix()),
		Status:        "published",
		CreatedAt:     nowRFC3339(),
		AssignedNodes: len(nodes),
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.PolicyRevision{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO policy_revisions (id, version, payload_json, status, created_by_account_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, item.Version, raw, item.Status, accountID, item.CreatedAt,
	); err != nil {
		return domain.PolicyRevision{}, err
	}
	if _, err := tx.Exec("DELETE FROM node_policy_assignments"); err != nil {
		return domain.PolicyRevision{}, err
	}
	for _, node := range nodes {
		snapshotJSON, err := policy.CompileForNode(node.ID, nodes, links, chains, rules)
		if err != nil {
			return domain.PolicyRevision{}, err
		}
		if _, err := tx.Exec(
			`INSERT INTO node_policy_assignments (node_id, policy_revision_id, snapshot_json, assigned_at) VALUES (?, ?, ?, ?)`,
			node.ID, item.ID, snapshotJSON, item.CreatedAt,
		); err != nil {
			return domain.PolicyRevision{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.PolicyRevision{}, err
	}
	return item, nil
}

func (s *MySQLStore) AuthenticateNodeToken(accessToken string) (string, bool) {
	var (
		nodeID    string
		expiresAt string
		status    string
		enabled   int
	)
	err := s.db.QueryRow(
		`SELECT t.node_id, t.expires_at, n.status, n.enabled
		 FROM node_api_tokens t
		 JOIN nodes n ON n.id = t.node_id
		 WHERE t.token_hash = ?`,
		accessToken,
	).Scan(&nodeID, &expiresAt, &status, &enabled)
	if err != nil {
		return "", false
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expiry) || enabled != 1 || status == "pending" {
		return "", false
	}
	return nodeID, true
}

func (s *MySQLStore) policyNodes() []domain.Node {
	all := s.ListNodes()
	items := make([]domain.Node, 0, len(all))
	for _, node := range all {
		if !node.Enabled || node.Status == "pending" {
			continue
		}
		items = append(items, node)
	}
	return items
}

func (s *MySQLStore) CleanupExpiredSessions() (int64, error) {
	result, err := s.db.Exec("DELETE FROM sessions WHERE expires_at <= ?", nowRFC3339())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *MySQLStore) CleanupExpiredBootstrapTokens() (int64, error) {
	result, err := s.db.Exec("DELETE FROM bootstrap_tokens WHERE expires_at <= ? OR consumed_at IS NOT NULL", nowRFC3339())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *MySQLStore) CleanupExpiredNodeTokens() (int64, error) {
	result, err := s.db.Exec("DELETE FROM node_api_tokens WHERE expires_at <= ?", nowRFC3339())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *MySQLStore) RefreshCertificateStatus(window time.Duration) error {
	now := time.Now().UTC()
	renewBefore := now.Add(window).Format(time.RFC3339)
	if _, err := s.db.Exec(
		"UPDATE certificates SET status = 'healthy', updated_at = ? WHERE not_after IS NOT NULL AND not_after > ?",
		now.Format(time.RFC3339), renewBefore,
	); err != nil {
		return err
	}
	if _, err := s.db.Exec(
		"UPDATE certificates SET status = 'expired', updated_at = ? WHERE not_after IS NOT NULL AND not_after <= ?",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	); err != nil {
		return err
	}
	_, err := s.db.Exec(
		"UPDATE certificates SET status = 'renew-soon', updated_at = ? WHERE not_after IS NOT NULL AND not_after > ? AND not_after <= ?",
		now.Format(time.RFC3339), now.Format(time.RFC3339), renewBefore,
	)
	return err
}

func (s *MySQLStore) RefreshNodeStatus(staleAfter time.Duration) error {
	staleAt := time.Now().UTC().Add(-staleAfter).Format(time.RFC3339)
	if _, err := s.db.Exec(
		"UPDATE nodes SET status = 'degraded', updated_at = ? WHERE id IN (SELECT node_id FROM node_health_snapshots WHERE heartbeat_at <= ?)",
		nowRFC3339(), staleAt,
	); err != nil {
		return err
	}
	_, err := s.db.Exec(
		"UPDATE nodes SET status = 'healthy', updated_at = ? WHERE id IN (SELECT node_id FROM node_health_snapshots WHERE heartbeat_at > ?)",
		nowRFC3339(), staleAt,
	)
	return err
}

func (s *MySQLStore) CleanupNodeHealthHistory(retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention).UTC().Format(time.RFC3339)
	result, err := s.db.Exec("DELETE FROM node_health_history WHERE heartbeat_at < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *MySQLStore) CreateGroup(input domain.CreateGroupInput) (domain.Group, error) {
	now := nowRFC3339()
	enabled := 1
	if input.Enabled != nil && !*input.Enabled {
		enabled = 0
	}
	item := domain.Group{
		ID:          newID("grp"),
		Name:        input.Name,
		Description: input.Description,
		Enabled:     enabled == 1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := s.db.Exec(
		`INSERT INTO groups (id, name, description, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, item.Name, item.Description, enabled, now, now,
	)
	return item, err
}

func (s *MySQLStore) UpdateGroup(id string, input domain.UpdateGroupInput) (domain.Group, error) {
	group, err := s.GetGroup(id)
	if err != nil {
		return domain.Group{}, err
	}
	if input.Name != nil {
		group.Name = *input.Name
	}
	if input.Description != nil {
		group.Description = *input.Description
	}
	if input.Enabled != nil {
		group.Enabled = *input.Enabled
	}
	now := nowRFC3339()
	group.UpdatedAt = now
	_, err = s.db.Exec(
		`UPDATE groups SET name = ?, description = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		group.Name, group.Description, boolToInt(group.Enabled), now, id,
	)
	if err != nil {
		return domain.Group{}, err
	}
	return group, nil
}

func (s *MySQLStore) DeleteGroup(id string) error {
	_, err := s.GetGroup(id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM groups WHERE id = ?", id)
	return err
}

func (s *MySQLStore) GetGroup(id string) (domain.Group, error) {
	var item domain.Group
	var enabled int
	err := s.db.QueryRow(
		`SELECT id, name, COALESCE(description, ''), enabled, created_at, updated_at FROM groups WHERE id = ?`,
		id,
	).Scan(&item.ID, &item.Name, &item.Description, &enabled, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.Group{}, err
	}
	item.Enabled = enabled == 1
	return item, nil
}

func (s *MySQLStore) ListGroups() ([]domain.Group, error) {
	rows, err := s.db.Query(
		`SELECT id, name, COALESCE(description, ''), enabled, created_at, updated_at FROM groups ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]domain.Group, 0)
	for rows.Next() {
		var item domain.Group
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		item.Enabled = enabled == 1
		groups = append(groups, item)
	}
	return groups, nil
}

func (s *MySQLStore) ListAccountGroups(accountID string) ([]domain.Group, error) {
	rows, err := s.db.Query(
		`SELECT g.id, g.name, COALESCE(g.description, ''), g.enabled, g.created_at, g.updated_at
		 FROM groups g
		 JOIN account_groups ag ON ag.group_id = g.id
		 WHERE ag.account_id = ? AND g.enabled = 1
		 ORDER BY g.name`,
		accountID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]domain.Group, 0)
	for rows.Next() {
		var item domain.Group
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		item.Enabled = enabled == 1
		groups = append(groups, item)
	}
	return groups, nil
}

func (s *MySQLStore) GetGroupScopes(groupID string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT scope_key FROM group_scopes WHERE group_id = ?`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	scopes := make([]string, 0)
	for rows.Next() {
		var scope string
		if err := rows.Scan(&scope); err != nil {
			continue
		}
		scopes = append(scopes, scope)
	}
	return scopes, nil
}

func (s *MySQLStore) AddAccountToGroup(accountID, groupID string) error {
	_, err := s.db.Exec("INSERT INTO account_groups (account_id, group_id) VALUES (?, ?)", accountID, groupID)
	return err
}

func (s *MySQLStore) RemoveAccountFromGroup(accountID, groupID string) error {
	_, err := s.db.Exec("DELETE FROM account_groups WHERE account_id = ? AND group_id = ?", accountID, groupID)
	return err
}

func (s *MySQLStore) ListGroupAccounts(groupID string) ([]domain.Account, error) {
	rows, err := s.db.Query(
		`SELECT a.id, a.account, r.name, a.status, a.must_rotate_password
		 FROM accounts a
		 JOIN account_groups ag ON ag.account_id = a.id
		 JOIN roles r ON r.id = a.role_id
		 WHERE ag.group_id = ?
		 ORDER BY a.account`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := make([]domain.Account, 0)
	for rows.Next() {
		var item domain.Account
		var mustRotate int
		if err := rows.Scan(&item.ID, &item.Account, &item.Role, &item.Status, &mustRotate); err != nil {
			continue
		}
		item.MustRotatePassword = mustRotate == 1
		accounts = append(accounts, item)
	}
	return accounts, nil
}

func (s *MySQLStore) SetGroupAccounts(groupID string, accountIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM account_groups WHERE group_id = ?", groupID); err != nil {
		return err
	}
	for _, accountID := range accountIDs {
		if _, err := tx.Exec("INSERT INTO account_groups (account_id, group_id) VALUES (?, ?)", accountID, groupID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *MySQLStore) SetGroupScopes(groupID string, scopeKeys []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM group_scopes WHERE group_id = ?", groupID); err != nil {
		return err
	}
	for _, key := range scopeKeys {
		if _, err := tx.Exec("INSERT INTO group_scopes (group_id, scope_key) VALUES (?, ?)", groupID, key); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *MySQLStore) GetNodeAgentPolicy(nodeID string) (domain.NodeAgentPolicy, bool) {
	var (
		policyID string
		payload  string
		version  string
	)
	err := s.db.QueryRow(
		`SELECT p.id, p.version, a.snapshot_json
		 FROM node_policy_assignments a
		 JOIN policy_revisions p ON p.id = a.policy_revision_id
		WHERE a.node_id = ?
		 ORDER BY a.assigned_at DESC
		 LIMIT 1`,
		nodeID,
	).Scan(&policyID, &version, &payload)
	if err == nil {
		return domain.NodeAgentPolicy{
			NodeID:           nodeID,
			PolicyRevisionID: version,
			PayloadJSON:      payload,
		}, true
	}
	if err != sql.ErrNoRows {
		return domain.NodeAgentPolicy{}, false
	}
	latestPolicyVersion, snapshotJSON, ok := s.compileLatestPolicyForNode(nodeID)
	if !ok {
		return domain.NodeAgentPolicy{}, false
	}
	_, _ = s.db.Exec(
		`INSERT INTO node_policy_assignments (node_id, policy_revision_id, snapshot_json, assigned_at)
		 SELECT ?, id, ?, ? FROM policy_revisions ORDER BY created_at DESC LIMIT 1`,
		nodeID, snapshotJSON, nowRFC3339(),
	)
	return domain.NodeAgentPolicy{
		NodeID:           nodeID,
		PolicyRevisionID: latestPolicyVersion,
		PayloadJSON:      snapshotJSON,
	}, true
}

func (s *MySQLStore) compileLatestPolicyForNode(nodeID string) (string, string, bool) {
	exists, err := s.exists(context.Background(), "SELECT 1 FROM nodes WHERE id = ? AND enabled = 1 AND status != 'pending'", nodeID)
	if err != nil || !exists {
		return "", "", false
	}
	var revisionID string
	var version string
	err = s.db.QueryRow(
		`SELECT id, version FROM policy_revisions ORDER BY created_at DESC LIMIT 1`,
	).Scan(&revisionID, &version)
	if err != nil {
		return "", "", false
	}
	snapshotJSON, err := policy.CompileForNode(nodeID, s.policyNodes(), s.ListNodeLinks(), s.ListChains(), s.ListRouteRules())
	if err != nil {
		return "", "", false
	}
	_ = revisionID
	return version, snapshotJSON, true
}

func (s *MySQLStore) UpsertNodeHeartbeat(input domain.NodeHeartbeatInput) (domain.NodeHealth, error) {
	now := nowRFC3339()
	status := heartbeatNodeStatus(input.ListenerStatus, input.CertStatus)
	tx, err := s.db.Begin()
	if err != nil {
		return domain.NodeHealth{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO node_health_snapshots (node_id, heartbeat_at, policy_revision_id, listener_status_json, cert_status_json, updated_at)
		 VALUES (?, ?, NULLIF(?, ''), ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   heartbeat_at = VALUES(heartbeat_at),
		   policy_revision_id = VALUES(policy_revision_id),
		   listener_status_json = VALUES(listener_status_json),
		   cert_status_json = VALUES(cert_status_json),
		   updated_at = VALUES(updated_at)`,
		input.NodeID, now, input.PolicyRevisionID, encodeJSONMap(input.ListenerStatus), encodeJSONMap(input.CertStatus), now,
	); err != nil {
		return domain.NodeHealth{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_health_history (node_id, heartbeat_at, policy_revision_id, listener_status_json, cert_status_json, created_at)
		 VALUES (?, ?, NULLIF(?, ''), ?, ?, ?)`,
		input.NodeID, now, input.PolicyRevisionID, encodeJSONMap(input.ListenerStatus), encodeJSONMap(input.CertStatus), now,
	); err != nil {
		return domain.NodeHealth{}, err
	}
	if _, err := tx.Exec("UPDATE nodes SET status = ?, updated_at = ? WHERE id = ?", status, now, input.NodeID); err != nil {
		return domain.NodeHealth{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.NodeHealth{}, err
	}
	return domain.NodeHealth{
		NodeID:           input.NodeID,
		HeartbeatAt:      now,
		PolicyRevisionID: input.PolicyRevisionID,
		ListenerStatus:   input.ListenerStatus,
		CertStatus:       input.CertStatus,
	}, nil
}

func heartbeatNodeStatus(listenerStatus map[string]string, certStatus map[string]string) string {
	for _, value := range listenerStatus {
		if value != "up" && value != "healthy" && value != "renewed" {
			return "degraded"
		}
	}
	for _, value := range certStatus {
		if value != "up" && value != "healthy" && value != "renewed" {
			return "degraded"
		}
	}
	return "healthy"
}

func (s *MySQLStore) RenewNodeCertificate(input domain.NodeCertRenewInput) (domain.NodeCertRenewResult, error) {
	now := nowRFC3339()
	notAfter := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	var certID string
	provider := "internal_ca"
	if input.CertType == "public" {
		provider = "lets_encrypt"
	}
	err := s.db.QueryRow(
		`SELECT id FROM certificates WHERE owner_type = 'node' AND owner_id = ? AND cert_type = ? LIMIT 1`,
		input.NodeID, input.CertType,
	).Scan(&certID)
	if err != nil {
		certID = newID("cert")
		_, err = s.db.Exec(
			`INSERT INTO certificates (id, owner_type, owner_id, cert_type, provider, status, not_before, not_after, created_at, updated_at)
			 VALUES (?, 'node', ?, ?, ?, ?, ?, ?, ?, ?)`,
			certID, input.NodeID, input.CertType, provider, "renewed", now, notAfter, now, now,
		)
	} else {
		_, err = s.db.Exec(
			`UPDATE certificates SET provider = ?, status = ?, not_before = ?, not_after = ?, updated_at = ? WHERE id = ?`,
			provider, "renewed", now, notAfter, now, certID,
		)
	}
	if err != nil {
		return domain.NodeCertRenewResult{}, err
	}
	return domain.NodeCertRenewResult{
		NodeID:   input.NodeID,
		CertType: input.CertType,
		Status:   "renewed",
		NotAfter: notAfter,
	}, nil
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

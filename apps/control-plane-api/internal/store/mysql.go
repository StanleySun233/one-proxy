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

	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/policy"
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
	schemaPath, err := resolveSchemaPath()
	if err != nil {
		return err
	}
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
	if err := s.gormDB.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		return err
	}
	if err := s.bootstrapAdmin(ctx); err != nil {
		return err
	}
	if err := s.bootstrapTopology(ctx); err != nil {
		return err
	}
	return nil
}

func resolveSchemaPath() (string, error) {
	candidates := []string{
		filepath.Join("apps", "control-plane-api", "schema", "001_init.sql"),
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

func (s *MySQLStore) bootstrapTopology(ctx context.Context) error {
	exists, err := s.exists(ctx, "SELECT 1 FROM nodes LIMIT 1")
	if err != nil || exists {
		return err
	}
	now := nowRFC3339()

	for _, node := range defaultNodes() {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO nodes (id, name, mode, public_host, public_port, scope_key, parent_node_id, enabled, status, created_at, updated_at)
			 VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?, ?, ?)`,
			node.ID, node.Name, node.Mode, node.PublicHost, node.PublicPort, node.ScopeKey, node.ParentNodeID, boolToInt(node.Enabled), node.Status, now, now,
		); err != nil {
			return err
		}
	}

	for _, link := range defaultNodeLinks() {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO node_links (id, source_node_id, target_node_id, link_type, trust_state, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			link.ID, link.SourceNodeID, link.TargetNodeID, link.LinkType, link.TrustState, now, now,
		); err != nil {
			return err
		}
	}

	for _, chain := range defaultChains() {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO chains (id, name, destination_scope, enabled, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			chain.ID, chain.Name, chain.DestinationScope, boolToInt(chain.Enabled), now, now,
		); err != nil {
			return err
		}
		for index, hop := range chain.Hops {
			if _, err := s.db.ExecContext(ctx,
				`INSERT INTO chain_hops (chain_id, hop_index, node_id) VALUES (?, ?, ?)`,
				chain.ID, index, hop,
			); err != nil {
				return err
			}
		}
	}

	for _, rule := range defaultRouteRules() {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO route_rules (id, priority, match_type, match_value, action_type, chain_id, destination_scope, enabled, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)`,
			rule.ID, rule.Priority, rule.MatchType, rule.MatchValue, rule.ActionType, rule.ChainID, rule.DestinationScope, boolToInt(rule.Enabled), now, now,
		); err != nil {
			return err
		}
	}

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO policy_revisions (id, version, payload_json, status, created_by_account_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"policy-rev-0007", "rev-0007", `{"seed":true}`, "published", "acct-admin", now,
	); err != nil {
		return err
	}

	for _, health := range defaultNodeHealth() {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO node_health_snapshots (node_id, heartbeat_at, policy_revision_id, listener_status_json, cert_status_json, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			health.NodeID, health.HeartbeatAt, "policy-rev-0007", encodeJSONMap(health.ListenerStatus), encodeJSONMap(health.CertStatus), now,
		); err != nil {
			return err
		}
	}

	for _, cert := range defaultCertificates() {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO certificates (id, owner_type, owner_id, cert_type, provider, status, not_before, not_after, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			cert.ID, cert.OwnerType, cert.OwnerID, cert.CertType, cert.Provider, cert.Status, cert.NotBefore, cert.NotAfter, now, now,
		); err != nil {
			return err
		}
	}

	for _, node := range defaultNodes() {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO node_policy_assignments (node_id, policy_revision_id, snapshot_json, assigned_at) VALUES (?, ?, ?, ?)`,
			node.ID, "policy-rev-0007", `{"seed":true}`, now,
		); err != nil {
			return err
		}
	}

	return nil
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

func (s *MySQLStore) CreateNode(input domain.CreateNodeInput) (domain.Node, error) {
	item := domain.Node{
		ID:           newID("node"),
		Name:         input.Name,
		Mode:         input.Mode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		Enabled:      true,
		Status:       "degraded",
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
	}
	now := nowRFC3339()
	_, err := s.db.Exec(
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
	if _, err := tx.Exec("UPDATE nodes SET status = ?, updated_at = ? WHERE id = ?", "degraded", now, nodeID); err != nil {
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
	node.Status = "degraded"
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
		expiresAt  string
		consumedAt sql.NullString
	)
	err := s.db.QueryRow(
		`SELECT id, expires_at, consumed_at FROM bootstrap_tokens WHERE token_hash = ?`,
		input.Token,
	).Scan(&tokenID, &expiresAt, &consumedAt)
	if err != nil {
		return domain.EnrollNodeResult{}, err
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expiry) || consumedAt.Valid {
		return domain.EnrollNodeResult{}, fmt.Errorf("invalid bootstrap token")
	}
	node, err := s.CreateNode(domain.CreateNodeInput{
		Name:         input.Name,
		Mode:         input.Mode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
	})
	if err != nil {
		return domain.EnrollNodeResult{}, err
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
	if _, err := tx.Exec("UPDATE bootstrap_tokens SET consumed_at = ? WHERE id = ?", now, tokenID); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_trust_materials (id, node_id, material_type, material_value, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		newID("trust"), node.ID, "enrollment_secret", enrollmentSecret, "pending", now, now,
	); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	if _, err := tx.Exec("UPDATE nodes SET status = ?, updated_at = ? WHERE id = ?", "pending", now, node.ID); err != nil {
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
	if _, err := tx.Exec("UPDATE nodes SET status = ?, updated_at = ? WHERE id = ?", "degraded", now, nodeID); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`UPDATE node_trust_materials SET status = ?, updated_at = ? WHERE node_id = ?`,
		"active", now, nodeID,
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
	node.Status = "degraded"
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
		 WHERE node_id = ? AND material_type = 'enrollment_secret' AND material_value = ?`,
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
	return domain.ApproveNodeEnrollmentResult{
		Node:          node,
		AccessToken:   accessToken,
		TrustMaterial: trustValue,
		ExpiresAt:     expiresAt,
	}, nil
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
		"UPDATE nodes SET status = 'degraded', updated_at = ? WHERE id NOT IN (SELECT node_id FROM node_health_snapshots)",
		nowRFC3339(),
	); err != nil {
		return err
	}
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
	_, err := s.db.Exec(
		`INSERT INTO node_health_snapshots (node_id, heartbeat_at, policy_revision_id, listener_status_json, cert_status_json, updated_at)
		 VALUES (?, ?, NULLIF(?, ''), ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   heartbeat_at = VALUES(heartbeat_at),
		   policy_revision_id = VALUES(policy_revision_id),
		   listener_status_json = VALUES(listener_status_json),
		   cert_status_json = VALUES(cert_status_json),
		   updated_at = VALUES(updated_at)`,
		input.NodeID, now, input.PolicyRevisionID, encodeJSONMap(input.ListenerStatus), encodeJSONMap(input.CertStatus), now,
	)
	if err != nil {
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

package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (s *MySQLStore) CreateBusinessAuditEvent(input domain.CreateBusinessAuditEventInput) (domain.BusinessAuditEvent, error) {
	id, err := s.nextID("audit_event")
	if err != nil {
		return domain.BusinessAuditEvent{}, err
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}
	if input.Outcome == "" {
		input.Outcome = domain.AuditOutcomeSuccess
	}
	if input.BeforeJSON == "" {
		input.BeforeJSON = "{}"
	}
	if input.AfterJSON == "" {
		input.AfterJSON = "{}"
	}
	if input.MetadataJSON == "" {
		input.MetadataJSON = "{}"
	}
	item := domain.BusinessAuditEvent{
		ID:           id,
		TenantID:     input.TenantID,
		OccurredAt:   input.OccurredAt.UTC(),
		ActorType:    input.ActorType,
		ActorID:      input.ActorID,
		ActorName:    input.ActorName,
		ActorIP:      input.ActorIP,
		ActorAgent:   input.ActorAgent,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		ResourceName: input.ResourceName,
		Outcome:      input.Outcome,
		Reason:       input.Reason,
		RequestID:    input.RequestID,
		BeforeJSON:   input.BeforeJSON,
		AfterJSON:    input.AfterJSON,
		MetadataJSON: input.MetadataJSON,
	}
	_, err = s.db.Exec(
		`INSERT INTO business_audit_events
		 (id, tenant_id, occurred_at, actor_type, actor_id, actor_name, actor_ip, actor_agent, action, resource_type, resource_id, resource_name, outcome, reason, request_id, before_json, after_json, metadata_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TenantID, formatAuditTime(item.OccurredAt), item.ActorType, item.ActorID, item.ActorName, item.ActorIP, item.ActorAgent,
		item.Action, item.ResourceType, item.ResourceID, item.ResourceName, item.Outcome, item.Reason, item.RequestID, item.BeforeJSON, item.AfterJSON, item.MetadataJSON,
	)
	return item, err
}

func (s *MySQLStore) ListBusinessAuditEvents(query domain.BusinessAuditQuery) (domain.BusinessAuditEventsResult, error) {
	limit := auditLimit(query.Limit)
	where, args := businessAuditWhere(query)
	rows, err := s.db.Query(
		`SELECT id, tenant_id, occurred_at, actor_type, actor_id, actor_name, actor_ip, actor_agent, action, resource_type, resource_id, resource_name, outcome, reason, request_id, before_json, after_json, metadata_json
		 FROM business_audit_events`+where+`
		 ORDER BY occurred_at DESC, id DESC
		 LIMIT ?`, append(args, limit)...,
	)
	if err != nil {
		return domain.BusinessAuditEventsResult{}, err
	}
	defer rows.Close()
	items := make([]domain.BusinessAuditEvent, 0)
	for rows.Next() {
		item, err := scanBusinessAuditEvent(rows)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	summary, err := s.businessAuditSummary(where, args)
	if err != nil {
		return domain.BusinessAuditEventsResult{}, err
	}
	return domain.BusinessAuditEventsResult{Items: items, Summary: summary}, nil
}

func (s *MySQLStore) CreateNetworkAuditSession(input domain.CreateNetworkAuditSessionInput) (domain.NetworkAuditSession, error) {
	id := input.ID
	var err error
	if id == "" {
		id, err = s.nextID("audit_session")
		if err != nil {
			return domain.NetworkAuditSession{}, err
		}
	}
	now := time.Now().UTC()
	if input.StartedAt.IsZero() {
		input.StartedAt = now
	}
	if input.EndedAt.IsZero() {
		input.EndedAt = input.StartedAt
	}
	if input.ReceivedAt.IsZero() {
		input.ReceivedAt = now
	}
	if input.Decision == "" {
		input.Decision = domain.NetworkDecisionAllow
	}
	if input.GovernanceMode == "" {
		input.GovernanceMode = "enforce"
	}
	if input.MatchedRuleID == "" {
		input.MatchedRuleID = input.RouteID
	}
	if input.DecisionSource == "" {
		if input.MatchedRuleID != "" {
			input.DecisionSource = "policy"
		} else {
			input.DecisionSource = "unknown"
		}
	}
	if input.MetadataJSON == "" {
		input.MetadataJSON = "{}"
	}
	input.MetadataJSON = networkAuditMetadataJSON(input.MetadataJSON, input.CacheStatus, input.CacheStoredAt)
	item := domain.NetworkAuditSession{
		ID:                 id,
		TenantID:           input.TenantID,
		StartedAt:          input.StartedAt.UTC(),
		EndedAt:            input.EndedAt.UTC(),
		ActorType:          input.ActorType,
		ActorID:            input.ActorID,
		TokenID:            input.TokenID,
		SourceIP:           input.SourceIP,
		EntryNodeID:        input.EntryNodeID,
		ExitNodeID:         input.ExitNodeID,
		TargetHost:         strings.ToLower(strings.TrimSpace(input.TargetHost)),
		TargetPort:         input.TargetPort,
		Scheme:             strings.ToLower(strings.TrimSpace(input.Scheme)),
		Method:             strings.ToUpper(strings.TrimSpace(input.Method)),
		RouteID:            input.RouteID,
		ScopeID:            input.ScopeID,
		ChainID:            input.ChainID,
		GovernanceMode:     input.GovernanceMode,
		PolicyRevision:     input.PolicyRevision,
		MatchedRuleID:      input.MatchedRuleID,
		MatchedRuleType:    input.MatchedRuleType,
		MatchedRulePattern: input.MatchedRulePattern,
		MatchedAction:      input.MatchedAction,
		DecisionSource:     input.DecisionSource,
		Decision:           input.Decision,
		DenyReason:         input.DenyReason,
		BytesIn:            input.BytesIn,
		BytesOut:           input.BytesOut,
		DurationMs:         input.DurationMs,
		StatusCode:         input.StatusCode,
		ErrorCode:          input.ErrorCode,
		CacheStatus:        input.CacheStatus,
		CacheStoredAt:      auditTimePtr(input.CacheStoredAt),
		ReceivedAt:         input.ReceivedAt.UTC(),
		MetadataJSON:       input.MetadataJSON,
	}
	_, err = s.db.Exec(
		`INSERT INTO network_audit_sessions
		 (id, tenant_id, started_at, ended_at, actor_type, actor_id, token_id, source_ip, entry_node_id, exit_node_id, target_host, target_port, scheme, method, route_id, scope_id, chain_id, governance_mode, policy_revision, matched_rule_id, matched_rule_type, matched_rule_pattern, matched_action, decision_source, decision, deny_reason, bytes_in, bytes_out, duration_ms, status_code, error_code, received_at, metadata_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   tenant_id = VALUES(tenant_id),
		   started_at = VALUES(started_at),
		   ended_at = VALUES(ended_at),
		   actor_type = VALUES(actor_type),
		   actor_id = VALUES(actor_id),
		   token_id = VALUES(token_id),
		   source_ip = VALUES(source_ip),
		   entry_node_id = VALUES(entry_node_id),
		   exit_node_id = VALUES(exit_node_id),
		   target_host = VALUES(target_host),
		   target_port = VALUES(target_port),
		   scheme = VALUES(scheme),
		   method = VALUES(method),
		   route_id = VALUES(route_id),
		   scope_id = VALUES(scope_id),
		   chain_id = VALUES(chain_id),
		   governance_mode = VALUES(governance_mode),
		   policy_revision = VALUES(policy_revision),
		   matched_rule_id = VALUES(matched_rule_id),
		   matched_rule_type = VALUES(matched_rule_type),
		   matched_rule_pattern = VALUES(matched_rule_pattern),
		   matched_action = VALUES(matched_action),
		   decision_source = VALUES(decision_source),
		   decision = VALUES(decision),
		   deny_reason = VALUES(deny_reason),
		   bytes_in = VALUES(bytes_in),
		   bytes_out = VALUES(bytes_out),
		   duration_ms = VALUES(duration_ms),
		   status_code = VALUES(status_code),
		   error_code = VALUES(error_code),
		   received_at = VALUES(received_at),
		   metadata_json = VALUES(metadata_json)`,
		item.ID, item.TenantID, formatAuditTime(item.StartedAt), formatAuditTime(item.EndedAt), item.ActorType, item.ActorID, item.TokenID, item.SourceIP,
		item.EntryNodeID, item.ExitNodeID, item.TargetHost, item.TargetPort, item.Scheme, item.Method, item.RouteID, item.ScopeID, item.ChainID,
		item.GovernanceMode, item.PolicyRevision, item.MatchedRuleID, item.MatchedRuleType, item.MatchedRulePattern, item.MatchedAction, item.DecisionSource,
		item.Decision, item.DenyReason, item.BytesIn, item.BytesOut, item.DurationMs, item.StatusCode, item.ErrorCode, formatAuditTime(item.ReceivedAt), item.MetadataJSON,
	)
	return item, err
}

func (s *MySQLStore) ListNetworkAuditSessions(query domain.NetworkAuditQuery) (domain.NetworkAuditSessionsResult, error) {
	limit := auditLimit(query.Limit)
	where, args := networkAuditWhere(query)
	rows, err := s.db.Query(
		`SELECT id, tenant_id, started_at, ended_at, actor_type, actor_id, token_id, source_ip, entry_node_id, exit_node_id, target_host, target_port, scheme, method, route_id, scope_id, chain_id, governance_mode, policy_revision, matched_rule_id, matched_rule_type, matched_rule_pattern, matched_action, decision_source, decision, deny_reason, bytes_in, bytes_out, duration_ms, status_code, error_code, received_at, metadata_json
		 FROM network_audit_sessions`+where+`
		 ORDER BY ended_at DESC, id DESC
		 LIMIT ?`, append(args, limit)...,
	)
	if err != nil {
		return domain.NetworkAuditSessionsResult{}, err
	}
	defer rows.Close()
	items := make([]domain.NetworkAuditSession, 0)
	for rows.Next() {
		item, err := scanNetworkAuditSession(rows)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	summary, err := s.networkAuditSummary(where, args)
	if err != nil {
		return domain.NetworkAuditSessionsResult{}, err
	}
	return domain.NetworkAuditSessionsResult{Items: items, Summary: summary}, nil
}

func (s *MySQLStore) GetAuditDashboard(query domain.AuditDashboardQuery) (domain.AuditDashboard, error) {
	summary, err := s.networkAuditSummary(networkDashboardWhere(query))
	if err != nil {
		return domain.AuditDashboard{}, err
	}
	recent, err := s.ListBusinessAuditEvents(domain.BusinessAuditQuery{
		TenantID: query.TenantID,
		From:     query.From,
		To:       query.To,
		Limit:    20,
	})
	if err != nil {
		return domain.AuditDashboard{}, err
	}
	summary.RecentBusiness = recent.Items
	return domain.AuditDashboard{NetworkAuditSummary: summary}, nil
}

func businessAuditWhere(query domain.BusinessAuditQuery) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	addStringFilter(&clauses, &args, "tenant_id", query.TenantID)
	addStringFilter(&clauses, &args, "actor_id", query.ActorID)
	addStringFilter(&clauses, &args, "actor_type", query.ActorType)
	addStringFilter(&clauses, &args, "resource_type", query.ResourceType)
	addStringFilter(&clauses, &args, "resource_id", query.ResourceID)
	addStringFilter(&clauses, &args, "action", query.Action)
	addStringFilter(&clauses, &args, "outcome", query.Outcome)
	addTimeFilter(&clauses, &args, "occurred_at", query.From, query.To)
	return auditWhere(clauses), args
}

func networkAuditWhere(query domain.NetworkAuditQuery) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	addStringFilter(&clauses, &args, "tenant_id", query.TenantID)
	addStringFilter(&clauses, &args, "actor_id", query.ActorID)
	addStringFilter(&clauses, &args, "token_id", query.TokenID)
	if query.NodeID != "" {
		clauses = append(clauses, "(entry_node_id = ? OR exit_node_id = ?)")
		args = append(args, query.NodeID, query.NodeID)
	}
	addStringFilter(&clauses, &args, "target_host", strings.ToLower(strings.TrimSpace(query.TargetHost)))
	addStringFilter(&clauses, &args, "route_id", query.RouteID)
	addStringFilter(&clauses, &args, "scope_id", query.ScopeID)
	addStringFilter(&clauses, &args, "chain_id", query.ChainID)
	addStringFilter(&clauses, &args, "deny_reason", query.DenyReason)
	addStringFilter(&clauses, &args, "error_code", query.ErrorCode)
	addStringFilter(&clauses, &args, "policy_revision", query.PolicyRevision)
	addStringFilter(&clauses, &args, "matched_rule_id", query.MatchedRuleID)
	addStringFilter(&clauses, &args, "decision_source", query.DecisionSource)
	addStringFilter(&clauses, &args, "decision", query.Decision)
	addTimeFilter(&clauses, &args, "ended_at", query.From, query.To)
	return auditWhere(clauses), args
}

func networkDashboardWhere(query domain.AuditDashboardQuery) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	addStringFilter(&clauses, &args, "tenant_id", query.TenantID)
	addTimeFilter(&clauses, &args, "ended_at", query.From, query.To)
	return auditWhere(clauses), args
}

func auditWhere(clauses []string) string {
	if len(clauses) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(clauses, " AND ")
}

func addStringFilter(clauses *[]string, args *[]any, column string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	*clauses = append(*clauses, column+" = ?")
	*args = append(*args, value)
}

func addTimeFilter(clauses *[]string, args *[]any, column string, from time.Time, to time.Time) {
	if !from.IsZero() {
		*clauses = append(*clauses, column+" >= ?")
		*args = append(*args, formatAuditTime(from))
	}
	if !to.IsZero() {
		*clauses = append(*clauses, column+" <= ?")
		*args = append(*args, formatAuditTime(to))
	}
}

func (s *MySQLStore) businessAuditSummary(where string, args []any) (domain.BusinessAuditSummary, error) {
	summary := domain.BusinessAuditSummary{
		OutcomeCount: map[string]int64{},
		ActionCount:  map[string]int64{},
		ResourceType: map[string]int64{},
		ActorCount:   []domain.AuditActorCount{},
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM business_audit_events`+where, args...).Scan(&summary.Total); err != nil {
		return summary, err
	}
	outcomes, err := s.countBy(where, args, "business_audit_events", "outcome", 20)
	if err != nil {
		return summary, err
	}
	summary.OutcomeCount = outcomes
	actions, err := s.countBy(where, args, "business_audit_events", "action", 30)
	if err != nil {
		return summary, err
	}
	summary.ActionCount = actions
	resourceTypes, err := s.countBy(where, args, "business_audit_events", "resource_type", 30)
	if err != nil {
		return summary, err
	}
	summary.ResourceType = resourceTypes
	rows, err := s.db.Query(
		`SELECT actor_type, actor_id, actor_name, COUNT(*)
		 FROM business_audit_events`+where+`
		 GROUP BY actor_type, actor_id, actor_name
		 ORDER BY COUNT(*) DESC
		 LIMIT 20`, args...,
	)
	if err != nil {
		return summary, err
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.AuditActorCount
		if err := rows.Scan(&item.ActorType, &item.ActorID, &item.ActorName, &item.Count); err != nil {
			continue
		}
		summary.ActorCount = append(summary.ActorCount, item)
	}
	return summary, nil
}

func (s *MySQLStore) networkAuditSummary(where string, args []any) (domain.NetworkAuditSummary, error) {
	summary := domain.NetworkAuditSummary{
		DecisionCount:   map[string]int64{},
		DenyReasonCount: map[string]int64{},
		ErrorCodeCount:  map[string]int64{},
		TopTargets:      []domain.AuditTargetTraffic{},
		ScenarioTraffic: []domain.AuditScenarioTraffic{},
		UserTraffic:     []domain.AuditActorTraffic{},
		NodeTraffic:     []domain.AuditNodeTraffic{},
		TenantTraffic:   []domain.AuditTenantTraffic{},
		RecentBusiness:  []domain.BusinessAuditEvent{},
	}
	if err := s.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(bytes_in), 0), COALESCE(SUM(bytes_out), 0), CAST(COALESCE(AVG(duration_ms), 0) AS SIGNED)
		 FROM network_audit_sessions`+where,
		args...,
	).Scan(&summary.Total, &summary.BytesIn, &summary.BytesOut, &summary.DurationAvgMs); err != nil {
		return summary, err
	}
	decisions, err := s.countBy(where, args, "network_audit_sessions", "decision", 20)
	if err != nil {
		return summary, err
	}
	summary.DecisionCount = decisions
	denyReasons, err := s.countBy(where, args, "network_audit_sessions", "deny_reason", 20)
	if err != nil {
		return summary, err
	}
	summary.DenyReasonCount = denyReasons
	errorCodes, err := s.countBy(where, args, "network_audit_sessions", "error_code", 50)
	if err != nil {
		return summary, err
	}
	delete(errorCodes, "")
	summary.ErrorCodeCount = errorCodes
	if err := s.queryTrafficRows(&summary, where, args); err != nil {
		return summary, err
	}
	return summary, nil
}

func (s *MySQLStore) countBy(where string, args []any, table string, column string, limit int) (map[string]int64, error) {
	rows, err := s.db.Query(
		`SELECT `+column+`, COUNT(*)
		 FROM `+table+where+`
		 GROUP BY `+column+`
		 ORDER BY COUNT(*) DESC
		 LIMIT ?`, append(args, limit)...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]int64{}
	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			continue
		}
		result[key] = count
	}
	return result, nil
}

func (s *MySQLStore) queryTrafficRows(summary *domain.NetworkAuditSummary, where string, args []any) error {
	targets, err := s.db.Query(
		`SELECT target_host, COALESCE(SUM(bytes_in), 0), COALESCE(SUM(bytes_out), 0), COUNT(*)
		 FROM network_audit_sessions`+where+`
		 GROUP BY target_host
		 ORDER BY COALESCE(SUM(bytes_in), 0) + COALESCE(SUM(bytes_out), 0) DESC
		 LIMIT 20`, args...,
	)
	if err != nil {
		return err
	}
	defer targets.Close()
	for targets.Next() {
		var item domain.AuditTargetTraffic
		if err := targets.Scan(&item.TargetHost, &item.BytesIn, &item.BytesOut, &item.Count); err != nil {
			continue
		}
		summary.TopTargets = append(summary.TopTargets, item)
	}

	scenarios, err := s.db.Query(
		`SELECT
		   COALESCE(NULLIF(route_id, ''), NULLIF(chain_id, ''), target_host) AS scenario_id,
		   COALESCE(NULLIF(matched_rule_pattern, ''), NULLIF(matched_rule_id, ''), NULLIF(chain_id, ''), target_host) AS scenario_name,
		   COALESCE(SUM(bytes_in), 0),
		   COALESCE(SUM(bytes_out), 0),
		   COUNT(*),
		   COALESCE(SUM(CASE WHEN decision = 'deny' OR error_code <> '' THEN 1 ELSE 0 END), 0)
		 FROM network_audit_sessions`+where+`
		 GROUP BY scenario_id, scenario_name
		 ORDER BY COALESCE(SUM(bytes_in), 0) + COALESCE(SUM(bytes_out), 0) DESC
		 LIMIT 20`, args...,
	)
	if err != nil {
		return err
	}
	defer scenarios.Close()
	for scenarios.Next() {
		var item domain.AuditScenarioTraffic
		if err := scenarios.Scan(&item.ScenarioID, &item.ScenarioName, &item.BytesIn, &item.BytesOut, &item.Count, &item.FailureCount); err != nil {
			continue
		}
		item.ErrorCodes = map[string]int64{}
		summary.ScenarioTraffic = append(summary.ScenarioTraffic, item)
	}
	for i := range summary.ScenarioTraffic {
		codes, err := s.errorCodesForScenario(where, args, summary.ScenarioTraffic[i].ScenarioID)
		if err != nil {
			continue
		}
		summary.ScenarioTraffic[i].ErrorCodes = codes
	}

	actors, err := s.db.Query(
		`SELECT actor_id, COALESCE(SUM(bytes_in), 0), COALESCE(SUM(bytes_out), 0), COUNT(*)
		 FROM network_audit_sessions`+where+`
		 GROUP BY actor_id
		 ORDER BY COALESCE(SUM(bytes_in), 0) + COALESCE(SUM(bytes_out), 0) DESC
		 LIMIT 20`, args...,
	)
	if err != nil {
		return err
	}
	defer actors.Close()
	for actors.Next() {
		var item domain.AuditActorTraffic
		if err := actors.Scan(&item.ActorID, &item.BytesIn, &item.BytesOut, &item.Count); err != nil {
			continue
		}
		summary.UserTraffic = append(summary.UserTraffic, item)
	}

	nodes, err := s.db.Query(
		`SELECT entry_node_id, COALESCE(SUM(bytes_in), 0), COALESCE(SUM(bytes_out), 0), COUNT(*)
		 FROM network_audit_sessions`+where+`
		 GROUP BY entry_node_id
		 ORDER BY COALESCE(SUM(bytes_in), 0) + COALESCE(SUM(bytes_out), 0) DESC
		 LIMIT 20`, args...,
	)
	if err != nil {
		return err
	}
	defer nodes.Close()
	for nodes.Next() {
		var item domain.AuditNodeTraffic
		if err := nodes.Scan(&item.NodeID, &item.BytesIn, &item.BytesOut, &item.Count); err != nil {
			continue
		}
		summary.NodeTraffic = append(summary.NodeTraffic, item)
	}

	tenants, err := s.db.Query(
		`SELECT tenant_id, COALESCE(SUM(bytes_in), 0), COALESCE(SUM(bytes_out), 0), COUNT(*)
		 FROM network_audit_sessions`+where+`
		 GROUP BY tenant_id
		 ORDER BY COALESCE(SUM(bytes_in), 0) + COALESCE(SUM(bytes_out), 0) DESC
		 LIMIT 20`, args...,
	)
	if err != nil {
		return err
	}
	defer tenants.Close()
	for tenants.Next() {
		var item domain.AuditTenantTraffic
		if err := tenants.Scan(&item.TenantID, &item.BytesIn, &item.BytesOut, &item.Count); err != nil {
			continue
		}
		summary.TenantTraffic = append(summary.TenantTraffic, item)
	}
	return nil
}

type businessAuditScanner interface {
	Scan(dest ...any) error
}

func scanBusinessAuditEvent(row businessAuditScanner) (domain.BusinessAuditEvent, error) {
	var item domain.BusinessAuditEvent
	var occurredAt string
	err := row.Scan(&item.ID, &item.TenantID, &occurredAt, &item.ActorType, &item.ActorID, &item.ActorName, &item.ActorIP, &item.ActorAgent, &item.Action, &item.ResourceType, &item.ResourceID, &item.ResourceName, &item.Outcome, &item.Reason, &item.RequestID, &item.BeforeJSON, &item.AfterJSON, &item.MetadataJSON)
	item.OccurredAt = parseAuditTime(occurredAt)
	return item, err
}

type networkAuditScanner interface {
	Scan(dest ...any) error
}

func scanNetworkAuditSession(row networkAuditScanner) (domain.NetworkAuditSession, error) {
	var item domain.NetworkAuditSession
	var startedAt string
	var endedAt string
	var receivedAt string
	err := row.Scan(&item.ID, &item.TenantID, &startedAt, &endedAt, &item.ActorType, &item.ActorID, &item.TokenID, &item.SourceIP, &item.EntryNodeID, &item.ExitNodeID, &item.TargetHost, &item.TargetPort, &item.Scheme, &item.Method, &item.RouteID, &item.ScopeID, &item.ChainID, &item.GovernanceMode, &item.PolicyRevision, &item.MatchedRuleID, &item.MatchedRuleType, &item.MatchedRulePattern, &item.MatchedAction, &item.DecisionSource, &item.Decision, &item.DenyReason, &item.BytesIn, &item.BytesOut, &item.DurationMs, &item.StatusCode, &item.ErrorCode, &receivedAt, &item.MetadataJSON)
	item.StartedAt = parseAuditTime(startedAt)
	item.EndedAt = parseAuditTime(endedAt)
	item.ReceivedAt = parseAuditTime(receivedAt)
	item.CacheStatus, item.CacheStoredAt = networkAuditCacheMetadata(item.MetadataJSON)
	return item, err
}

func (s *MySQLStore) errorCodesForScenario(where string, args []any, scenarioID string) (map[string]int64, error) {
	if scenarioID == "" {
		return map[string]int64{}, nil
	}
	scenarioWhere := where
	scenarioArgs := append([]any(nil), args...)
	condition := "COALESCE(NULLIF(route_id, ''), NULLIF(chain_id, ''), target_host) = ? AND error_code <> ''"
	if scenarioWhere == "" {
		scenarioWhere = " WHERE " + condition
	} else {
		scenarioWhere += " AND " + condition
	}
	scenarioArgs = append(scenarioArgs, scenarioID)
	return s.countBy(scenarioWhere, scenarioArgs, "network_audit_sessions", "error_code", 20)
}

func networkAuditMetadataJSON(raw string, cacheStatus string, cacheStoredAt time.Time) string {
	metadata := map[string]any{}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &metadata)
	}
	if cacheStatus != "" {
		metadata["cacheStatus"] = cacheStatus
	}
	if !cacheStoredAt.IsZero() {
		metadata["cacheStoredAt"] = cacheStoredAt.UTC().Format(time.RFC3339)
	}
	body, err := json.Marshal(metadata)
	if err != nil || len(body) == 0 {
		return "{}"
	}
	return string(body)
}

func networkAuditCacheMetadata(raw string) (string, *time.Time) {
	metadata := map[string]any{}
	if raw == "" || json.Unmarshal([]byte(raw), &metadata) != nil {
		return "", nil
	}
	cacheStatus, _ := metadata["cacheStatus"].(string)
	cacheStoredAtRaw, _ := metadata["cacheStoredAt"].(string)
	return cacheStatus, auditTimePtr(parseAuditTime(cacheStoredAtRaw))
}

func auditTimePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func auditLimit(limit int) int {
	if limit <= 0 {
		return 100
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func formatAuditTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func parseAuditTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

var _ businessAuditScanner = (*sql.Row)(nil)
var _ networkAuditScanner = (*sql.Row)(nil)

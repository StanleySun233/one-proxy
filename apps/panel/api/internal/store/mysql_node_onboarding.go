package store

import (
	"database/sql"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

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
	taskID, err := s.nextID("onboarding_task")
	if err != nil {
		return domain.NodeOnboardingTask{}, err
	}
	item := domain.NodeOnboardingTask{
		ID:                   taskID,
		Mode:                 input.Mode,
		PathID:               input.PathID,
		TargetNodeID:         input.TargetNodeID,
		TargetHost:           input.TargetHost,
		TargetPort:           input.TargetPort,
		Status:               domain.TaskStatusPlanned,
		StatusMessage:        "task created",
		RequestedByAccountID: accountID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	_, err = s.db.Exec(
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

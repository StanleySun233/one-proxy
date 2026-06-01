package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *SeedStore) ListNodeOnboardingTasks() []domain.NodeOnboardingTask {
	return []domain.NodeOnboardingTask{}
}

func (s *SeedStore) CreateNodeOnboardingTask(accountID string, input domain.CreateNodeOnboardingTaskInput) (domain.NodeOnboardingTask, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	return domain.NodeOnboardingTask{
		ID:                   s.nextID("onboarding_task"),
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
	}, nil
}

func (s *SeedStore) UpdateNodeOnboardingTaskStatus(taskID string, status string, statusMessage string) (domain.NodeOnboardingTask, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	return domain.NodeOnboardingTask{
		ID:            taskID,
		Status:        status,
		StatusMessage: statusMessage,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

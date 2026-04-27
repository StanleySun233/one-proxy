package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (r *Router) handleNodeOnboardingTasks(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.NodeOnboardingTasks())
	case http.MethodPost:
		account, ok := accountFromContext(req.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid_access_token")
			return
		}
		var payload domain.CreateNodeOnboardingTaskInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateNodeOnboardingTask(account.ID, payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleNodeOnboardingTaskByID(w http.ResponseWriter, req *http.Request) {
	taskID := resourceID(req.URL.Path, "/api/v1/node-onboarding-tasks/")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "missing_task_id")
		return
	}
	if req.Method != http.MethodPatch {
		writeMethodNotAllowed(w, "PATCH")
		return
	}
	var payload domain.UpdateNodeOnboardingTaskStatusInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	item, err := r.service.UpdateNodeOnboardingTaskStatus(taskID, payload)
	if err != nil {
		writeServiceError(w, req, err, "update_failed")
		return
	}
	writeSuccess(w, http.StatusOK, item)
}

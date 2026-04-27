package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func resourceID(path string, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}

func (r *Router) handleOverview(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.Overview())
}

func (r *Router) handleExtensionBootstrap(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.ExtensionBootstrap(account))
}

func (r *Router) handleCertificates(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.Certificates())
}

func (r *Router) handleEnums(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	field := req.URL.Query().Get("field")
	var items []domain.FieldEnum
	var err error
	if field != "" {
		items, err = r.service.ListFieldEnumsByField(field)
	} else {
		items, err = r.service.ListFieldEnums()
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_enums_failed")
		return
	}
	type enumEntry struct {
		Name string      `json:"name"`
		Meta interface{} `json:"meta,omitempty"`
	}
	grouped := make(map[string]map[string]enumEntry)
	for _, item := range items {
		if _, ok := grouped[item.Field]; !ok {
			grouped[item.Field] = make(map[string]enumEntry)
		}
		entry := enumEntry{Name: item.Name}
		if item.Meta != nil && *item.Meta != "" && *item.Meta != "{}" {
			var meta interface{}
			if err := json.Unmarshal([]byte(*item.Meta), &meta); err == nil {
				entry.Meta = meta
			}
		}
		grouped[item.Field][item.Value] = entry
	}
	if field != "" {
		if m, ok := grouped[field]; ok {
			writeSuccess(w, http.StatusOK, m)
		} else {
			writeSuccess(w, http.StatusOK, map[string]enumEntry{})
		}
		return
	}
	writeSuccess(w, http.StatusOK, grouped)
}

func (r *Router) handlePolicyRevisions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.PolicyRevisions())
}

func (r *Router) handlePolicyPublish(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	item, err := r.service.PublishPolicy(account.ID)
	if err != nil {
		writeServiceError(w, req, err, "publish_failed")
		return
	}
	writeSuccess(w, http.StatusOK, item)
}

func (r *Router) handleNodeScopes(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.NodeScopes())
}

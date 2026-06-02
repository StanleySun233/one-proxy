package linkservice

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/store"
)

type Service struct {
	store store.Store
}

type Error struct {
	Status  int
	Code    string
	Message string
}

func New(store store.Store) *Service {
	return &Service{store: store}
}

func (e *Error) Error() string {
	return e.Message
}

func invalidInput(code string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: code, Message: code}
}

func (s *Service) isValidEnum(field, value string) bool {
	items, err := s.store.ListFieldEnumsByField(field)
	if err != nil {
		return true
	}
	for _, item := range items {
		if item.Value == value {
			return true
		}
	}
	return false
}

func nodeByID(items []domain.Node, nodeID string) (domain.Node, bool) {
	for _, item := range items {
		if item.ID == nodeID {
			return item, true
		}
	}
	return domain.Node{}, false
}

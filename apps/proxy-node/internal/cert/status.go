package cert

import "sync"

type StatusReporter struct {
	mu     sync.RWMutex
	values map[string]string
}

func NewStatusReporter(initial map[string]string) *StatusReporter {
	cloned := make(map[string]string, len(initial))
	for key, value := range initial {
		cloned[key] = value
	}
	return &StatusReporter{values: cloned}
}

func (r *StatusReporter) Set(certType string, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[certType] = status
}

func (r *StatusReporter) Snapshot() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cloned := make(map[string]string, len(r.values))
	for key, value := range r.values {
		cloned[key] = value
	}
	return cloned
}

package store

import (
	"encoding/json"
)

func decodeJSONMap(raw string) map[string]string {
	if raw == "" {
		return map[string]string{}
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return map[string]string{}
	}
	return result
}

func encodeJSONMap(value map[string]string) string {
	if value == nil {
		return "{}"
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func decodeJSONStringSlice(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return []string{}
	}
	return result
}

func encodeJSONStringSlice(value []string) string {
	value = normalizeStringSlice(value)
	raw, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func normalizeStringSlice(value []string) []string {
	if value == nil {
		return []string{}
	}
	return value
}

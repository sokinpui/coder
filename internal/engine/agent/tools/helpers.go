package tools

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func parseDuration(val any) time.Duration {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case string:
		d, err := time.ParseDuration(strings.TrimSpace(v))
		if err == nil {
			return d
		}
		if sec, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return time.Duration(sec) * time.Second
		}
	case int:
		return time.Duration(v) * time.Second
	case int64:
		return time.Duration(v) * time.Second
	case float64:
		return time.Duration(v) * time.Second
	}
	return 0
}

func parseInt(val any) int {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(v))
		return i
	}
	return 0
}

func parseInt64(val any) int64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		i, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return i
	}
	return 0
}

func parseString(val any) string {
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func extractJSONField(arguments, field string) string {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return ""
	}

	var rawString string
	if err := json.Unmarshal([]byte(trimmed), &rawString); err == nil && strings.HasPrefix(strings.TrimSpace(rawString), "{") {
		trimmed = strings.TrimSpace(rawString)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(trimmed), &data); err != nil {
		if !strings.HasPrefix(trimmed, "{") {
			return trimmed
		}
		return ""
	}

	if val, ok := data[field].(string); ok {
		return strings.TrimSpace(val)
	}
	return ""
}

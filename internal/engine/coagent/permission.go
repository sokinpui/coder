package coagent

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
)

type PermissionAction string

const (
	ActionAllow PermissionAction = "allow"
	ActionAsk   PermissionAction = "ask"
	ActionDeny  PermissionAction = "deny"
)

type PermissionManager struct {
	mu            sync.RWMutex
	config        map[string]any
	alwaysAllowed map[string]bool
}

func NewPermissionManager(config map[string]any) *PermissionManager {
	return &PermissionManager{
		config:        config,
		alwaysAllowed: make(map[string]bool),
	}
}

func (pm *PermissionManager) SetConfig(config map[string]any) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.config = config
}

func (pm *PermissionManager) AlwaysAllow(toolName string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.alwaysAllowed[toolName] = true
}

func (pm *PermissionManager) Check(toolName, arguments string) PermissionAction {
	pm.mu.RLock()
	if pm.alwaysAllowed[toolName] {
		pm.mu.RUnlock()
		return ActionAllow
	}
	cfg := pm.config
	pm.mu.RUnlock()

	if len(cfg) == 0 {
		return ActionAllow
	}

	target := extractTarget(toolName, arguments)

	if toolRule, exists := cfg[toolName]; exists {
		if action, matched := pm.evaluateRule(toolRule, target); matched {
			return action
		}
	}

	if globalRule, exists := cfg["*"]; exists {
		if action, matched := pm.evaluateRule(globalRule, target); matched {
			return action
		}
	}

	return ActionAllow
}

func (pm *PermissionManager) evaluateRule(rule any, target string) (PermissionAction, bool) {
	switch v := rule.(type) {
	case string:
		return parseAction(v), true
	case map[string]any:
		return pm.evaluateMap(v, target)
	case map[string]string:
		converted := make(map[string]any, len(v))
		for k, val := range v {
			converted[k] = val
		}
		return pm.evaluateMap(converted, target)
	default:
		return ActionAllow, false
	}
}

func (pm *PermissionManager) evaluateMap(rules map[string]any, target string) (PermissionAction, bool) {
	if val, ok := rules[target]; ok {
		if str, ok := val.(string); ok {
			return parseAction(str), true
		}
	}

	var bestPattern string
	var bestAction PermissionAction

	for pattern, val := range rules {
		if pattern == "*" || pattern == target {
			continue
		}
		str, ok := val.(string)
		if !ok {
			continue
		}
		if matchWildcard(pattern, target) && len(pattern) > len(bestPattern) {
			bestPattern = pattern
			bestAction = parseAction(str)
		}
	}

	if bestPattern != "" {
		return bestAction, true
	}

	if starVal, ok := rules["*"]; ok {
		if str, ok := starVal.(string); ok {
			return parseAction(str), true
		}
	}

	return ActionAllow, false
}

func parseAction(val string) PermissionAction {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "ask":
		return ActionAsk
	case "deny":
		return ActionDeny
	case "allow":
		return ActionAllow
	default:
		return ActionAllow
	}
}

func matchWildcard(pattern, target string) bool {
	if pattern == "*" || pattern == target {
		return true
	}

	if matched, err := filepath.Match(pattern, target); err == nil && matched {
		return true
	}

	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == target
	}
	if !strings.HasPrefix(target, parts[0]) {
		return false
	}
	target = target[len(parts[0]):]
	for i := 1; i < len(parts)-1; i++ {
		idx := strings.Index(target, parts[i])
		if idx == -1 {
			return false
		}
		target = target[idx+len(parts[i]):]
	}
	return strings.HasSuffix(target, parts[len(parts)-1])
}

func extractTarget(toolName, arguments string) string {
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

	if toolName == "bash" {
		if cmd, ok := data["command"].(string); ok {
			return strings.TrimSpace(cmd)
		}
	}

	if path, ok := data["path"].(string); ok {
		return strings.TrimSpace(path)
	}

	return trimmed
}

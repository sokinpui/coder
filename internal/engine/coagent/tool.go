package coagent

import (
	"context"
	"fmt"
	"sync"
)

type ToolDeclaration struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type Tool interface {
	Declaration() ToolDeclaration
	Execute(ctx context.Context, arguments string) (string, error)
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Declaration().Name] = tool
}

func (r *Registry) Declarations() []ToolDeclaration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decls := make([]ToolDeclaration, 0, len(r.tools))
	for _, t := range r.tools {
		decls = append(decls, t.Declaration())
	}
	return decls
}

func (r *Registry) Execute(ctx context.Context, name, arguments string) (string, error) {
	r.mu.RLock()
	tool, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("tool '%s' not found", name)
	}
	return tool.Execute(ctx, arguments)
}

var DefaultRegistry = NewRegistry()

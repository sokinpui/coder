package coagent

import (
	"context"
	"fmt"
	"sync"

	"github.com/sokinpui/coder/internal/types"
)

type ToolDeclaration = types.ToolDeclaration

type Tool interface {
	Declaration() ToolDeclaration
	Execute(ctx context.Context, arguments string) (string, error)
}

type ImageProducerTool interface {
	ExecuteWithImages(ctx context.Context, arguments string) (string, []types.Message, error)
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

func (r *Registry) Execute(ctx context.Context, name, arguments string) (string, []types.Message, error) {
	r.mu.RLock()
	tool, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return "", nil, fmt.Errorf("tool '%s' not found", name)
	}
	if imgTool, ok := tool.(ImageProducerTool); ok {
		return imgTool.ExecuteWithImages(ctx, arguments)
	}
	out, err := tool.Execute(ctx, arguments)
	return out, nil, err
}

var DefaultRegistry = NewRegistry()

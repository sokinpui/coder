package coder

import (
	"context"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/generation"
	"github.com/sokinpui/coder/internal/types"
)

type Runtime struct {
	Config    config.ModelConfig
	Generator *generation.Generator
}

func NewRuntime(cfg *config.Config) (*Runtime, error) {
	gen, err := generation.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Runtime{
		Config:    cfg.Coder.ModelConfig(),
		Generator: gen,
	}, nil
}

func (r *Runtime) Execute(ctx context.Context, systemInstruction string, messages []types.ChatMessage, streamChan chan<- types.StreamChunk) {
	cfg := r.Config
	r.Generator.GenerateTask(ctx, systemInstruction, messages, nil, streamChan, &cfg)
}

func (r *Runtime) GenerateTitle(ctx context.Context, prompt string) (string, error) {
	return r.Generator.GenerateTitle(ctx, prompt)
}

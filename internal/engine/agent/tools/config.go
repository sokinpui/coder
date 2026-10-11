package tools

import (
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/generation"
)

func BuildRegistry(toolsCfg map[string]any, gen *generation.Generator, secConfig *config.ModelConfig) *Registry {
	reg := DefaultRegistry.Clone()
	ConfigureRegistry(reg, toolsCfg, gen, secConfig)
	return reg
}

func ConfigureRegistry(reg *Registry, toolsCfg map[string]any, gen *generation.Generator, secConfig *config.ModelConfig) {
	if reg == nil {
		return
	}
	if gen != nil && secConfig != nil {
		reg.AttachGenerator(gen, secConfig)
	}
	if len(toolsCfg) > 0 {
		reg.Configure(toolsCfg)
	}
}

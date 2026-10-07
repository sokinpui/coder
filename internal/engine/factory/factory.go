package factory

import (
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/engine/coder"
)

func init() {
	engine.RegisterSessionFactory(NewSession)
}

func NewSession(cfg *config.Config, mode string, instruction string, contextFiles []string) (engine.EngineSession, error) {
	norm := engine.NormalizeMode(mode)
	if norm == engine.ModeAgent {
		sess, err := coagent.NewSession(cfg)
		if err != nil {
			return nil, err
		}
		if instruction != "" {
			sess.Instruction = instruction
		}
		return sess, nil
	}
	return coder.New(cfg, norm, instruction, contextFiles)
}

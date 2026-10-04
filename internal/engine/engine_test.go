package engine_test

import (
	"testing"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/engine/coder"
)

func TestEngineSessionInterfaces(t *testing.T) {
	cfg := config.DefaultConfig()

	coderSess, err := coder.New(&cfg, coder.ModeCoder, "", nil)
	if err != nil {
		t.Fatalf("coder.New failed: %v", err)
	}
	var _ engine.EngineSession = coderSess

	agentSess, err := coagent.NewSession(&cfg)
	if err != nil {
		t.Fatalf("coagent.NewSession failed: %v", err)
	}
	var _ engine.EngineSession = agentSess
}

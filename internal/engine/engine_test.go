package engine_test

import (
	"testing"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/engine/coder"
	"github.com/sokinpui/coder/internal/types"
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

	if !agentSess.Capabilities().Has(engine.CapShell) {
		t.Fatal("expected coagent session to have CapShell capability")
	}

	out, ok := agentSess.ExecuteCommand("/shell echo hello")
	if !ok {
		t.Fatal("expected /shell command execution to succeed")
	}
	if out.Type != types.ShellExecutionStarted || out.Payload != "echo hello" {
		t.Fatalf("unexpected command output: %+v", out)
	}

	messages := agentSess.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message in coagent session, got %d", len(messages))
	}
	if messages[0].Type != types.ShellCmdMessage {
		t.Fatalf("expected ShellCmdMessage, got %v", messages[0].Type)
	}
	if messages[0].Content != "/shell echo hello" {
		t.Fatalf("unexpected message content: %s", messages[0].Content)
	}
}

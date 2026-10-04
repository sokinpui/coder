package engine_test

import (
	"testing"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/engine/coder"
	"github.com/sokinpui/coder/internal/engine/commands"
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

	out, ok = agentSess.ExecuteCommand("! echo world")
	if !ok {
		t.Fatal("expected ! command execution to succeed")
	}
	if out.Type != types.ShellExecutionStarted || out.Payload != "echo world" {
		t.Fatalf("unexpected command output for !: %+v", out)
	}

	out, ok = coderSess.ExecuteCommand("@ main.go")
	if !ok {
		t.Fatal("expected @ command execution to succeed")
	}
	if !out.IsContext {
		t.Fatalf("expected @ command to have IsContext true: %+v", out)
	}

	out, ok = agentSess.ExecuteCommand("@")
	if ok {
		t.Fatal("expected @ without args to fail in agent session")
	}

	out, ok = agentSess.ExecuteCommand("/file")
	if ok {
		t.Fatal("expected /file without args to fail in agent session")
	}

	out, ok = agentSess.ExecuteCommand("@ main.go")
	if !ok {
		t.Fatal("expected @ main.go to succeed in agent session")
	}
	if !out.IsAgentFileRead {
		t.Fatalf("expected IsAgentFileRead to be true: %+v", out)
	}
	if !coagent.DefaultTracker.HasRead("main.go") {
		t.Fatal("expected main.go to be marked as read in agent DefaultTracker")
	}

	agentMsgs := agentSess.GetMessages()
	if len(agentMsgs) < 4 {
		t.Fatalf("expected at least 4 messages in agent session, got %d", len(agentMsgs))
	}

	if commands.IsCommand("@") {
		t.Fatal("expected single @ not to be treated as a command")
	}

	out, ok = coderSess.ExecuteCommand("@")
	if ok {
		t.Fatal("expected single @ command execution to fail without arguments")
	}
	if out.Payload != "Usage: @ <paths...>" {
		t.Fatalf("unexpected output for single @: %+v", out)
	}

	out, ok = coderSess.ExecuteCommand("/file")
	if ok {
		t.Fatal("expected bare /file command execution to fail without arguments")
	}
	if out.Payload != "Usage: /file <paths...>" {
		t.Fatalf("unexpected output for bare /file: %+v", out)
	}

	if len(coderSess.GetContextFiles()) == 0 {
		t.Fatal("expected context files to be preserved after bare /file invocation")
	}

	out, ok = coderSess.ExecuteCommand("/clear_context")
	if !ok {
		t.Fatal("expected /clear_context to succeed")
	}
	if len(coderSess.GetContextFiles()) != 0 {
		t.Fatalf("expected context files to be empty after /clear_context, got: %v", coderSess.GetContextFiles())
	}

	contextFiles := coderSess.GetContextFiles()

	out, ok = coderSess.ExecuteCommand("! echo test")
	if !ok {
		t.Fatal("expected ! command execution on coder session to succeed")
	}
	if out.Type != types.ShellExecutionStarted || out.Payload != "echo test" {
		t.Fatalf("unexpected command output: %+v", out)
	}
}

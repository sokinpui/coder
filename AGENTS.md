# AGENTS.md - Operational Context for Coding Agents

## Overview & Stack
- **Project**: Coder (`github.com/sokinpui/coder`)
- **Description**: Terminal-centric AI development suite featuring a one-step diff editor (Coder mode), an autonomous coding agent (Agent mode), and standalone utilities (`coder-server`, `itf`, `sf`, `pcat`, `pti`).
- **Language/Runtime**: Go 1.26
- **Key Dependencies**: Cobra, Viper, Charmbracelet (`bubbletea`, `bubbles`, `lipgloss`, `glamour`), Wazero, SQLite (`modernc.org/sqlite`), Go-Pdfium.

---

## Essential Commands

### Setup & Dependencies
```bash
go mod download
```

### Build
```bash
# Build all packages/binaries
go build ./...

# Install binaries locally via install script
./install.sh
```

### Testing
```bash
# Run tests for a specific package (e.g., sf)
go test ./pkg/sf

# Run tests across internal packages
go test ./internal/...
```

### Quality & Linting
```bash
go vet ./...
```

---

## Architecture & Directory Layout
- `cmd/`: Main entrypoints for binaries:
  - `coder`: Main CLI application (Coder, Agent, and Chat modes).
  - `coder-server`: Headless JSON-RPC / WebSocket / HTTP server.
  - `itf`, `sf`, `pcat`, `pti`: Standalone CLI utilities.
- `internal/`: Core application logic (private):
  - `engine/`: AI execution engines, prompt builders, agent tool definitions, generation handlers, context compactors.
  - `server/`: Server implementation for `coder-server`.
  - `ui/`: Terminal user interface components built with Bubble Tea (`bubbletea`, `bubbles`, `lipgloss`, `markdown`).
  - `config/`: Configuration management (`viper`).
  - `project/`: Project context and file discovery.
  - `templates/`: Prompt templates and initialization logic.
- `pkg/`: Reusable public packages:
  - `itf/`: Insert To File patch parser and transactional patch applicator.
  - `sf/`: Search Fast parallel directory walker.
  - `pti/`: PDF to Image rendering utility.
  - `pcat/`: Prompt Cat file concatenator.
  - `version/`: Version metadata.

---

## Code Style & Conventions
- **Go Idioms**: Follow standard Go formatting (`gofmt`), explicit error handling (`if err != nil`), and meaningful naming.
- **CLI & Config**: Use `spf13/cobra` for command routing and `spf13/viper` for configurations.
- **UI Components**: Follow Bubble Tea patterns (`Model`, `Init`, `Update`, `View`) for interactive terminal interfaces.
- **Modularity**: Keep concerns cleanly separated between `cmd/` (CLI wiring), `internal/` (business logic & UI), and `pkg/` (standalone libraries).

---

## Agent Rules & Guardrails
- **Read-Before-Edit**: Always inspect files using `read` or file search tools before performing any text replacement or patch application (`write` or `itf`).
- **Protected Files**: Do not modify generated build artifacts or sensitive credentials/keys.
- **Testing**: Run relevant package tests (`go test`) after making code modifications to ensure no regressions.

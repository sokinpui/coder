<p align="center">
  <img src="assets/icon.svg" alt="Coder Logo" width="128" height="128" />
</p>

<h1 align="center">Coder</h1>

<p align="center">
  Terminal-centric AI development suite: one-step diff editor and autonomous coding agent.
</p>

<p align="center">
  <a href="README.zh.md">简体中文</a>
</p>

## Quick Install

Installs `coder` and `co`:

```bash
curl -fsSL https://raw.githubusercontent.com/sokinpui/coder/main/quickinstall.sh | bash
```

To include all standalone utilities (`itf`, `sf`, `pcat`, `pti`), append `--all`:

```bash
curl -fsSL https://raw.githubusercontent.com/sokinpui/coder/main/quickinstall.sh | bash -s -- --all
```

## Install with Go

```bash
go install github.com/sokinpui/coder/cmd/coder@latest
go install github.com/sokinpui/coder/cmd/co@latest
```

## Local Build

```bash
git clone https://github.com/sokinpui/coder.git
cd coder
./install.sh
```

## Coder

**`coder`** is a **one-step AI code editor**. It is designed for developers who prefer full control over their project context and changes.

- **Human-in-the-loop**: You curate the exact context files provided to the model.
- **Unified Diff Application**: The model returns standardized diffs and file lifecycle operations applied via `itf`.
- **Safe & Reversible**: Every applied modification can be reviewed and undone with `/undo`.

## Co

**`co`** is an **autonomous coding agent**. It operates in an iterative tool-calling loop to solve complex tasks independently.

- **Autonomous Tool Execution**: Self-directed file reading, string replacements, file creation, and shell command execution.
- **Read-before-edit Guard**: Rejects text replacements on files the agent has not yet inspected.
- **Visual & PDF Context**: Native multi-modal support for images and PDF documents.
- **Configurable Permissions**: Fine-grained security policies (`allow`, `ask`, `deny`) on tool and command execution.

## Usage

Launch the one-step editor:

```bash
coder [files...]
```

> Detailed user guide, CLI flags, shortcuts, and commands: [**docs/coder/README.md**](docs/coder/README.md)

Launch the autonomous agent:

```bash
co [prompt]
```

> Detailed agent manual, tool architecture, and workflows: [**docs/coagent/README.md**](docs/coagent/README.md)

## Endpoint Configuration

Coder connects to any OpenAI-compatible API endpoint. Set your API key via environment variable:

```bash
export CODER_API_KEY="your-api-key"
```

Configure endpoints and models in `~/.config/coder/config.yaml` (global) or `.coder/config.yaml` (project-local):

```yaml
server:
  url: http://localhost:9001/v1
  protocol: responses # "responses" (/v1/responses) or "chat" (/v1/chat/completions)

title:
  modelcode: aisrp/gemini-flash-lite-latest

coder:
  modelcode: aisrp/gemini-flash-latest
  reasoningeffort: high
  context:
    dirs: ["."]

agent:
  modelcode: aisrp/gemini-flash-lite-latest
  reasoningeffort: high
  max_iterations: 50
  permission:
    bash: ask
    write:
      ".env*": deny
      "*": ask
```

## Coder Suite

- **[coder.nvim](https://github.com/sokinpui/coder.nvim)**: Neovim plugin integrating Coder directly into your editor buffer.
- **[coder.flutter](https://github.com/sokinpui/coder.flutter)**: Cross-platform GUI client (Desktop, Mobile, Web) connecting via WebSocket (`coder --ws`).
- **[itf](./pkg/itf/README.md)**: Insert To File — parser and patching utility with transactional undo/redo.
- **[sf](./pkg/sf/README.md)**: Search Fast — parallel directory walker respecting `.gitignore`.
- **[pti](./pkg/pti/README.md)**: PDF To Image — renders PDF documents into images for vision models.
- **[pcat](./pkg/pcat/README.md)**: Prompt Cat — concatenates and formats files into syntax-highlighted markdown code blocks.

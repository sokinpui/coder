# Coder

Coder is a TUI AI coding assistant which try to address copy-and-paste overwhelming.

Ask Generation and apply code change directly within `coder`

## Documentation

Full documentation for the project can be found in the `docs/` directory.

- **[Installation & Usage](./docs/installation/README.md)**: How to install, configure, and use the application.
- **[Architecture](./docs/architecture/README.md)**: An overview of the project's architecture and modular design.
- **[Developer Guide](./docs/develop/README.md)**: Information for contributors and developers.
- **[API Reference](./docs/api/README.md)**: Details on the gRPC API contract.

## Quick Start

1.  **Prerequisites:** Ensure you have Go, Git, `protoc`, `pcat`, and `ripgrep` installed.
2.  **Clone:** `git clone https://github.com/your-org/coder.git && cd coder`
3.  **Install Dependencies:** `go mod tidy`
4.  **Generate gRPC Code:** `./gen.sh`
5.  **Build:** `go build -o coder cmd/coder/main.go`
6.  **Run:** `./coder` (must be run from within a Git repository)

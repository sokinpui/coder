# Installation and Usage

This guide provides instructions for installing, configuring, and running the `coder` application.

## Prerequisites

- **Go**: Version 1.21 or higher.
- **Git**: Required for context loading and history management.
- **`protoc`**: The Protocol Buffers compiler, for generating gRPC code.
- **External Tools**: The following command-line tools must be installed and available in your `PATH`:
  - `pcat`: Used to concatenate files with syntax highlighting.
  - `ripgrep` (`rg`): Used to filter files (e.g., exclude markdown) when loading project source.
  - `itf` (optional): An interactive testing framework used by the `:itf` command.
- **AI Service**: Access to a running instance of the gRPC AI generation service.

## Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/your-org/coder.git
    cd coder
    ```

2.  **Install Go dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Generate gRPC code:**
    This step is only necessary if you modify `protos/generate.proto`.
    ```bash
    ./gen.sh
    ```

4.  **Build the application:**
    ```bash
    go build -o coder cmd/coder/main.go
    ```

## Configuration

- **gRPC Server Address**: The application connects to the AI service at `localhost:50051` by default. This is configured in `internal/config/config.go`.
- **AI Parameters**: The default model, temperature, and other generation parameters are also set in `internal/config/config.go`. These can be changed at runtime using the `:model` command.

### Context Directory

To provide the AI with specific context about your project, create a directory named `Context` at the root of your Git repository. Place any relevant documentation files (e.g., `.md`, `.txt`) inside this directory. They will be automatically loaded and included in the prompt.

Example:

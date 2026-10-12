---
name: init
description: Initialize AGENTS.md guide for the repository
---

# Repository Initialization (AGENTS.md)

Your task is to inspect this codebase and generate a root `AGENTS.md` file tailored for AI coding agents.
`AGENTS.md` serves as the authoritative, persistent operational context for agents working in this repository.

## Workflow

1. **Pre-check Existing Configuration (Guard Clause)**:
   - Check whether `AGENTS.md` already exists in the current working directory.
   - If `AGENTS.md` already exists, do not overwrite or modify it. Inform the user that `AGENTS.md` already exists and stop immediately.

2. **Analyze Repository Typology & Structure**:
   - Determine the repository layout:
     - **Monorepo / Workspaces**: pnpm/yarn/npm/bun workspaces, Cargo workspaces, Go multi-module/workspaces, Gradle multi-project, Nx, Turborepo.
     - **Single-module / Application / Library**: Go, Rust, Python, Node/TypeScript, Java/Kotlin, C/C++, etc.
     - **Polyglot / Multi-service**: Repositories containing independent backend, frontend, or infrastructure directories.
   - Inspect manifests, configuration files, and package lockfiles (e.g., `go.mod`, `Cargo.toml`, `package.json`, `pyproject.toml`, `pom.xml`, `build.gradle`, `CMakeLists.txt`, `Makefile`, `Justfile`, `Taskfile`).
   - Check for existing instructions and rules to incorporate:
     - `.cursorrules` or `.cursor/rules/**`
     - `.github/copilot-instructions.md`
     - `CLAUDE.md`
     - `.windsurfrules` or `.clinerules`

3. **Discover Verified Verification & Execution Commands**:
   - Inspect CI workflows (e.g., `.github/workflows/`, `.gitlab-ci.yml`), build scripts, and task runners to find verified commands.
   - Capture exact, runnable commands for:
     - **Setup**: Dependency installation and environment preparation.
     - **Build**: Compilation, bundling, or code generation commands (e.g., protobuf, sqlc, openapi).
     - **Tests**:
       - Full test suite execution command.
       - **Single test execution**: The exact command syntax to run an individual test file or specific test case (crucial for fast agent feedback loops).
     - **Quality**: Linters, formatters, and static analysis/type-checking commands.
   - Do not speculate or invent unverified commands. Only include commands backed by manifests, scripts, or verified toolchains.

4. **Identify Conventions & Operational Guardrails**:
   - Identify core directory boundaries, entry points, and module responsibilities.
   - Note coding conventions: error handling idioms, import conventions, type annotations, and naming patterns.
   - Identify protected paths and agent gotchas:
     - Generated files that agents must not manually edit.
     - Required environment variables or configuration templates (e.g., `.env.example`).
     - Repository-specific constraints and architectural boundaries.

5. **Generate `AGENTS.md`**:
   - Create `AGENTS.md` in the current working directory.
   - Keep the file concise, high-density, and actionable (~100–150 lines). Avoid conversational filler.
   - Recommended structure:
     - **Overview & Stack**: Brief description of the project, architecture type, language, and runtime stack.
     - **Essential Commands**: Setup, build, lint, format, typecheck, full test suite, and single-test command examples.
     - **Architecture & Directory Layout**: Key directories and their architectural roles.
     - **Code Style & Conventions**: Practical rules for imports, error handling, types, and testing conventions.
     - **Agent Rules & Guardrails**: Non-negotiable constraints, protected files, code generation workflows, and common pitfalls.
   - Output a clean, structured file ready for source control.

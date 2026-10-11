---
name: init
description: Initialize or update the AGENTS.md file with repository guidelines, build instructions, and architecture overview
---

# Repository Initialization (AGENTS.md)

Your task is to inspect this codebase and generate or update the root `AGENTS.md` file.
`AGENTS.md` serves as the authoritative guide for coding agents working in this repository.

## Workflow

1. **Analyze Project Structure & Environment**:
   - Inspect key files and directory structure at the repository root.
   - Look for manifest and package files (e.g. `go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, `pom.xml`, `build.gradle`, `CMakeLists.txt`).
   - Identify primary languages, frameworks, runtimes, and dependencies.

2. **Discover Build, Test, and Quality Commands**:
   - Inspect build scripts, `Makefile`, task files, CI workflows (e.g. `.github/workflows/`), and package scripts.
   - Identify the exact commands to:
     - Install dependencies
     - Build or compile the project
     - Run the test suite (unit and integration tests)
     - Run linters, formatters, and type checkers
   - Use the `bash` tool to run non-destructive inspection or verification commands if needed.

3. **Analyze Architecture & Code Conventions**:
   - Identify key entry points and core modules.
   - Note key design patterns, directory responsibilities, and coding conventions.
   - Check if an existing `AGENTS.md` or similar instruction file already exists, and preserve valid existing information.

4. **Write or Update `AGENTS.md`**:
   - Use the `write` tool to create or overwrite `AGENTS.md` at the repository root.
   - Structure `AGENTS.md` clearly:
     - **Overview**: Brief summary of the project.
     - **Architecture & Directory Structure**: Key directories and their purposes.
     - **Development Workflow**: Exact build, test, and lint commands.
     - **Coding Conventions & Guidelines**: Naming, error handling, style, and rules for agents.
   - Keep the file concise, practical, and factual. Do not speculate or invent unverified commands.

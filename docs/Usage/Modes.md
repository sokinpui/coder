# Application Modes

Coder provides different application modes to tailor the AI's behavior and context for specific tasks. You can switch modes at any time using the `:mode <name>` command in the TUI or the mode selector in the Web UI.

Each mode affects two key aspects of the application's operation:

1.  **System Prompt**: The set of instructions given to the AI to define its role and expertise (e.g., as a programmer or a technical writer).
2.  **Context Gathering**: The types of files included from your project source code to provide relevant context for the AI's responses.

## Available Modes

### `Coding`

This is the default mode, optimized for writing and modifying source code.

-   **System Prompt**: Instructs the AI to act as an expert programmer, focusing on code quality, modularity, and best practices. The full prompt is defined in `internal/core/Roles/coding.md`.
-   **Context Gathering**: Gathers all source code files from the project but **excludes** Markdown files (`*.md`). This prevents documentation from consuming the context window when the primary task is coding.

### `Documenting`

This mode is designed for writing documentation, such as READMEs, architectural overviews, or API references.

-   **System Prompt**: Instructs the AI to act as an expert technical writer, emphasizing clarity, conciseness, and objective language. The full prompt is defined in `internal/core/Roles/documenting.md`.
-   **Context Gathering**: Gathers all source code files from the project, **including** Markdown files (`*.md`). This allows the AI to reference existing documentation and maintain consistency.

### `Auto`

This mode provides a balanced approach for tasks that involve both coding and documentation.

-   **System Prompt**: Combines the instructions for both an expert programmer and an expert technical writer. The full prompt is defined in `internal/core/Roles/auto-mode.md`.
-   **Context Gathering**: Gathers all source code files, **including** Markdown files (`*.md`), similar to the `Documenting` mode.

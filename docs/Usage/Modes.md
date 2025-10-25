# Application Modes

Coder provides different application modes to tailor the AI's behavior and context for specific tasks. You can switch modes at any time using the `:mode <name>` command in the TUI.

Each mode affects two key aspects of the application's operation:

1.  **System Prompt**: The set of instructions given to the AI to define its role and expertise (e.g., as a programmer or a technical writer).
2.  **Context Gathering**: The types of files included from your project source code to provide relevant context for the AI's responses.

## Available Modes

### `Coding`

This is the default mode, optimized for writing and modifying source code.

-   **System Prompt**: Instructs the AI to act as an expert programmer, focusing on code quality, modularity, and best practices. The full prompt is defined in `internal/prompt/Roles/coding.md`.
-   **Context Gathering**: Gathers all source code files from the project but **excludes** Markdown files (`*.md`) and other non-code files by default. This prevents documentation from consuming the context window when the primary task is coding.
    - To include a Markdown file or any other specific file, use the `:file <path>` command.

### `Documenting`

This mode is designed for writing documentation, such as READMEs, architectural overviews, or API references.

-   **System Prompt**: Instructs the AI to act as an expert technical writer, emphasizing clarity, conciseness, and objective language. The full prompt is defined in `internal/prompt/Roles/documenting.md`.
-   **Context Gathering**: Gathers source code using the same default exclusions as `Coding` mode (which ignores `*.md` files). This allows the AI to reference source code while writing documentation.
    - To make the AI read existing documentation, you must explicitly include the Markdown files using the `:file <path-to-doc.md>` command.

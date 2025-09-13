# Coder - Architecture and Design

This document outlines the high-level architecture, modular design, and user interface (UI) design principles of the `coder` project.

## Overall Architecture

`coder` follows a client-server architecture, where the `coder` application itself acts as a rich terminal-based client that interacts with an external AI generation service via gRPC. The application is designed to be run from within a Git repository, allowing it to provide highly contextualized responses based on the project's codebase and user-defined documents.

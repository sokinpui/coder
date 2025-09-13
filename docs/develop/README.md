# Coder - Developer Guide

This document provides an overview of the `coder` project's structure, key components, and internal workflows to help developers understand, maintain, and contribute to the codebase.

## Project Goal

`coder` is an interactive, AI-powered assistant designed to be run from within a Git repository. Its primary function is to assist developers by generating code, providing explanations, and interacting with the project context, leveraging a gRPC-based AI generation service.

## Project Structure

The project follows a standard Go project layout with a `cmd` directory for the main application entry point and an `internal` directory for private packages.

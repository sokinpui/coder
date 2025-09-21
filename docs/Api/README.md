# API Reference

The Coder Web UI (`coder-web`) communicates with its Go backend via a WebSocket API. This document outlines the message formats used in this communication.

The WebSocket endpoint is `/ws`.

## Message Structure

All messages, in both directions, follow a common JSON structure:

```json
{
  "type": "message-type-string",
  "payload": { ... }
}
```

-   `type`: A string identifying the message type.
-   `payload`: A JSON object or primitive containing the message data.

## Client-to-Server Messages

Messages sent from the web client to the server.

### `userInput`

Sent when the user submits a prompt or command.

-   **Type**: `userInput`
-   **Payload**: `string` - The text from the input box.

### `cancelGeneration`

Sent to stop an in-progress AI generation.

-   **Type**: `cancelGeneration`
-   **Payload**: (empty)

### `regenerateFrom`

Sent to regenerate an AI response from a specific point in the conversation.

-   **Type**: `regenerateFrom`
-   **Payload**: `number` - The index of the user message to regenerate from.

### `applyItf`

Sent when the user clicks the "Apply" button on an AI message containing a diff.

-   **Type**: `applyItf`
-   **Payload**: `string` - The content of the AI message to be piped to `itf`.

### `editMessage`

Sent to update the content of a user's message.

-   **Type**: `editMessage`
-   **Payload**: `object`
    -   `index`: `number` - The index of the message to edit.
    -   `content`: `string` - The new content for the message.

### `branchFrom`

Sent to create a new session that is a fork of the current one up to a certain message.

-   **Type**: `branchFrom`
-   **Payload**: `number` - The index of the last message to include in the new session.

### `deleteMessage`

Sent to delete a message from the conversation.

-   **Type**: `deleteMessage`
-   **Payload**: `number` - The index of the message to delete.

### `listHistory`

Requests the list of saved conversations.

-   **Type**: `listHistory`
-   **Payload**: (empty)

### `loadConversation`

Requests to load a specific conversation from history.

-   **Type**: `loadConversation`
-   **Payload**: `string` - The filename of the conversation to load.

### `getSourceTree`

Requests the project's file tree.

-   **Type**: `getSourceTree`
-   **Payload**: (empty)

### `getFileContent`

Requests the content of a specific file.

-   **Type**: `getFileContent`
-   **Payload**: `string` - The relative path of the file.

### `getGitGraphLog`

Requests the Git commit log for the repository.

-   **Type**: `getGitGraphLog`
-   **Payload**: (empty)

### `getCommitDiff`

Requests the diff for a specific Git commit.

-   **Type**: `getCommitDiff`
-   **Payload**: `string` - The commit hash.

## Server-to-Client Messages

Messages sent from the server to the web client.

### `initialState`

Sent once upon successful connection.

-   **Type**: `initialState`
-   **Payload**: `object` - Contains initial state like `cwd`, `title`, `mode`, `model`, `tokenCount`, `availableModes`, and `availableModels`.

### `stateUpdate`

Sent when configuration state (like mode or model) changes.

-   **Type**: `stateUpdate`
-   **Payload**: `object` - Contains the updated state fields (`mode`, `model`, `tokenCount`).

### `messageUpdate`

Sent when a new, complete message (e.g., a command result) is added to the history.

-   **Type**: `messageUpdate`
-   **Payload**: `object`
    -   `type`: `string` - The sender type (`User`, `AI`, `System`, `Command`, `Result`, `Error`).
    -   `content`: `string` - The message content.

### `generationChunk`

Sent repeatedly for each piece of a streaming AI response.

-   **Type**: `generationChunk`
-   **Payload**: `string` - A piece of the generated text.

### `generationEnd`

Sent when an AI generation stream is complete.

-   **Type**: `generationEnd`
-   **Payload**: (empty)

### `newSession`

Sent after a `:new` command is processed, instructing the client to clear its state.

-   **Type**: `newSession`
-   **Payload**: (empty)

### `truncateMessages`

Sent after a `regenerateFrom` or `branchFrom` action, instructing the client to shorten its message list to a specific length.

-   **Type**: `truncateMessages`
-   **Payload**: `number` - The new length of the message array.

### `titleUpdate`

Sent when the conversation title is generated or changed.

-   **Type**: `titleUpdate`
-   **Payload**: `string` - The new title.

### `historyList`

The response to a `listHistory` request.

-   **Type**: `historyList`
-   **Payload**: `array` - A list of history item objects, each with `filename`, `title`, and `modifiedAt`.

### `sessionLoaded`

The response to a `loadConversation` request, containing the full state of the loaded session.

-   **Type**: `sessionLoaded`
-   **Payload**: `object` - Contains `messages`, `title`, `mode`, `model`, and `tokenCount`.

### `sourceTree`

The response to a `getSourceTree` request.

-   **Type**: `sourceTree`
-   **Payload**: `object` - The root node of the file tree.

### `fileContent`

The response to a `getFileContent` request.

-   **Type**: `fileContent`
-   **Payload**: `object`
    -   `path`: `string` - The path of the file.
    -   `content`: `string` - The content of the file.

### `gitGraphLog`

The response to a `getGitGraphLog` request.

-   **Type**: `gitGraphLog`
-   **Payload**: `array` - A list of Git log entry objects.

### `commitDiff`

The response to a `getCommitDiff` request.

-   **Type**: `commitDiff`
-   **Payload**: `object`
    -   `hash`: `string` - The commit hash.
    -   `diff`: `string` - The diff content.

### `error`

Sent when a server-side error occurs.

-   **Type**: `error`
-   **Payload**: `string` - The error message.

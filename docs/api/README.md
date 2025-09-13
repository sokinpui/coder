# API Reference

The `coder` application communicates with an AI generation service via a gRPC API. This document defines the service contract based on the `protos/generate.proto` file.

## Service: `Generate`

The `Generate` service provides a single method for streaming AI-generated content.

### RPC: `GenerateTask`

This is a server-streaming RPC. The client sends a single `Request` message, and the server streams back a sequence of `Response` messages.

- **Request**: `Request`
- **Stream Response**: `Response`

### Messages

#### `Request`

The `Request` message contains all the information needed to generate a response.

| Field | Type | Description |
| :--- | :--- | :--- |
| `prompt` | `string` | The full prompt string to be sent to the AI model. |
| `model_code` | `string` | The identifier for the AI model to be used (e.g., "gemini-2.5-pro"). |
| `stream` | `bool` | If `true`, the server will stream the response back in chunks. |
| `config` | `GenerationConfig` | (Optional) Configuration parameters for the generation process. |

#### `Response`

The `Response` message contains a chunk of the generated output.

| Field | Type | Description |
| :--- | :--- | :--- |
| `output_string` | `string` | A segment of the AI-generated text. |

#### `GenerationConfig`

The `GenerationConfig` message allows for fine-tuning the behavior of the AI model. All fields are optional.

| Field | Type | Description |
| :--- | :--- | :--- |
| `temperature` | `float` | Controls the randomness of the output. A lower value makes the output more deterministic. |
| `top_p` | `float` | The cumulative probability cutoff for nucleus sampling. |
| `top_k` | `int32` | The number of highest probability vocabulary tokens to keep for top-k-filtering. |
| `output_length` | `int32` | The maximum number of tokens to generate. |

# codex-mcp-go

<div align="center">

**MCP Protocol Wrapper for Codex CLI**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/dl/) [![MCP Compatible](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io) [![NPM Version](https://img.shields.io/npm/v/@zenfun510/codex-mcp-go)](https://www.npmjs.com/package/@zenfun510/codex-mcp-go)

⭐ **If you find this useful, please give us a Star! Your support keeps us going~** ⭐

[简体中文](./README.md) | English

</div>

---

## Introduction

`codex-mcp-go` is an MCP (Model Context Protocol) server implementation in Go. It wraps the OpenAI Codex CLI, enabling it to be called as an MCP tool by AI clients like Claude Code, Roo Code, and KiloCode.

Key Features:
- **Session Management**: Maintains multi-turn conversation context via `SESSION_ID`.
- **Sandbox Control**: Provides security policies like `read-only` and `workspace-write`.
- **Concurrency**: Supports concurrent client calls using Go routines.
- **Single Binary**: Compiles to a single binary with no runtime dependencies.

---

## Quick Start

### 1. Prerequisites

This tool relies on OpenAI's `codex` CLI. Please ensure you have installed and configured it.

**Install Codex CLI:**

```bash
# Install via npm (Recommended)
npm i -g @openai/codex

# Or refer to the official repository
# https://github.com/openai/codex-cli
```

### 2. Build

Requires Go 1.24+.

```bash
git clone https://github.com/w31r4/codex-mcp-go.git
cd codex-mcp-go
go build -o codex-mcp-go cmd/server/main.go
```

### 3. Configure MCP Client

Choose the configuration method based on your AI client.

#### Method A: Using npx (Recommended)

<details>
<summary><strong>Claude Code</strong></summary>

```bash
claude mcp add codex -s user --transport stdio -- npx -y @zenfun510/codex-mcp-go
```
</details>

<details>
<summary><strong>Roo Code (VSCode / Cursor)</strong></summary>

Add to Roo Code's MCP settings:

```json
{
  "mcpServers": {
    "codex": {
      "command": "npx",
      "args": ["-y", "@zenfun510/codex-mcp-go"],
      "env": {}
    }
  }
}
```

Config file paths:
- VSCode: `~/.config/Code/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json`
- Cursor: `~/.config/Cursor/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json`
</details>

<details>
<summary><strong>KiloCode</strong></summary>

Add to `~/.kilocode/mcp.json`:

```json
{
  "mcpServers": {
    "codex": {
      "command": "npx",
      "args": ["-y", "@zenfun510/codex-mcp-go"],
      "env": {}
    }
  }
}
```
</details>

<details>
<summary><strong>Cursor (Native MCP)</strong></summary>

1. Open Cursor Settings -> Features -> MCP
2. Click "Add New MCP Server"
3. Fill in the configuration:
   - Name: `codex`
   - Type: `stdio`
   - Command: `npx`
   - Args: `-y @zenfun510/codex-mcp-go`
</details>

#### Method B: Using Local Binary

If you have built the binary via `go build` (assuming path is `/path/to/codex-mcp-go`), use the following configuration:

<details>
<summary><strong>Claude Code</strong></summary>

```bash
claude mcp add codex -s user --transport stdio -- /path/to/codex-mcp-go
```
</details>

<details>
<summary><strong>Roo Code / KiloCode / Generic JSON</strong></summary>

```json
{
  "mcpServers": {
    "codex": {
      "command": "/path/to/codex-mcp-go",
      "args": [],
      "env": {
        "OPENAI_API_KEY": "your-api-key"
      }
    }
  }
}
```
</details>

<details>
<summary><strong>Cursor (Native MCP)</strong></summary>

1. Open Cursor Settings -> Features -> MCP
2. Click "Add New MCP Server"
3. Fill in the configuration:
   - Name: `codex`
   - Type: `stdio`
   - Command: `/path/to/codex-mcp-go`
   - Args: (Leave empty)
</details>

### 4. Verify

```bash
cat <<'EOF' | ./codex-mcp-go
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"0.1.0","capabilities":{}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
EOF
```
The MCP handshake requires `initialize` before `tools/list`. If the response contains the `codex` tool, it is working.

---

## Tool Parameters

Tool Name: `codex`

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `PROMPT` | `string` | ✅ | - | Instruction sent to Codex |
| `cd` | `string` | ✅ | - | Working directory path |
| `sandbox` | `string` | ❌ | `"workspace-write"` | Policy: `read-only` / `workspace-write` / `danger-full-access` |
| `SESSION_ID` | `string` | ❌ | `""` | Session ID for multi-turn conversations |
| `skip_git_repo_check` | `bool` | ❌ | `true` | Allow running in non-Git directories |
| `return_all_messages` | `bool` | ❌ | `false` | Return full reasoning logs |
| `image` | `[]string` | ❌ | `[]` | Attached image paths |
| `model` | `string` | ❌ | `""` | Specify model |
| `yolo` | `bool` | ❌ | `true` | Skip all confirmations (Default enabled to prevent timeouts) |
| `profile` | `string` | ❌ | `""` | Specify configuration profile |

---

## Feature Comparison

### 1. vs Official Codex CLI

| Feature | Official Codex CLI | CodexMCP (This Tool) |
|---------|--------------------|----------------------|
| **Basic Codex Call** | ✅ | ✅ |
| **Multi-turn Chat** | ❌ | ✅ (via Session Management) |
| **Reasoning Trace** | ❌ | ✅ (Full Log Capture) |
| **Parallel Tasks** | ❌ | ✅ (MCP Protocol Support) |
| **Error Handling** | ❌ | ✅ (Structured Error Return) |

### 2. vs Python Version (codexmcp)

| Feature | Go Version (codex-mcp-go) | Python Version (codexmcp) |
|---------|---------------------------|---------------------------|
| **Deployment** | Single binary, zero dependencies | Requires Python env & deps |
| **Startup Speed** | 🚀 Very Fast | 🐢 Slower (Interpreter startup) |
| **Resource Usage** | 📉 Low | 📈 Higher |
| **Concurrency** | Goroutine (Efficient) | Threading |
| **Use Case** | Production, Low-level Service | Secondary Dev, Prototyping |

---

## Troubleshooting

*   **Connection Failed**: Check if `codex` CLI is in PATH, or verify Go version >= 1.24.
*   **Permission Denied**: Check if the binary has execution permissions (`chmod +x`).
*   **Session Lost**: Ensure the client correctly passes the `SESSION_ID` returned from the previous call.

---

## License

This project is licensed under the [MIT License](./LICENSE).

---

## Acknowledgements

This project is heavily inspired by [codexmcp](https://github.com/GuDaStudio/codexmcp) (Python implementation). Thanks to the GuDaStudio team for their pioneering work in exploring Codex MCP integration.

Codex's capabilities in detail and bug fixing are evident to all, but sometimes it feels slightly lacking in global perspective. Therefore, my current workflow uses Gemini 3.0 Pro's KiloCode as the main planner, while Codex can be responsible for the implementation of complex functions and bug fixing.

---

<div align="center">

**Thanks again for your interest! Don't forget to Star~ 🌟**

</div>

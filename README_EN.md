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

### 1. Build

Requires Go 1.24+.

```bash
git clone https://github.com/w31r4/codex-mcp-go.git
cd codex-mcp-go
go build -o codex-mcp-go cmd/server/main.go
```

### 2. Configure MCP Client

Choose the configuration method based on your AI client.

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

<details>
<summary><strong>Generic Configuration (JSON)</strong></summary>

For other MCP-compatible clients:

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

### 3. Verify

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./codex-mcp-go
```
If it returns JSON data containing the `codex` tool, it is running correctly.

---

## Tool Parameters

Tool Name: `codex`

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `PROMPT` | `string` | ✅ | - | Instruction sent to Codex |
| `cd` | `string` | ✅ | - | Working directory path |
| `sandbox` | `string` | ❌ | `"read-only"` | Policy: `read-only` / `workspace-write` / `danger-full-access` |
| `SESSION_ID` | `string` | ❌ | `""` | Session ID for multi-turn conversations |
| `skip_git_repo_check` | `bool` | ❌ | `true` | Allow running in non-Git directories |
| `return_all_messages` | `bool` | ❌ | `false` | Return full reasoning logs |
| `image` | `[]string` | ❌ | `[]` | Attached image paths |
| `model` | `string` | ❌ | `""` | Specify model |
| `yolo` | `bool` | ❌ | `false` | Skip all confirmations (Use with caution) |
| `profile` | `string` | ❌ | `""` | Specify configuration profile |

---

## Version Comparison (Go vs Python)

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

## Acknowledgements

This project is heavily inspired by [codexmcp](https://github.com/GuDaStudio/codexmcp) (Python implementation). Thanks to the GuDaStudio team for their pioneering work in exploring Codex MCP integration.

---

<div align="center">

**Thanks again for your interest! Don't forget to Star~ 🌟**

</div>

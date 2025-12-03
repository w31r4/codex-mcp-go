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

## Recommended System Prompts

To get the best experience, it is recommended to configure the appropriate system prompt based on the client type you are using.

### 1. Agent Mode

Suitable for Agents capable of autonomous planning and execution of multi-step tasks. Add the following to your Agent configuration or as the initial instruction for the task:

<details>
<summary><strong>Click to expand detailed prompt</strong></summary>

```text
## Core Instruction for CodeX MCP

At any moment, you must consider how the current process can collaborate with Codex, and how to invoke the MCP tools provided by Codex as a guarantee for your objective and comprehensive analysis.
You **MUST execute** the following steps:
**1** After forming a preliminary analysis of the user's requirements, inform Codex of the user's requirements and your initial thoughts, and ask it to refine the requirement analysis and implementation plan.
**2** Before implementing specific coding tasks, **you MUST ask Codex for a code implementation prototype (require Codex to provide ONLY a unified diff patch, strictly forbidding any real modification to the code)**. After obtaining the code prototype, you **can only use it as a logical reference and rewrite the code modification** to form enterprise production-level, highly readable, and highly maintainable code before implementing the specific programming modification task.
**3** Whenever a tangible coding action is completed, **you MUST immediately use Codex to review the code changes and the degree of completion of the corresponding requirements**.
**4** Codex can only provide references; you **MUST have your own thinking and even need to question Codex's answers**. Believing everything in books is worse than having no books at all. The ultimate mission for you and Codex is to reach a unified, comprehensive, and precise opinion, so you must constantly debate to find the only way to truth.

## Codex Tool Invocation Specification

1. Tool Overview
   The codex MCP provides a tool `codex` for executing AI-assisted coding tasks. This tool is **invoked via the MCP protocol**, without the need to use the command line.

2. Tool Parameters
   **Required** Parameters:
   - PROMPT (string): The task instruction sent to Codex.
   - cd (Path): The root path of the working directory where Codex executes the task.

   Optional Parameters:
   - sandbox (string): Sandbox policy, possible values:
     - "read-only" (Default): Read-only mode, safest.
     - "workspace-write": Allows writing in the workspace.
     - "danger-full-access": Full access permission.
   - SESSION_ID (UUID | null): Used to continue a previous session for multi-turn interaction with Codex. Defaults to None (start a new session).
   - skip_git_repo_check (boolean): Whether to allow running in a non-Git repository. Defaults to False.
   - return_all_messages (boolean): Whether to return all messages (including reasoning, tool calls, etc.). Defaults to False.
   - image (List[Path] | null): Attach one or more image files to the initial prompt. Defaults to None.
   - model (string | null): Specify the model to use. Defaults to None (use user default config).
   - yolo (boolean | null): Run all commands without approval (skip sandbox). Defaults to False.
   - profile (string | null): Configuration profile name loaded from `~/.codex/config.toml`. Defaults to None (use user default config).

3. Invocation Specification
   **MUST Comply**:
   - Every time the codex tool is called, the returned SESSION_ID must be saved for subsequent dialogue.
   - The cd parameter must point to an existing directory, otherwise the tool will fail silently.
   - It is strictly forbidden for Codex to make actual modifications to the code. Use sandbox="read-only" to avoid accidents, and ask Codex to provide only a unified diff patch.

   Recommended Usage:
   - If you need to track Codex's reasoning process and tool calls in detail, set return_all_messages=True.
   - For tasks such as precise positioning, debugging, and rapid code prototyping, prioritize using the codex tool.
```
</details>

### 2. Copilot Mode

Suitable for assistants running as IDE plugins. Recommended to add to `.clinerules` (Roo Code) or "Rules for AI" (Cursor):

<details>
<summary><strong>Click to expand rule prompt</strong></summary>

```text
# Codex MCP Tool Rules

You have access to the `codex` tool, which wraps the OpenAI Codex CLI. Use it for complex code generation, debugging, and analysis.

## Workflow
1.  **Consultation**: Before writing complex code, ask Codex for a plan or analysis.
2.  **Prototyping**: Ask Codex for a `unified diff patch` to solve the problem.
    *   **IMPORTANT**: Always use `sandbox="read-only"` when asking for code.
    *   **IMPORTANT**: Do NOT let Codex apply changes directly.
3.  **Implementation**: Read the Codex-generated diff, understand it, and then apply the changes yourself using your own file editing tools.
4.  **Review**: After applying changes, you can ask Codex to review the code.

## Tool Usage
-   **Session**: Always capture and reuse `SESSION_ID` for multi-turn tasks.
-   **Path**: Ensure `cd` is set to the current workspace root.
-   **Safety**: Default to `sandbox="read-only"`. Only use `workspace-write` if explicitly instructed by the user and you are confident in the operation.
```
</details>

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
| `model` | `string` | ❌ | `""` | Prohibited unless explicitly allowlisted |
| `yolo` | `bool` | ❌ | `false` | Skip all confirmations (non-interactive) |
| `profile` | `string` | ❌ | `""` | Prohibited unless explicitly allowlisted |
| `timeout_seconds` | `int` | ❌ | `1800` | Total timeout (seconds) for the codex invocation (cap: 1800) |
| `no_output_seconds` | `int` | ❌ | `0` | Kill the run if no output for this many seconds (0 disables) |

**Runtime behavior:** Codex invocations default to a 30m total timeout (capped at 30m) with an optional no-output watchdog (disabled by default); failures/non-zero exits or error lines are surfaced with recent output. For slow networks or MCP clients with shorter RPC timeouts, keep `timeout_seconds=1800` on the tool call to avoid premature cancellation.
**Defaults:** `sandbox=read-only`, `yolo=false`, `skip_git_repo_check=false`; `model/profile` are rejected unless you explicitly allowlist them; `timeout_seconds=1800` (capped at 1800), `no_output_seconds=0` (disabled).

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

# Codex4KiloMCP

<div align="center">

**Seamlessly integrate AI coding assistants with Codex**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/dl/) [![MCP Compatible](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io)

</div>

---

## 1. Project Overview

Codex4KiloMCP is a Go-based MCP (Model Context Protocol) server that serves as a bridge for Codex CLI, enabling various AI coding assistants (such as Claude Code, Roo Code, KiloCode, etc.) to seamlessly collaborate with Codex.

In the current AI-assisted programming ecosystem:
- **AI Coding Assistants** (Claude Code/Roo Code/KiloCode): Handle architecture design, requirement analysis, and code refactoring
- **Codex**: Handles code generation, bug fixing, and code review
- **Codex4KiloMCP**: Manages session context, supporting multi-turn conversations and parallel tasks

Compared to the official Codex MCP implementation, Codex4KiloMCP introduces **enterprise-grade features** such as **session persistence**, **parallel execution**, and **reasoning trace**.

---

## 2. Quick Start

### 0. Prerequisites

Please ensure you have **installed** and **configured** the following tools:

- [Codex CLI](https://developers.openai.com/codex/quickstart) - OpenAI's programming assistant
- [Go 1.24+](https://golang.org/dl/) - Go language environment
- MCP-compatible AI client (Claude Code, Roo Code, KiloCode, etc.)

### 1. Installation Steps

#### 1.1 Build Project

```bash
# Clone repository
git clone https://github.com/your-repo/codex4kilomcp.git
cd codex4kilomcp

# Build binary
go build -o codex4kilomcp cmd/server/main.go
```

#### 1.2 Configure MCP Client

Choose the configuration method based on your AI client:

<details>
<summary><strong>Claude Code Configuration</strong></summary>

```bash
# Remove official Codex MCP (if installed)
claude mcp remove codex

# Add Codex4KiloMCP
claude mcp add codex -s user --transport stdio -- /path/to/codex4kilomcp
```

Verify installation:
```bash
claude mcp list
# Should display: codex: /path/to/codex4kilomcp - ✓ Connected
```

</details>

<details>
<summary><strong>Roo Code Configuration</strong></summary>

Add to Roo Code's MCP configuration:

```json
{
  "mcpServers": {
    "codex": {
      "command": "/path/to/codex4kilomcp",
      "args": [],
      "env": {}
    }
  }
}
```

Configuration path:
- VSCode: `~/.config/Code/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json`
- Cursor: `~/.config/Cursor/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json`

</details>

<details>
<summary><strong>KiloCode Configuration</strong></summary>

Add to KiloCode's MCP configuration:

```json
{
  "mcpServers": {
    "codex": {
      "command": "/path/to/codex4kilomcp",
      "args": [],
      "env": {}
    }
  }
}
```

Configuration path: `~/.kilocode/mcp.json`

</details>

<details>
<summary><strong>Other MCP-Compatible Clients</strong></summary>

Generic configuration format:

```json
{
  "mcpServers": {
    "codex": {
      "command": "/path/to/codex4kilomcp",
      "args": [],
      "env": {}
    }
  }
}
```

</details>

#### 1.3 Verify Installation

Test MCP server with:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | /path/to/codex4kilomcp
```

Should return available tools list containing `codex` tool.

---

## 3. Tool Description

### codex Tool

Execute non-interactive Codex sessions to complete AI-assisted programming tasks.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `PROMPT` | `string` | ✅ | - | Task instruction sent to Codex |
| `cd` | `string` | ✅ | - | Codex working directory root path |
| `sandbox` | `string` | ❌ | `"read-only"` | Sandbox policy: `read-only` / `workspace-write` / `danger-full-access` |
| `SESSION_ID` | `string` | ❌ | `""` | Session ID (empty starts new session) |
| `skip_git_repo_check` | `bool` | ❌ | `true` | Allow running in non-Git repositories |
| `return_all_messages` | `bool` | ❌ | `false` | Return complete reasoning information |
| `image` | `[]string` | ❌ | `[]` | Attach images to initial prompt |
| `model` | `string` | ❌ | `""` | Specify model to use |
| `yolo` | `bool` | ❌ | `false` | Run all commands without approval (skip sandbox) |
| `profile` | `string` | ❌ | `""` | Configuration profile name |

#### Return Values

**Success:**
```json
{
  "success": true,
  "SESSION_ID": "550e8400-e29b-41d4-a716-446655440000",
  "agent_messages": "Codex response content...",
  "all_messages": [...]  // Only included when return_all_messages=true
}
```

**Failure:**
```json
{
  "success": false,
  "error": "Error description"
}
```

---

## 4. Usage Examples

### Example 1: Code Review

```json
{
  "PROMPT": "Review the code in src/main.go and suggest improvements",
  "cd": "/path/to/project",
  "sandbox": "read-only",
  "return_all_messages": true
}
```

### Example 2: Generate Test Code

```json
{
  "PROMPT": "Generate unit tests for the calculate function in math.go",
  "cd": "/path/to/project",
  "sandbox": "workspace-write"
}
```

### Example 3: Multi-turn Conversation

First turn:
```json
{
  "PROMPT": "Help me design a REST API for user management",
  "cd": "/path/to/project"
}
```

Second turn (using returned SESSION_ID):
```json
{
  "PROMPT": "Now implement the authentication middleware",
  "cd": "/path/to/project",
  "SESSION_ID": "550e8400-e29b-41d4-a716-446655440000"
}
```

---

## 5. Go Version vs Python Version

| Feature | Go Version (codex4kilomcp) | Python Version (codexmcp) |
|---------|---------------------------|---------------------------|
| **Core Features** | ✅ Full support | ✅ Full support |
| **Session Persistence** | ✅ Supported | ✅ Supported |
| **Reasoning Trace** | ✅ Supported | ✅ Supported |
| **Parallel Execution** | ✅ Supported | ✅ Supported |
| **Error Handling** | ✅ Enhanced | ✅ Standard |
| **Performance** | ⚡ Higher (compiled) | 🐍 Good (interpreted) |
| **Memory Usage** | 📦 Lower | 📦 Higher |
| **Startup Speed** | 🚀 Faster | 🚶 Slower |
| **Cross-platform** | ✅ Windows/Linux/macOS | ✅ Windows/Linux/macOS |
| **Dependency Management** | go.mod | uv/pip |
| **Use Cases** | Production, resource-constrained | Development, rapid iteration |

**Recommendation:**
- **Go Version**: Suitable for production deployment, resource-sensitive environments, high-performance scenarios
- **Python Version**: Suitable for rapid development, Python ecosystem integration, prototyping

---

## 6. Troubleshooting

### Issue 1: MCP Server Fails to Start

**Symptom**: Client shows connection failure

**Solution**:
1. Check if codex is installed: `which codex`
2. Check Go version: `go version` (requires 1.24+)
3. Check binary permissions: `chmod +x codex4kilomcp`
4. Manual test: `./codex4kilomcp`

### Issue 2: Codex Command Execution Failed

**Symptom**: Returns "codex command failed"

**Solution**:
1. Check Codex CLI configuration: `codex --help`
2. Check API key is set: `echo $OPENAI_API_KEY`
3. Check working directory exists: `ls -la /path/to/project`

### Issue 3: Empty SESSION_ID

**Symptom**: Returns "Failed to get SESSION_ID"

**Solution**:
1. Check Codex version supports `--json` output
2. Check network connectivity
3. Try setting `return_all_messages: true` to see detailed errors

### Issue 4: Insufficient Permissions

**Symptom**: Sandbox policy restrictions cause operation failures

**Solution**:
1. Use `sandbox: "workspace-write"` to allow writes
2. Or use `yolo: true` to completely skip sandbox (not recommended)
3. Check file and directory permissions: `ls -la`

---

## 7. Advanced Configuration

### Environment Variables

```bash
# Set log level
export CODEX4KILOMCP_LOG_LEVEL=debug

# Set timeout (seconds)
export CODEX4KILOMCP_TIMEOUT=300
```

### Configuration File Example

Create `~/.codex4kilomcp/config.toml`:

```toml
[server]
timeout = 300
log_level = "info"

[defaults]
sandbox = "read-only"
skip_git_repo_check = true
return_all_messages = false
```

---

## 8. Development & Contribution

### Project Structure

```
codex4kilomcp/
├── cmd/server/         # Main program entry
├── internal/
│   ├── mcp/           # MCP server implementation
│   └── codex/         # Codex client wrapper
├── go.mod             # Go module definition
└── README.md          # This documentation
```

### Development Setup

```bash
# Clone repository
git clone https://github.com/your-repo/codex4kilomcp.git
cd codex4kilomcp

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o codex4kilomcp cmd/server/main.go
```

### Contribution Guidelines

- Follow [Conventional Commits](https://www.conventionalcommits.org/)
- Run tests before committing: `go test ./...`
- Update documentation

---

## 9. License

This project is licensed under the [MIT License](LICENSE).

Copyright (c) 2025 [guda.studio](mailto:gudaclaude@gmail.com)

---

## 10. Acknowledgments

- [OpenAI Codex](https://github.com/openai/codex) - Powerful programming assistant
- [Model Context Protocol](https://modelcontextprotocol.io) - Unified AI tool protocol
- [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk) - Go language MCP implementation

---

<div align="center">

**⭐ Star this project to show your support!**

</div>
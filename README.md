# codex-mcp-go

<div align="center">

**Codex CLI 的 MCP 协议封装实现**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/dl/) [![MCP Compatible](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io)

⭐ **如果觉得好用，请给个 Star 吧！您的支持是我们更新的动力~** ⭐

[English](./README_EN.md) | 简体中文

</div>

---

## 简介

`codex-mcp-go` 是一个基于 Go 语言实现的 MCP (Model Context Protocol) 服务器。它封装了 OpenAI 的 Codex CLI，使其能够作为 MCP 工具被 Claude Code、Roo Code、KiloCode 等 AI 客户端调用。

主要特性：
- **会话管理**：支持 `SESSION_ID` 维持多轮对话上下文。
- **沙箱控制**：提供 `read-only`、`workspace-write` 等安全策略。
- **并发支持**：基于 Go 协程，支持多客户端并发调用。
- **单文件部署**：编译为单一二进制文件，无运行时依赖。

---

## 快速开始

### 1. 安装

#### 方式一：使用 npx (推荐)

无需安装 Go 环境，直接运行：

```bash
npx @w31r4/codex-mcp-go
```

#### 方式二：手动下载

从 [Releases](https://github.com/w31r4/codex-mcp-go/releases) 页面下载对应平台的二进制文件。

#### 方式三：源码构建

需要 Go 1.24+ 环境。

```bash
git clone https://github.com/w31r4/codex-mcp-go.git
cd codex-mcp-go
go build -o codex-mcp-go cmd/server/main.go
```

### 2. 配置 MCP 客户端

根据您使用的 AI 客户端，选择对应的配置方式。

<details>
<summary><strong>Claude Code</strong></summary>

```bash
claude mcp add codex -s user --transport stdio -- npx -y @w31r4/codex-mcp-go
```
</details>

<details>
<summary><strong>Roo Code (VSCode / Cursor)</strong></summary>

在 Roo Code 的 MCP 设置中添加：

```json
{
  "mcpServers": {
    "codex": {
      "command": "npx",
      "args": ["-y", "@w31r4/codex-mcp-go"],
      "env": {
        "OPENAI_API_KEY": "your-api-key"
      }
    }
  }
}
```

配置文件路径参考：
- VSCode: `~/.config/Code/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json`
- Cursor: `~/.config/Cursor/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json`
</details>

<details>
<summary><strong>KiloCode</strong></summary>

在 `~/.kilocode/mcp.json` 中添加：

```json
{
  "mcpServers": {
    "codex": {
      "command": "npx",
      "args": ["-y", "@w31r4/codex-mcp-go"],
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

1. 打开 Cursor 设置 -> Features -> MCP
2. 点击 "Add New MCP Server"
3. 填写配置：
   - Name: `codex`
   - Type: `stdio`
   - Command: `npx`
   - Args: `-y @w31r4/codex-mcp-go`
</details>

<details>
<summary><strong>通用配置 (JSON)</strong></summary>

适用于其他支持 MCP 的客户端：

```json
{
  "mcpServers": {
    "codex": {
      "command": "npx",
      "args": ["-y", "@w31r4/codex-mcp-go"],
      "env": {
        "OPENAI_API_KEY": "your-api-key"
      }
    }
  }
}
```
</details>

### 3. 验证

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./codex-mcp-go
```
若返回包含 `codex` 工具的 JSON 数据，即表示运行正常。

---

## 工具参数说明

工具名称：`codex`

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `PROMPT` | `string` | ✅ | - | 发送给 Codex 的指令 |
| `cd` | `string` | ✅ | - | 工作目录路径 |
| `sandbox` | `string` | ❌ | `"read-only"` | 策略：`read-only` / `workspace-write` / `danger-full-access` |
| `SESSION_ID` | `string` | ❌ | `""` | 会话 ID，用于多轮对话 |
| `skip_git_repo_check` | `bool` | ❌ | `true` | 允许在非 Git 目录运行 |
| `return_all_messages` | `bool` | ❌ | `false` | 返回完整推理日志 |
| `image` | `[]string` | ❌ | `[]` | 附加图片路径 |
| `model` | `string` | ❌ | `""` | 指定模型 |
| `yolo` | `bool` | ❌ | `false` | 跳过所有确认（慎用） |
| `profile` | `string` | ❌ | `""` | 指定配置文件 |

---

## 版本对比 (Go vs Python)

| 特性 | Go 版本 (codex-mcp-go) | Python 版本 (codexmcp) |
|------|------------------------|----------------------|
| **部署** | 单二进制文件，零依赖 | 需 Python 环境及依赖 |
| **启动速度** | 🚀 极快 | 🐢 较慢 (解释器启动) |
| **资源占用** | 📉 低 | 📈 较高 |
| **并发模型** | Goroutine (高效) | Threading |
| **适用场景** | 生产环境、底层服务 | 二次开发、原型验证 |

---

## 故障排查

*   **连接失败**：检查 `codex` CLI 是否在 PATH 中，或确认 Go 版本 >= 1.24。
*   **无权限**：检查二进制文件是否有执行权限 (`chmod +x`)。
*   **Session 丢失**：确保客户端正确传递了上一次调用返回的 `SESSION_ID`。

---

## 致谢

本项目深受 [codexmcp](https://github.com/GuDaStudio/codexmcp) (Python 实现) 的启发。感谢 GuDaStudio 团队在探索 Codex MCP 集成方面所做的开创性工作。

---

<div align="center">

**再次感谢您的关注！别忘了点个 Star 哦~ 🌟**

</div>

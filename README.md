# codex-mcp-go

<div align="center">

**Codex CLI 的 MCP 协议封装实现**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/dl/) [![MCP Compatible](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io) [![NPM Version](https://img.shields.io/npm/v/@zenfun510/codex-mcp-go)](https://www.npmjs.com/package/@zenfun510/codex-mcp-go)

⭐ **如果觉得好用，请给个 Star 吧！您的支持是我们更新的动力~** ⭐

[English](./README_EN.md) | 简体中文

</div>

---

## 简介

`codex-mcp-go` 是一个基于 Go 语言实现的 MCP (Model Context Protocol) 服务器。它封装了 OpenAI 的 Codex CLI，使其能够作为 MCP 工具被 Claude Code、Roo Code、KiloCode 等 AI 客户端调用。
codex 在细节和 bug 修复方面的能力有目共睹, 但是很多时候用起来会感觉稍微缺乏全局视野, 所以我目前的工作流是使用 gemini 3.0 pro 的 kilocode 作为主要的规划者, codex 可以负责复杂功能的落地和 bug 修复

主要特性：
- **会话管理**：支持 `SESSION_ID` 维持多轮对话上下文。
- **沙箱控制**：提供 `read-only`、`workspace-write` 等安全策略。
- **并发支持**：基于 Go 协程，支持多客户端并发调用。
- **单文件部署**：编译为单一二进制文件，无运行时依赖。

---

## 快速开始

### 1. 前置要求

本工具依赖 OpenAI 的 `codex` CLI。请确保您已安装并配置好它。

**安装 Codex CLI:**

```bash
# 使用 npm 安装 (推荐)
npm i -g @openai/codex

# 或者参考官方仓库
# https://github.com/openai/codex-cli
```

### 2. 安装 MCP Server

#### 方式一：使用 npx (推荐)

无需安装 Go 环境，直接运行：

```bash
npx @zenfun510/codex-mcp-go
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

### 3. 配置 MCP 客户端

根据您使用的 AI 客户端，选择对应的配置方式。

#### 方式 A：使用 npx (推荐)

<details>
<summary><strong>Claude Code</strong></summary>

```bash
claude mcp add codex -s user --transport stdio -- npx -y @zenfun510/codex-mcp-go
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
      "args": ["-y", "@zenfun510/codex-mcp-go"],
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
      "args": ["-y", "@zenfun510/codex-mcp-go"],
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
   - Args: `-y @zenfun510/codex-mcp-go`
</details>

#### 方式 B：使用本地二进制文件

如果您已通过 `go build` 构建了二进制文件（假设路径为 `/path/to/codex-mcp-go`），可使用以下配置：

<details>
<summary><strong>Claude Code</strong></summary>

```bash
claude mcp add codex -s user --transport stdio -- /path/to/codex-mcp-go
```
</details>

<details>
<summary><strong>Roo Code / KiloCode / 通用 JSON 配置</strong></summary>

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

1. 打开 Cursor 设置 -> Features -> MCP
2. 点击 "Add New MCP Server"
3. 填写配置：
   - Name: `codex`
   - Type: `stdio`
   - Command: `/path/to/codex-mcp-go`
   - Args: (留空)
</details>

### 4. 验证

```bash
cat <<'EOF' | ./codex-mcp-go
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"0.1.0","capabilities":{}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
EOF
```
需先完成 `initialize` 握手，然后才能调用 `tools/list`。若返回包含 `codex` 工具的 JSON 数据，即表示运行正常。

---

## 推荐的系统提示词 (System Prompts)

为了获得最佳体验，建议根据您使用的客户端类型配置相应的系统提示词。

### 1. 智能体模式

适用于能够自主规划和执行多步任务的 Agent。请将以下内容添加到您的 Agent 配置或作为任务的初始指令：

<details>
<summary><strong>点击展开详细提示词</strong></summary>

```text
## Core Instruction for CodeX MCP

在任何时刻，你必须思考当前过程可以如何与codex进行协作，如何调用Codex 为你提供的MCP工具作为你客观全面分析的保障。
其中你**务必执行**以下几个步骤：
**1** 在你对用户需求形成初步分析后，将用户需求、初始思路告知codex，并要求其完善需求分析和实施计划。
**2** 在实施具体编码任务前，**必须向codex索要代码实现原型（要求codex仅给出unified diff patch，严禁对代码做任何真实修改）**。在获取代码原型后，你**只能以此为逻辑参考，再次对代码修改进行重写**，形成企业生产级别、可读性极高、可维护性极高的代码后，才能实施具体编程修改任务。
**3** 无论何时，只要完成切实编码行为后，**必须立即使用codex review代码改动和对应需求完成程度**。
**4** codex只能给出参考，你**必须有自己的思考，甚至需要对codex的回答提出置疑**。尽信书则不如无书，你与codex的最终使命都是达成统一、全面、精准的意见，所以你们必须不断争辩已找到通向真理的唯一途径。

## Codex Tool Invocation Specification

1. 工具概述
   codex MCP 提供了一个工具 `codex`，用于执行 AI 辅助的编码任务。该工具**通过 MCP 协议调用**，无需使用命令行。

2. 工具参数
   **必选**参数：
   - PROMPT (string): 发送给 codex 的任务指令
   - cd (Path): codex 执行任务的工作目录根路径

   可选参数：
   - sandbox (string): 沙箱策略，可选值：
     - "read-only" (默认): 只读模式，最安全
     - "workspace-write": 允许在工作区写入
     - "danger-full-access": 完全访问权限
   - SESSION_ID (UUID | null): 用于继续之前的会话以与codex进行多轮交互，默认为 None（开启新会话）
   - skip_git_repo_check (boolean): 是否允许在非 Git 仓库中运行，默认 False
   - return_all_messages (boolean): 是否返回所有消息（包括推理、工具调用等），默认 False
   - image (List[Path] | null): 附加一个或多个图片文件到初始提示词，默认为 None
   - model (string | null): 指定使用的模型，默认为 None（使用用户默认配置）
   - yolo (boolean | null): 无需审批运行所有命令（跳过沙箱），默认 False
   - profile (string | null): 从 `~/.codex/config.toml` 加载的配置文件名称，默认为 None（使用用户默认配置）

3. 调用规范
   **必须遵守**：
   - 每次调用 codex 工具时，必须保存返回的 SESSION_ID，以便后续继续对话
   - cd 参数必须指向存在的目录，否则工具会静默失败
   - 严禁codex对代码进行实际修改，使用 sandbox="read-only" 以避免意外，并要求codex仅给出unified diff patch即可

   推荐用法：
   - 如需详细追踪 codex 的推理过程和工具调用，设置 return_all_messages=True
   - 对于精准定位、debug、代码原型快速编写等任务，优先使用 codex 工具
```
</details>

### 2. 辅助编程模式

适用于作为 IDE 插件运行的助手。建议添加到 `.clinerules` (Roo Code) 或 "Rules for AI" (Cursor) 中：

<details>
<summary><strong>点击展开规则提示词</strong></summary>

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
| `model` | `string` | ❌ | `""` | 默认禁止，除非显式允许 |
| `yolo` | `bool` | ❌ | `false` | 跳过所有确认（非交互） |
| `profile` | `string` | ❌ | `""` | 默认禁止，除非显式允许 |
| `timeout_seconds` | `int` | ❌ | `1800` | Codex 调用的总超时（秒，最多 1800） |
| `no_output_seconds` | `int` | ❌ | `0` | 无输出达到该秒数后终止运行（0 表示关闭） |

**运行时行为：** 默认 30 分钟总超时（上限 30 分钟），无输出看门狗默认关闭；出现错误行、非零退出会携带最近输出返回，便于定位卡住原因。若网络慢或 MCP 客户端自身有较短的 RPC 超时，调用时保持 `timeout_seconds=1800`，以避免过早被取消。
**默认策略：** `sandbox=read-only`、`yolo=false`、`skip_git_repo_check=false`；`model/profile` 默认拒绝，需显式放行；`timeout_seconds=1800`（最多 1800）、`no_output_seconds=0`（关闭）。

---

## 功能对比

### 1. 与官方 Codex CLI 对比

| 特性 | 官方 Codex CLI | CodexMCP (本工具) |
|------|----------------|-------------------|
| **基本 Codex 调用** | ✅ | ✅ |
| **多轮对话** | ❌ | ✅ (通过 Session 管理) |
| **推理详情追踪** | ❌ | ✅ (完整日志捕获) |
| **并行任务支持** | ❌ | ✅ (MCP 协议支持) |
| **错误处理** | ❌ | ✅ (结构化错误返回) |

### 2. 与 Python 版本 (codexmcp) 对比

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

## 开源协议

本项目采用 [MIT License](./LICENSE) 开源协议。

---

## 致谢

本项目深受 [codexmcp](https://github.com/GuDaStudio/codexmcp) (Python 实现) 的启发。感谢 GuDaStudio 团队在探索 Codex MCP 集成方面所做的开创性工作。

---

<div align="center">

**再次感谢您的关注！别忘了点个 Star 哦~ 🌟**

</div>

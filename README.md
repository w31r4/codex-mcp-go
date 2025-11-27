# Codex4KiloMCP

<div align="center">

**让 AI 编程助手与 Codex 无缝协作**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/dl/) [![MCP Compatible](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io)

</div>

---

## 一、项目简介

Codex4KiloMCP 是一个基于 Go 语言实现的 MCP (Model Context Protocol) 服务器，作为 Codex CLI 的桥梁，让各种 AI 编程助手（如 Claude Code、Roo Code、KiloCode 等）能够与 Codex 无缝协作。

在当前 AI 辅助编程生态中：
- **AI 编程助手**（Claude Code/Roo Code/KiloCode）：负责架构设计、需求分析、代码重构
- **Codex**：负责代码生成、bug 定位、代码审查
- **Codex4KiloMCP**：管理会话上下文，支持多轮对话与并行任务

相比官方 Codex MCP 实现，Codex4KiloMCP 引入了**会话持久化**、**并行执行**和**推理追踪**等企业级特性。

---

## 二、快速开始

### 0. 前置要求

请确保您已成功**安装**和**配置**以下工具：

- [Codex CLI](https://developers.openai.com/codex/quickstart) - OpenAI 的编程助手
- [Go 1.24+](https://golang.org/dl/) - Go 语言环境
- 支持 MCP 的 AI 客户端（Claude Code、Roo Code、KiloCode 等）

### 1. 安装步骤

#### 1.1 构建项目

```bash
# 克隆仓库
git clone https://github.com/your-repo/codex4kilomcp.git
cd codex4kilomcp

# 构建二进制文件
go build -o codex4kilomcp cmd/server/main.go
```

#### 1.2 配置 MCP 客户端

根据您使用的 AI 客户端，选择对应的配置方式：

<details>
<summary><strong>Claude Code 配置</strong></summary>

```bash
# 移除官方 Codex MCP（如果已安装）
claude mcp remove codex

# 添加 Codex4KiloMCP
claude mcp add codex -s user --transport stdio -- /path/to/codex4kilomcp
```

验证安装：
```bash
claude mcp list
# 应显示: codex: /path/to/codex4kilomcp - ✓ Connected
```

</details>

<details>
<summary><strong>Roo Code 配置</strong></summary>

在 Roo Code 的 MCP 配置中添加：

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

配置路径：
- VSCode: `~/.config/Code/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json`
- Cursor: `~/.config/Cursor/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json`

</details>

<details>
<summary><strong>KiloCode 配置</strong></summary>

在 KiloCode 的 MCP 配置中添加：

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

配置路径：`~/.kilocode/mcp.json`

</details>

<details>
<summary><strong>其他 MCP 兼容客户端</strong></summary>

通用配置格式：

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

#### 1.3 验证安装

运行以下命令测试 MCP 服务器：

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | /path/to/codex4kilomcp
```

应返回可用的工具列表，包含 `codex` 工具。

---

## 三、工具说明

### codex 工具

执行非交互式 Codex 会话，完成 AI 辅助编程任务。

#### 参数说明

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `PROMPT` | `string` | ✅ | - | 发送给 Codex 的任务指令 |
| `cd` | `string` | ✅ | - | Codex 工作目录根路径 |
| `sandbox` | `string` | ❌ | `"read-only"` | 沙箱策略：`read-only` / `workspace-write` / `danger-full-access` |
| `SESSION_ID` | `string` | ❌ | `""` | 会话 ID（空则开启新会话） |
| `skip_git_repo_check` | `bool` | ❌ | `true` | 是否允许在非 Git 仓库运行 |
| `return_all_messages` | `bool` | ❌ | `false` | 是否返回完整推理信息 |
| `image` | `[]string` | ❌ | `[]` | 附加图片文件到初始提示词 |
| `model` | `string` | ❌ | `""` | 指定使用的模型 |
| `yolo` | `bool` | ❌ | `false` | 无需审批运行所有命令（跳过沙箱） |
| `profile` | `string` | ❌ | `""` | 配置文件名称 |

#### 返回值

**成功时：**
```json
{
  "success": true,
  "SESSION_ID": "550e8400-e29b-41d4-a716-446655440000",
  "agent_messages": "Codex 的回复内容...",
  "all_messages": [...]  // 仅当 return_all_messages=true 时包含
}
```

**失败时：**
```json
{
  "success": false,
  "error": "错误信息描述"
}
```

---

## 四、使用示例

### 示例 1：代码审查

```json
{
  "PROMPT": "Review the code in src/main.go and suggest improvements",
  "cd": "/path/to/project",
  "sandbox": "read-only",
  "return_all_messages": true
}
```

### 示例 2：生成测试代码

```json
{
  "PROMPT": "Generate unit tests for the calculate function in math.go",
  "cd": "/path/to/project",
  "sandbox": "workspace-write"
}
```

### 示例 3：多轮对话

第一轮：
```json
{
  "PROMPT": "Help me design a REST API for user management",
  "cd": "/path/to/project"
}
```

第二轮（使用返回的 SESSION_ID）：
```json
{
  "PROMPT": "Now implement the authentication middleware",
  "cd": "/path/to/project",
  "SESSION_ID": "550e8400-e29b-41d4-a716-446655440000"
}
```

---

## 五、Go 版本 vs Python 版本

| 特性 | Go 版本 (codex4kilomcp) | Python 版本 (codexmcp) |
|------|------------------------|----------------------|
| **核心功能** | ✅ 完整支持 | ✅ 完整支持 |
| **会话持久化** | ✅ 支持 | ✅ 支持 |
| **推理追踪** | ✅ 支持 | ✅ 支持 |
| **并行执行** | ✅ 支持 | ✅ 支持 |
| **错误处理** | ✅ 增强 | ✅ 标准 |
| **性能** | ⚡ 更高（编译型） | 🐍 良好（解释型） |
| **内存占用** | 📦 更低 | 📦 较高 |
| **启动速度** | 🚀 更快 | 🚶 较慢 |
| **跨平台** | ✅ Windows/Linux/macOS | ✅ Windows/Linux/macOS |
| **依赖管理** | go.mod | uv/pip |
| **适用场景** | 生产环境、资源受限 | 开发环境、快速迭代 |

**选择建议：**
- **Go 版本**：适合生产部署、资源敏感环境、需要高性能场景
- **Python 版本**：适合快速开发、Python 生态集成、原型验证

---

## 六、故障排查

### 问题 1：MCP 服务器无法启动

**症状**：客户端显示连接失败

**解决方案**：
1. 检查 codex 是否已安装：`which codex`
2. 检查 Go 版本：`go version`（需要 1.24+）
3. 检查二进制文件权限：`chmod +x codex4kilomcp`
4. 手动测试：`./codex4kilomcp`

### 问题 2：Codex 命令执行失败

**症状**：返回 "codex command failed"

**解决方案**：
1. 检查 Codex CLI 是否配置正确：`codex --help`
2. 检查 API 密钥是否设置：`echo $OPENAI_API_KEY`
3. 检查工作目录是否存在：`ls -la /path/to/project`

### 问题 3：SESSION_ID 为空

**症状**：返回 "Failed to get SESSION_ID"

**解决方案**：
1. 检查 Codex 版本是否支持 `--json` 输出
2. 检查网络连接是否正常
3. 尝试设置 `return_all_messages: true` 查看详细错误

### 问题 4：权限不足

**症状**：沙箱策略限制导致操作失败

**解决方案**：
1. 使用 `sandbox: "workspace-write"` 允许写入
2. 或使用 `yolo: true` 完全跳过沙箱（不推荐）
3. 检查文件和目录权限：`ls -la`

---

## 七、高级配置

### 环境变量

```bash
# 设置日志级别
export CODEX4KILOMCP_LOG_LEVEL=debug

# 设置超时时间（秒）
export CODEX4KILOMCP_TIMEOUT=300
```

### 配置文件示例

创建 `~/.codex4kilomcp/config.toml`：

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

## 八、开发与贡献

### 项目结构

```
codex4kilomcp/
├── cmd/server/         # 主程序入口
├── internal/
│   ├── mcp/           # MCP 服务器实现
│   └── codex/         # Codex 客户端封装
├── go.mod             # Go 模块定义
└── README.md          # 本文档
```

### 开发环境搭建

```bash
# 克隆仓库
git clone https://github.com/your-repo/codex4kilomcp.git
cd codex4kilomcp

# 安装依赖
go mod download

# 运行测试
go test ./...

# 构建
go build -o codex4kilomcp cmd/server/main.go
```

### 提交规范

- 遵循 [Conventional Commits](https://www.conventionalcommits.org/)
- 提交前运行测试：`go test ./...`
- 更新文档

---

## 九、许可证

本项目采用 [MIT License](LICENSE) 开源协议。

Copyright (c) 2025 [guda.studio](mailto:gudaclaude@gmail.com)

---

## 十、致谢

- [OpenAI Codex](https://github.com/openai/codex) - 强大的编程助手
- [Model Context Protocol](https://modelcontextprotocol.io) - 统一的 AI 工具协议
- [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk) - Go 语言 MCP 实现

---

<div align="center">

**用 🌟 为本项目助力~**

</div>
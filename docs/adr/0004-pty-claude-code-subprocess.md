# ADR 0004: Relay 通过 PTY 子进程操控 Claude Code CLI

## 状态

提议

## 上下文

Relay Daemon 需要让手机端对电脑上运行的 Claude Code 会话进行读写。核心问题：Relay 如何与 Claude Code 通信？

两个候选方案：

### 方案 A：PTY 子进程（推荐）
Relay 用伪终端（Pseudo-Terminal）启动 `claude` CLI 作为子进程，通过 pty 的 stdin/stdout 交互。
- 手机消息 → WebSocket → Relay → pty stdin → Claude Code 进程
- Claude Code 输出 → pty stdout → Relay 解析 → WebSocket → 手机
- Session 文件由 Claude Code 自己管理

### 方案 B：直接调 API
Relay 绕过 Claude Code，直接调用 Anthropic API（或 OpenRouter 等）。
- 手机消息 → Relay → API → 流式响应 → 手机
- 需要自己在 Relay 中实现所有 Claude Code 功能（工具调用、文件读写、LSP 等）

## 决策

采用 **方案 A（PTY 子进程）**。

## 理由

1. **零功能重实现** — Claude Code 的所有能力（代码编辑、文件操作、Terminal 执行、LSP 搜索）通过子进程自动获得，Relay 只需转发文本流
2. **一致性** — 手机端看到的响应与桌面端完全一致，包括 Markdown 渲染、代码块等
3. **项目上下文自动加载** — Claude Code 启动时自动加载 `.claude/` 下的项目配置、规则等
4. **Session 管理零额外工作** — Claude Code 自己写 session JSON，Relay 只需读目录

方案 B 的代价（重实现 Claude Code 所有工具链）远大于方案 A 的代价（处理 PTY 流）。

## 后果

- Relay 在 Windows 上需使用 ConPTY API（`microsoft/go-winio` 或 `github.com/creack/pty`）
- 需要解析 Claude Code 输出的 ANSI escape codes（`github.com/acarl005/stripansi`）
- 子进程崩溃/挂起时需有看门狗（watchdog）自动恢复
- PTY 不支持 Windows 的某些边缘场景（如旧版 Windows 10 的 ConPTY bug）

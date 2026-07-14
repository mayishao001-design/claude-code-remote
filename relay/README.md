# Claude Code Remote — Relay Daemon

手机远程遥控电脑上 Claude Code 的中转服务。

## 架构

```
[iPhone IPA] ←→ Tailscale ←→ [Relay Daemon] ←PTY→ [Claude Code CLI]
```

- **Relay**: Go 写的常驻 Windows 服务，提供 REST + WebSocket API
- **连接**: 通过 Tailscale 加密 mesh VPN，手机在外用 4G/5G 直连电脑
- **鉴权**: Bearer Token（首次启动自动生成）
- **通信**: REST 查列表/详情，WebSocket 流式收发消息

## 快速开始

### 前置条件

1. [Go 1.22+](https://go.dev/dl/) — 编译 Relay
2. [Claude Code](https://claude.ai/code) — 电脑上已安装 `claude` CLI
3. [Tailscale](https://tailscale.com/download) — 手机和电脑都装上，登录同账号

### 编译运行

```bash
cd relay
go mod tidy
go build -o relay.exe ./cmd/relay

# 首次运行
./relay.exe
```

首次启动会在 `~/.claude-remote/` 下生成：
- `auth.json` — Bearer Token（手机连接时需要）
- `projects.json` — 空项目模板

### 配置项目

编辑 `~/.claude-remote/projects.json`：

```json
[
  { "name": "后端API", "path": "D:/projects/api-server" },
  { "name": "前端App", "path": "D:/projects/mobile-app" },
  { "name": "博客",    "path": "D:/projects/blog" }
]
```

### 连接手机

1. 电脑查 Tailscale IP：`tailscale ip -4` → `100.x.x.x`
2. 手机 IPA 填入地址：`http://100.x.x.x:9943`
3. 填入 `auth.json` 中的 Token
4. 看到项目列表 → 选项目 → 选 session → 开聊

## REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/health` | 健康检查 |
| `GET` | `/api/v1/projects` | 项目列表 |
| `GET` | `/api/v1/sessions` | 会话列表（`?archived=true&project=xxx`） |
| `GET` | `/api/v1/sessions/:id` | 会话详情 |
| `POST` | `/api/v1/sessions/:id/interrupt` | 中断生成 |
| `DELETE` | `/api/v1/sessions/:id` | 删除会话 |
| `WS` | `/api/v1/ws?token=xxx` | WebSocket 实时通信 |

## WebSocket 协议

手机 → Relay：

```json
{"type":"send_message", "session_id":"abc", "text":"继续写"}
{"type":"start_session", "project":"后端API", "text":"初始提示"}
{"type":"interrupt", "session_id":"abc"}
{"type":"ping"}
```

Relay → 手机：

```json
{"type":"stream_chunk", "session_id":"abc", "text":"正在分析..."}
{"type":"stream_end", "session_id":"abc"}
{"type":"stream_error", "session_id":"abc", "error":"进程崩溃"}
{"type":"pong"}
```

## 项目结构

```
relay/
├── cmd/relay/main.go              # 入口
├── internal/
│   ├── api/
│   │   ├── router.go              # Gin 路由 + 鉴权中间件
│   │   ├── handlers.go            # REST 处理器
│   │   └── ws.go                  # WebSocket 处理器
│   ├── claude/
│   │   ├── session.go             # 会话文件读写
│   │   ├── process.go             # PTY 子进程管理
│   │   └── stream.go              # 流式缓冲区
│   ├── config/
│   │   └── config.go              # projects + auth 配置
│   └── relay/
│       └── relay.go               # 核心调度
└── go.mod
```

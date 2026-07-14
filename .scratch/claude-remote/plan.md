# Claude Code Remote — 设计方案

手机遥控电脑上 Claude Code 的完整架构与实施计划。

---

## 一、系统总览

```
┌──────────────────────────────────────────────────────────────────────┐
│ iPhone (TrollStore IPA / SwiftUI)                                     │
│                                                                       │
│  ProjectList → SessionList → ChatView (流式消息 + 中断)               │
│                              ↑                                       │
│                     [WebSocket / REST]   ▲ Tailscale (100.x.x.x)     │
│                                          │ 加密 + 设备身份            │
│                              ↓           │ Bearer Token 鉴权          │
│  PC (Relay Daemon / Go) ─────────────────┘                           │
│                                                                       │
│  ┌─ HTTP Server ──┬─ REST: GET /api/v1/* ───────────────┐            │
│  │ (gin / fiber)  └─ WS:   /api/v1/ws (流式 + 控制)    │            │
│  └───────────────────────┬──────────────────────────────┘            │
│                          ↓                                           │
│  ┌──────────────────────┴──────────────────────────────┐             │
│  │ Session Manager                                       │            │
│  │ • 解析 .claude/sessions/*.json → SessionSummary       │           │
│  │ • PTY 子进程管理 claude CLI                           │           │
│  │ • stdin 注入消息 / stdout 捕获流式输出                │           │
│  │ • SIGINT 中断 / 进程看门狗                            │           │
│  └──────────────────────┬──────────────────────────────┘             │
│                          ↓                                           │
│  ┌──────────────────────┴──────────────────────────────┐             │
│  │ Project Config (~/.claude-remote/projects.json)       │            │
│  │ [{name:"后端", path:"D:/project/api"}, ...]          │            │
│  └─────────────────────────────────────────────────────┘             │
│                          ↓                                           │
│              ┌─ Claude Code CLI (子进程 PTY) ─┐                      │
│              │ claude ←→ 工具调用 ←→ 代码编辑   │                     │
│              │ session → .claude/sessions/*     │                     │
│              └──────────────────────────────────┘                     │
└──────────────────────────────────────────────────────────────────────┘
```

## 二、核心数据模型

### Project（预配项目）

```json
{
  "name": "后端API",
  "path": "D:/projects/api-server",
  "emoji": "🔧"
}
```

配置保存在 `~/.claude-remote/projects.json`，启动时加载，手机端只读列表。

### Session（Claude Code 会话）

来源：`.claude/sessions/<session-id>.json`，由 Claude Code 自己管理。
Relay 负责读取并解析为手机端可显示的摘要。

```json
{
  "id": "abc123",
  "project": "后端API",
  "title": "修复登录接口超时",
  "messageCount": 12,
  "lastMessageAt": "2026-07-14T10:30:00+08:00",
  "archived": false
}
```

手机端消息模型：

```json
{
  "role": "user" | "assistant",
  "text": "消息内容（Markdown）",
  "timestamp": "..."
}
```

### Stream Chunk（流式帧）

```json
{
  "type": "stream_chunk",
  "session_id": "abc123",
  "text": "经过分析..."
}
```

## 三、Relay API 规范

### REST 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/health` | 健康检查 `{"status":"ok"}` |
| `GET` | `/api/v1/projects` | 项目列表 |
| `GET` | `/api/v1/sessions` | 会话列表（含 project 过滤参数 `?project=xxx`） |
| `GET` | `/api/v1/sessions/:id` | 会话详情（消息历史） |
| `POST` | `/api/v1/sessions/:id/interrupt` | 中断当前生成 |
| `DELETE` | `/api/v1/sessions/:id` | 删除会话 |

### WebSocket (`/api/v1/ws`)

**手机→Relay：**

```json
{"type":"send_message",  "session_id":"abc123", "text":"继续写"}
{"type":"start_session", "project":"后端API",   "text":"初始提示"}
{"type":"interrupt",     "session_id":"abc123"}
{"type":"ping"}
```

**Relay→手机：**

```json
{"type":"stream_chunk",  "session_id":"abc123", "text":"正在分析..."}
{"type":"stream_end",    "session_id":"abc123"}
{"type":"stream_error",  "session_id":"abc123", "error":"进程崩溃"}
{"type":"session_updated","session":{...}}
{"type":"pong"}
```

### 鉴权

- Relay 首次启动生成随机 Bearer Token（32 字节 hex）
- 存于 `~/.claude-remote/auth.json`
- 手机首次连接需扫描二维码或手动输入 token
- 所有请求带 `Authorization: Bearer <token>`

## 四、Relay Daemon 实现（Go）

### 项目结构

```
relay/
├── cmd/relay/main.go           # 入口：启动 HTTP 服务
├── internal/
│   ├── api/
│   │   ├── router.go           # Gin 路由注册
│   │   ├── middleware.go       # 鉴权中间件 + CORS + 日志
│   │   ├── projects.go         # GET /projects 处理
│   │   ├── sessions.go         # GET /sessions 处理
│   │   └── ws.go               # WebSocket 升级 + 消息分发
│   ├── claude/
│   │   ├── process.go          # PTY 子进程管理（启动/停止/看门狗）
│   │   ├── session.go          # 读取/解析 .claude/sessions/*.json
│   │   └── stream.go           # stdout 捕获 → chunk 回调
│   ├── config/
│   │   └── config.go           # 项目配置 + auth 读写
│   └── relay/
│       └── relay.go            # 核心：session manager + 消息路由
├── go.mod
└── go.sum
```

### 关键依赖

- `github.com/gin-gonic/gin` — HTTP + WebSocket 框架
- `github.com/gorilla/websocket` — WebSocket 升级
- `github.com/creack/pty` — PTY 子进程管理（Windows 用 ConPTY）
- `github.com/acarl005/stripansi` — ANSI 转义码剥离
- `github.com/fsnotify/fsnotify` — 监听 session 目录变化

### PTY 子进程管理核心逻辑

```
StartSession(project):
  1. 切换到 project.path
  2. pty.Start("claude") → 获取 pty fd
  3. goroutine: 读取 pty.stdout → stripANSI → 按行/块推送 WebSocket
  4. 记录子进程 PID

SendMessage(session_id, text):
  1. 写入 pty.stdin: text 换行
  2. 开始流式转发

Interrupt(session_id):
  1. 发送 SIGINT 到子进程
  2. 等待进程自行退出或 5s 后强制 kill
  3. 通知手机端 stream_end

Watchdog:
  每 30s 检查子进程是否存活
  崩溃或卡顿时 → 通知手机 → 自动重启进程 + 恢复上下文
```

## 五、IPA 实现（SwiftUI）

### 最低要求

- iOS 16.0+（保证 SwiftUI 完整支持）
- TrollStore 安装（无 App Store 签名限制）

### 项目结构

```
CCRemote/
├── App/
│   ├── CCRemoteApp.swift          # @main，Root View
│   ├── ContentView.swift          # 主 TabView / NavigationStack
│   └── AppState.swift             # 全局应用状态（@Observable）
├── Models/
│   ├── Project.swift              # Codable，对应 Relay 返回
│   ├── SessionListItem.swift      # 会话摘要
│   ├── Session.swift              # 会话详情（含消息数组）
│   └── StreamChunk.swift          # WebSocket 帧模型
├── ViewModels/
│   ├── ConnectionViewModel.swift  # Relay 地址 + Token + 连接状态
│   ├── ProjectViewModel.swift     # 项目列表加载
│   ├── SessionListViewModel.swift # 会话列表 + 轮询刷新
│   └── ChatViewModel.swift        # WebSocket 管理 + 消息发送 + 流式接收
├── Views/
│   ├── SetupView.swift            # 首次引导：输入 Relay 地址 + Token
│   ├── ProjectListView.swift      # 项目列表（带有 emoji 图标）
│   ├── SessionListView.swift      # 会话列表（标题 + 时间 + 消息数）
│   ├── ChatView.swift             # 消息流 + 输入框 + 中断按钮
│   ├── MessageBubble.swift        # 单条消息渲染（支持 Markdown）
│   └── ConnectionBadge.swift      # 顶部连接状态指示器
├── Services/
│   ├── RelayAPI.swift             # URLSession + async/await HTTP 请求
│   ├── WebSocketManager.swift     # WebSocketKit / Starscream 封装
│   └── NetworkMonitor.swift       # NWPathMonitor 网络可达性
├── Utils/
│   ├── KeychainHelper.swift       # Keychain 存储地址和 Token
│   └── DateFormatter.swift        # 相对时间格式化
└── Resources/
    ├── Assets.xcassets/
    └── Info.plist
```

### 核心流程

```
App启动 → Keychain 有 Token/地址？
  ├── 否 → 显示 SetupView（输入地址 + Token）
  └── 是 → 连接检测 → ProjectListView
              ↓ 选项目
           SessionListView（该项目的 recent 会话列表）
              ↓ 选会话 → │
           ChatView        │ 右上 "+" → StartSession(初始提示)
              ↓             │
           WebSocket 连接 → Relay → Claude Code PTY
              ↓
           流式接收并渲染
              ↓
           发送新消息 / 中断
```

### 关键实现细节

**WebSocket 重连：**
- 网络中断时自动重连（指数退避：1s → 2s → 4s → 8s → max 30s）
- 重连后重新订阅当前 session 的流
- NetworkMonitor 监听网络状态变化

**流式文本渲染：**
- stream_chunk 逐片追加到 `@Published var streamingText: String`
- 使用 `Text(streamingText)` 配合 `.animation()` 让文字平滑增长
- 支持 Markdown（使用 MarkdownUI 或 SwiftUI 内置渲染）

**中断逻辑：**
- 点击 Interrupt 按钮 → WebSocket 发送 `interrupt` → Relay → SIGINT → 子进程停止
- 长按 Interrupt → 强制终止（SIGKILL + 重启 session）

## 六、部署方式

### Relay Daemon

```
Windows 部署：
1. 下载编译好的 relay.exe
2. 第一次运行自动生成：
   - ~/.claude-remote/projects.json（空模板）
   - ~/.claude-remote/auth.json（随机 Token）
3. 编辑 projects.json 添加项目
4. relay.exe 常驻后台（可选注册为 Windows Service）
```

### IPA 安装

```
1. 用 Xcode 编译 ipa/ 目录下的项目
2. 产物为 CCRemote.app / .ipa
3. 通过 TrollStore 安装
```

**编译 IPA 的方式（选一个）：**
- 本地 Mac + Xcode 直接 build → .ipa
- GitHub Actions CI（免费 Mac mini runner）→ 自动编译 release
- 直接在手机上用 TrollStore 的 URL 安装功能（需要分发）

### Tailscale 设置

```
1. 电脑安装 Tailscale，登录
2. 手机安装 Tailscale，登录同一账号
3. 电脑上确认 Tailscale IP：`tailscale ip -4` → 100.x.x.x
4. Relay Daemon 监听 :9943（或自定义端口）
5. 手机端填入：http://100.x.x.x:9943 + Token
```

## 七、实施路线图

### Phase 1：Relay Daemon 核心（1-2 天）

| 任务 | 产出 |
|------|------|
| Go 项目脚手架 + gin 路由 | 可编译的 relay 骨架 |
| 配置文件（projects + auth） | JSON 文件读写 |
| Session 文件解析器 | 读取 .claude/sessions/ 返回摘要 |
| REST API（health / projects / sessions） | curl 可验证的端点 |
| PTY 子进程管理器 | 启动/停止 claude 并捕获输出 |
| WebSocket + 流式转发 | 手机可接收流式文本 |
| 中断功能 | INTERRUPT → SIGINT → 停止 |

**验证方式：** 电脑上 `curlor wscat` 连接 relay，发消息看能否调通 Claude Code

### Phase 2：IPA MVP（1-2 天）

| 任务 | 产出 |
|------|------|
| SwiftUI 项目脚手架 + 导航 | 可运行的空白 app |
| SetupView（地址 + Token） | 连接配置 |
| KeychainHelper | 凭证安全存储 |
| ProjectListView | 显示项目列表 |
| SessionListView | 显示 recent 会话 |
| ChatViewModel + WebSocket | 发送消息 + 流式接收 |
| ChatView + MessageBubble | 消息显示 + 输入框 |
| Interrupt 按钮 | 停止生成 |
| ConnectionBadge | 连接状态指示 |

**验证方式：** 手机上安装 testflight 或直接 TrollStore 安装，连接电脑测试

### Phase 3：体验打磨（1-2 天）

| 任务 | 说明 |
|------|------|
| Session 自动刷新/轮询 | 手机回到前台自动刷新会话列表 |
| 网络断开重连 | 指数退避重连 |
| 消息 Markdown 渲染 | 代码块加亮、链接可点 |
| 消息时间戳格式 | 相对时间（刚刚 / 5分钟前 / 昨天）|
| 多项目切换 | 切换项目时自动加载对应 session |
| 暗黑模式支持 | SwiftUI Dark Mode |
| 错误提示 | 连接失败、进程崩溃等 toast |

### Phase 4：进阶功能（按需）

| 功能 | 说明 |
|------|------|
| 会话归档显示 | 显示 archived session |
| 文件变更通知 | Claude 修改了哪些文件 |
| 消息复制 / 分享 | 长按消息弹出菜单 |
| 语音输入 | iOS 原生语音转文字 |
| 多 session 并行 | 同时连接多个 Claude Code 会话 |
| 远程文件浏览器 | 浏览项目文件 |
| 推送通知 | 后台收到消息时推送 |
| 自动启动 | relay 注册为 Windows 服务，开机自启 |

## 八、技术风险与应对

| 风险 | 影响 | 应对 |
|------|------|------|
| Windows ConPTY 兼容性问题 | relay 无法管理子进程 | 备选方案：用 `os/exec` 管道模式降级运行 |
| Claude Code 输出含大量 ANSI | 解析不完整导致乱码 | stripansi + 启发式过滤 |
| 子进程内存泄漏 / 卡死 | session 无法继续 | 看门狗 + 超时强制重启 |
| Tailscale 在外网延迟高 | 流式响应卡顿 | 消息分块优化 + 本地缓存 |
| IPA 编译需要 Mac | 无法产出 IPA | GitHub Actions 免费 Mac runner |
| Claude Code CLI 版本变更 | 输出格式不兼容 | 适配层隔离 + 版本检测 |

## 九、MVP 验收标准

1. 电脑启动 relay，命令行看到 `Listening on :9943`
2. 手机装 IPA，填入 Tailscale IP + Token，看到连接成功
3. 看到项目列表（预配在 projects.json 中的）
4. 点项目 → 看到该项目的 recent session 列表
5. 点已有 session → 看到历史消息
6. 底部输入文字发送 → 看到 Claude Code 流式回复
7. 点中断按钮 → Claude 停止生成
8. 退出 app 重进 → 会话继续可接

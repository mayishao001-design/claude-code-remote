# CCRemote — iPhone 遥控 Claude Code

**TrollStore 安装，无需签名。**

在手机上远程操控电脑上的 Claude Code，看到会话列表、流式对话、中断生成。

## 编译方法

### 方案 A：GitHub Actions（推荐，无需 Mac）

1. 把代码推到 GitHub
2. GitHub → Actions → Build IPA → Run workflow
3. 等几分钟，下载 artifact 中的 `CCRemote.ipa`
4. 用 TrollStore 安装

### 方案 B：本地 Mac + Xcode

```bash
# 1. 安装 XcodeGen
brew install xcodegen

# 2. 生成 xcodeproj
cd ipa
xcodegen generate

# 3. 编译（unsigned，TrollStore 可装）
xcodebuild \
  -project CCRemote.xcodeproj \
  -scheme CCRemote \
  -sdk iphoneos \
  -configuration Release \
  CODE_SIGN_IDENTITY="" \
  CODE_SIGNING_REQUIRED=NO \
  CODE_SIGNING_ALLOWED=NO \
  build

# 4. 打包 IPA
mkdir Payload
cp -r build/Build/Products/Release-iphoneos/CCRemote.app Payload/
zip -r CCRemote.ipa Payload/
```

### 方案 C：Swift Playground（测试用）

直接在 Playground 里跑 `Views/` 下的 SwiftUI 组件。

## 首次使用

1. 电脑上启动 Relay：`relay.exe`
2. 查电脑 Tailscale IP：`tailscale ip -4`
3. 记下 Relay 打印的 Token
4. 手机上打开 CCRemote → 输入地址 `http://100.x.x.x:9943` + Token
5. 看到项目列表 → 选项目 → 开聊

## 项目结构

```
ipa/CCRemote/
├── App/
│   ├── CCRemoteApp.swift     # @main 入口
│   ├── ContentView.swift     # 主导航 / 设置
│   └── AppState.swift        # 全局状态
├── Models/
│   ├── Project.swift         # 项目模型
│   ├── ClaudeSession.swift   # 会话模型
│   └── StreamChunk.swift     # WebSocket 帧
├── ViewModels/
│   ├── ConnectionViewModel   # 连接配置
│   ├── ProjectViewModel      # 项目列表
│   ├── SessionListViewModel  # 会话列表
│   └── ChatViewModel         # 聊天 + 流式
├── Views/
│   ├── SetupView.swift       # 首次引导
│   ├── ProjectListView.swift # 项目选择
│   ├── SessionListView.swift # 会话列表
│   ├── ChatView.swift        # 聊天界面
│   ├── MessageBubble.swift   # 消息气泡
│   └── ConnectionBadge.swift # 状态指示
├── Services/
│   ├── RelayAPI.swift        # REST 客户端
│   ├── WebSocketManager.swift # WebSocket 流式
│   └── NetworkMonitor.swift  # 网络检测
├── Utils/
│   ├── KeychainHelper.swift  # 安全存储
│   └── DateFormatters.swift  # 时间格式化
├── Info.plist
└── project.yml               # XcodeGen 配置
```

## 技术栈

- **SwiftUI** + iOS 16+
- **URLSession** (async/await) — REST API
- **URLSessionWebSocketTask** — 流式通信
- **Keychain** — 安全存储凭证
- **无外部依赖** — 纯 Foundation + SwiftUI

## IPA 要求

- iOS 16.0+
- TrollStore 安装
- Tailscale（或同局域网）

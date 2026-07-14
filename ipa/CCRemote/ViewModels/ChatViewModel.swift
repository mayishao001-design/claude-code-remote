import Foundation

/// 聊天界面状态
@Observable
final class ChatViewModel {
    enum State: Equatable {
        case idle
        case connecting
        case streaming
        case error(String)
    }

    var state: State = .idle
    var messages: [ChatMessage] = []
    var currentStreamingText = ""
    var currentSessionID: String?

    private let api: RelayAPI
    private var ws: WebSocketManager?

    struct ChatMessage: Identifiable {
        let id: String
        let role: String     // "user" | "assistant"
        let text: String
    }

    init(api: RelayAPI) {
        self.api = api
    }

    // MARK: - Session Management

    /// 加载已有会话历史
    func loadSession(id: String) async {
        do {
            let session = try await api.getSession(id: id)
            currentSessionID = session.id
            messages = session.messages?
                .filter { $0.role == "user" || $0.role == "assistant" }
                .map { ChatMessage(id: UUID().uuidString, role: $0.role, text: $0.content) } ?? []
        } catch {
            state = .error(error.localizedDescription)
        }
    }

    /// 连接 WebSocket
    func connectWS(baseURL: String, token: String) {
        let mgr = WebSocketManager(baseURL: baseURL, token: token)

        mgr.onChunk = { [weak self] text in
            self?.currentStreamingText += text
        }

        mgr.onEnd = { [weak self] in
            guard let self else { return }
            // 流结束 → 把累积文本作为一条 assistant 消息
            if !self.currentStreamingText.isEmpty {
                self.messages.append(ChatMessage(
                    id: UUID().uuidString,
                    role: "assistant",
                    text: self.currentStreamingText
                ))
                self.currentStreamingText = ""
            }
            self.state = .idle
        }

        mgr.onError = { [weak self] error in
            self?.state = .error(error)
        }

        mgr.onSessionUpdated = { [weak self] in
            // 可以触发 session 列表刷新
        }

        mgr.connect()
        self.ws = mgr
    }

    // MARK: - Actions

    func sendMessage(_ text: String) {
        guard let sessionID = currentSessionID, !text.isEmpty else { return }

        messages.append(ChatMessage(id: UUID().uuidString, role: "user", text: text))
        state = .streaming
        currentStreamingText = ""

        ws?.sendMessage(sessionId: sessionID, text: text)
    }

    func startNewSession(project: String, initialPrompt: String?) {
        state = .connecting
        currentStreamingText = ""

        if let prompt = initialPrompt, !prompt.isEmpty {
            messages.append(ChatMessage(id: UUID().uuidString, role: "user", text: prompt))
        }

        ws?.startSession(project: project, initialPrompt: initialPrompt)
        state = .streaming
    }

    func interrupt() {
        guard let sessionID = currentSessionID else { return }
        ws?.interrupt(sessionId: sessionID)
    }

    func disconnect() {
        ws?.disconnect()
        ws = nil
        state = .idle
    }
}

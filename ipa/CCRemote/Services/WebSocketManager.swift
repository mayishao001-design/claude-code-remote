import Foundation

/// WebSocket 管理器（纯 URLSessionWebSocketTask，无外部依赖）
/// 管理连接生命周期、自动重连、消息流式接收
final class WebSocketManager: ObservableObject {
    private var task: URLSessionWebSocketTask?
    private let session = URLSession(configuration: .default)
    private var reconnectTimer: Timer?

    @Published var isConnected = false
    @Published var lastError: String?

    private let baseURL: String
    private let token: String

    // 回调
    var onChunk: ((String) -> Void)?
    var onEnd: (() -> Void)?
    var onError: ((String) -> Void)?
    var onSessionUpdated: (() -> Void)?

    init(baseURL: String, token: String) {
        self.baseURL = baseURL.hasSuffix("/") ? String(baseURL.dropLast()) : baseURL
        self.token = token
    }

    func connect() {
        disconnect()

        var wsURL = baseURL
            .replacingOccurrences(of: "https://", with: "wss://")
            .replacingOccurrences(of: "http://", with: "ws://")
        wsURL += "/api/v1/ws?token=\(token)"

        guard let url = URL(string: wsURL) else {
            lastError = "无效的 WebSocket URL"
            return
        }

        task = session.webSocketTask(with: url)
        task?.resume()
        isConnected = true
        lastError = nil
        receiveMessage()
    }

    func disconnect() {
        reconnectTimer?.invalidate()
        reconnectTimer = nil
        task?.cancel(with: .normalClosure, reason: nil)
        task = nil
        isConnected = false
    }

    func sendMessage(sessionId: String, text: String) {
        let msg = ClientMessage(type: "send_message", sessionId: sessionId, text: text, project: nil)
        send(msg)
    }

    func startSession(project: String, initialPrompt: String?) {
        let msg = ClientMessage(type: "start_session", sessionId: nil, text: initialPrompt, project: project)
        send(msg)
    }

    func interrupt(sessionId: String) {
        let msg = ClientMessage(type: "interrupt", sessionId: sessionId, text: nil, project: nil)
        send(msg)
    }

    func ping() {
        let msg = ClientMessage(type: "ping", sessionId: nil, text: nil, project: nil)
        send(msg)
    }

    private func send(_ msg: ClientMessage) {
        guard let data = try? JSONEncoder().encode(msg),
              let text = String(data: data, encoding: .utf8) else { return }

        task?.send(.string(text)) { [weak self] error in
            if let e = error {
                DispatchQueue.main.async {
                    self?.handleError(e.localizedDescription)
                }
            }
        }
    }

    private func receiveMessage() {
        task?.receive { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success(let message):
                    switch message {
                    case .string(let text):
                        self?.handle(text)
                    case .data(let data):
                        if let text = String(data: data, encoding: .utf8) {
                            self?.handle(text)
                        }
                    @unknown default:
                        break
                    }
                    self?.receiveMessage()
                case .failure(let error):
                    self?.handleError(error.localizedDescription)
                }
            }
        }
    }

    private func handle(_ text: String) {
        guard let data = text.data(using: .utf8),
              let msg = try? JSONDecoder().decode(ServerMessage.self, from: data) else { return }

        switch msg.type {
        case "stream_chunk":
            if let t = msg.text { onChunk?(t) }
        case "stream_end":
            onEnd?()
        case "stream_error":
            if let e = msg.error { onError?(e) }
        case "session_updated":
            onSessionUpdated?()
        case "pong":
            break
        default:
            break
        }
    }

    private func handleError(_ error: String) {
        lastError = error
        isConnected = false
        onError?(error)

        reconnectTimer?.invalidate()
        reconnectTimer = Timer.scheduledTimer(withTimeInterval: 3, repeats: false) { [weak self] _ in
            self?.connect()
        }
    }
}

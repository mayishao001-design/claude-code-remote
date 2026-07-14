import Foundation

/// 聊天界面状态
final class ChatViewModel: ObservableObject {
    enum State: Equatable {
        case idle
        case connecting
        case streaming
        case error(String)
    }

    @Published var state: State = .idle
    @Published var messages: [ChatMessage] = []
    @Published var currentStreamingText = ""
    @Published var currentSessionID: String?

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

    @MainActor
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

    func connectWS(baseURL: String, token: String) {
        let mgr = WebSocketManager(baseURL: baseURL, token: token)

        mgr.onChunk = { [weak self] text in
            DispatchQueue.main.async {
                self?.currentStreamingText += text
            }
        }

        mgr.onEnd = { [weak self] in
            guard let self else { return }
            DispatchQueue.main.async {
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
        }

        mgr.onError = { [weak self] error in
            DispatchQueue.main.async {
                self?.state = .error(error)
            }
        }

        mgr.connect()
        self.ws = mgr
    }

    // MARK: - Actions

    func sendMessage(_ text: String) {
        guard let sessionID = currentSessionID, !text.isEmpty else { return }
        let msg = ChatMessage(id: UUID().uuidString, role: "user", text: text)
        messages.append(msg)
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

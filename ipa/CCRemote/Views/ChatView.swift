import SwiftUI

/// 聊天主界面
struct ChatView: View {
    @State var viewModel: ChatViewModel
    let baseURL: String
    let token: String
    let project: String
    var existingSessionID: String?

    @State private var inputText = ""
    @State private var scrollToBottom = false
    @FocusState private var inputFocused: Bool

    var body: some View {
        VStack(spacing: 0) {
            // 消息列表
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 12) {
                        ForEach(viewModel.messages) { msg in
                            MessageBubble(message: msg)
                        }

                        // 流式内容
                        if !viewModel.currentStreamingText.isEmpty {
                            MessageBubble(
                                message: ChatViewModel.ChatMessage(
                                    id: "streaming",
                                    role: "assistant",
                                    text: viewModel.currentStreamingText
                                ),
                                isStreaming: true
                            )
                            .id("streaming")
                        }
                    }
                    .padding()
                }
                .onChange(of: viewModel.messages.count) { _, _ in
                    scrollToBottom(proxy)
                }
                .onChange(of: viewModel.currentStreamingText) { _, _ in
                    scrollToBottom(proxy)
                }
            }

            // 状态指示 & 中断
            HStack {
                if case .streaming = viewModel.state {
                    Button(action: { viewModel.interrupt() }) {
                        Label("中断", systemImage: "stop.fill")
                            .font(.caption)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(.red.opacity(0.1))
                            .cornerRadius(8)
                    }
                    .buttonStyle(.plain)
                } else if case .connecting = viewModel.state {
                    HStack(spacing: 6) {
                        ProgressView()
                            .scaleEffect(0.7)
                        Text("连接中...")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }
                Spacer()
            }
            .padding(.horizontal)
            .padding(.vertical, 4)

            // 输入栏
            HStack(spacing: 8) {
                TextField("输入消息...", text: $inputText)
                    .textFieldStyle(.roundedBorder)
                    .focused($inputFocused)
                    .disabled(viewModel.state == .streaming || viewModel.state == .connecting)

                Button(action: send) {
                    Image(systemName: "arrow.up.circle.fill")
                        .font(.title2)
                        .foregroundColor(inputText.trimmingCharacters(in: .whitespaces).isEmpty ? .gray : .accentColor)
                }
                .disabled(inputText.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            .padding()
            .background(.bar)
        }
        .navigationTitle("对话")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await initializeSession()
            viewModel.connectWS(baseURL: baseURL, token: token)
        }
        .onDisappear {
            viewModel.disconnect()
        }
    }

    // MARK: - Init

    private func initializeSession() async {
        if let sessionID = existingSessionID {
            await viewModel.loadSession(id: sessionID)
        }
    }

    // MARK: - Send

    private func send() {
        let text = inputText.trimmingCharacters(in: .whitespaces)
        guard !text.isEmpty else { return }

        inputText = ""

        if viewModel.currentSessionID == nil {
            // 新会话
            viewModel.startNewSession(project: project, initialPrompt: text)
        } else {
            viewModel.sendMessage(text)
        }
    }

    // MARK: - Scroll

    private func scrollToBottom(_ proxy: ScrollViewProxy) {
        withAnimation(.easeOut(duration: 0.15)) {
            if !viewModel.currentStreamingText.isEmpty {
                proxy.scrollTo("streaming", anchor: .bottom)
            } else if let last = viewModel.messages.last {
                proxy.scrollTo(last.id, anchor: .bottom)
            }
        }
    }
}

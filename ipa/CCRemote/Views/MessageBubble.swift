import SwiftUI

/// 单条消息气泡
struct MessageBubble: View {
    let message: ChatViewModel.ChatMessage
    var isStreaming: Bool = false

    private var isUser: Bool { message.role == "user" }

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            if isUser {
                Spacer(minLength: 60)
            } else {
                Image(systemName: "cpu")
                    .font(.caption)
                    .foregroundColor(.accentColor)
                    .frame(width: 24, height: 24)
                    .background(Color.accentColor.opacity(0.1))
                    .clipShape(RoundedRectangle(cornerRadius: 6))
            }

            VStack(alignment: isUser ? .trailing : .leading, spacing: 4) {
                Text(.init(message.text))
                    .font(.body)
                    .textSelection(.enabled)
                    .foregroundColor(isUser ? .white : .primary)

                if isStreaming {
                    HStack(spacing: 3) {
                        ForEach(0..<3) { i in
                            Circle()
                                .fill(.secondary)
                                .frame(width: 4, height: 4)
                                .opacity(isStreaming ? blinkOpacity(at: i) : 1)
                        }
                    }
                }
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 10)
            .background(isUser ? Color.accentColor : Color(.systemGray6))
            .clipShape(RoundedRectangle(cornerRadius: 16))
            .overlay(alignment: isUser ? .bottomTrailing : .bottomLeading) {
                if !isStreaming {
                    Text(DateFormatters.format(Date()))
                        .font(.caption2)
                        .foregroundColor(.secondary)
                        .padding(.top, 2)
                }
            }

            if !isUser {
                Spacer(minLength: 60)
            }
        }
    }

    private func blinkOpacity(at index: Int) -> Double {
        let phase = Date().timeIntervalSince1970 * 2
        let shift = Double(index) * 0.5
        return 0.3 + 0.7 * sin(phase + shift) * sin(phase + shift)
    }
}

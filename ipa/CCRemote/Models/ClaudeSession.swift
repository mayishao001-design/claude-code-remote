import Foundation

/// 会话摘要（列表页使用）
struct SessionListItem: Codable, Identifiable {
    let id: String
    let title: String
    let project: String
    let projectPath: String?
    let messageCount: Int
    let lastMessageAt: String?
    let archived: Bool

    var lastMessageDate: Date? {
        guard let s = lastMessageAt else { return nil }
        let fmt = ISO8601DateFormatter()
        fmt.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return fmt.date(from: s) ?? ISO8601DateFormatter().date(from: s)
    }
}

/// 会话详情（含消息历史）
struct ClaudeSession: Codable, Identifiable {
    let id: String
    let title: String?
    let project: String?
    let messages: [SessionMessage]?
    let archived: Bool?
}

/// 单条消息
struct SessionMessage: Codable, Identifiable {
    let id: String?
    let role: String       // "user" | "assistant"
    let content: String
}

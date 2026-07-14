import Foundation

/// WebSocket 通信帧
struct ServerMessage: Codable {
    let type: String       // stream_chunk | stream_end | stream_error | session_updated | pong
    let sessionId: String?
    let text: String?
    let error: String?

    enum CodingKeys: String, CodingKey {
        case type
        case sessionId = "session_id"
        case text
        case error
    }
}

/// 手机→Relay 消息
struct ClientMessage: Encodable {
    let type: String       // send_message | start_session | interrupt | ping
    let sessionId: String?
    let text: String?
    let project: String?

    enum CodingKeys: String, CodingKey {
        case type
        case sessionId = "session_id"
        case text
        case project
    }
}

/// REST API 通用响应
struct SessionListResponse: Codable {
    let sessions: [SessionListItem]
}

struct ProjectListResponse: Codable {
    let projects: [Project]
}

struct SessionDetailResponse: Codable {
    let session: ClaudeSession
}

struct HealthResponse: Codable {
    let status: String
}

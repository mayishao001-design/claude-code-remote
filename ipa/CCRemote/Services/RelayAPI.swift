import Foundation

/// Relay REST API 客户端
actor RelayAPI {
    private let baseURL: String
    private let token: String
    private let session: URLSession

    init(baseURL: String, token: String) {
        self.baseURL = baseURL.hasSuffix("/") ? String(baseURL.dropLast()) : baseURL
        self.token = token
        self.session = URLSession(configuration: .default)
    }

    // MARK: - Auth

    var requestHeaders: [String: String] {
        ["Authorization": "Bearer \(token)"]
    }

    // MARK: - Health

    func health() async throws -> Bool {
        let (_, response) = try await get("/api/v1/health")
        return (response as? HTTPURLResponse)?.statusCode == 200
    }

    // MARK: - Projects

    func listProjects() async throws -> [Project] {
        let (data, _) = try await get("/api/v1/projects")
        let resp = try JSONDecoder().decode(ProjectListResponse.self, from: data)
        return resp.projects
    }

    // MARK: - Sessions

    func listSessions(archived: Bool? = nil, project: String? = nil) async throws -> [SessionListItem] {
        var path = "/api/v1/sessions"
        var params: [String] = []
        if let a = archived { params.append("archived=\(a)") }
        if let p = project { params.append("project=\(p.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? p)") }
        if !params.isEmpty { path += "?" + params.joined(separator: "&") }

        let (data, _) = try await get(path)
        let resp = try JSONDecoder().decode(SessionListResponse.self, from: data)
        return resp.sessions
    }

    func getSession(id: String) async throws -> ClaudeSession {
        let (data, _) = try await get("/api/v1/sessions/\(id)")
        let resp = try JSONDecoder().decode(SessionDetailResponse.self, from: data)
        return resp.session
    }

    func interruptSession(id: String) async throws {
        let _ = try await post("/api/v1/sessions/\(id)/interrupt", body: nil)
    }

    func deleteSession(id: String) async throws {
        let _ = try await delete("/api/v1/sessions/\(id)")
    }

    // MARK: - HTTP

    private func get(_ path: String) async throws -> (Data, URLResponse) {
        var req = URLRequest(url: URL(string: baseURL + path)!)
        req.httpMethod = "GET"
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        req.timeoutInterval = 10
        return try await session.data(for: req)
    }

    private func post(_ path: String, body: Encodable?) async throws -> (Data, URLResponse) {
        var req = URLRequest(url: URL(string: baseURL + path)!)
        req.httpMethod = "POST"
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.timeoutInterval = 10
        if let b = body {
            req.httpBody = try JSONEncoder().encode(b)
        }
        return try await session.data(for: req)
    }

    private func delete(_ path: String) async throws -> (Data, URLResponse) {
        var req = URLRequest(url: URL(string: baseURL + path)!)
        req.httpMethod = "DELETE"
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        req.timeoutInterval = 10
        return try await session.data(for: req)
    }
}

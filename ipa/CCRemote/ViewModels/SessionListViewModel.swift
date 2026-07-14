import Foundation

/// 会话列表
@Observable
final class SessionListViewModel {
    var sessions: [SessionListItem] = []
    var isLoading = false
    var error: String?

    private let api: RelayAPI

    init(api: RelayAPI) {
        self.api = api
    }

    func load(project: String? = nil) async {
        isLoading = true
        error = nil
        do {
            sessions = try await api.listSessions(project: project)
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    func deleteSession(id: String) async {
        do {
            try await api.deleteSession(id: id)
            sessions.removeAll { $0.id == id }
        } catch {
            self.error = error.localizedDescription
        }
    }
}

import Foundation

/// 会话列表
final class SessionListViewModel: ObservableObject {
    @Published var sessions: [SessionListItem] = []
    @Published var isLoading = false
    @Published var error: String?

    private let api: RelayAPI

    init(api: RelayAPI) {
        self.api = api
    }

    @MainActor
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

    @MainActor
    func deleteSession(id: String) async {
        do {
            try await api.deleteSession(id: id)
            sessions.removeAll { $0.id == id }
        } catch {
            self.error = error.localizedDescription
        }
    }
}

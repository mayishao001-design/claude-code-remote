import Foundation

/// 项目列表
@Observable
final class ProjectViewModel {
    var projects: [Project] = []
    var isLoading = false
    var error: String?

    private let api: RelayAPI

    init(api: RelayAPI) {
        self.api = api
    }

    func load() async {
        isLoading = true
        error = nil
        do {
            projects = try await api.listProjects()
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }
}

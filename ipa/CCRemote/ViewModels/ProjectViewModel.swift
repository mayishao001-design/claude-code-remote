import Foundation

/// 项目列表
final class ProjectViewModel: ObservableObject {
    @Published var projects: [Project] = []
    @Published var isLoading = false
    @Published var error: String?

    private let api: RelayAPI

    init(api: RelayAPI) {
        self.api = api
    }

    @MainActor
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

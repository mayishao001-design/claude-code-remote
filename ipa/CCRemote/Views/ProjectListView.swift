import SwiftUI

/// 项目列表
struct ProjectListView: View {
    @ObservedObject var viewModel: ProjectViewModel
    let baseURL: String
    let token: String

    var body: some View {
        List {
            if viewModel.isLoading && viewModel.projects.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity)
            } else if viewModel.projects.isEmpty {
                ContentUnavailableView(
                    "暂无项目",
                    systemImage: "folder",
                    description: Text("在电脑上编辑 ~/.claude-remote/projects.json 添加项目")
                )
            } else {
                ForEach(viewModel.projects) { project in
                    NavigationLink(destination: sessionList(for: project)) {
                        HStack(spacing: 14) {
                            Image(systemName: "folder.fill")
                                .font(.title2)
                                .foregroundColor(.accentColor)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(project.name)
                                    .font(.headline)
                                Text(project.path)
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                    .lineLimit(1)
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }
            }
        }
        .listStyle(.plain)
        .navigationTitle("项目")
        .refreshable {
            await viewModel.load()
        }
        .task {
            await viewModel.load()
        }
    }

    private func sessionList(for project: Project) -> some View {
        let api = RelayAPI(baseURL: baseURL, token: token)
        let sessionVM = SessionListViewModel(api: api)
        return SessionListView(
            viewModel: sessionVM,
            project: project,
            baseURL: baseURL,
            token: token
        )
    }
}

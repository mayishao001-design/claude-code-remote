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
                emptyView("暂无项目", icon: "folder", hint: "在电脑上编辑 ~/.claude-remote/projects.json 添加项目")
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

    @ViewBuilder
    private func emptyView(_ title: String, icon: String, hint: String) -> some View {
        VStack(spacing: 12) {
            Spacer().frame(height: 60)
            Image(systemName: icon)
                .font(.system(size: 48))
                .foregroundColor(.secondary.opacity(0.5))
            Text(title)
                .font(.headline)
                .foregroundColor(.secondary)
            Text(hint)
                .font(.caption)
                .foregroundColor(.secondary.opacity(0.7))
                .multilineTextAlignment(.center)
            Spacer()
        }
        .frame(maxWidth: .infinity)
        .listRowBackground(Color.clear)
        .listRowSeparator(.hidden)
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

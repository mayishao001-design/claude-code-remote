import SwiftUI

/// 会话列表
struct SessionListView: View {
    @ObservedObject var viewModel: SessionListViewModel
    let project: Project
    let baseURL: String
    let token: String

    @State private var showNewChat = false
    @State private var newChatPrompt = ""

    var body: some View {
        List {
            if viewModel.isLoading && viewModel.sessions.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity)
            } else if viewModel.sessions.isEmpty {
                emptyView("暂无会话", icon: "message", hint: "在电脑上启动 Claude Code 后将出现在这里")
            } else {
                ForEach(viewModel.sessions) { session in
                    NavigationLink(destination: chatView(for: session)) {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(session.title)
                                .font(.headline)
                                .lineLimit(1)

                            HStack(spacing: 8) {
                                Text("\(session.messageCount) 条消息")
                                    .font(.caption)
                                    .foregroundColor(.secondary)

                                if let date = session.lastMessageDate {
                                    Text(DateFormatters.format(date))
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                }
                            }
                        }
                        .padding(.vertical, 2)
                    }
                    .swipeActions(edge: .trailing) {
                        Button(role: .destructive) {
                            Task { await viewModel.deleteSession(id: session.id) }
                        } label: {
                            Label("删除", systemImage: "trash")
                        }
                    }
                }
            }
        }
        .listStyle(.plain)
        .navigationTitle(project.name)
        .toolbarRole(.editor)
        .toolbar {
            ToolbarItemGroup(placement: .navigationBarTrailing) {
                Button(action: { showNewChat = true }) {
                    Image(systemName: "plus")
                }
            }
        }
        .refreshable {
            await viewModel.load(project: project.name)
        }
        .task {
            await viewModel.load(project: project.name)
        }
        .alert("新建会话", isPresented: $showNewChat) {
            TextField("初始提示（可选）", text: $newChatPrompt)
            Button("开始") { startNewSession() }
            Button("取消", role: .cancel) { newChatPrompt = "" }
        } message: {
            Text("输入你想让 Claude 处理的问题")
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

    private func chatView(for session: SessionListItem) -> some View {
        let api = RelayAPI(baseURL: baseURL, token: token)
        let vm = ChatViewModel(api: api)
        return ChatView(
            viewModel: vm,
            baseURL: baseURL,
            token: token,
            project: project.name,
            existingSessionID: session.id
        )
    }

    private func startNewSession() {
        // TODO: implement new session navigation
    }
}

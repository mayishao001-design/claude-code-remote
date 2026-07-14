import SwiftUI

/// 主界面：项目列表或连接配置
struct ContentView: View {
    @ObservedObject var connVM: ConnectionViewModel
    @State private var showSettings = false

    var body: some View {
        NavigationStack {
            if let baseURL = KeychainHelper.load(key: "relay_url"),
               let token = KeychainHelper.load(key: "relay_token") {
                let api = RelayAPI(baseURL: baseURL, token: token)
                let projectVM = ProjectViewModel(api: api)

                ProjectListView(
                    viewModel: projectVM,
                    baseURL: baseURL,
                    token: token
                )
                .toolbar {
                    ToolbarItem(placement: .navigationBarLeading) {
                        ConnectionBadge(
                            isConnected: true,
                            lastError: nil
                        )
                    }
                    ToolbarItem(placement: .navigationBarTrailing) {
                        Button(action: { showSettings = true }) {
                            Image(systemName: "gearshape")
                        }
                    }
                }
                .sheet(isPresented: $showSettings) {
                    NavigationStack {
                        settingsView(baseURL: baseURL)
                    }
                }
            }
        }
    }

    private func settingsView(baseURL: String) -> some View {
        Form {
            Section("连接") {
                LabeledContent("地址", value: baseURL)
                LabeledContent("Token", value: String(connVM.relayToken.prefix(8)) + "...")
            }

            Section {
                Button(role: .destructive) {
                    connVM.reset()
                    showSettings = false
                } label: {
                    Label("重置连接", systemImage: "arrow.counterclockwise")
                }
            }
        }
        .navigationTitle("设置")
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                Button("完成") { showSettings = false }
            }
        }
    }
}

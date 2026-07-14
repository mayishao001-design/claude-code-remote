import Foundation

/// 全局应用状态
final class AppState: ObservableObject {
    @Published var isConnected = false
    @Published var relayURL: String = ""
    @Published var relayToken: String = ""

    var relayAPI: RelayAPI? {
        guard !relayURL.isEmpty, !relayToken.isEmpty else { return nil }
        return RelayAPI(baseURL: relayURL, token: relayToken)
    }

    init() {
        loadCredentials()
    }

    private func loadCredentials() {
        relayURL = KeychainHelper.load(key: "relay_url") ?? ""
        relayToken = KeychainHelper.load(key: "relay_token") ?? ""
        isConnected = !relayURL.isEmpty && !relayToken.isEmpty
    }
}

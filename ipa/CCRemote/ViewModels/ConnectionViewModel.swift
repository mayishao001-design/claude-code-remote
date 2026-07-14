import Foundation

/// 连接配置管理
@Observable
final class ConnectionViewModel {
    var relayURL: String = ""
    var relayToken: String = ""
    var isConfigured: Bool = false
    var isConnecting = false
    var connectionError: String?

    private let apiKey = "relay_url"
    private let tokenKey = "relay_token"

    init() {
        loadSaved()
    }

    private func loadSaved() {
        relayURL = KeychainHelper.load(key: apiKey) ?? ""
        relayToken = KeychainHelper.load(key: tokenKey) ?? ""
        isConfigured = !relayURL.isEmpty && !relayToken.isEmpty
    }

    /// 保存并测试连接
    func saveAndTest() async -> Bool {
        guard !relayURL.isEmpty, !relayToken.isEmpty else {
            connectionError = "请输入地址和 Token"
            return false
        }

        isConnecting = true
        connectionError = nil

        let api = RelayAPI(baseURL: relayURL, token: relayToken)
        do {
            let ok = try await api.health()
            if ok {
                KeychainHelper.save(key: apiKey, value: relayURL)
                KeychainHelper.save(key: tokenKey, value: relayToken)
                isConfigured = true
                isConnecting = false
                return true
            } else {
                connectionError = "连接失败：服务器返回异常"
            }
        } catch {
            connectionError = "连接失败：\(error.localizedDescription)"
        }

        isConnecting = false
        return false
    }

    /// 重置配置
    func reset() {
        KeychainHelper.delete(key: apiKey)
        KeychainHelper.delete(key: tokenKey)
        relayURL = ""
        relayToken = ""
        isConfigured = false
        connectionError = nil
    }
}

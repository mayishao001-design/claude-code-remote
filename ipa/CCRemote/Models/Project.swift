import Foundation

/// 预配项目（来自 Relay projects.json）
struct Project: Codable, Identifiable, Hashable {
    let name: String
    let path: String

    var id: String { name }
}

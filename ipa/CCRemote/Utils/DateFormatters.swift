import Foundation

/// 通用日期格式化
enum DateFormatters {
    static let relative: RelativeDateTimeFormatter = {
        let f = RelativeDateTimeFormatter()
        f.unitsStyle = .short
        f.locale = Locale(identifier: "zh_CN")
        return f
    }()

    static let short: DateFormatter = {
        let f = DateFormatter()
        f.dateStyle = .short
        f.timeStyle = .short
        f.locale = Locale(identifier: "zh_CN")
        return f
    }()

    static func format(_ date: Date?) -> String {
        guard let d = date else { return "" }
        let elapsed = -d.timeIntervalSinceNow
        if elapsed < 3600 * 24 {
            return relative.localizedString(for: d, relativeTo: Date())
        }
        return short.string(from: d)
    }
}

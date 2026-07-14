import SwiftUI

/// 连接状态指示器
struct ConnectionBadge: View {
    let isConnected: Bool
    let lastError: String?

    var body: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(isConnected ? Color.green : Color.red)
                .frame(width: 8, height: 8)

            Text(isConnected ? "已连接" : "断开")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 4)
        .background(.ultraThinMaterial)
        .cornerRadius(12)
    }
}

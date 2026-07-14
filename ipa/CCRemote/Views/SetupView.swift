import SwiftUI

/// 首次引导：输入 Relay 地址 + Token
struct SetupView: View {
    @State var connVM: ConnectionViewModel
    let onComplete: () -> Void

    @State private var showError = false

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                Spacer()

                Image(systemName: "antenna.radiowaves.left.and.right")
                    .font(.system(size: 60))
                    .foregroundColor(.accentColor)

                Text("连接 Relay")
                    .font(.title).bold()

                Text("输入电脑上的 Relay 地址和 Token\n两者都在 Relay 首次启动时显示")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)

                VStack(alignment: .leading, spacing: 4) {
                    Text("Relay 地址").font(.caption).foregroundColor(.secondary)
                    TextField("http://100.x.x.x:9943", text: $connVM.relayURL)
                        .textContentType(.URL)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                        .textFieldStyle(.roundedBorder)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text("Token").font(.caption).foregroundColor(.secondary)
                    SecureField("输入 Token", text: $connVM.relayToken)
                        .textFieldStyle(.roundedBorder)
                }

                if let err = connVM.connectionError, showError {
                    Text(err)
                        .font(.caption)
                        .foregroundColor(.red)
                        .multilineTextAlignment(.center)
                }

                Button(action: testAndSave) {
                    if connVM.isConnecting {
                        ProgressView()
                            .progressViewStyle(.circular)
                            .tint(.white)
                    } else {
                        Text("连接并保存")
                            .frame(maxWidth: .infinity)
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(connVM.isConnecting)
                .padding(.top, 8)

                Spacer()
            }
            .padding(32)
        }
    }

    private func testAndSave() {
        showError = true
        Task {
            let ok = await connVM.saveAndTest()
            if ok {
                onComplete()
            }
        }
    }
}

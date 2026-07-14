import SwiftUI

/// 应用入口
@main
struct CCRemoteApp: App {
    @State private var connVM = ConnectionViewModel()
    @State private var isSetup = false

    var body: some Scene {
        WindowGroup {
            if connVM.isConfigured || isSetup {
                ContentView(connVM: connVM)
            } else {
                SetupView(connVM: connVM) {
                    isSetup = true
                }
            }
        }
    }
}

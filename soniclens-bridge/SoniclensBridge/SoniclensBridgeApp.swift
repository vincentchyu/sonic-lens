import SwiftUI

#if os(macOS)
@main
struct SoniclensBridgeApp: App {
    @StateObject private var store = AppStore()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(store)
        }
    }
}
#endif

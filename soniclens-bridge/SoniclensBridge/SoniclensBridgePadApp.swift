import SwiftUI

#if os(iOS)
@main
struct SoniclensBridgePadApp: App {
    @StateObject private var store = AppStore()

    var body: some Scene {
        WindowGroup {
            PadRootView()
                .environmentObject(store)
                .environment(store.playbackStore)
                .environment(store.favoriteStore)
                .environmentObject(store.insightCoordinator)
        }
    }
}
#endif

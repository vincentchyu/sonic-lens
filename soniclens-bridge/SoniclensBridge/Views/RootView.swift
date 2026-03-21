import SwiftUI

struct RootView: View {
    @EnvironmentObject private var store: AppStore

    var body: some View {
        ZStack {
            if store.currentServer == nil {
                ConnectionView()
            } else {
                #if os(macOS)
                MacRootView()
                #else
                PadRootView()
                #endif
            }
        }
    }
}

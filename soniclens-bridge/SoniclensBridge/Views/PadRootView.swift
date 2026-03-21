import SwiftUI

#if os(iOS)
struct PadRootView: View {
    @EnvironmentObject private var store: AppStore

    var body: some View {
        ZStack {
            if store.currentServer == nil {
                ConnectionView()
            } else {
                PadAppLayoutView()
            }
        }
    }
}
#endif

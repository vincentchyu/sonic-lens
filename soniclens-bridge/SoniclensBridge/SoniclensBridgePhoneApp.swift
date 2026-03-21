import SwiftUI

#if os(iOS)
@main
struct SoniclensBridgePhoneApp: App {
    @StateObject private var store = AppStore()

    var body: some Scene {
        WindowGroup {
            PhoneRootView()
                .environmentObject(store)
        }
    }
}
#endif

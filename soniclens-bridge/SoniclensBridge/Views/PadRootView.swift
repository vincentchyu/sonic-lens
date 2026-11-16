import SwiftUI

#if os(iOS)
struct PadRootView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(ConnectionRecoveryStore.self) private var connectionRecoveryStore
    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        ZStack {
            if shouldShowBootstrappingConnection || connectionRecoveryStore.isBootstrapping {
                ConnectionBootstrapView(
                    status: store.connectionStatus,
                    server: connectionRecoveryStore.server
                ) {
                    store.cancelConnection()
                }
            } else if connectionRecoveryStore.isRecoveryRequired {
                ConnectionRecoveryDecisionView(
                    server: connectionRecoveryStore.server,
                    status: store.connectionStatus,
                    onReconnect: {
                        guard let server = connectionRecoveryStore.server else { return }
                        Task {
                            _ = await store.connect(server)
                        }
                    },
                    onDisconnect: {
                        store.disconnect()
                    },
                    onOffline: {
                        store.enterOfflineMode(preserving: connectionRecoveryStore.server)
                    }
                )
            } else if store.currentServer == nil && !store.isOfflineBrowsing {
                ConnectionView()
            } else {
                PadAppLayoutView()
            }
        }
        .task {
            Task { await store.refreshMobileScrobblesIfNeeded() }
            await store.bootstrapConnectionIfNeeded()
        }
        .onChange(of: scenePhase) { _, phase in
            guard phase == .active else { return }
            Task { await store.refreshMobileScrobblesIfNeeded() }
        }
    }

    private var shouldShowBootstrappingConnection: Bool {
        store.shouldShowBootstrappingConnection
    }
}
#endif

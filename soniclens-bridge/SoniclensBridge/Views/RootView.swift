import SwiftUI

struct RootView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(ConnectionRecoveryStore.self) private var connectionRecoveryStore

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
                    }
                )
            } else if store.currentServer == nil {
                ConnectionView()
            } else {
                #if os(macOS)
                MacRootView()
                #else
                PadRootView()
                #endif
            }
        }
        .task {
            await store.bootstrapConnectionIfNeeded()
        }
    }

    private var shouldShowBootstrappingConnection: Bool {
        store.shouldShowBootstrappingConnection
    }
}

import SwiftUI
import OSLog

struct ConnectionView: View {
    @EnvironmentObject private var store: AppStore
    @StateObject private var discovery = BonjourDiscovery()

    @State private var manualHost: String = ""
    @State private var manualPort: String = "8082"
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "ConnectionView")

    var body: some View {
        ZStack {
            AppBackground()

            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    ContentHeader(title: "连接 音眸轨迹")

                    ConnectionStatusView(status: store.connectionStatus)
                    if store.connectionStatus.phase.isInFlight {
                        HStack {
                            Button("取消连接") {
                                store.cancelConnection()
                            }
                            .buttonStyle(.bordered)
                            Spacer()
                        }
                        .padding(.top, -12)
                    }

                    if !store.recentServers.isEmpty {
                        GroupBox("最近连接") {
                            VStack(alignment: .leading, spacing: 8) {
                                ForEach(store.recentServers) { server in
                                    let isConnecting = store.isConnecting(to: server)
                                    Button(action: {
                                        logger.info("点击最近连接服务端 \(server.displayName, privacy: .public)")
                                        Task { await store.connect(server) }
                                    }) {
                                        HStack {
                                            Text(server.displayName)
                                                .font(.subheadline)
                                            Spacer()
                                            if isConnecting {
                                                ProgressView()
                                                    .controlSize(.small)
                                                Text(store.connectionStatus.phase.inlineStatusTitle)
                                                    .font(.caption)
                                                    .foregroundColor(.secondary)
                                            } else {
                                                Image(systemName: "chevron.right")
                                                    .foregroundColor(.secondary)
                                            }
                                        }
                                    }
                                    .buttonStyle(.plain)
                                    .disabled(store.isConnecting && !isConnecting)
                                }
                            }
                            .padding(.top, 6)
                        }
                    }

                    GroupBox("自动发现") {
                        VStack(alignment: .leading, spacing: 10) {
                            HStack {
                                Text("局域网扫描")
                                Spacer()
                                if discovery.isScanning {
                                    ProgressView()
                                        .controlSize(.small)
                                }
                                Button(discovery.isScanning ? "扫描中..." : "刷新") { refreshDiscovery() }
                                    .disabled(discovery.isScanning)
                            }

                            if let message = discovery.lastErrorMessage, discovery.candidates.isEmpty {
                                Text(message)
                                    .foregroundColor(.secondary)
                            } else if discovery.isScanning && discovery.candidates.isEmpty {
                                Text("正在扫描局域网服务器...")
                                    .foregroundColor(.secondary)
                            } else if discovery.candidates.isEmpty {
                                Text("未发现局域网服务器")
                                    .foregroundColor(.secondary)
                            } else {
                                ForEach(discovery.candidates) { candidate in
                                    let server = candidate.toConfig()
                                    let isConnecting = store.isConnecting(to: server)
                                    Button(action: {
                                        logger.info("点击自动发现服务端 \(candidate.name, privacy: .public)")
                                        Task { await store.connect(server) }
                                    }) {
                                        HStack(alignment: .center, spacing: 12) {
                                            VStack(alignment: .leading, spacing: 2) {
                                                Text(candidate.name)
                                                Text(candidate.detailText)
                                                    .font(.caption)
                                                    .foregroundColor(.secondary)
                                                    .lineLimit(2)
                                            }
                                            Spacer()
                                            if isConnecting {
                                                ProgressView()
                                                    .controlSize(.small)
                                                Text(store.connectionStatus.phase.inlineStatusTitle)
                                                    .font(.caption)
                                                    .foregroundColor(.secondary)
                                            }
                                        }
                                    }
                                    .buttonStyle(.plain)
                                    .disabled(store.isConnecting && !isConnecting)
                                }
                            }

                            if let finishedAt = discovery.lastScanFinishedAt {
                                Text("上次扫描：\(finishedAt.formatted(date: .omitted, time: .standard))")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                        }
                        .padding(.top, 6)
                    }

                    if let server = store.currentServer {
                        GroupBox("当前连接") {
                            HStack {
                                Text(server.displayName)
                                    .font(.subheadline)
                                Spacer()
                                Button("断开") { store.disconnect() }
                                    .buttonStyle(.bordered)
                            }
                            .padding(.top, 6)
                        }
                    }

                    GroupBox("手动输入") {
                        VStack(alignment: .leading, spacing: 10) {
                            TextField("IP 或主机名", text: $manualHost)
                                .textFieldStyle(RoundedBorderTextFieldStyle())
                            TextField("端口", text: $manualPort)
                                .textFieldStyle(RoundedBorderTextFieldStyle())
                            Button("连接") {
                                guard let port = Int(manualPort), !manualHost.isEmpty else { return }
                                logger.info("点击手动连接，主机 \(manualHost, privacy: .public)，端口 \(port, privacy: .public)")
                                let server = ServerConfig(name: "手动输入", host: manualHost, port: port)
                                Task { await store.connect(server) }
                            }
                            .buttonStyle(.borderedProminent)
                            .disabled(store.isConnecting || manualHost.isEmpty || Int(manualPort) == nil)
                        }
                        .padding(.top, 6)
                    }
                }
                .padding(32)
            }
        }
        .onAppear {
            logger.info("连接页出现，准备刷新扫描与最近连接")
            refreshDiscovery()
            store.loadRecentServers()
        }
        .onDisappear {
            logger.info("连接页消失，停止局域网扫描")
            discovery.stop()
        }
    }

    private func refreshDiscovery() {
        logger.debug("手动触发局域网扫描刷新")
        discovery.start()
    }
}

struct ConnectionStatusView: View {
    let status: ConnectionStatus

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                if status.phase.isInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
                Text(status.message)
                    .font(.subheadline)
                    .foregroundColor(statusColor)
                Spacer()
            }
            if let detail = status.detail, !detail.isEmpty {
                Text(detail)
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .lineLimit(status.phase.isInFlight ? 2 : 1)
            }
        }
        .padding(10)
        .background(Color.gray.opacity(0.12))
        .cornerRadius(10)
    }

    private var statusColor: Color {
        switch status.phase {
        case .connected:
            return .green
        case .resolving, .healthCheck, .establishingRealtime:
            return .orange
        case .failed:
            return .red
        case .cancelled:
            return .secondary
        case .idle:
            return .secondary
        }
    }
}

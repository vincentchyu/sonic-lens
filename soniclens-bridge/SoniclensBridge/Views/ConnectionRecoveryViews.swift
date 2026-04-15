import SwiftUI

struct ConnectionBootstrapView: View {
    let status: ConnectionStatus
    let server: ServerConfig?
    let onCancel: () -> Void

    var body: some View {
        let displayStatus = status.phase == .idle
            ? ConnectionStatus(
                phase: .resolving,
                message: "正在准备恢复...",
                detail: server?.displayName
            )
            : status

        ZStack {
            AppBackground()

            ScrollView {
                VStack(spacing: 18) {
                    Image(systemName: "arrow.clockwise.circle.fill")
                        .font(.system(size: 40, weight: .semibold))
                        .foregroundStyle(.orange)

                    VStack(spacing: 8) {
                        Text("正在恢复上次连接")
                            .font(.title2.weight(.semibold))

                        Text("我们会先做一次静默健康检查，成功后直接进入主页。")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                            .multilineTextAlignment(.center)
                    }

                    if let server {
                        Text(server.displayName)
                            .font(.subheadline.weight(.medium))
                            .foregroundColor(.secondary)
                            .multilineTextAlignment(.center)
                    }

                    ConnectionStatusView(status: displayStatus)

                    Button("取消恢复", role: .cancel) {
                        onCancel()
                    }
                    .buttonStyle(.bordered)
                }
                .frame(maxWidth: 520)
                .padding(24)
                .glassCard(cornerRadius: 28)
                .padding(.horizontal, 24)
                .frame(maxWidth: .infinity, minHeight: 320)
            }
        }
    }
}

struct ConnectionRecoveryDecisionView: View {
    let server: ServerConfig?
    let status: ConnectionStatus
    let onReconnect: () -> Void
    let onDisconnect: () -> Void

    var body: some View {
        ZStack {
            AppBackground()

            ScrollView {
                VStack(spacing: 18) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 40, weight: .semibold))
                        .foregroundStyle(.red)

                    VStack(spacing: 8) {
                        Text("连接失效")
                            .font(.title2.weight(.semibold))

                        Text("当前服务端暂时不可用。你可以退出当前连接，或重新连接后继续使用。")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                            .multilineTextAlignment(.center)
                    }

                    if let server {
                        VStack(spacing: 6) {
                            Text(server.displayName)
                                .font(.subheadline.weight(.medium))
                            Text("我们会保留这次连接上下文，直到你做决定。")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                        .multilineTextAlignment(.center)
                    }

                    ConnectionStatusView(status: status)

                    HStack(spacing: 12) {
                        Button("退出当前连接", role: .destructive) {
                            onDisconnect()
                        }
                        .buttonStyle(.bordered)

                        Button("重新连接") {
                            onReconnect()
                        }
                        .buttonStyle(.borderedProminent)
                    }
                    .frame(maxWidth: .infinity)
                }
                .frame(maxWidth: 560)
                .padding(24)
                .glassCard(cornerRadius: 28)
                .padding(.horizontal, 24)
                .frame(maxWidth: .infinity, minHeight: 360)
            }
        }
    }
}

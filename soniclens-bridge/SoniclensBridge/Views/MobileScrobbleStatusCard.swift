import SwiftUI
#if os(iOS)
import UIKit

struct MobileScrobbleStatusCard: View {
    @EnvironmentObject private var store: AppStore
    @Environment(AppleMusicScrobbleCoordinator.self) private var coordinator
    @Environment(\.openURL) private var openURL
    @State private var isShowingDetails = false

    var body: some View {
        GlassPanel(cornerRadius: 18, padding: 14) {
            HStack(alignment: .center, spacing: 12) {
                Button {
                    isShowingDetails = true
                } label: {
                    HStack(alignment: .center, spacing: 12) {
                        iconView

                        VStack(alignment: .leading, spacing: 4) {
                            HStack(spacing: 6) {
                                Text("户外 Apple Music 补录")
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(SonicTheme.textPrimary)

                                if coordinator.pendingCount > 0 {
                                    Text("\(coordinator.pendingCount) 条待同步")
                                        .font(.caption2.weight(.semibold))
                                        .padding(.horizontal, 7)
                                        .padding(.vertical, 2)
                                        .background(Color.orange.opacity(0.16), in: Capsule())
                                        .foregroundStyle(.orange)
                                }

                                Image(systemName: "chevron.right")
                                    .font(.caption2.weight(.semibold))
                                    .foregroundStyle(SonicTheme.textSecondary.opacity(0.7))
                            }

                            Text(subtitleText)
                                .font(.caption)
                                .foregroundStyle(SonicTheme.textSecondary)
                                .multilineTextAlignment(.leading)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)

                actionView
            }
        }
        .sheet(isPresented: $isShowingDetails) {
            MobileScrobbleDetailSheet()
                .environmentObject(store)
                .environment(coordinator)
        }
    }

    private var iconView: some View {
        Image(systemName: iconName)
            .font(.system(size: 18, weight: .semibold))
            .foregroundStyle(iconColor)
            .frame(width: 34, height: 34)
            .background(iconColor.opacity(0.14), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
    }

    private var iconName: String {
        switch coordinator.authorizationState {
        case .notDetermined:
            return "music.note.house"
        case .denied, .restricted:
            return "exclamationmark.shield"
        case .authorized:
            return coordinator.pendingCount > 0 ? "arrow.triangle.2.circlepath.circle" : "checkmark.circle"
        case .unsupported:
            return "music.note"
        }
    }

    private var iconColor: Color {
        switch coordinator.authorizationState {
        case .notDetermined:
            return .orange
        case .denied, .restricted:
            return .red
        case .authorized:
            return coordinator.pendingCount > 0 ? .orange : .green
        case .unsupported:
            return .secondary
        }
    }

    private var subtitleText: String {
        switch coordinator.authorizationState {
        case .notDetermined:
            return "开启媒体库权限后，户外用手机 Apple Music 听歌会在连接服务端时自动补录"
        case .denied, .restricted:
            return "媒体库权限已关闭，请前往系统设置开启以捕获户外听歌记录"
        case .authorized:
            if coordinator.isScanning || coordinator.isSyncing {
                return "正在扫描本地媒体库并同步听歌记录..."
            }
            if coordinator.pendingCount > 0 {
                if store.isConnectionHealthy {
                    return "已连接服务端，点击可查看本地待同步曲目明细"
                }
                return "已缓存至本地离线队列，点击可查看扫描结果明细"
            }
            if !store.isConnectionHealthy {
                if coordinator.syncHistory.isEmpty {
                    return "离线监控中 · 新听歌将暂存本地 · 点击查看明细"
                }
                return "离线监控中 · 暂无新增待同步（历史累计已同步 \(coordinator.syncHistory.count) 条） · 点击查看明细"
            }
            if let message = coordinator.statusMessage, !message.isEmpty {
                return "\(message) · 点击查看明细"
            }
            return "本机媒体库与系统播放器双引擎已就绪 · 点击查看明细"
        case .unsupported:
            return "当前设备不支持媒体库扫描"
        }
    }

    @ViewBuilder
    private var actionView: some View {
        switch coordinator.authorizationState {
        case .notDetermined:
            Button("开启授权") {
                Task {
                    await store.requestMobileScrobbleAuthorization()
                }
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.small)
        case .denied, .restricted:
            Button("系统设置") {
                if let settingsURL = URL(string: UIApplication.openSettingsURLString) {
                    openURL(settingsURL)
                }
            }
            .buttonStyle(.bordered)
            .controlSize(.small)
        case .authorized:
            if coordinator.isScanning || coordinator.isSyncing {
                ProgressView()
                    .controlSize(.small)
            } else {
                Button(coordinator.pendingCount > 0 && store.isConnectionHealthy ? "立即同步" : "立即扫描") {
                    Task {
                        await store.refreshMobileScrobblesIfNeeded()
                    }
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
            }
        case .unsupported:
            EmptyView()
        }
    }
}

private struct MobileScrobbleDetailSheet: View {
    @EnvironmentObject private var store: AppStore
    @Environment(AppleMusicScrobbleCoordinator.self) private var coordinator
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List {
                Section {
                    HStack(spacing: 8) {
                        Circle()
                            .fill(store.isConnectionHealthy ? Color.green : Color.orange)
                            .frame(width: 8, height: 8)
                        Text(connectionStateSummary)
                            .font(.caption.weight(.medium))
                            .foregroundStyle(store.isConnectionHealthy ? Color.green : Color.orange)
                    }

                    HStack(spacing: 12) {
                        metricBadge(title: "基线曲目", value: "\(coordinator.baselineTrackCount)", tint: .blue)
                        metricBadge(title: "待同步", value: "\(coordinator.pendingItems.count)", tint: .orange)
                        metricBadge(title: "累计已同步", value: "\(coordinator.syncHistory.count)", tint: .green)
                    }
                    .padding(.vertical, 4)

                    if let statusMessage = coordinator.statusMessage, !statusMessage.isEmpty {
                        Text(statusMessage)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                } header: {
                    Text("本机采集状态")
                }

                Section {
                    if coordinator.pendingItems.isEmpty {
                        Text("当前无待同步记录；在户外用 Apple Music 听歌后，扫描捕获的歌曲会暂存于此，连接服务端后自动同步并推送 Last.fm。")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                            .padding(.vertical, 4)
                    } else {
                        ForEach(coordinator.pendingItems) { item in
                            VStack(alignment: .leading, spacing: 6) {
                                HStack(alignment: .firstTextBaseline, spacing: 8) {
                                    Text(item.title)
                                        .font(.subheadline.weight(.semibold))
                                        .lineLimit(1)
                                    Spacer()
                                    tagBadge(text: item.captureSourceTitle, tint: .orange)
                                }

                                Text("\(item.artist) · \(item.album)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                    .lineLimit(1)

                                Text("播放时间：\(formatTimestamp(item.playedAtTimestamp))")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                            .padding(.vertical, 2)
                        }
                    }
                } header: {
                    Text("本地待同步队列（\(coordinator.pendingItems.count)）")
                }

                Section {
                    if coordinator.syncHistory.isEmpty {
                        Text("暂无已同步记录")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                            .padding(.vertical, 4)
                    } else {
                        ForEach(coordinator.syncHistory) { item in
                            VStack(alignment: .leading, spacing: 6) {
                                HStack(alignment: .firstTextBaseline, spacing: 8) {
                                    Text(item.title)
                                        .font(.subheadline.weight(.semibold))
                                        .lineLimit(1)
                                    Spacer()
                                    tagBadge(text: item.statusDisplayTitle, tint: historyTint(for: item))
                                }

                                Text("\(item.artist) · \(item.album)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                    .lineLimit(1)

                                HStack(spacing: 8) {
                                    Text("播放：\(formatTimestamp(item.playedAtTimestamp))")
                                    Text("·")
                                    Text("来源：\(item.captureSourceTitle)")
                                }
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                            }
                            .padding(.vertical, 2)
                        }
                    }
                } header: {
                    Text("最近同步记录（\(coordinator.syncHistory.count)）")
                }
            }
            .navigationTitle("户外补录明细")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    if coordinator.isScanning || coordinator.isSyncing {
                        ProgressView()
                            .controlSize(.small)
                    } else if coordinator.authorizationState == .authorized {
                        Button(coordinator.pendingCount > 0 && store.isConnectionHealthy ? "扫描并同步" : "立即扫描") {
                            Task {
                                await store.refreshMobileScrobblesIfNeeded()
                            }
                        }
                    }
                }

                ToolbarItem(placement: .topBarTrailing) {
                    Button("完成") {
                        dismiss()
                    }
                }
            }
            .task {
                await coordinator.reloadDetailSnapshot()
            }
        }
    }

    private var connectionStateSummary: String {
        if store.isConnectionHealthy, let server = store.currentServer {
            return "已连接服务端（\(server.displayName)）· 自动补录已启用"
        }
        return "当前未连接服务端（离线模式）· 新记录暂存本地，连接后自动补录"
    }

    private func metricBadge(title: String, value: String, tint: Color) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.headline.weight(.bold))
                .foregroundStyle(tint)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 10)
        .padding(.vertical, 8)
        .background(tint.opacity(0.09), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
    }

    private func tagBadge(text: String, tint: Color) -> some View {
        Text(text)
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 7)
            .padding(.vertical, 2)
            .background(tint.opacity(0.14), in: Capsule())
            .foregroundStyle(tint)
    }

    private func historyTint(for item: MobileScrobbleHistoryItem) -> Color {
        if item.status == "duplicate" {
            return .secondary
        }
        return item.scrobbled ? .green : .orange
    }

    private func formatTimestamp(_ timestamp: Int64) -> String {
        guard timestamp > 0 else { return "未知" }
        return Date(timeIntervalSince1970: TimeInterval(timestamp)).formatted(date: .abbreviated, time: .shortened)
    }
}
#endif

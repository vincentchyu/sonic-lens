import SwiftUI
import Combine

struct TrackDetailView: View {
    @EnvironmentObject private var store: AppStore
    @StateObject private var viewModel = TrackDetailViewModel()

    let track: Track
    @State private var selectedTab: TrackDetailTab = .info
    @State private var previewTime: TimeInterval = 0

    var body: some View {
        ZStack {
            if let message = viewModel.errorMessage {
                ErrorBanner(message: message)
                    .padding(16)
            }

            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    TrackDetailHeader(track: track, playCount: track.playCount)

                    Picker("内容", selection: $selectedTab) {
                        Text("信息").tag(TrackDetailTab.info)
                        Text("歌词").tag(TrackDetailTab.lyrics)
                        Text("音眸").tag(TrackDetailTab.insights)
                    }
                    .pickerStyle(.segmented)
                    .tint(SonicTheme.primary)
                    .frame(width: 320)

                    if selectedTab == .info {
                        infoSection
                    } else if selectedTab == .lyrics {
                        lyricsSection
                    } else {
                        insightsSection
                    }
                }
                .padding(32)
            }

            if viewModel.isLoading {
                LoadingOverlay()
            }
        }
        .navigationTitle("曲目详情")
        .toolbar {
            Menu {
                Button("导出：基础信息") {
                    exportSnapshotPNG(infoSection.padding(32), suggestedFilename: "\(track.artist)-\(track.track)-信息")
                }
                Button("导出：歌词") {
                    exportSnapshotPNG(lyricsSection.padding(32), suggestedFilename: "\(track.artist)-\(track.track)-歌词")
                }
                Button("导出：音眸") {
                    exportSnapshotPNG(insightsSection.padding(32), suggestedFilename: "\(track.artist)-\(track.track)-音眸")
                }
            } label: {
                Label("导出快照", systemImage: "square.and.arrow.up")
            }
        }
        .onAppear {
            if let server = store.currentServer {
                Task {
                    await viewModel.load(
                        using: server,
                        artist: track.artist,
                        album: track.album,
                        track: track.track,
                        trackNumber: track.trackNumber,
                        discNumber: track.discNumber
                    )
                }
            }
        }
        .onReceive(store.$currentServer) { server in
            guard let server else { return }
            Task {
                await viewModel.load(
                    using: server,
                    artist: track.artist,
                    album: track.album,
                    track: track.track,
                    trackNumber: track.trackNumber,
                    discNumber: track.discNumber
                )
            }
        }
    }

    private var lyricsSection: some View {
        DetailSectionCard(title: "歌词") {
            HStack {
                Spacer()
                if viewModel.lyricLines.contains(where: { $0.time != nil }) {
                    Text("LRC")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 8))
                }
            }

            if viewModel.lyricLines.isEmpty {
                Text("暂无歌词")
                    .foregroundColor(.secondary)
            } else {
                if let duration = track.duration, duration > 0, viewModel.lyricLines.contains(where: { $0.time != nil }) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("时间预览（仅用于歌词高亮，不会控制播放器）")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        Slider(value: $previewTime, in: 0...TimeInterval(duration), step: 1)
                    }
                }

                LyricsPane(lines: viewModel.lyricLines, currentLineID: viewModel.currentLineID(forPreviewTime: previewTime))
                    .frame(minHeight: 260)
            }
        }
    }

    private var infoSection: some View {
        DetailSectionCard(title: "基础信息") {
            VStack(alignment: .leading, spacing: 10) {
                InfoRow(title: "曲目", value: track.track)
                InfoRow(title: "艺术家", value: track.artist)
                InfoRow(title: "专辑", value: track.album)
                InfoRow(title: "播放次数", value: "\(track.playCount)")
                if let duration = track.duration {
                    InfoRow(title: "时长", value: formatDuration(duration))
                }
                if let disc = track.discNumber {
                    InfoRow(title: "碟号", value: "\(disc)")
                }
                if let no = track.trackNumber {
                    InfoRow(title: "曲序", value: "\(no)")
                }
            }
            .padding(12)
            .background(Color.primary.opacity(0.04), in: RoundedRectangle(cornerRadius: 12))
        }
    }

    private var insightsSection: some View {
        DetailSectionCard(title: "音眸") {
            if viewModel.insights.isEmpty {
                Text("暂无解析")
                    .foregroundColor(.secondary)
            } else {
                InsightPrimaryContentView(
                    insight: viewModel.insights.primaryInsight,
                    style: .detail,
                    emptyTitle: "暂无音眸",
                    emptySubtitle: "当前曲目还没有可展示的音眸内容。"
                )
            }
        }
    }

    private func formatDuration(_ duration: Int64) -> String {
        let totalSeconds = Int(duration)
        let minutes = totalSeconds / 60
        let seconds = totalSeconds % 60
        return String(format: "%02d:%02d", minutes, seconds)
    }
}

enum TrackDetailTab {
    case info
    case lyrics
    case insights
}

struct LyricsPane: View {
    let lines: [LyricLine]
    let currentLineID: UUID?

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 12) {
                    if lines.isEmpty {
                        Text("暂无歌词")
                            .font(.body)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(lines) { line in
                            let isCurrent = line.id == currentLineID
                            Group {
                                if line.isSectionLabel {
                                    Text(line.text)
                                        .font(.system(size: 11, weight: .semibold))
                                        .tracking(2.0)
                                        .textCase(.uppercase)
                                        .foregroundStyle(.secondary)
                                        .frame(maxWidth: .infinity, alignment: .center)
                                        .padding(.vertical, 6)
                                } else {
                                    Text(line.text)
                                        .font(isCurrent ? .title3 : .body)
                                        .fontWeight(isCurrent ? .semibold : .regular)
                                        .foregroundStyle(isCurrent ? Color.primary : Color.secondary)
                                        .padding(.vertical, 4)
                                        .padding(.horizontal, 6)
                                        .background(isCurrent ? Color.accentColor.opacity(0.12) : Color.clear)
                                        .clipShape(RoundedRectangle(cornerRadius: 6))
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                }
                            }
                            .id(line.id)
                        }
                    }
                }
                .padding(16)
            }
            .mask(
                LinearGradient(
                    colors: [.clear, .black, .black, .clear],
                    startPoint: .top,
                    endPoint: .bottom
                )
            )
            .onChange(of: currentLineID) { _, lineID in
                guard let lineID else { return }
                withAnimation(.easeInOut(duration: 0.35)) {
                    proxy.scrollTo(lineID, anchor: .center)
                }
            }
        }
    }
}

struct InfoRow: View {
    let title: String
    let value: String

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 10) {
            Text(title)
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: 64, alignment: .leading)
            Text(value)
                .font(.body)
                .foregroundStyle(.primary)
            Spacer()
        }
    }
}

struct InsightDetailCard: View {
    let insight: Insight

    var body: some View {
        InsightPrimaryContentView(
            insight: insight,
            style: .detail,
            emptyTitle: "暂无音眸",
            emptySubtitle: "当前曲目还没有可展示的音眸内容。"
        )
    }
}

struct TrackDetailHeader: View {
    let track: Track
    let playCount: Int
    @EnvironmentObject private var store: AppStore

    var body: some View {
        HStack(spacing: 24) {
            RoundedRectangle(cornerRadius: 16)
                .fill(
                    LinearGradient(
                        colors: [
                            Color.accentColor.opacity(0.55),
                            Color.accentColor.opacity(0.18)
                        ],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )
                // TODO: 后端提供 artwork 后改为 AsyncImage 显示真实封面
                .frame(width: 160, height: 160)
                .overlay(
                    Image(systemName: "music.note")
                        .font(.system(size: 34, weight: .semibold))
                        .foregroundStyle(.secondary)
                )
                .shadow(color: Color.black.opacity(0.16), radius: 24, x: 0, y: 16)

            VStack(alignment: .leading, spacing: 10) {
                Text(track.track)
                    .font(.title2)
                    .fontWeight(.semibold)
                Text("\(track.artist) · \(track.album)")
                    .font(.body)
                    .foregroundStyle(.secondary)

                HStack(spacing: 16) {
                    DetailMetaChip(title: "播放次数", value: "\(playCount)")
                    if let duration = track.duration {
                        DetailMetaChip(title: "时长", value: formatDuration(duration))
                    }
                }
            }
            Spacer()
            FavoriteButton(
                isFavorite: store.isFavorite(
                    artist: track.artist,
                    album: track.album,
                    track: track.track,
                    trackNumber: track.trackNumber,
                    discNumber: track.discNumber
                ),
                action: {
                    Task {
                        await store.toggleFavorite(
                            artist: track.artist,
                            album: track.album,
                            track: track.track,
                            trackNumber: track.trackNumber,
                            discNumber: track.discNumber
                        )
                    }
                }
            )
        }
        .padding(18)
        .glassCard(cornerRadius: 16)
    }

    private func formatDuration(_ duration: Int64) -> String {
        let totalSeconds = Int(duration)
        let minutes = totalSeconds / 60
        let seconds = totalSeconds % 60
        return String(format: "%02d:%02d", minutes, seconds)
    }
}

struct DetailMetaChip: View {
    let title: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.caption)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(Color.primary.opacity(0.06), in: RoundedRectangle(cornerRadius: 8))
    }
}

struct FavoriteButton: View {
    let isFavorite: Bool
    let action: () -> Void
    @State private var isHovered = false

    var body: some View {
        Button(action: action) {
            Image(systemName: isFavorite ? "heart.fill" : "heart")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(isFavorite ? Color.red : SonicTheme.textPrimary)
                .frame(width: 32, height: 32)
                .background(
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .fill(isHovered ? Color.primary.opacity(0.12) : Color.primary.opacity(0.06))
                )
        }
        .buttonStyle(.plain)
        .buttonStyle(PressableButtonStyle())
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
        .help(isFavorite ? "取消收藏" : "收藏")
    }
}

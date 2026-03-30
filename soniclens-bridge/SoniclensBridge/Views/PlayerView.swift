import SwiftUI

struct PlayerView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(PlaybackStore.self) private var playbackStore
    @StateObject private var viewModel = PlayerViewModel()
    @State private var selectedTab: PlayerTab = .lyrics
    @State private var currentNowPlaying: NowPlaying
    @State private var isFavorited = false
    @State private var favoriteStatus: FavoriteStatus = .none

    enum FavoriteStatus {
        case none
        case partial
        case full
    }

    let nowPlaying: NowPlaying
    let onClose: () -> Void

    init(nowPlaying: NowPlaying, onClose: @escaping () -> Void) {
        self.nowPlaying = nowPlaying
        self.onClose = onClose
        _currentNowPlaying = State(initialValue: nowPlaying)
    }

    var body: some View {
        ZStack {
            AmbientBackgroundView(
                gradient: LinearGradient(
                    colors: [
                        SonicTheme.dynamicColor(
                            light: .sonicRGBA(0.40, 0.49, 0.92, 1),
                            dark: .sonicRGBA(0.12, 0.16, 0.25, 1)
                        ),
                        SonicTheme.dynamicColor(
                            light: .sonicRGBA(0.46, 0.29, 0.64, 1),
                            dark: .sonicRGBA(0.05, 0.07, 0.12, 1)
                        )
                    ],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                ),
                orbs: [
                    AmbientOrb(
                        color: SonicTheme.lyricsAccent.opacity(0.55),
                        size: 520,
                        blur: 120,
                        opacity: 0.6,
                        offsetFrom: CGSize(width: -220, height: -260),
                        offsetTo: CGSize(width: -120, height: -180),
                        duration: 20
                    ),
                    AmbientOrb(
                        color: SonicTheme.primary.opacity(0.45),
                        size: 460,
                        blur: 130,
                        opacity: 0.5,
                        offsetFrom: CGSize(width: 240, height: 140),
                        offsetTo: CGSize(width: 180, height: 220),
                        duration: 26
                    ),
                    AmbientOrb(
                        color: SonicTheme.lyricsAccent.opacity(0.3),
                        size: 360,
                        blur: 140,
                        opacity: 0.4,
                        offsetFrom: CGSize(width: 80, height: 40),
                        offsetTo: CGSize(width: 120, height: -40),
                        duration: 30
                    )
                ]
            )

            VStack(spacing: 16) {
                LyricsTopBar(
                    nowPlaying: currentNowPlaying,
                    isFavorited: $isFavorited,
                    favoriteStatus: favoriteStatus,
                    onShowInsights: { selectedTab = .insights },
                    onClose: onClose,
                    onRefresh: reloadLyrics
                )

                if let bannerText = viewModel.playbackState.bannerText {
                    PlaybackStatusBanner(text: bannerText)
                }

                Picker("内容", selection: $selectedTab) {
                    Text("歌词").tag(PlayerTab.lyrics)
                    Text("音眸").tag(PlayerTab.insights)
                }
                .pickerStyle(.segmented)
                .tint(SonicTheme.lyricsAccent)
                .frame(width: 260)

                if selectedTab == .lyrics {
                    LyricsLiveSection(lines: viewModel.lyricLines, currentLineID: viewModel.currentLineID)
                } else {
                    InsightLiveSection(items: viewModel.insights)
                }

                Spacer(minLength: 12)
            }
            .padding(24)
        }
        .overlay(alignment: .bottom) {
            PlayerProgressBar(
                currentTime: viewModel.currentTime,
                duration: currentNowPlaying.duration ?? 0
            )
            .padding(.horizontal, 32)
            .padding(.bottom, 16)
        }
        .task {
            if let server = store.currentServer {
                await viewModel.load(
                    using: server,
                    artist: currentNowPlaying.artist,
                    album: currentNowPlaying.album,
                    track: currentNowPlaying.track,
                    trackNumber: currentNowPlaying.trackNumber,
                    discNumber: currentNowPlaying.discNumber
                )
                viewModel.startProgress(position: currentNowPlaying.position, positionMs: currentNowPlaying.positionMs)
            }
        }
        .onChange(of: playbackStore.nowPlaying?.track) { _, _ in
            guard let updated = playbackStore.nowPlaying else { return }
            currentNowPlaying = updated
            updateFavoriteStatus(from: updated)
            if let server = store.currentServer {
                Task {
                    await viewModel.load(
                        using: server,
                        artist: updated.artist,
                        album: updated.album,
                        track: updated.track,
                        trackNumber: updated.trackNumber,
                        discNumber: updated.discNumber
                    )
                    viewModel.startProgress(position: updated.position, positionMs: updated.positionMs)
                }
            }
        }
        .onChange(of: playbackStore.nowPlaying?.position) { _, position in
            viewModel.syncProgress(position: position, positionMs: playbackStore.nowPlaying?.positionMs)
        }
        .onChange(of: playbackStore.nowPlaying?.positionMs) { _, positionMs in
            viewModel.syncProgress(position: playbackStore.nowPlaying?.position, positionMs: positionMs)
        }
        .onDisappear {
            viewModel.stopProgress()
        }
        .onAppear {
            updateFavoriteStatus(from: currentNowPlaying)
        }
    }

    private func reloadLyrics() {
        guard let server = store.currentServer else { return }
        Task {
            await viewModel.load(
                using: server,
                artist: currentNowPlaying.artist,
                album: currentNowPlaying.album,
                track: currentNowPlaying.track,
                trackNumber: currentNowPlaying.trackNumber,
                discNumber: currentNowPlaying.discNumber
            )
            viewModel.startProgress(position: currentNowPlaying.position, positionMs: currentNowPlaying.positionMs)
        }
    }

    private func updateFavoriteStatus(from nowPlaying: NowPlaying) {
        let apple = nowPlaying.isAppleMusicFav ?? false
        let lastfm = nowPlaying.isLastFmFav ?? false
        if apple && lastfm {
            favoriteStatus = .full
            isFavorited = true
        } else if apple || lastfm {
            favoriteStatus = .partial
            isFavorited = true
        } else {
            favoriteStatus = .none
            isFavorited = false
        }
    }
}

enum PlayerTab {
    case lyrics
    case insights
}

struct LyricsTopBar: View {
    @EnvironmentObject private var store: AppStore
    let nowPlaying: NowPlaying
    @Binding var isFavorited: Bool
    let favoriteStatus: PlayerView.FavoriteStatus
    let onShowInsights: () -> Void
    let onClose: () -> Void
    let onRefresh: () -> Void

    var body: some View {
        GlassPanel(cornerRadius: 12, padding: 12) {
            Grid(alignment: .center, horizontalSpacing: 12, verticalSpacing: 0) {
                GridRow {
                    VStack(alignment: .leading, spacing: 6) {
                        if let disc = nowPlaying.discNumber, disc > 0 {
                            TrackInfoBadge(text: "DISC \(disc)")
                        }
                        if let track = nowPlaying.trackNumber, track > 0 {
                            TrackInfoBadge(text: "TRACK \(track)")
                        }
                        TrackInfoBadge(text: "实时")
                        TrackInfoBadge(text: "歌词")
                    }
                    .frame(minWidth: 140, alignment: .leading)

                    VStack(spacing: 4) {
                        Text(nowPlaying.track)
                            .font(.system(size: 18, weight: .bold))
                            .foregroundColor(SonicTheme.textPrimary)
                            .lineLimit(1)
                        Text([nowPlaying.artist, nowPlaying.album].compactMap { $0 }.joined(separator: " · "))
                            .font(.subheadline)
                            .foregroundColor(SonicTheme.textSecondary)
                            .lineLimit(1)
                    }
                    .frame(maxWidth: .infinity)

                    VStack(alignment: .trailing, spacing: 6) {
                        favoriteButton
                        Button("赏析") { onShowInsights() }
                            .buttonStyle(LyricsButtonStyle(accent: SonicTheme.lyricsAccent, isFilled: false))
                        Button("返回看板") { onClose() }
                            .buttonStyle(LyricsButtonStyle(accent: SonicTheme.lyricsAccent, isFilled: false))
                        Button("刷新歌词") { onRefresh() }
                            .buttonStyle(LyricsButtonStyle(accent: SonicTheme.lyricsAccent, isFilled: false))
                    }
                    .frame(minWidth: 140, alignment: .trailing)
                }
            }
        }
    }

    @ViewBuilder
    private var favoriteButton: some View {
        switch favoriteStatus {
        case .full:
            favoriteButtonImpl(text: "★★ 已收藏", isFilled: true)
        case .partial:
            favoriteButtonImpl(text: "★ 已收藏", isFilled: true)
        case .none:
            favoriteButtonImpl(text: "收藏", isFilled: false)
        }
    }

    private func favoriteButtonImpl(text: String, isFilled: Bool) -> some View {
        Button(text) {
            Task {
                await store.toggleFavorite(
                    artist: nowPlaying.artist,
                    album: nowPlaying.album,
                    track: nowPlaying.track,
                    trackNumber: nowPlaying.trackNumber,
                    discNumber: nowPlaying.discNumber
                )
                isFavorited = store.isFavorite(
                    artist: nowPlaying.artist,
                    album: nowPlaying.album,
                    track: nowPlaying.track,
                    trackNumber: nowPlaying.trackNumber,
                    discNumber: nowPlaying.discNumber
                )
            }
        }
        .buttonStyle(LyricsButtonStyle(accent: SonicTheme.lyricsAccent, isFilled: isFilled))
    }
}

struct TrackInfoBadge: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.system(size: 10, weight: .bold))
            .padding(.vertical, 4)
            .padding(.horizontal, 8)
            .background(SonicTheme.lyricsAccent)
            .foregroundColor(.white)
            .cornerRadius(6)
    }
}

struct LyricsButtonStyle: ButtonStyle {
    let accent: Color
    let isFilled: Bool

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 11, weight: .semibold))
            .padding(.vertical, 4)
            .frame(width: 84)
            .background(isFilled ? accent : accent.opacity(0.12))
            .foregroundColor(isFilled ? .white : accent)
            .cornerRadius(6)
            .overlay(
                RoundedRectangle(cornerRadius: 6)
                    .stroke(accent.opacity(isFilled ? 0.0 : 0.35), lineWidth: 1)
            )
            .opacity(configuration.isPressed ? 0.8 : 1.0)
    }
}

struct LyricsLiveSection: View {
    let lines: [LyricLine]
    let currentLineID: UUID?

    var body: some View {
        ScrollViewReader { proxy in
            GlassPanel(cornerRadius: 12, padding: 0) {
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 12) {
                        if lines.isEmpty {
                            Text("暂无歌词")
                                .foregroundColor(SonicTheme.textSecondary)
                        } else {
                            ForEach(lines) { line in
                                let isCurrent = line.id == currentLineID
                                Group {
                                    if line.isSectionLabel {
                                        Text(line.text)
                                            .font(.system(size: 11, weight: .semibold))
                                            .tracking(2.2)
                                            .textCase(.uppercase)
                                            .foregroundColor(SonicTheme.textSecondary.opacity(0.78))
                                            .frame(maxWidth: .infinity, alignment: .center)
                                            .padding(.vertical, 6)
                                    } else {
                                        Text(line.text)
                                            .font(.system(size: isCurrent ? 22 : 18, weight: isCurrent ? .semibold : .regular))
                                            .foregroundColor(isCurrent ? SonicTheme.textPrimary : SonicTheme.textSecondary)
                                            .padding(.vertical, 4)
                                            .padding(.horizontal, 6)
                                            .background(isCurrent ? SonicTheme.lyricsAccent.opacity(0.12) : Color.clear)
                                            .cornerRadius(6)
                                            .frame(maxWidth: .infinity, alignment: .leading)
                                    }
                                }
                                .id(line.id)
                            }
                        }
                    }
                    .padding(18)
                }
            }
            .onChange(of: currentLineID) { _, lineID in
                guard let lineID else { return }
                withAnimation(.easeInOut(duration: 0.35)) {
                    proxy.scrollTo(lineID, anchor: .center)
                }
            }
        }
    }
}

struct InsightLiveSection: View {
    let items: [Insight]

    var body: some View {
        GlassPanel(cornerRadius: 12, padding: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if items.isEmpty {
                        Text("暂无解析")
                            .foregroundColor(SonicTheme.textSecondary)
                    } else {
                        ForEach(items) { insight in
                            GlassPanel(cornerRadius: 12, padding: 14) {
                                Text(insight.analysisSummary ?? "")
                                    .font(.system(size: 16))
                                    .foregroundColor(SonicTheme.textPrimary)
                            }
                        }
                    }
                }
                .padding(16)
            }
        }
    }
}

struct PlayerProgressBar: View {
    let currentTime: TimeInterval
    let duration: Int

    var body: some View {
        ZStack(alignment: .bottomTrailing) {
            Capsule()
                .fill(SonicTheme.progressTrack)
                .frame(height: 6)
                .overlay(
                    GeometryReader { geo in
                        Capsule()
                            .fill(SonicTheme.progressFill)
                            .frame(width: geo.size.width * progress, height: 6, alignment: .leading)
                    }
                )

            HStack(spacing: 10) {
                Text(formatTime(currentTime))
                Text(formatTime(TimeInterval(duration)))
            }
            .font(.caption2)
            .padding(.vertical, 4)
            .padding(.horizontal, 8)
            .background(SonicTheme.card)
            .foregroundColor(SonicTheme.textPrimary)
            .overlay(
                RoundedRectangle(cornerRadius: 8)
                    .stroke(SonicTheme.glassBorder, lineWidth: 1)
            )
            .cornerRadius(8)
            .offset(y: -16)
        }
        .frame(height: 30)
    }

    private var progress: CGFloat {
        guard duration > 0 else { return 0 }
        return min(max(CGFloat(currentTime) / CGFloat(duration), 0), 1)
    }

    private func formatTime(_ seconds: TimeInterval) -> String {
        let total = max(Int(seconds), 0)
        let minutes = total / 60
        let secs = total % 60
        return String(format: "%02d:%02d", minutes, secs)
    }
}

import SwiftUI

#if os(iOS)
private enum PadNowPlayingTab: String, CaseIterable {
    case lyrics = "歌词"
    case insights = "音眸"
}

struct PadNowPlayingView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(FavoriteActionStore.self) private var favoriteActionStore
    @Environment(PlaybackStore.self) private var playbackStore
    @Environment(\.scenePhase) private var scenePhase
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled
    @StateObject private var viewModel = PlayerViewModel()
    @State private var palette: LiquidPalette = .fallback
    @State private var lastArtworkURL: String?
    @State private var lyricsFollowMode = true
    @State private var selectedTab: PadNowPlayingTab = .lyrics
    @State private var animate = false
    @State private var favoriteNoticeDismissTask: Task<Void, Never>?

    let nowPlaying: NowPlaying
    let onClose: () -> Void

    var body: some View {
        GeometryReader { geo in
            let landscape = geo.size.width > geo.size.height
            let current = currentNowPlaying
            let topInset = max(geo.safeAreaInsets.top + 10, 34)

            ZStack {
                NowPlayingLiquidBackground(
                    palette: palette,
                    animate: .constant(animate && scenePhase == .active),
                    isWindowFullscreen: landscape
                )

                VStack(spacing: landscape ? 28 : 0) {
                    PadNowPlayingTopBar(
                        favoriteStatus: favoriteStatus,
                        favoriteActionLoading: favoriteActionLoading,
                        favoriteActionNotice: favoriteActionNotice,
                        lyricsFollowMode: $lyricsFollowMode,
                        selectedTab: $selectedTab,
                        statusBannerText: viewModel.playbackState.bannerText,
                        onFavorite: toggleFavorite,
                        onClose: onClose
                    )
                    .padding(.horizontal, 24)
                    .padding(.top, topInset)

                    if landscape {
                        HStack(alignment: .center, spacing: 36) {
                            PadNowPlayingArtworkColumn(
                                nowPlaying: current,
                                insightTeaser: viewModel.insights.primaryInsight?.teaserText,
                                favoriteStatusTagText: favoriteStatusTagText,
                                favoriteStatusTagTone: favoriteStatusTagTone
                            )
                            .frame(maxWidth: 420)
                            //.offset(x: 80)   // 向右移动 80
                            PadNowPlayingContentColumn(
                                viewModel: viewModel,
                                selectedTab: selectedTab,
                                lyricsFollowMode: $lyricsFollowMode,
                                performanceModeEnabled: performanceModeEnabled
                            )
                            //.offset(x: 80)   // 向右移动 80
                            .padding(.bottom, 60)
                        }
                        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .center)
                        .padding(.horizontal, 50)
                         //.offset(x: 80)   // 向右移动 80
                        
                    } else {
                        ScrollView(.vertical, showsIndicators: false) {
                            VStack(spacing: 24) {
                                PadNowPlayingArtworkColumn(
                                    nowPlaying: current,
                                    insightTeaser: viewModel.insights.primaryInsight?.teaserText,
                                    favoriteStatusTagText: favoriteStatusTagText,
                                    favoriteStatusTagTone: favoriteStatusTagTone
                                )
                                PadNowPlayingContentColumn(
                                    viewModel: viewModel,
                                    selectedTab: selectedTab,
                                    lyricsFollowMode: $lyricsFollowMode,
                                    performanceModeEnabled: performanceModeEnabled
                                )
                            }
                            .padding(.horizontal, 20)
                            .padding(.top, 16)
                            .padding(.bottom, 24)
                        }
                        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
                    }
                }
                .padding(.bottom, landscape ? 54 : max(geo.safeAreaInsets.bottom + 74, 92))

                VStack {
                    Spacer()

                    PadNowPlayingProgressSection(
                        currentTime: viewModel.currentTime,
                        duration: TimeInterval(current.duration ?? 0),
                        progress: progress
                    )
                    .padding(.horizontal, landscape ? 28 : 20)
                    .padding(.bottom, max(geo.safeAreaInsets.bottom, 12))
                }
            }
        }
        .ignoresSafeArea()
        .task {
            animate = true
            await refreshNowPlaying(forcePaletteRefresh: true)
        }
        .onChange(of: trackIdentity) { _, _ in
            Task { await refreshNowPlaying(forcePaletteRefresh: true) }
        }
        .onChange(of: playbackStore.nowPlaying?.artwork) { _, artwork in
            Task { await updatePalette(for: artwork) }
        }
        .onChange(of: playbackSyncToken) { _, token in
            viewModel.syncProgress(
                position: token.position,
                positionMs: token.positionMs,
                receivedAt: token.receivedAt ?? nowPlaying.receivedAt
            )
        }
        .onChange(of: favoriteActionStore.state) { _, state in
            handleFavoriteActionStateChange(state)
        }
        .onChange(of: playbackStore.hasActiveNowPlaying) { _, hasActive in
            if !hasActive {
                onClose()
            }
        }
        .onDisappear {
            viewModel.stopProgress()
            favoriteNoticeDismissTask?.cancel()
            favoriteNoticeDismissTask = nil
        }
    }

    private var currentNowPlaying: NowPlaying {
        playbackStore.nowPlaying ?? nowPlaying
    }

    private var trackIdentity: String {
        let active = currentNowPlaying
        return "\(active.artist)::\(active.album ?? "")::\(active.track)"
    }

    private var playbackSyncToken: PlaybackProgressSyncToken {
        PlaybackProgressSyncToken(nowPlaying: playbackStore.nowPlaying)
    }

    private var favoriteStatus: NowPlayingFavoriteStatus {
        .init(projection: currentNowPlaying.favoriteProjection)
    }

    private var favoriteActionLoading: Bool {
        favoriteActionStore.state.isLoading(matching: currentNowPlaying)
    }

    private var favoriteActionNotice: FavoriteActionNotice? {
        favoriteActionStore.state.notice(matching: currentNowPlaying)
    }

    private var favoriteStatusTagText: String? {
        if favoriteActionLoading {
            return "收藏处理中"
        }
        return favoriteStatus.badgeTitle
    }

    private var favoriteStatusTagTone: NowPlayingArtworkStatusTagTone? {
        if favoriteActionLoading {
            return .loading
        }
        switch favoriteStatus {
        case .full:
            return .success
        case .partial:
            return .warning
        case .pending, .unfavoritePending:
            return .loading
        case .none:
            return nil
        }
    }

    private var progress: Double {
        let duration = TimeInterval(currentNowPlaying.duration ?? 0)
        guard duration > 0 else { return 0 }
        return min(viewModel.currentTime / duration, 1)
    }

    private func toggleFavorite() {
        guard favoriteStatus.allowsFavoriteAction else { return }
        let active = currentNowPlaying
        Task {
            await store.setFavorite(
                artist: active.artist,
                album: active.album,
                track: active.track,
                trackNumber: active.trackNumber,
                discNumber: active.discNumber,
                favorite: true
            )
        }
    }

    private func refreshNowPlaying(forcePaletteRefresh: Bool) async {
        guard let server = store.currentServer else { return }
        let active = currentNowPlaying

        viewModel.startProgress(
            position: active.position,
            positionMs: active.positionMs,
            receivedAt: active.receivedAt
        )
        await Task.yield()

        await viewModel.load(
            using: server,
            artist: active.artist,
            album: active.album,
            track: active.track,
            trackNumber: active.trackNumber,
            discNumber: active.discNumber
        )

        guard forcePaletteRefresh || lastArtworkURL != active.artwork else { return }
        lastArtworkURL = active.artwork
        await updatePalette(for: active.artwork)
    }

    private func updatePalette(for artworkURL: String?) async {
        guard let artworkURL, !artworkURL.isEmpty else {
            withAnimation(.easeInOut(duration: 0.35)) {
                palette = .fallback
            }
            return
        }

        if let extracted = await ArtworkPaletteExtractor.palette(for: artworkURL) {
            withAnimation(.easeInOut(duration: 0.45)) {
                palette = extracted
            }
        }
    }

    private func handleFavoriteActionStateChange(_ state: FavoriteActionState) {
        favoriteNoticeDismissTask?.cancel()
        favoriteNoticeDismissTask = nil

        guard state.notice(matching: currentNowPlaying) != nil else { return }
        favoriteNoticeDismissTask = Task { @MainActor in
            try? await Task.sleep(nanoseconds: 2_000_000_000)
            guard !Task.isCancelled else { return }
            withAnimation(.easeInOut(duration: 0.2)) {
                favoriteActionStore.clear()
            }
        }
    }
}
#endif

private struct PadNowPlayingTopBar: View {
    let favoriteStatus: NowPlayingFavoriteStatus
    let favoriteActionLoading: Bool
    let favoriteActionNotice: FavoriteActionNotice?
    @Binding var lyricsFollowMode: Bool
    @Binding var selectedTab: PadNowPlayingTab
    let statusBannerText: String?
    let onFavorite: () -> Void
    let onClose: () -> Void
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 14) {
                HStack(spacing: 10) {
                    Text("正在播放")
                        .font(.system(size: 28, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)

                    if let statusBannerText {
                        PlaybackStatusBanner(text: statusBannerText)
                    }
                }

                Spacer()

                Picker("", selection: $selectedTab) {
                    ForEach(PadNowPlayingTab.allCases, id: \.self) { tab in
                        Text(tab.rawValue).tag(tab)
                    }
                }
                .pickerStyle(.segmented)
                .frame(maxWidth: 220)

                Button(action: {
                    withAnimation(.easeInOut(duration: 0.16)) {
                        lyricsFollowMode.toggle()
                    }
                }) {
                    Label(lyricsFollowMode ? "跟随播放" : "自由浏览", systemImage: lyricsFollowMode ? "dot.radiowaves.left.and.right" : "hand.draw")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(.white)
                        .padding(.horizontal, 14)
                        .padding(.vertical, 10)
                        .background(Color.white.opacity(0.12), in: Capsule())
                }
                .buttonStyle(.plain)

                NowPlayingFavoriteButton(
                    status: favoriteStatus,
                    isLoading: favoriteActionLoading,
                    action: onFavorite
                )

                Button(action: onClose) {
                    Image(systemName: "xmark")
                        .font(.system(size: 16, weight: .bold))
                        .foregroundStyle(.white)
                        .frame(width: 40, height: 40)
                        .background {
                            if performanceModeEnabled {
                                Circle()
                                    .fill(SonicTheme.card.opacity(0.92))
                            } else {
                                Circle()
                                    .fill(.ultraThinMaterial)
                            }
                        }
                }
                .buttonStyle(.plain)
            }

            if let favoriteActionNotice {
                FavoriteActionNoticeBanner(notice: favoriteActionNotice)
                    .padding(.leading, 2)
                    .transition(.move(edge: .top).combined(with: .opacity))
            } else if favoriteActionLoading {
                PlaybackStatusBanner(text: "收藏处理中")
            }
        }
    }
}

private struct PadNowPlayingArtworkColumn: View {
    let nowPlaying: NowPlaying
    let insightTeaser: String?
    let favoriteStatusTagText: String?
    let favoriteStatusTagTone: NowPlayingArtworkStatusTagTone?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            NowPlayingArtwork(
                artworkURL: nowPlaying.artwork,
                fallbackTitle: nowPlaying.displayAlbumTitle ?? nowPlaying.track,
                badgeText: nowPlaying.sampleRateDisplayText,
                genreText: nowPlaying.genre,
                statusTagText: favoriteStatusTagText,
                statusTagTone: favoriteStatusTagTone
            )
                .frame(maxWidth: .infinity, alignment: .leading)

            VStack(alignment: .leading, spacing: 8) {
                Text(nowPlaying.track)
                    .font(.system(size: 32, weight: .bold, design: .rounded))
                    .foregroundStyle(.white)
                    .lineLimit(3)

                Text([nowPlaying.artist, nowPlaying.displayAlbumTitle].compactMap { $0 }.joined(separator: " · "))
                    .font(.system(size: 17, weight: .medium))
                    .foregroundStyle(.white.opacity(0.78))
                    .lineLimit(2)

                DiscTrackBadgeRow(discNumber: nowPlaying.discNumber, trackNumber: nowPlaying.trackNumber)
            }
            .padding(.top, 0)

            VStack(alignment: .leading, spacing: 8) {
                Text("沉浸模式")
                    .font(.headline)
                    .foregroundStyle(.white.opacity(0.9))
                Text(insightTeaser ?? "专注当前播放，把封面、歌词和音眸洞察放到同一块大画布里。")
                    .font(.subheadline)
                    .foregroundStyle(.white.opacity(0.56))
                    .fixedSize(horizontal: false, vertical: true)
            }
            .padding(.top, 14)
        }
    }
}

private struct PadNowPlayingContentColumn: View {
    @ObservedObject var viewModel: PlayerViewModel
    let selectedTab: PadNowPlayingTab
    @Binding var lyricsFollowMode: Bool
    let performanceModeEnabled: Bool

    var body: some View {
        Group {
            if selectedTab == .lyrics {
                NowPlayingLyricsPanel(
                    lines: viewModel.lyricLines,
                    currentLineID: viewModel.currentLineID,
                    highlightedIndex: viewModel.currentLineIndex,
                    followMode: $lyricsFollowMode
                )
                .frame(maxWidth: .infinity, minHeight: 560)
                .frame(maxHeight: .infinity, alignment: .center)
            } else {
                PadNowPlayingInsightPanel(items: viewModel.insights)
                    .frame(maxHeight: .infinity, alignment: .top)
            }
        }
        .padding(.horizontal, 4)
        .padding(.top, 2)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: selectedTab == .lyrics ? .center : .top)
    }
}

private struct PadNowPlayingInsightPanel: View {
    let items: [Insight]

    var body: some View {
        ScrollView(.vertical, showsIndicators: false) {
            InsightPrimaryContentView(
                insight: items.primaryInsight,
                style: .immersive,
                emptyTitle: "暂无音眸",
                emptySubtitle: "当前曲目还没有生成洞察内容。"
            )
            .padding(.horizontal, 12)
            .padding(.vertical, 16)
        }
        .mask(
            LinearGradient(
                colors: [.clear, .black.opacity(0.94), .black, .black.opacity(0.94), .clear],
                startPoint: .top,
                endPoint: .bottom
            )
        )
    }
}

private struct PadNowPlayingProgressSection: View {
    let currentTime: TimeInterval
    let duration: TimeInterval
    let progress: Double

    var body: some View {
        VStack(spacing: 8) {
            HStack {
                Text(formatTime(currentTime))
                Spacer()
                Text(formatTime(duration))
            }
            .font(.system(size: 13, weight: .semibold, design: .monospaced))
            .foregroundStyle(.white.opacity(0.82))

            ProgressBarView(progress: progress)
                .frame(height: 5)
        }
        .padding(.horizontal, 2)
        .padding(.vertical, 4)
    }

    private func formatTime(_ seconds: TimeInterval) -> String {
        let total = max(Int(seconds), 0)
        return String(format: "%02d:%02d", total / 60, total % 60)
    }
}

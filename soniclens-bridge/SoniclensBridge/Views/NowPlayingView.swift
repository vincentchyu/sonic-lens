import SwiftUI
import NukeUI
#if os(iOS)
import UIKit
#endif

enum MacNowPlayingTab: String, CaseIterable {
    case lyrics = "歌词"
    case insights = "音眸"
}

struct NowPlayingView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(FavoriteActionStore.self) private var favoriteActionStore
    @Environment(PlaybackStore.self) private var playbackStore
    @Environment(\.scenePhase) private var scenePhase
    @StateObject private var viewModel = PlayerViewModel()
    @State private var animate = false
    @State private var palette: LiquidPalette = .fallback
    @State private var lastArtworkURL: String?
    @State private var lyricsFollowMode = true
    @State private var isWindowFullscreen = false
    @State private var selectedTab: MacNowPlayingTab = .lyrics
    @State private var topBarHeight: CGFloat = 0
    @State private var progressBarHeight: CGFloat = 0
    @State private var favoriteNoticeDismissTask: Task<Void, Never>?

    let nowPlaying: NowPlaying
    let onClose: () -> Void

    var body: some View {
        GeometryReader { geo in
            nowPlayingCanvas(geo: geo)
        }
        .task {
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
        .onDisappear {
            animate = false
            viewModel.stopProgress()
            favoriteNoticeDismissTask?.cancel()
            favoriteNoticeDismissTask = nil
        }
        .onAppear {
            animate = true
            refreshFullscreenState()
        }
        .modifier(FullscreenStateObserver(onChange: refreshFullscreenState))
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

    private var duration: TimeInterval {
        TimeInterval(currentNowPlaying.duration ?? 0)
    }

    private var progress: Double {
        guard duration > 0 else { return 0 }
        return min(viewModel.currentTime / duration, 1.0)
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
            track: active.track
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

    private func refreshFullscreenState() {
        #if os(macOS)
        let window = NSApp.keyWindow ?? NSApp.mainWindow
        isWindowFullscreen = window?.styleMask.contains(.fullScreen) ?? false
        #else
        isWindowFullscreen = false
        #endif
    }

    private func handleFavoriteActionStateChange(_ state: FavoriteActionState) {
        favoriteNoticeDismissTask?.cancel()
        favoriteNoticeDismissTask = nil

        guard state.notice(matching: currentNowPlaying) != nil else { return }
        favoriteNoticeDismissTask = Task { @MainActor in
            try? await Task.sleep(nanoseconds: 3_500_000_000)
            guard !Task.isCancelled else { return }
            withAnimation(.easeInOut(duration: 0.2)) {
                favoriteActionStore.clear()
            }
        }
    }

    private func adaptivePanelViewportHeight(for windowHeight: CGFloat) -> CGFloat {
        let reservedHeight = topBarHeight + progressBarHeight + 28 + 52 + 46 + 10
        return max(0, windowHeight - reservedHeight)
    }

    @ViewBuilder
    private func nowPlayingCanvas(geo: GeometryProxy) -> some View {
        let displayNowPlaying = currentNowPlaying
        let panelViewportHeight = adaptivePanelViewportHeight(for: geo.size.height)

        ZStack {
            NowPlayingLiquidBackground(
                palette: palette,
                animate: .constant(animate && scenePhase == .active),
                isWindowFullscreen: isWindowFullscreen
            )

            nowPlayingMainContent(
                displayNowPlaying: displayNowPlaying,
                panelViewportHeight: panelViewportHeight
            )

            nowPlayingProgressOverlay
        }
        .clipped()
    }

    @ViewBuilder
    private func nowPlayingMainContent(
        displayNowPlaying: NowPlaying,
        panelViewportHeight: CGFloat
    ) -> some View {
        VStack(spacing: 28) {
            NowPlayingTopBar(
                favoriteStatus: favoriteStatus,
                favoriteActionLoading: favoriteActionLoading,
                favoriteActionNotice: favoriteActionNotice,
                lyricsFollowMode: $lyricsFollowMode,
                selectedTab: $selectedTab,
                statusBannerText: viewModel.playbackState.bannerText,
                onFavorite: {
                    guard favoriteStatus.allowsFavoriteAction else { return }
                    Task {
                        await store.setFavorite(
                            artist: displayNowPlaying.artist,
                            album: displayNowPlaying.album,
                            track: displayNowPlaying.track,
                            trackNumber: displayNowPlaying.trackNumber,
                            discNumber: displayNowPlaying.discNumber,
                            favorite: true
                        )
                    }
                },
                onClose: onClose
            )
            .readHeight { topBarHeight = $0 }

            HStack(alignment: .top, spacing: 48) {
                NowPlayingLeftPanel(
                    nowPlaying: displayNowPlaying,
                    insightSummary: viewModel.insights.primaryInsight?.teaserText,
                    favoriteStatusTagText: favoriteStatusTagText,
                    favoriteStatusTagTone: favoriteStatusTagTone
                )
                .frame(maxWidth: 380)
                .frame(height: panelViewportHeight, alignment: .topLeading)
                .clipped()

                Group {
                    if selectedTab == .lyrics {
                        NowPlayingLyricsPanel(
                            lines: viewModel.lyricLines,
                            currentLineID: viewModel.currentLineID,
                            highlightedIndex: viewModel.currentLineIndex,
                            followMode: $lyricsFollowMode
                        )
                    } else {
                        MacNowPlayingInsightPanel(
                            items: viewModel.insights,
                            selectedInsightIndex: $viewModel.selectedInsightIndex,
                            insightViewMode: $viewModel.insightViewMode
                        )
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
            }
            .frame(maxWidth: .infinity, maxHeight: panelViewportHeight, alignment: .topLeading)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .padding(.horizontal, 34)
        .padding(.top, 52)
        .padding(.bottom, 46)
    }

    private var nowPlayingProgressOverlay: some View {
        VStack {
            Spacer()
            NowPlayingBottomProgressBar(
                currentTime: viewModel.currentTime,
                duration: duration,
                progress: progress
            )
            .readHeight { progressBarHeight = $0 }
            .padding(.horizontal, 22)
            .padding(.bottom, 10)
        }
    }
}

private struct FullscreenStateObserver: ViewModifier {
    let onChange: () -> Void

    func body(content: Content) -> some View {
        #if os(macOS)
        content
            .onReceive(NotificationCenter.default.publisher(for: NSWindow.didEnterFullScreenNotification)) { _ in
                onChange()
            }
            .onReceive(NotificationCenter.default.publisher(for: NSWindow.didExitFullScreenNotification)) { _ in
                onChange()
            }
        #else
        content
        #endif
    }
}

private struct HeightReporter: ViewModifier {
    let onChange: (CGFloat) -> Void

    func body(content: Content) -> some View {
        content.background(
            GeometryReader { proxy in
                Color.clear
                    .preference(key: MeasuredHeightKey.self, value: proxy.size.height)
            }
        )
        .onPreferenceChange(MeasuredHeightKey.self, perform: onChange)
    }
}

private struct MeasuredHeightKey: PreferenceKey {
    static let defaultValue: CGFloat = 0

    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = max(value, nextValue())
    }
}

private extension View {
    func readHeight(_ onChange: @escaping (CGFloat) -> Void) -> some View {
        modifier(HeightReporter(onChange: onChange))
    }
}

struct NowPlayingTopBar: View {
    let favoriteStatus: NowPlayingFavoriteStatus
    let favoriteActionLoading: Bool
    let favoriteActionNotice: FavoriteActionNotice?
    @Binding var lyricsFollowMode: Bool
    @Binding var selectedTab: MacNowPlayingTab
    let statusBannerText: String?
    let onFavorite: () -> Void
    let onClose: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .top) {
                HStack(spacing: 10) {
                    Text("正在播放")
                        .font(.system(size: 28, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)

                    if let statusBannerText {
                        PlaybackStatusBanner(text: statusBannerText)
                    }
                }

                Spacer()
                VStack(alignment: .trailing, spacing: 8) {
                    HStack(spacing: 10) {
                        Picker("", selection: $selectedTab) {
                            ForEach(MacNowPlayingTab.allCases, id: \.self) { tab in
                                Text(tab.rawValue).tag(tab)
                            }
                        }
                        .pickerStyle(.segmented)
                        .frame(width: 220)

                        Button(action: {
                            withAnimation(.easeInOut(duration: 0.16)) {
                                lyricsFollowMode.toggle()
                            }
                        }) {
                            HStack(spacing: 6) {
                                Image(systemName: lyricsFollowMode ? "dot.radiowaves.left.and.right" : "hand.draw")
                                    .font(.system(size: 11, weight: .semibold))
                                Text(lyricsFollowMode ? "跟随播放" : "自由浏览")
                                    .font(.system(size: 11, weight: .semibold))
                            }
                            .foregroundStyle(.white.opacity(0.96))
                            .padding(.horizontal, 10)
                            .padding(.vertical, 8)
                            .background(
                                RoundedRectangle(cornerRadius: 10, style: .continuous)
                                    .fill(lyricsFollowMode ? Color.blue.opacity(0.28) : Color.white.opacity(0.12))
                            )
                            .overlay(
                                RoundedRectangle(cornerRadius: 10, style: .continuous)
                                    .stroke(.white.opacity(lyricsFollowMode ? 0.22 : 0.12), lineWidth: 1)
                            )
                        }
                        .buttonStyle(.plain)

                        NowPlayingFavoriteButton(
                            status: favoriteStatus,
                            isLoading: favoriteActionLoading,
                            action: onFavorite
                        )
                        Button(action: onClose) {
                            Image(systemName: "xmark")
                                .font(.system(size: 14, weight: .bold))
                                .frame(width: 32, height: 32)
                                .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 10))
                        }
                        .buttonStyle(.plain)
                    }

                    if favoriteActionLoading {
                        MacFavoriteActionToast(
                            tone: .loading,
                            title: "收藏处理中",
                            message: "正在同步 Apple Music / Last.fm"
                        )
                        .transition(.move(edge: .top).combined(with: .opacity).combined(with: .scale(scale: 0.96)))
                    } else if let favoriteActionNotice {
                        MacFavoriteActionToast(
                            tone: favoriteActionNotice.style == .success ? .success : .failure,
                            title: favoriteActionNotice.style == .success ? "收藏成功" : "收藏失败",
                            message: favoriteActionNotice.message
                        )
                        .transition(.move(edge: .top).combined(with: .opacity).combined(with: .scale(scale: 0.96)))
                    }
                }
            }
        }
        .padding(.trailing, 6)
    }
}

private struct MacFavoriteActionToast: View {
    enum Tone {
        case loading
        case success
        case failure
    }

    let tone: Tone
    let title: String
    let message: String

    var body: some View {
        HStack(spacing: 10) {
            Group {
                switch tone {
                case .loading:
                    ProgressView()
                        .controlSize(.small)
                case .success:
                    Image(systemName: "checkmark.circle.fill")
                case .failure:
                    Image(systemName: "xmark.octagon.fill")
                }
            }
            .font(.system(size: 12, weight: .bold))
            .foregroundStyle(foregroundColor)

            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.system(size: 12, weight: .bold))
                    .foregroundStyle(foregroundColor)

                Text(message)
                    .font(.system(size: 11, weight: .medium))
                    .foregroundStyle(foregroundColor.opacity(0.8))
                    .lineLimit(1)
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
        .frame(minWidth: 240, alignment: .leading)
        .background(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .fill(backgroundColor)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .stroke(borderColor, lineWidth: 1)
        )
        .shadow(color: shadowColor, radius: 16, x: 0, y: 6)
    }

    private var foregroundColor: Color {
        switch tone {
        case .loading:
            return Color.orange.opacity(0.97)
        case .success:
            return Color.green.opacity(0.97)
        case .failure:
            return Color.red.opacity(0.97)
        }
    }

    private var backgroundColor: Color {
        switch tone {
        case .loading:
            return Color.orange.opacity(0.18)
        case .success:
            return Color.green.opacity(0.16)
        case .failure:
            return Color.red.opacity(0.16)
        }
    }

    private var borderColor: Color {
        switch tone {
        case .loading:
            return Color.orange.opacity(0.3)
        case .success:
            return Color.green.opacity(0.28)
        case .failure:
            return Color.red.opacity(0.28)
        }
    }

    private var shadowColor: Color {
        switch tone {
        case .loading:
            return Color.orange.opacity(0.15)
        case .success:
            return Color.green.opacity(0.12)
        case .failure:
            return Color.red.opacity(0.12)
        }
    }
}

struct MacNowPlayingInsightPanel: View {
    let items: [Insight]
    @Binding var selectedInsightIndex: Int
    @Binding var insightViewMode: InsightViewMode

    private var currentInsight: Insight? {
        guard !items.isEmpty else { return nil }
        if selectedInsightIndex < items.count {
            return items[selectedInsightIndex]
        }
        return items.first
    }

    var body: some View {
        HStack(alignment: .top, spacing: 24) {
            ScrollView(.vertical, showsIndicators: false) {
                if insightViewMode == .history {
                    InsightHistoryList(
                        insights: items,
                        selectedIndex: Binding(
                            get: { selectedInsightIndex },
                            set: { 
                                selectedInsightIndex = $0
                                insightViewMode = .current
                            }
                        )
                    )
                    .padding(.horizontal, 14)
                    .padding(.vertical, 18)
                } else {
                    InsightPrimaryContentView(
                        insight: currentInsight,
                        style: .immersive,
                        emptyTitle: "暂无音眸",
                        emptySubtitle: "当前曲目还没有生成洞察内容。"
                    )
                    .padding(.horizontal, 14)
                    .padding(.vertical, 18)
                    .id("insight-\(selectedInsightIndex)")
                }
            }

            if items.count > 1 {
                InsightVersionPicker(
                    viewMode: $insightViewMode,
                    historyCount: items.count,
                    axis: .vertical
                )
                .padding(.top, 6)
            }
        }
        .mask(
            LinearGradient(
                colors: [.clear, .black.opacity(0.94), .black, .black.opacity(0.94), .clear],
                startPoint: .top,
                endPoint: .bottom
            )
        )
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }
}

struct NowPlayingLeftPanel: View {
    let nowPlaying: NowPlaying
    let insightSummary: String?
    let favoriteStatusTagText: String?
    let favoriteStatusTagTone: NowPlayingArtworkStatusTagTone?

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            NowPlayingArtwork(
                artworkURL: nowPlaying.artwork,
                fallbackTitle: nowPlaying.displayAlbumTitle ?? nowPlaying.track,
                badgeText: nowPlaying.sampleRateDisplayText,
                statusTagText: favoriteStatusTagText,
                statusTagTone: favoriteStatusTagTone
            )

            VStack(alignment: .leading, spacing: 8) {
                Text(nowPlaying.track)
                    .font(.system(size: 30, weight: .bold))
                    .foregroundStyle(.white)
                    .lineLimit(2)

                Text([nowPlaying.artist, nowPlaying.displayAlbumTitle].compactMap { $0 }.joined(separator: " · "))
                    .font(.system(size: 16, weight: .medium))
                    .foregroundStyle(Color.white.opacity(0.82))
                    .lineLimit(2)
                DiscTrackBadgeRow(discNumber: nowPlaying.discNumber, trackNumber: nowPlaying.trackNumber)
                
            }

            NowPlayingAdaptiveSummarySection(
                title: "沉浸模式",
                text: insightSummary ?? "专注当前播放，把封面、歌词和音眸洞察放到同一块大画布里。"
            )
            .padding(.top, 0)
        }
    }
}

private struct NowPlayingAdaptiveSummarySection: View {
    let title: String
    let text: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.headline)
                .foregroundStyle(.white.opacity(0.9))

            ViewThatFits(in: .vertical) {
                Text(text)
                    .font(.subheadline)
                    .foregroundStyle(.white.opacity(0.56))
                    .fixedSize(horizontal: false, vertical: true)

                Text(text)
                    .font(.subheadline)
                    .foregroundStyle(.white.opacity(0.56))
                    .lineLimit(6)
                    .truncationMode(.tail)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct NowPlayingBottomProgressBar: View {
    let currentTime: TimeInterval
    let duration: TimeInterval
    let progress: Double

    @State private var isHovered = false

    var body: some View {
        VStack(spacing: 6) {
            if isHovered {
                HStack {
                    Text(formatTime(currentTime))
                    Spacer()
                    Text(formatTime(duration))
                }
                .font(.system(size: 12, weight: .semibold, design: .monospaced))
                .foregroundStyle(.white.opacity(0.86))
                .transition(.opacity.combined(with: .move(edge: .bottom)))
            }

            ProgressBarView(progress: progress)
                .frame(height: 5)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 6)
        .contentShape(Rectangle())
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.14)) {
                isHovered = hovering
            }
        }
    }

    private func formatTime(_ seconds: TimeInterval) -> String {
        let total = max(Int(seconds), 0)
        let minutes = total / 60
        let secs = total % 60
        return String(format: "%02d:%02d", minutes, secs)
    }
}

struct DiscTrackBadgeRow: View {
    let discNumber: Int?
    let trackNumber: Int?

    private var hasDisc: Bool { (discNumber ?? 0) > 0 }
    private var hasTrack: Bool { (trackNumber ?? 0) > 0 }

    var body: some View {
        HStack(spacing: 8) {
            if hasDisc {
                CapsuleBadge(text: "DISC \(discNumber ?? 0)")
            }
            if hasTrack {
                CapsuleBadge(text: "TRACK \(trackNumber ?? 0)")
            }
        }
        .opacity((hasDisc || hasTrack) ? 1 : 0)
        .frame(height: (hasDisc || hasTrack) ? nil : 0)
    }
}

struct CapsuleBadge: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.system(size: 11, weight: .bold, design: .monospaced))
            .foregroundStyle(.white.opacity(0.94))
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(
                Capsule(style: .continuous)
                    .fill(.white.opacity(0.14))
            )
            .overlay(
                Capsule(style: .continuous)
                    .stroke(.white.opacity(0.26), lineWidth: 1)
            )
    }
}

struct NowPlayingFavoriteButton: View {
    let status: NowPlayingFavoriteStatus
    let isLoading: Bool
    let action: () -> Void
    @State private var isHovered = false

    var body: some View {
        Button(action: action) {
            Group {
                if isLoading {
                    ProgressView()
                        .controlSize(.small)
                        .tint(iconColor)
                } else {
                    Image(systemName: iconName)
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(iconColor)
                }
            }
            .frame(width: 32, height: 32)
            .background(
                RoundedRectangle(cornerRadius: 10)
                    .fill(backgroundOpacity)
            )
        }
        .buttonStyle(.plain)
        .buttonStyle(PressableButtonStyle())
        .disabled(!status.allowsFavoriteAction || isLoading)
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
        .help(helpText)
    }

    private var iconName: String {
        if isLoading {
            return "clock.fill"
        }
        switch status {
        case .full: return "star.fill"
        case .partial: return "star.leadinghalf.filled"
        case .pending: return "clock.fill"
        case .unfavoritePending: return "clock.fill"
        case .none: return "star"
        }
    }

    private var iconColor: Color {
        if isLoading {
            return Color.orange.opacity(0.96)
        }
        switch status {
        case .none:
            return Color.white.opacity(0.95)
        case .pending:
            return Color.orange.opacity(0.96)
        case .unfavoritePending:
            return Color.white.opacity(0.94)
        case .partial, .full:
            return Color.yellow.opacity(0.95)
        }
    }

    private var backgroundOpacity: Color {
        if isLoading {
            return Color.orange.opacity(isHovered ? 0.28 : 0.18)
        }
        switch status {
        case .full:
            return Color.yellow.opacity(isHovered ? 0.34 : 0.26)
        case .partial:
            return Color.yellow.opacity(isHovered ? 0.24 : 0.16)
        case .pending:
            return Color.orange.opacity(isHovered ? 0.28 : 0.18)
        case .unfavoritePending:
            return Color.white.opacity(isHovered ? 0.24 : 0.16)
        case .none:
            return Color.white.opacity(isHovered ? 0.20 : 0.12)
        }
    }

    private var helpText: String {
        if isLoading {
            return "收藏处理中"
        }
        switch status {
        case .full: return "已同步到 Apple Music + Last.fm"
        case .partial: return "已在单平台收藏，点击补全双平台收藏"
        case .pending: return "收藏已记录，等待后端归因同步"
        case .unfavoritePending: return "取消收藏已记录，等待后端归因同步"
        case .none: return "收藏到 Apple Music / Last.fm"
        }
    }
}

enum NowPlayingFavoriteStatus {
    case none
    case partial
    case full
    case pending
    case unfavoritePending

    init(projection: TrackFavoriteProjection) {
        switch projection.favoriteState {
        case .favoritePending:
            self = .pending
        case .unfavoritePending:
            self = .unfavoritePending
        case .favorited, .notFavorited:
            if projection.appleMusic && projection.lastfm {
                self = .full
            } else if projection.appleMusic || projection.lastfm {
                self = .partial
            } else {
                self = .none
            }
        }
    }

    var allowsFavoriteAction: Bool {
        switch self {
        case .none, .partial:
            return true
        case .full, .pending, .unfavoritePending:
            return false
        }
    }

    var badgeTitle: String? {
        switch self {
        case .none:
            return nil
        case .partial:
            return "已在单平台收藏"
        case .full:
            return "已双端收藏"
        case .pending:
            return "收藏已记录，等待归因"
        case .unfavoritePending:
            return "取消收藏处理中"
        }
    }
}

struct NowPlayingFavoriteStatusBadge: View {
    let status: NowPlayingFavoriteStatus
    let title: String

    var body: some View {
        Label(title, systemImage: iconName)
            .font(.caption.weight(.semibold))
            .foregroundStyle(foregroundColor)
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(
                Capsule(style: .continuous)
                    .fill(backgroundColor)
            )
            .overlay(
                Capsule(style: .continuous)
                    .stroke(borderColor, lineWidth: 1)
            )
    }

    private var iconName: String {
        switch status {
        case .none:
            return "star"
        case .partial:
            return "star.leadinghalf.filled"
        case .full:
            return "star.fill"
        case .pending, .unfavoritePending:
            return "clock.fill"
        }
    }

    private var foregroundColor: Color {
        switch status {
        case .pending:
            return Color.orange.opacity(0.96)
        case .unfavoritePending:
            return Color.white.opacity(0.92)
        case .none:
            return Color.white.opacity(0.92)
        case .partial, .full:
            return Color.yellow.opacity(0.96)
        }
    }

    private var backgroundColor: Color {
        switch status {
        case .pending:
            return Color.orange.opacity(0.16)
        case .unfavoritePending:
            return Color.white.opacity(0.12)
        case .none:
            return Color.white.opacity(0.12)
        case .partial, .full:
            return Color.yellow.opacity(0.16)
        }
    }

    private var borderColor: Color {
        switch status {
        case .pending:
            return Color.orange.opacity(0.28)
        case .unfavoritePending:
            return Color.white.opacity(0.16)
        case .none:
            return Color.white.opacity(0.16)
        case .partial, .full:
            return Color.yellow.opacity(0.24)
        }
    }
}

struct NowPlayingArtwork: View {
    let artworkURL: String?
    var fallbackTitle: String? = nil
    var badgeText: String? = nil
    var statusTagText: String? = nil
    var statusTagTone: NowPlayingArtworkStatusTagTone? = nil
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    var body: some View {
        ArtworkSquareView(
            artworkURL: artworkURL,
            fallbackTitle: fallbackTitle,
            size: 340,
            cornerRadius: 18,
            style: .vivid
        )
        .frame(width: 340, height: 340)
        .overlay(
            RoundedRectangle(cornerRadius: 18)
                .stroke(.white.opacity(0.16), lineWidth: 1)
        )
        .overlay(alignment: .topTrailing) {
            HStack(spacing: 8) {
                if let badgeText, !badgeText.isEmpty {
                    NowPlayingCornerTag(text: badgeText)
                }

                if let statusTagText, !statusTagText.isEmpty, let statusTagTone {
                    NowPlayingArtworkStatusTag(text: statusTagText, tone: statusTagTone)
                }
            }
            .padding(12)
        }
        .shadow(
            color: .black.opacity(performanceModeEnabled ? 0.18 : 0.28),
            radius: performanceModeEnabled ? 20 : 32,
            x: 0,
            y: performanceModeEnabled ? 14 : 24
        )
    }
}

struct NowPlayingCornerTag: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.system(size: 11, weight: .bold, design: .rounded))
            .foregroundStyle(.white.opacity(0.96))
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(
                Capsule(style: .continuous)
                    .fill(Color.black.opacity(0.46))
            )
            .overlay(
                Capsule(style: .continuous)
                    .stroke(.white.opacity(0.18), lineWidth: 1)
            )
            .shadow(color: .black.opacity(0.14), radius: 10, x: 0, y: 4)
    }
}

enum NowPlayingArtworkStatusTagTone {
    case loading
    case success
    case warning
    case neutral
}

struct NowPlayingArtworkStatusTag: View {
    let text: String
    let tone: NowPlayingArtworkStatusTagTone

    var body: some View {
        HStack(spacing: 6) {
            Group {
                switch tone {
                case .loading:
                    ProgressView()
                        .controlSize(.mini)
                case .success:
                    Image(systemName: "star.fill")
                case .warning:
                    Image(systemName: "star.leadinghalf.filled")
                case .neutral:
                    Image(systemName: "star")
                }
            }
            .font(.system(size: 11, weight: .bold))
            .foregroundStyle(iconColor)

            Text(text)
                .font(.system(size: 11, weight: .bold))
                .foregroundStyle(.white)
                .lineLimit(1)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 7)
        .background(
            Capsule(style: .continuous)
                .fill(backgroundColor)
        )
        .overlay(
            Capsule(style: .continuous)
                .stroke(borderColor, lineWidth: 1)
        )
        .shadow(color: shadowColor, radius: 10, x: 0, y: 4)
    }

    private var iconColor: Color {
        switch tone {
        case .loading:
            return Color.orange.opacity(0.98)
        case .success:
            return Color.green.opacity(0.98)
        case .warning:
            return Color.yellow.opacity(0.98)
        case .neutral:
            return Color.white.opacity(0.9)
        }
    }

    private var backgroundColor: Color {
        switch tone {
        case .loading:
            return Color.black.opacity(0.58)
        case .success:
            return Color.black.opacity(0.50)
        case .warning:
            return Color.black.opacity(0.52)
        case .neutral:
            return Color.black.opacity(0.52)
        }
    }

    private var borderColor: Color {
        switch tone {
        case .loading:
            return Color.orange.opacity(0.24)
        case .success:
            return Color.green.opacity(0.26)
        case .warning:
            return Color.yellow.opacity(0.24)
        case .neutral:
            return Color.white.opacity(0.16)
        }
    }

    private var shadowColor: Color {
        switch tone {
        case .loading:
            return Color.black.opacity(0.18)
        case .success:
            return Color.black.opacity(0.16)
        case .warning:
            return Color.black.opacity(0.16)
        case .neutral:
            return Color.black.opacity(0.16)
        }
    }
}

struct NowPlayingLyricsPanel: View {
    let lines: [LyricLine]
    let currentLineID: UUID?
    let highlightedIndex: Int?
    @Binding var followMode: Bool
    var scrollRequestToken: Int = 0
    var scrollEnabled: Bool = true
    var bottomContentInset: CGFloat = 24
    var onUserScroll: (() -> Void)? = nil

    private let visibleRadius = 5
    @State private var didNotifyUserScroll = false
    @State private var suppressFollowAnimation = true

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView(.vertical, showsIndicators: false) {
                LazyVStack(alignment: .center, spacing: 8) {
                    if lines.isEmpty {
                        Text("暂无歌词")
                            .font(.system(size: 18, weight: .semibold))
                            .foregroundStyle(.white.opacity(0.72))
                    } else {
                        ForEach(Array(lines.enumerated()), id: \.element.id) { item in
                            let style = lyricStyle(for: item.offset, highlightedIndex: highlightedIndex)
                            Group {
                                if item.element.isSectionLabel {
                                    Text(item.element.text)
                                        .font(.system(size: 11, weight: .semibold))
                                        .tracking(2.4)
                                        .textCase(.uppercase)
                                        .foregroundStyle(.white.opacity(0.5))
                                        .frame(maxWidth: .infinity, alignment: .center)
                                        .padding(.vertical, 6)
                                } else {
                                    Text(item.element.text)
                                        .font(.system(size: style.size, weight: style.weight))
                                        .lineSpacing(1)
                                        .multilineTextAlignment(.center)
                                        .foregroundStyle(.white)
                                        .opacity(style.opacity)
                                        .scaleEffect(style.scale)
                                        .frame(maxWidth: .infinity, alignment: .center)
                                }
                            }
                            .id(item.element.id)
                        }
                    }
                }
                .padding(.horizontal, 6)
                .padding(.vertical, 20)
                .padding(.bottom, bottomContentInset)
            }
            .scrollDisabled(!scrollEnabled)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
            .mask(
                LinearGradient(
                    colors: [.clear, .black.opacity(0.94), .black, .black.opacity(0.94), .clear],
                    startPoint: .top,
                    endPoint: .bottom
                )
            )
            .onChange(of: currentLineID) { _, lineID in
                guard followMode else { return }
                guard let lineID else { return }
                if suppressFollowAnimation {
                    proxy.scrollTo(lineID, anchor: .center)
                } else {
                    withAnimation(.easeInOut(duration: 0.2)) {
                        proxy.scrollTo(lineID, anchor: .center)
                    }
                }
            }
            .task {
                try? await Task.sleep(nanoseconds: 450_000_000)
                guard !Task.isCancelled else { return }
                suppressFollowAnimation = false
            }
            .onAppear {
                if let currentLineID, followMode {
                    proxy.scrollTo(currentLineID, anchor: .center)
                }
            }
            .onChange(of: scrollRequestToken) { _, _ in
                guard let currentLineID else { return }
                withAnimation(.easeInOut(duration: 0.2)) {
                    proxy.scrollTo(currentLineID, anchor: .center)
                }
            }
            .simultaneousGesture(
                DragGesture(minimumDistance: 8)
                    .onChanged { _ in
                        guard !didNotifyUserScroll else { return }
                        didNotifyUserScroll = true
                        onUserScroll?()
                    }
                    .onEnded { _ in
                        didNotifyUserScroll = false
                    }
            )
        }
    }

    private func lyricStyle(for index: Int, highlightedIndex: Int?) -> LyricTypography {
        guard let highlightedIndex else {
            return LyricTypography(size: 20, weight: .medium, opacity: 0.58, scale: 0.98)
        }
        let distance = abs(index - highlightedIndex)
        if distance == 0 {
            return LyricTypography(size: 30, weight: .bold, opacity: 1.0, scale: 1.0)
        }
        if distance > visibleRadius {
            return LyricTypography(size: 18, weight: .regular, opacity: 0.22, scale: 0.92)
        }
        let falloff = Double(distance) / Double(visibleRadius)
        let size = CGFloat(30.0 - falloff * 10.0) // 30 -> 20
        let opacity = 0.86 - falloff * 0.46       // 0.86 -> 0.40
        let scale = 1.0 - falloff * 0.06          // 1.00 -> 0.94
        let weight: Font.Weight = distance <= 2 ? .semibold : .medium
        return LyricTypography(size: size, weight: weight, opacity: opacity, scale: scale)
    }
}

private struct LyricTypography {
    let size: CGFloat
    let weight: Font.Weight
    let opacity: Double
    let scale: CGFloat
}

struct NowPlayingLiquidBackground: View {
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    let palette: LiquidPalette
    @Binding var animate: Bool
    let isWindowFullscreen: Bool

    var body: some View {
        let simplified = performanceModeEnabled || automaticallySimplified

        GeometryReader { geo in
            ZStack {
                LinearGradient(
                    colors: [palette.base, palette.depth],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )

                if !simplified {
                    liquidBlob(
                        color: palette.blobA.opacity(isWindowFullscreen ? 0.45 : 0.55),
                        size: max(geo.size.width * (isWindowFullscreen ? 0.46 : 0.50), 300),
                        offsetFrom: CGSize(width: -geo.size.width * 0.20, height: -geo.size.height * 0.18),
                        offsetTo: CGSize(width: -geo.size.width * 0.10, height: -geo.size.height * 0.08),
                        duration: isWindowFullscreen ? 28 : 22,
                        blurRadius: isWindowFullscreen ? 34 : 42
                    )
                    if !isWindowFullscreen {
                        liquidBlob(
                            color: palette.blobB.opacity(0.48),
                            size: max(geo.size.width * 0.40, 260),
                            offsetFrom: CGSize(width: geo.size.width * 0.34, height: geo.size.height * 0.24),
                            offsetTo: CGSize(width: geo.size.width * 0.18, height: geo.size.height * 0.30),
                            duration: 26,
                            blurRadius: 40
                        )
                    }
                } else {
                    liquidBlob(
                        color: palette.blobA.opacity(0.28),
                        size: max(geo.size.width * 0.38, 220),
                        offsetFrom: CGSize(width: -geo.size.width * 0.16, height: -geo.size.height * 0.12),
                        offsetTo: CGSize(width: -geo.size.width * 0.10, height: -geo.size.height * 0.06),
                        duration: 18,
                        blurRadius: 24
                    )
                }

                LinearGradient(
                    colors: [.black.opacity(0.28), .clear, .black.opacity(0.46)],
                    startPoint: .top,
                    endPoint: .bottom
                )
            }
        }
    }

    private var automaticallySimplified: Bool {
        #if os(iOS)
        UIDevice.current.userInterfaceIdiom == .phone
        #else
        false
        #endif
    }

    @ViewBuilder
    private func liquidBlob(
        color: Color,
        size: CGFloat,
        offsetFrom: CGSize,
        offsetTo: CGSize,
        duration: Double,
        blurRadius: CGFloat
    ) -> some View {
        Circle()
            .fill(color)
            .frame(width: size, height: size)
            .blur(radius: blurRadius)
            .offset(animate ? offsetTo : offsetFrom)
            .animation(.easeInOut(duration: duration).repeatForever(autoreverses: true), value: animate)
    }
}

struct LiquidPalette {
    let base: Color
    let depth: Color
    let blobA: Color
    let blobB: Color
    let blobC: Color

    static let fallback = LiquidPalette(
        base: Color(platformColor: .sonicRGBA(0.20, 0.16, 0.12, 1)),
        depth: Color(platformColor: .sonicRGBA(0.07, 0.06, 0.05, 1)),
        blobA: Color(platformColor: .sonicRGBA(0.70, 0.41, 0.18, 1)),
        blobB: Color(platformColor: .sonicRGBA(0.40, 0.25, 0.14, 1)),
        blobC: Color(platformColor: .sonicRGBA(0.53, 0.33, 0.16, 1))
    )

    static func from(averageColor: PlatformColor) -> LiquidPalette {
        let srgb = averageColor.sonicSRGB
        let base = srgb.adjustedSaturation(1.18).adjustedBrightness(0.40)
        let depth = srgb.mixed(with: .black, amount: 0.78).adjustedSaturation(1.08)
        let blobA = srgb.adjustedSaturation(1.32).adjustedBrightness(0.86)
        let blobB = srgb.adjustedSaturation(1.12).adjustedBrightness(0.68)
        let blobC = srgb.mixed(with: .sonicWhite(1, alpha: 1), amount: 0.16)

        return LiquidPalette(
            base: Color(platformColor: base),
            depth: Color(platformColor: depth),
            blobA: Color(platformColor: blobA),
            blobB: Color(platformColor: blobB),
            blobC: Color(platformColor: blobC)
        )
    }
}

actor ArtworkPaletteStore {
    static let shared = ArtworkPaletteStore()

    private var cache: [String: LiquidPalette] = [:]
    private var inFlight: [String: Task<LiquidPalette?, Never>] = [:]
    private let session: URLSession = {
        let config = URLSessionConfiguration.default
        config.requestCachePolicy = .returnCacheDataElseLoad
        config.urlCache = URLCache.shared
        config.timeoutIntervalForRequest = 5
        config.timeoutIntervalForResource = 8
        return URLSession(configuration: config)
    }()

    func palette(for artworkURL: String) async -> LiquidPalette? {
        if let cached = cache[artworkURL] {
            return cached
        }

        if let running = inFlight[artworkURL] {
            return await running.value
        }

        let task = Task<LiquidPalette?, Never> { [session] in
            guard let url = URL(string: artworkURL) else { return nil }
            do {
                let (data, _) = try await session.data(from: url)
                guard let image = PlatformImage(data: data), let average = image.averageSRGBColor() else {
                    return nil
                }
                return LiquidPalette.from(averageColor: average)
            } catch {
                return nil
            }
        }
        inFlight[artworkURL] = task

        let resolved = await task.value
        inFlight.removeValue(forKey: artworkURL)
        if let resolved {
            cache[artworkURL] = resolved
        }
        return resolved
    }
}

enum ArtworkPaletteExtractor {
    static func palette(for artworkURL: String) async -> LiquidPalette? {
        await ArtworkPaletteStore.shared.palette(for: artworkURL)
    }
}

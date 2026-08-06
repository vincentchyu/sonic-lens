import SwiftUI

#if os(iOS)
private enum PhoneNowPlayingTab: String, CaseIterable {
    case lyrics = "歌词"
    case insights = "音眸"
}

private enum PhoneImmersiveMode {
    case artworkFocused
    case lyricsFocused
}

struct PhoneNowPlayingView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(FavoriteActionStore.self) private var favoriteActionStore
    @Environment(PlaybackStore.self) private var playbackStore
    @Environment(\.scenePhase) private var scenePhase
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled
    @StateObject private var viewModel = PlayerViewModel()
    @State private var palette: LiquidPalette = .fallback
    @State private var lastArtworkURL: String?
    @State private var selectedTab: PhoneNowPlayingTab = .lyrics
    @State private var immersiveMode: PhoneImmersiveMode = .artworkFocused
    @State private var lyricsFollowMode = true
    @State private var lyricsScrollRequestToken = 0
    @State private var animate = false
    @State private var favoriteNoticeDismissTask: Task<Void, Never>?

    let nowPlaying: NowPlaying
    let onClose: () -> Void

    var body: some View {
        GeometryReader { geo in
            ZStack {
                liquidBackground
                content(geo: geo)
            }
            .overlay(alignment: .bottom) {
                progressDock(bottomInset: max(geo.safeAreaInsets.bottom, 8))
            }
        }
        .ignoresSafeArea()
        .task {
            animate = !performanceModeEnabled
            await refreshNowPlaying(forcePaletteRefresh: true)
        }
        .onChange(of: trackIdentity) { _, _ in
            Task { await refreshNowPlaying(forcePaletteRefresh: true) }
            withAnimation(.interactiveSpring(response: 0.34, dampingFraction: 0.86)) {
                immersiveMode = .artworkFocused
                lyricsFollowMode = true
            }
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

    private var favoriteStatusTagTone: NowPlayingArtworkStatusTagTone {
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
            return .neutral
        }
    }

    private var liquidBackground: some View {
        NowPlayingLiquidBackground(
            palette: palette,
            animate: .constant(animate && scenePhase == .active),
            isWindowFullscreen: false
        )
    }

    @ViewBuilder
    private func content(geo: GeometryProxy) -> some View {
        let current = currentNowPlaying
        let topInset: CGFloat = 25
        let progressDockHeight = max(geo.safeAreaInsets.bottom, 8) + 46
        let expandedLyricsHeight = max(geo.size.height - topInset - 220 - progressDockHeight, 360)

        VStack(spacing: 18) {
            PhoneNowPlayingTopBar(
                statusBannerText: viewModel.playbackState.bannerText,
                favoriteActionLoading: favoriteActionLoading,
                favoriteActionNotice: favoriteActionNotice
            )
                .padding(.top, topInset)

            if immersiveMode == .artworkFocused {
                ScrollView(.vertical, showsIndicators: false) {
                    VStack(spacing: 18) {
                        artworkSection(current: current)
                        metadataSection(current: current)
                        tabPicker
                        tabPanel(
                            lyricsHeight: 430,
                            lyricsInteractive: false,
                            lyricsBottomInset: 24
                        )
                        .environment(\.sonicPerformanceModeEnabled, performanceModeEnabled)
                    }
                    .padding(.horizontal, 20)
                    .padding(.bottom, 28 + progressDockHeight)
                }
            } else {
                VStack(spacing: 16) {
                    compactArtworkHeader(current: current)
                    tabPicker
                    tabPanel(
                        lyricsHeight: expandedLyricsHeight,
                        lyricsInteractive: true,
                        lyricsBottomInset: progressDockHeight + 18
                    )
                    .environment(\.sonicPerformanceModeEnabled, performanceModeEnabled)
                }
                .padding(.horizontal, 20)
                .padding(.bottom, 8)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
            }
        }
        .padding(.horizontal, 16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
    }

    @ViewBuilder
    private func artworkSection(current: NowPlaying) -> some View {
        NowPlayingArtwork(
            artworkURL: current.artwork,
            fallbackTitle: current.album ?? current.track,
            badgeText: current.sampleRateDisplayText,
            genreText: current.genre,
            statusTagText: favoriteStatusTagText,
            statusTagTone: favoriteStatusTagText == nil ? nil : favoriteStatusTagTone
        )
            .frame(maxWidth: 286)
            .contentShape(Rectangle())
            .highPriorityGesture(dismissGesture)
            .onTapGesture {
                withAnimation(.interactiveSpring(response: 0.34, dampingFraction: 0.86)) {
                    immersiveMode = .artworkFocused
                }
            }
    }

    @ViewBuilder
    private func compactArtworkHeader(current: NowPlaying) -> some View {
        Button {
            withAnimation(.interactiveSpring(response: 0.34, dampingFraction: 0.86)) {
                immersiveMode = .artworkFocused
            }
        } label: {
            HStack(spacing: 12) {
                compactArtworkThumbnail(
                    urlString: current.artwork,
                    fallbackTitle: current.album ?? current.track
                )

                VStack(alignment: .leading, spacing: 3) {
                    Text(current.track)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(.white)
                        .lineLimit(1)

                    Text("点击封面回到完整专辑区")
                        .font(.caption)
                        .foregroundStyle(.white.opacity(0.66))
                        .lineLimit(1)
                }

                Spacer(minLength: 8)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .background {
                if performanceModeEnabled {
                    RoundedRectangle(cornerRadius: 16, style: .continuous)
                        .fill(SonicTheme.card.opacity(0.92))
                } else {
                    RoundedRectangle(cornerRadius: 16, style: .continuous)
                        .fill(.ultraThinMaterial)
                }
            }
            .overlay(
                RoundedRectangle(cornerRadius: 16, style: .continuous)
                    .stroke(.white.opacity(0.14), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
    }

    @ViewBuilder
    private func compactArtworkThumbnail(urlString: String?, fallbackTitle: String?) -> some View {
        ArtworkSquareView(
            artworkURL: urlString,
            fallbackTitle: fallbackTitle,
            size: 56,
            cornerRadius: 12,
            style: .vivid
        )
        .frame(width: 56, height: 56)
        .overlay(
            RoundedRectangle(cornerRadius: 12, style: .continuous)
                .stroke(.white.opacity(0.16), lineWidth: 1)
        )
    }

    @ViewBuilder
    private func metadataSection(current: NowPlaying) -> some View {
        VStack(alignment: .center, spacing: 8) {
            ZStack(alignment: .trailing) {
                Text(current.track)
                    .font(.system(size: 28, weight: .bold, design: .rounded))
                    .foregroundStyle(.white)
                    .multilineTextAlignment(.center)
                    .lineLimit(1) // 强制只显示一行，不换行
                    .minimumScaleFactor(0.6) //文字放不下时，最小可以缩到原字号的 60%
                    .allowsTightening(true) // 系统会先稍微压紧字符间距，还是不够才继续缩小
                    .frame(maxWidth: .infinity, alignment: .center)
                    .padding(.horizontal, 56)

                NowPlayingFavoriteButton(
                    status: favoriteStatus,
                    isLoading: favoriteActionLoading,
                    action: toggleFavorite
                )
            }

            Text([current.artist, current.album].compactMap { $0 }.joined(separator: " · "))
                .font(.subheadline.weight(.medium))
                .foregroundStyle(Color.white.opacity(0.74))
                .multilineTextAlignment(.center)

            DiscTrackBadgeRow(discNumber: current.discNumber, trackNumber: current.trackNumber)

            if let insightTeaser = viewModel.insights.primaryInsight?.teaserText, !insightTeaser.isEmpty {
                Text(insightTeaser)
                    .font(.footnote)
                    .foregroundStyle(Color.white.opacity(0.7))
                    .lineLimit(2)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity)
    }

    private var tabPicker: some View {
        Picker("", selection: $selectedTab) {
            ForEach(PhoneNowPlayingTab.allCases, id: \.self) { tab in
                Text(tab.rawValue).tag(tab)
            }
        }
        .pickerStyle(.segmented)
        .padding(.horizontal, 2)
    }

    @ViewBuilder
    private func tabPanel(
        lyricsHeight: CGFloat,
        lyricsInteractive: Bool,
        lyricsBottomInset: CGFloat
    ) -> some View {
        if selectedTab == .lyrics {
            if lyricsInteractive {
                ZStack(alignment: .topTrailing) {
                    PhoneImmersiveLyricsPanel(
                        lines: viewModel.lyricLines,
                        currentLineID: viewModel.currentLineID,
                        highlightedIndex: viewModel.currentLineIndex,
                        followMode: $lyricsFollowMode,
                        scrollRequestToken: lyricsScrollRequestToken,
                        scrollEnabled: true,
                        viewportHeight: lyricsHeight,
                        bottomContentInset: lyricsBottomInset,
                        onUserScroll: {
                            guard lyricsFollowMode else { return }
                            lyricsFollowMode = false
                        }
                    )

                    if !lyricsFollowMode {
                        Button {
                            withAnimation(.easeInOut(duration: 0.2)) {
                                lyricsFollowMode = true
                                lyricsScrollRequestToken += 1
                            }
                        } label: {
                            Label("回到当前行", systemImage: "dot.radiowaves.left.and.right")
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(.white)
                                .padding(.horizontal, 10)
                                .padding(.vertical, 8)
                                .background {
                                    if performanceModeEnabled {
                                        Capsule()
                                            .fill(SonicTheme.card.opacity(0.92))
                                    } else {
                                        Capsule()
                                            .fill(.ultraThinMaterial)
                                    }
                                }
                                .overlay(
                                    Capsule().stroke(.white.opacity(0.18), lineWidth: 1)
                                )
                        }
                        .buttonStyle(.plain)
                        .padding(.top, 12)
                        .padding(.trailing, 12)
                    }
                }
            } else {
                PhoneLyricsPreviewPanel(
                    lines: viewModel.lyricLines,
                    currentLineID: viewModel.currentLineID,
                    highlightedIndex: viewModel.currentLineIndex,
                    onExpand: {
                        withAnimation(.interactiveSpring(response: 0.34, dampingFraction: 0.86)) {
                            immersiveMode = .lyricsFocused
                        }
                    }
                )
            }
        } else {
            PhoneNowPlayingInsightPanel(items: viewModel.insights)
                .frame(minHeight: max(lyricsHeight, 320))
        }
    }

    private func progressDock(bottomInset: CGFloat) -> some View {
        PlayerProgressBar(
            currentTime: viewModel.currentTime,
            duration: currentNowPlaying.duration ?? 0
        )
        .padding(.horizontal, 20)
        .padding(.top, 8)
        .padding(.bottom, bottomInset)
        .background(
            LinearGradient(
                colors: [.clear, .black.opacity(0.28)],
                startPoint: .top,
                endPoint: .bottom
            )
            .allowsHitTesting(false)
        )
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

    private var dismissGesture: some Gesture {
        DragGesture(minimumDistance: 12, coordinateSpace: .global)
            .onEnded { value in
                let translation = value.translation.height
                let predicted = value.predictedEndTranslation.height
                guard translation > 120 || predicted > 180 else { return }
                onClose()
            }
    }
}

private struct PhoneNowPlayingTopBar: View {
    let statusBannerText: String?
    let favoriteActionLoading: Bool
    let favoriteActionNotice: FavoriteActionNotice?

    var body: some View {
        VStack(spacing: 8) {
            HStack {
                Spacer(minLength: 0)

                HStack(spacing: 10) {
                    Text("正在播放")
                        .font(.headline.weight(.semibold))
                        .foregroundStyle(.white)

                    if let statusBannerText {
                        PlaybackStatusBanner(text: statusBannerText)
                    }
                }

                Spacer(minLength: 0)
            }

            if let favoriteActionNotice {
                FavoriteActionNoticeBanner(notice: favoriteActionNotice)
            } else if favoriteActionLoading {
                PlaybackStatusBanner(text: "收藏处理中")
            }
        }
        .frame(maxWidth: .infinity)
    }
}

private struct PhoneNowPlayingInsightPanel: View {
    let items: [Insight]

    var body: some View {
        ScrollView(.vertical, showsIndicators: false) {
            InsightPrimaryContentView(
                insight: items.primaryInsight,
                style: .phoneImmersive,
                emptyTitle: "暂无音眸",
                emptySubtitle: "当前曲目还没有生成洞察内容。"
            )
            .padding(.horizontal, 4)
            .padding(.vertical, 8)
        }
        .mask(
            LinearGradient(
                colors: [.clear, .black.opacity(0.94), .black, .black.opacity(0.94), .clear],
                startPoint: .top,
                endPoint: .bottom
            )
        )
        .frame(maxWidth: .infinity, minHeight: 320, alignment: .top)
    }
}

private struct PhoneImmersiveLyricsPanel: View {
    let lines: [LyricLine]
    let currentLineID: UUID?
    let highlightedIndex: Int?
    @Binding var followMode: Bool
    var scrollRequestToken: Int = 0
    var scrollEnabled: Bool = true
    let viewportHeight: CGFloat
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
                            .frame(maxWidth: .infinity, minHeight: max(viewportHeight - 40, 180))
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
            .frame(maxWidth: .infinity)
            .frame(height: viewportHeight, alignment: .top)
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

    private func lyricStyle(for index: Int, highlightedIndex: Int?) -> PhoneLyricTypography {
        guard let highlightedIndex else {
            return PhoneLyricTypography(size: 20, weight: .medium, opacity: 0.58, scale: 0.98)
        }

        let distance = abs(index - highlightedIndex)
        if distance == 0 {
            return PhoneLyricTypography(size: 30, weight: .bold, opacity: 1.0, scale: 1.0)
        }
        if distance > visibleRadius {
            return PhoneLyricTypography(size: 18, weight: .regular, opacity: 0.22, scale: 0.92)
        }

        let falloff = Double(distance) / Double(visibleRadius)
        let size = CGFloat(30.0 - falloff * 10.0)
        let opacity = 0.86 - falloff * 0.46
        let scale = 1.0 - falloff * 0.06
        let weight: Font.Weight = distance <= 2 ? .semibold : .medium
        return PhoneLyricTypography(size: size, weight: weight, opacity: opacity, scale: scale)
    }
}

private struct PhoneLyricTypography {
    let size: CGFloat
    let weight: Font.Weight
    let opacity: Double
    let scale: CGFloat
}

private struct PhoneLyricsPreviewPanel: View {
    let lines: [LyricLine]
    let currentLineID: UUID?
    let highlightedIndex: Int?
    let onExpand: () -> Void

    private let previewLineLimit = 7

    private var previewLines: [LyricLine] {
        let normalized = lines.filter { !$0.text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
        guard !normalized.isEmpty else { return [] }

        if let currentLineID,
           let highlightedIndex {
            let normalizedIndex = normalized.firstIndex(where: { $0.id == currentLineID }) ?? highlightedIndex
            let visibleCount = min(previewLineLimit, normalized.count)
            let leadingContext = visibleCount / 2
            let start = min(max(normalizedIndex - leadingContext, 0), max(normalized.count - visibleCount, 0))
            let end = min(start + visibleCount, normalized.count)
            return Array(normalized[start..<end])
        }

        return Array(normalized.prefix(previewLineLimit))
    }

    var body: some View {
        VStack(spacing: 4) {
            HStack {
                Text("歌词预览")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.white.opacity(0.58))
                    .tracking(1.2)

                Spacer()
                
            }

            if previewLines.isEmpty {
                Text("暂无歌词")
                    .font(.system(size: 17, weight: .semibold))
                    .foregroundStyle(.white.opacity(0.72))
                    .frame(maxWidth: .infinity, minHeight: 180, alignment: .top)
            } else {
                VStack(spacing: 10) {
                    ForEach(Array(previewLines.enumerated()), id: \.element.id) { item in
                        let style = previewTypography(for: item.element)
                        Text(item.element.text)
                            .font(.system(size: style.size, weight: style.weight))
                            .tracking(style.tracking)
                            .multilineTextAlignment(.center)
                            .foregroundStyle(.white)
                            .opacity(style.opacity)
                            .frame(maxWidth: .infinity, alignment: .center)
                    }
                }
                .frame(maxWidth: .infinity, minHeight: 180, alignment: .top)
            }
        }
        .padding(.horizontal, 10)
        .padding(.top, 2)
        .contentShape(Rectangle())
        .onTapGesture(perform: onExpand)
    }

    private func previewTypography(for line: LyricLine) -> (size: CGFloat, weight: Font.Weight, opacity: Double, tracking: CGFloat) {
        if line.isSectionLabel {
            return (11, .semibold, 0.42, 2.4)
        }

        if line.id == currentLineID {
            return (20, .semibold, 1.0, 0)
        }

        return (16, .medium, 0.54, 0)
    }
}
#endif

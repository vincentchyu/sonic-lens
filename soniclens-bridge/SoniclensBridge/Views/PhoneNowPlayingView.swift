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

    let nowPlaying: NowPlaying
    let onClose: () -> Void

    var body: some View {
        GeometryReader { geo in
            ZStack {
                liquidBackground
                content(geo: geo)
            }
        }
        .ignoresSafeArea()
        .safeAreaInset(edge: .bottom, spacing: 0) {
            progressDock(bottomInset: 8)
        }
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
        .onChange(of: store.nowPlaying?.artwork) { _, artwork in
            Task { await updatePalette(for: artwork) }
        }
        .onChange(of: store.nowPlaying?.position) { _, position in
            viewModel.syncProgress(position: position, positionMs: store.nowPlaying?.positionMs)
        }
        .onChange(of: store.nowPlaying?.positionMs) { _, positionMs in
            viewModel.syncProgress(position: store.nowPlaying?.position, positionMs: positionMs)
        }
        .onDisappear {
            viewModel.stopProgress()
        }
    }

    private var currentNowPlaying: NowPlaying {
        store.nowPlaying ?? nowPlaying
    }

    private var trackIdentity: String {
        let active = currentNowPlaying
        return "\(active.artist)::\(active.album ?? "")::\(active.track)"
    }

    private var favoriteStatus: NowPlayingFavoriteStatus {
        .init(projection: currentNowPlaying.favoriteProjection)
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
        let expandedLyricsHeight = max(geo.size.height - topInset - 220, 420)

        VStack(spacing: 18) {
            PhoneNowPlayingTopBar()
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
                            lyricsBottomInset: 42
                        )
                        .environment(\.sonicPerformanceModeEnabled, performanceModeEnabled)
                    }
                    .padding(.horizontal, 20)
                    .padding(.bottom, 28)
                }
            } else {
                VStack(spacing: 16) {
                    compactArtworkHeader(current: current)
                    tabPicker
                    tabPanel(
                        lyricsHeight: expandedLyricsHeight,
                        lyricsInteractive: true,
                        lyricsBottomInset: 88
                    )
                    .environment(\.sonicPerformanceModeEnabled, performanceModeEnabled)
                }
                .padding(.horizontal, 20)
                .padding(.bottom, 12)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
            }
        }
        .padding(.horizontal, 16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
    }

    @ViewBuilder
    private func artworkSection(current: NowPlaying) -> some View {
        NowPlayingArtwork(artworkURL: current.artwork, fallbackTitle: current.album ?? current.track)
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
                
                NowPlayingFavoriteButton(status: favoriteStatus, action: toggleFavorite)
            }

            Text([current.artist, current.album].compactMap { $0 }.joined(separator: " · "))
                .font(.subheadline.weight(.medium))
                .foregroundStyle(Color.white.opacity(0.74))
                .multilineTextAlignment(.center)

            DiscTrackBadgeRow(discNumber: current.discNumber, trackNumber: current.trackNumber)

            if let badgeTitle = favoriteStatus.badgeTitle {
                NowPlayingFavoriteStatusBadge(status: favoriteStatus, title: badgeTitle)
            }

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

        await viewModel.load(
            using: server,
            artist: active.artist,
            album: active.album,
            track: active.track,
            trackNumber: active.trackNumber,
            discNumber: active.discNumber
        )
        viewModel.startProgress(position: active.position, positionMs: active.positionMs)

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
    var body: some View {
        ZStack {
            Text("正在播放")
                .font(.headline.weight(.semibold))
                .foregroundStyle(.white)
        }
        .frame(maxWidth: .infinity)
    }
}

private struct PhoneNowPlayingInsightPanel: View {
    let items: [Insight]

    var body: some View {
        GlassPanel(cornerRadius: 22, padding: 0) {
            ScrollView {
                InsightPrimaryContentView(
                    insight: items.primaryInsight,
                    style: .phoneCompact,
                    emptyTitle: "暂无音眸",
                    emptySubtitle: "当前曲目还没有生成洞察内容。"
                )
                .padding(16)
            }
            .frame(minHeight: 320)
        }
    }
}

private struct PhoneImmersiveLyricsPanel: View {
    let lines: [LyricLine]
    let currentLineID: UUID?
    @Binding var followMode: Bool
    var scrollRequestToken: Int = 0
    var scrollEnabled: Bool = true
    let viewportHeight: CGFloat
    var bottomContentInset: CGFloat = 24
    var onUserScroll: (() -> Void)? = nil

    private let visibleRadius = 5
    @State private var didNotifyUserScroll = false

    var body: some View {
        let highlightedIndex = currentLineID.flatMap { id in
            lines.firstIndex(where: { $0.id == id })
        }

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
                withAnimation(.easeInOut(duration: 0.2)) {
                    proxy.scrollTo(lineID, anchor: .center)
                }
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
    let onExpand: () -> Void

    private var previewLines: [LyricLine] {
        let normalized = lines.filter { !$0.text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
        guard !normalized.isEmpty else { return [] }

        if let currentLineID,
           let highlightedIndex = normalized.firstIndex(where: { $0.id == currentLineID }) {
            let visibleCount = min(5, normalized.count)
            let start = min(max(highlightedIndex - 2, 0), max(normalized.count - visibleCount, 0))
            let end = min(start + visibleCount, normalized.count)
            return Array(normalized[start..<end])
        }

        return Array(normalized.prefix(5))
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

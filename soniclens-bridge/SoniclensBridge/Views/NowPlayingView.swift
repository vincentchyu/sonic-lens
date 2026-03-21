import SwiftUI
import NukeUI

enum MacNowPlayingTab: String, CaseIterable {
    case lyrics = "歌词"
    case insights = "音眸"
}

struct NowPlayingView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled
    @Environment(\.scenePhase) private var scenePhase
    @StateObject private var viewModel = PlayerViewModel()
    @State private var animate = false
    @State private var palette: LiquidPalette = .fallback
    @State private var lastArtworkURL: String?
    @State private var lyricsFollowMode = true
    @State private var isWindowFullscreen = false
    @State private var selectedTab: MacNowPlayingTab = .lyrics

    let nowPlaying: NowPlaying
    let onClose: () -> Void

    var body: some View {
        let displayNowPlaying = currentNowPlaying

        ZStack {
            NowPlayingLiquidBackground(
                palette: palette,
                animate: .constant(animate && scenePhase == .active),
                isWindowFullscreen: isWindowFullscreen
            )

            VStack(spacing: 28) {
                NowPlayingTopBar(
                    favoriteStatus: favoriteStatus,
                    lyricsFollowMode: $lyricsFollowMode,
                    selectedTab: $selectedTab,
                    onFavorite: {
                        guard favoriteStatus != .full else { return }
                        Task {
                            await store.setFavorite(
                                artist: displayNowPlaying.artist,
                                album: displayNowPlaying.album,
                                track: displayNowPlaying.track,
                                favorite: true
                            )
                        }
                    },
                    onClose: onClose
                )

                HStack(alignment: .center, spacing: 48) {
                    NowPlayingLeftPanel(
                        nowPlaying: displayNowPlaying,
                        insightSummary: viewModel.insights.primaryInsight?.teaserText
                    )
                    .frame(maxWidth: 380)

                    Group {
                        if selectedTab == .lyrics {
                            NowPlayingLyricsPanel(
                                lines: viewModel.lyricLines,
                                currentLineID: viewModel.currentLineID,
                                isSimplified: performanceModeEnabled,
                                followMode: $lyricsFollowMode
                            )
                        } else {
                            MacNowPlayingInsightPanel(items: viewModel.insights)
                        }
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
            }
            .padding(.horizontal, 34)
            .padding(.top, 52)
            .padding(.bottom, 46)

            VStack {
                Spacer()
                NowPlayingBottomProgressBar(
                    currentTime: viewModel.currentTime,
                    duration: duration,
                    progress: progress
                )
                .padding(.horizontal, 22)
                .padding(.bottom, 10)
            }
        }
        .task {
            await refreshNowPlaying(forcePaletteRefresh: true)
        }
        .onChange(of: trackIdentity) { _, _ in
            Task { await refreshNowPlaying(forcePaletteRefresh: true) }
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
        .onAppear {
            animate = true
            refreshFullscreenState()
        }
        .modifier(FullscreenStateObserver(onChange: refreshFullscreenState))
    }

    private var currentNowPlaying: NowPlaying {
        store.nowPlaying ?? nowPlaying
    }

    private var trackIdentity: String {
        let active = currentNowPlaying
        return "\(active.artist)::\(active.album ?? "")::\(active.track)"
    }

    private var duration: TimeInterval {
        TimeInterval(currentNowPlaying.duration ?? 0)
    }

    private var progress: Double {
        guard duration > 0 else { return 0 }
        return min(viewModel.currentTime / duration, 1.0)
    }

    private var favoriteStatus: NowPlayingFavoriteStatus {
        let apple = currentNowPlaying.isAppleMusicFav ?? false
        let lastfm = currentNowPlaying.isLastFmFav ?? false
        if apple && lastfm {
            return .full
        }
        if apple || lastfm {
            return .partial
        }
        return .none
    }

    private func refreshNowPlaying(forcePaletteRefresh: Bool) async {
        guard let server = store.currentServer else { return }
        let active = currentNowPlaying

        await viewModel.load(
            using: server,
            artist: active.artist,
            album: active.album,
            track: active.track
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

    private func refreshFullscreenState() {
        #if os(macOS)
        let window = NSApp.keyWindow ?? NSApp.mainWindow
        isWindowFullscreen = window?.styleMask.contains(.fullScreen) ?? false
        #else
        isWindowFullscreen = false
        #endif
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

struct NowPlayingTopBar: View {
    let favoriteStatus: NowPlayingFavoriteStatus
    @Binding var lyricsFollowMode: Bool
    @Binding var selectedTab: MacNowPlayingTab
    let onFavorite: () -> Void
    let onClose: () -> Void

    var body: some View {
        HStack {
            Text("正在播放")
                .font(.system(size: 28, weight: .bold, design: .rounded))
                .foregroundStyle(.white)

            Spacer()
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

                NowPlayingFavoriteButton(status: favoriteStatus, action: onFavorite)
                Button(action: onClose) {
                    Image(systemName: "xmark")
                        .font(.system(size: 14, weight: .bold))
                        .frame(width: 32, height: 32)
                        .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 10))
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.trailing, 6)
    }
}

struct MacNowPlayingInsightPanel: View {
    let items: [Insight]

    var body: some View {
        ScrollView(.vertical, showsIndicators: false) {
            InsightPrimaryContentView(
                insight: items.primaryInsight,
                style: .immersive,
                emptyTitle: "暂无音眸",
                emptySubtitle: "当前曲目还没有生成洞察内容。"
            )
            .padding(.horizontal, 14)
            .padding(.vertical, 18)
        }
        .mask(
            LinearGradient(
                colors: [.clear, .black.opacity(0.94), .black, .black.opacity(0.94), .clear],
                startPoint: .top,
                endPoint: .bottom
            )
        )
        .frame(maxWidth: .infinity)
        .frame(height: 520)
    }
}

struct NowPlayingLeftPanel: View {
    let nowPlaying: NowPlaying
    let insightSummary: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            NowPlayingArtwork(artworkURL: nowPlaying.artwork)

            VStack(alignment: .leading, spacing: 8) {
                Text(nowPlaying.track)
                    .font(.system(size: 30, weight: .bold))
                    .foregroundStyle(.white)
                    .lineLimit(2)

                Text([nowPlaying.artist, nowPlaying.album].compactMap { $0 }.joined(separator: " · "))
                    .font(.system(size: 16, weight: .medium))
                    .foregroundStyle(Color.white.opacity(0.82))
                    .lineLimit(2)

                DiscTrackBadgeRow(discNumber: nowPlaying.discNumber, trackNumber: nowPlaying.trackNumber)
            }

            VStack(alignment: .leading, spacing: 8) {
                Text("沉浸模式")
                    .font(.headline)
                    .foregroundStyle(.white.opacity(0.9))
                Text(insightSummary ?? "专注当前播放，把封面、歌词和音眸洞察放到同一块大画布里。")
                    .font(.subheadline)
                    .foregroundStyle(.white.opacity(0.56))
                    .fixedSize(horizontal: false, vertical: true)
            }
            .padding(.top, 6)
        }
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
    let action: () -> Void
    @State private var isHovered = false

    var body: some View {
        Button(action: action) {
            Image(systemName: iconName)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(status == .none ? Color.white.opacity(0.95) : Color.yellow.opacity(0.95))
                .frame(width: 32, height: 32)
                .background(
                    RoundedRectangle(cornerRadius: 10)
                        .fill(backgroundOpacity)
                )
        }
        .buttonStyle(.plain)
        .buttonStyle(PressableButtonStyle())
        .disabled(status == .full)
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
        .help(helpText)
    }

    private var iconName: String {
        switch status {
        case .full: return "star.fill"
        case .partial: return "star.leadinghalf.filled"
        case .none: return "star"
        }
    }

    private var backgroundOpacity: Color {
        switch status {
        case .full:
            return Color.yellow.opacity(isHovered ? 0.34 : 0.26)
        case .partial:
            return Color.yellow.opacity(isHovered ? 0.24 : 0.16)
        case .none:
            return Color.white.opacity(isHovered ? 0.20 : 0.12)
        }
    }

    private var helpText: String {
        switch status {
        case .full: return "已同步到 Apple Music + Last.fm"
        case .partial: return "已在单平台收藏，点击补全双平台收藏"
        case .none: return "收藏到 Apple Music / Last.fm"
        }
    }
}

enum NowPlayingFavoriteStatus {
    case none
    case partial
    case full
}

struct NowPlayingArtwork: View {
    let artworkURL: String?

    var body: some View {
        Group {
            if let artworkURL, let url = URL(string: artworkURL) {
                LazyImage(url: url) { state in
                    if let image = state.image {
                        image
                            .resizable()
                            .scaledToFill()
                    } else {
                        placeholder
                    }
                }
            } else {
                placeholder
            }
        }
        .frame(width: 340, height: 340)
        .clipShape(RoundedRectangle(cornerRadius: 18))
        .overlay(
            RoundedRectangle(cornerRadius: 18)
                .stroke(.white.opacity(0.16), lineWidth: 1)
        )
        .shadow(color: .black.opacity(0.28), radius: 32, x: 0, y: 24)
    }

    private var placeholder: some View {
        RoundedRectangle(cornerRadius: 18)
            .fill(
                LinearGradient(
                    colors: [Color.accentColor.opacity(0.58), Color.accentColor.opacity(0.22)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
            .overlay(
                Image(systemName: "music.note")
                    .font(.system(size: 46, weight: .bold))
                    .foregroundStyle(.white.opacity(0.6))
            )
    }
}

struct NowPlayingLyricsPanel: View {
    let lines: [LyricLine]
    let currentLineID: UUID?
    let isSimplified: Bool
    @Binding var followMode: Bool
    var scrollRequestToken: Int = 0
    var scrollEnabled: Bool = true
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
            .frame(height: isSimplified ? 430 : 520)
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
        GeometryReader { geo in
            ZStack {
                LinearGradient(
                    colors: [palette.base, palette.depth],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )

                if !performanceModeEnabled {
                    liquidBlob(
                        color: palette.blobA.opacity(isWindowFullscreen ? 0.45 : 0.55),
                        size: max(geo.size.width * (isWindowFullscreen ? 0.50 : 0.54), 320),
                        offsetFrom: CGSize(width: -geo.size.width * 0.20, height: -geo.size.height * 0.18),
                        offsetTo: CGSize(width: -geo.size.width * 0.10, height: -geo.size.height * 0.08),
                        duration: isWindowFullscreen ? 28 : 22,
                        blurRadius: isWindowFullscreen ? 42 : 54
                    )
                    if !isWindowFullscreen {
                        liquidBlob(
                            color: palette.blobB.opacity(0.48),
                            size: max(geo.size.width * 0.46, 300),
                            offsetFrom: CGSize(width: geo.size.width * 0.34, height: geo.size.height * 0.24),
                            offsetTo: CGSize(width: geo.size.width * 0.18, height: geo.size.height * 0.30),
                            duration: 26,
                            blurRadius: 54
                        )
                    }
                }

                LinearGradient(
                    colors: [.black.opacity(0.28), .clear, .black.opacity(0.46)],
                    startPoint: .top,
                    endPoint: .bottom
                )
            }
        }
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

enum ArtworkPaletteExtractor {
    private static var cache: [String: LiquidPalette] = [:]

    static func palette(for artworkURL: String) async -> LiquidPalette? {
        if let cached = cache[artworkURL] {
            return cached
        }

        guard let url = URL(string: artworkURL) else { return nil }
        do {
            let (data, _) = try await URLSession.shared.data(from: url)
            guard let image = PlatformImage(data: data), let average = image.averageSRGBColor() else {
                return nil
            }
            let extracted = LiquidPalette.from(averageColor: average)
            cache[artworkURL] = extracted
            return extracted
        } catch {
            return nil
        }
    }
}

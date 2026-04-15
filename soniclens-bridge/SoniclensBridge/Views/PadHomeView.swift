import SwiftUI

#if os(iOS)
struct PadHomeView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(PlaybackStore.self) private var playbackStore
    @StateObject private var viewModel = HomeViewModel()
    @State private var selectedRecentTrack: Track?
    @State private var selectedAlbumID: Int64?

    let onOpenNowPlaying: () -> Void

    var body: some View {
        ZStack {
            AmbientBackgroundView(
                gradient: LinearGradient(
                    colors: [SonicTheme.background, SonicTheme.background.opacity(0.98), Color.accentColor.opacity(0.10)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                ),
                orbs: [
                    AmbientOrb(color: SonicTheme.primary.opacity(0.26), size: 520, blur: 140, opacity: 0.7, offsetFrom: CGSize(width: -220, height: -280), offsetTo: CGSize(width: -140, height: -160), duration: 20),
                    AmbientOrb(color: SonicTheme.secondaryAccent.opacity(0.18), size: 420, blur: 150, opacity: 0.7, offsetFrom: CGSize(width: 280, height: 120), offsetTo: CGSize(width: 180, height: 220), duration: 26)
                ],
                renderingStyle: .staticHome
            )

            ScrollView {
                LazyVStack(alignment: .leading, spacing: 24) {
                    if let message = viewModel.errorMessage {
                        ErrorBanner(message: message)
                    }

                    PadHomeHero(nowPlaying: playbackStore.nowPlaying, onOpenNowPlaying: onOpenNowPlaying)

                    let hotPresentation = viewModel.hotModulePresentation
                    let trendSnapshot = viewModel.trendSnapshot
                    let selectedAccent = hotPresentation.primaryAccentKey

                    VStack(alignment: .leading, spacing: 14) {
                        // HotModuleSectionHeader(
                        //     title: "听觉版图",
                        //     subtitle: "在 iPad 上先铺开热门专辑、艺术家和曲目，再把口味与播放渠道压进同一张聆听画像里。",
                        //     accentKey: selectedAccent
                        //  )

                        DashboardTrendSection(
                            points: trendSnapshot.points,
                            hourlyData: trendSnapshot.hourlyData,
                            title: "聆听趋势",
                            subtitle: "按日期展开 24 小时播放分布，保持时间轴比例，可横向浏览完整 90 天。",
                            heatmapLayout: .fixedWidthScrollable(cellWidth: 7),
                            axisLabelStyle: .dayStride(step: 3, rotationDegrees: 42)
                        )
                        .equatable()
                        .frame(maxWidth: .infinity)

                        ListeningProfileCard(
                            summaryText: hotPresentation.combinedSummaryText,
                            footnoteText: hotPresentation.profileFootnoteText,
                            genres: Array(hotPresentation.genres.prefix(3)),
                            sources: Array(hotPresentation.sources.prefix(3)),
                            accentKey: selectedAccent,
                            layoutStyle: .split
                        )

                        AdaptiveWidthContainer(minWidth: 900) {
                            Grid(horizontalSpacing: 16, verticalSpacing: 16) {
                                GridRow {
                                    VStack(spacing: 16) {
                                        AlbumShelfCard(
                                            items: Array(hotPresentation.albums.prefix(6)),
                                            artworkBaseURL: store.currentServer?.artworkBaseURL,
                                            style: .rail,
                                            collectionCount: hotPresentation.totalAlbumsCount,
                                            accentKey: selectedAccent,
                                            onAlbumTap: { selectedAlbumID = $0 }
                                        )
                                        .frame(minWidth: 300, maxWidth: .infinity, maxHeight: .infinity)

                                        ArtistLadderCard(
                                            items: Array(hotPresentation.artists.prefix(6)),
                                            artworkBaseURL: store.currentServer?.artworkBaseURL,
                                            collectionCount: hotPresentation.totalArtistsCount,
                                            accentKey: selectedAccent
                                        )
                                        .frame(minWidth: 300, maxWidth: .infinity, maxHeight: .infinity)
                                    }
                                    .frame(minWidth: 300, maxWidth: .infinity, maxHeight: .infinity)

                                        TrackShelfCard(
                                            items: Array(hotPresentation.tracks.prefix(6)),
                                            artworkBaseURL: store.currentServer?.artworkBaseURL,
                                            style: .rail,
                                            totalTracksCount: hotPresentation.totalTracksCount,
                                            accentKey: selectedAccent,
                                            onTrackTap: { selectedRecentTrack = $0.bridgeTrack }
                                        )
                                    .frame(minWidth: 320, maxWidth: .infinity, maxHeight: .infinity)
                                }
                            }
                        } narrowContent: {
                            VStack(spacing: 16) {
                                AlbumShelfCard(
                                    items: Array(hotPresentation.albums.prefix(6)),
                                    artworkBaseURL: store.currentServer?.artworkBaseURL,
                                    style: .rail,
                                    collectionCount: hotPresentation.totalAlbumsCount,
                                    accentKey: selectedAccent,
                                    onAlbumTap: { selectedAlbumID = $0 }
                                )

                                ArtistLadderCard(
                                    items: Array(hotPresentation.artists.prefix(6)),
                                    artworkBaseURL: store.currentServer?.artworkBaseURL,
                                    collectionCount: hotPresentation.totalArtistsCount,
                                    accentKey: selectedAccent
                                )

                                TrackShelfCard(
                                    items: Array(hotPresentation.tracks.prefix(6)),
                                    artworkBaseURL: store.currentServer?.artworkBaseURL,
                                    style: .rail,
                                    totalTracksCount: hotPresentation.totalTracksCount,
                                    accentKey: selectedAccent,
                                    onTrackTap: { selectedRecentTrack = $0.bridgeTrack }
                                )
                            }
                        }
                    }

                    RecentPlaysSection(
                        items: viewModel.recentPlays,
                        artworkURLs: viewModel.recentPlayArtworkURLs,
                        totalPlaysCount: hotPresentation.totalPlaysCount,
                        accentKey: selectedAccent,
                        onTrackTap: { selectedRecentTrack = $0 }
                    )
                }
                .padding(.horizontal, 24)
                .padding(.top, 20)
                .padding(.bottom, 36)
            }
        }
        .overlay {
            if viewModel.isLoading && viewModel.stats == nil {
                LoadingOverlay()
            }
        }
        .task {
            if let server = store.currentServer {
                await viewModel.load(using: server)
            }
        }
        .onChange(of: store.currentServer) { _, server in
            guard let server else { return }
            Task { await viewModel.load(using: server) }
        }
        .onReceive(NotificationCenter.default.publisher(for: .recentPlaysDidUpdate)) { _ in
            guard let server = store.currentServer else { return }
            Task { await viewModel.refreshRecentPlays(using: server) }
        }
        .navigationDestination(item: $selectedRecentTrack) { track in
            TrackDetailView(track: track)
        }
        .navigationDestination(item: $selectedAlbumID) { albumID in
            albumDetailDestination(albumID: albumID)
        }
    }
}
#endif

private struct PadHomeHero: View {
    let nowPlaying: NowPlaying?
    let onOpenNowPlaying: () -> Void

    var body: some View {
        GlassPanel(cornerRadius: 28, padding: 0) {
            VStack(alignment: .leading, spacing: 18) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("SonicLens Bridge for iPad")
                        .font(.system(size: 14, weight: .semibold, design: .rounded))
                        .foregroundStyle(TrendHeatmapView.levelFour)
                    Text("声之透镜 · 深度解析 · 聆听之印记")
                        .font(.system(size: 28, weight: .bold, design: .rounded))
                        .foregroundStyle(SonicTheme.textPrimary)
                    Text("音乐不仅是流动的空气，更是我们生命中不曾停歇的数字资产")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                }

                if let nowPlaying {
                    Button(action: onOpenNowPlaying) {
                        HStack(spacing: 16) {
                            PlaybackArtworkView(artworkURL: nowPlaying.artwork)
                                .frame(width: 60, height: 60)

                            VStack(alignment: .leading, spacing: 4) {
                                Text("正在播放")
                                    .font(.caption.weight(.semibold))
                                    .foregroundStyle(SonicTheme.textSecondary)
                                Text(nowPlaying.track)
                                    .font(.title3.weight(.semibold))
                                    .foregroundStyle(SonicTheme.textPrimary)
                                    .lineLimit(1)
                                Text([nowPlaying.artist, nowPlaying.displayAlbumTitle].compactMap { $0 }.joined(separator: " · "))
                                    .font(.subheadline)
                                    .foregroundStyle(SonicTheme.textSecondary)
                                    .lineLimit(1)
                            }

                            Spacer()

                            Label("进入沉浸播放中", systemImage: "arrow.up.left.and.arrow.down.right")
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(TrendHeatmapView.levelFour)
                                .padding(.horizontal, 14)
                                .padding(.vertical, 10)
                                .background(TrendHeatmapView.levelFour.opacity(0.10), in: Capsule())
                        }
                        .padding(18)
                        .background(
                            SonicTheme.dynamicColor(
                                light: .sonicRGBA(0.99, 0.98, 0.95, 0.82),
                                dark: .sonicWhite(1, alpha: 0.08)
                            ),
                            in: RoundedRectangle(cornerRadius: 22)
                        )
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(24)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background {
                ZStack {
                    LinearGradient(
                        colors: [
                            SonicTheme.dynamicColor(light: .sonicRGBA(0.99, 0.98, 0.95, 1), dark: .sonicRGBA(0.10, 0.10, 0.12, 1)),
                            SonicTheme.dynamicColor(light: .sonicRGBA(0.96, 0.95, 0.91, 1), dark: .sonicRGBA(0.07, 0.08, 0.10, 1))
                        ],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                    RadialGradient(
                        colors: [TrendHeatmapView.levelFour.opacity(0.14), Color.clear],
                        center: .topLeading,
                        startRadius: 0,
                        endRadius: 280
                    )
                    RadialGradient(
                        colors: [SonicTheme.primary.opacity(0.035), Color.clear],
                        center: .bottomTrailing,
                        startRadius: 0,
                        endRadius: 220
                    )
                }
            }
        }
    }
}

private struct AdaptiveWidthContainer<Content: View, NarrowContent: View>: View {
    let minWidth: CGFloat
    private let content: Content
    private let narrowContent: NarrowContent

    init(
        minWidth: CGFloat,
        @ViewBuilder content: () -> Content,
        @ViewBuilder narrowContent: () -> NarrowContent
    ) {
        self.minWidth = minWidth
        self.content = content()
        self.narrowContent = narrowContent()
    }

    var body: some View {
        ViewThatFits(in: .horizontal) {
            content
                .frame(minWidth: minWidth, alignment: .leading)
            narrowContent
        }
    }
}

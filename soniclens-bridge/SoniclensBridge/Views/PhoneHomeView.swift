import SwiftUI

#if os(iOS)
struct PhoneHomeView: View {
    @EnvironmentObject private var store: AppStore
    @Environment(PlaybackStore.self) private var playbackStore
    @StateObject private var viewModel = HomeViewModel(defaultTrendRange: 30)
    @State private var selectedRecentTrack: Track?
    @State private var selectedAlbumID: Int64?
    @State private var selectedProfileDetail: ListeningProfileDetailMode?
    @State private var isShowingTrendDetail = false
    @State private var showAllArtists = false
    @State private var showAllAlbums = false
    @State private var showAllTracks = false

    private let profilePreviewCount = 3
    private let profileExpandedCount = 10

    let onOpenNowPlaying: () -> Void

    var body: some View {
        ZStack {
            AmbientBackgroundView(
                gradient: LinearGradient(
                    colors: [SonicTheme.background, SonicTheme.background.opacity(0.96), Color.accentColor.opacity(0.12)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                ),
                orbs: [
                    AmbientOrb(color: SonicTheme.primary.opacity(0.22), size: 360, blur: 120, opacity: 0.68, offsetFrom: CGSize(width: -140, height: -220), offsetTo: CGSize(width: -80, height: -120), duration: 18),
                    AmbientOrb(color: SonicTheme.secondaryAccent.opacity(0.14), size: 280, blur: 110, opacity: 0.62, offsetFrom: CGSize(width: 180, height: 120), offsetTo: CGSize(width: 120, height: 180), duration: 24)
                ],
                renderingStyle: .staticHome
            )

            ScrollView {
                LazyVStack(alignment: .leading, spacing: 18) {
                    if let message = viewModel.errorMessage {
                        ErrorBanner(message: message)
                    }

                    PhoneHomeHero(nowPlaying: playbackStore.nowPlaying, onOpenNowPlaying: onOpenNowPlaying)

                    let hotPresentation = viewModel.hotModulePresentation
                    let trendSnapshot = viewModel.trendSnapshot
                    let selectedAccent = hotPresentation.primaryAccentKey

                    VStack(alignment: .leading, spacing: 14) {
                        // HotModuleSectionHeader(
                        //     title: "听觉版图",
                        //     subtitle: "把最近最热的专辑、创作者和曲目压进首屏摘要，再把口味流派和播放渠道拆成两个更好扫读的卡片。",
                        //    accentKey: selectedAccent
                        // )

                        DashboardTrendSection(
                            points: trendSnapshot.points,
                            hourlyData: trendSnapshot.hourlyData,
                            title: "聆听趋势",
                            subtitle: "聚焦最近 30 天，保留更清晰的日期列和时段密度。",
                            heatmapHeight: 264,
                            heatmapLayout: .fitted(minCellWidth: 6),
                            axisLabelStyle: .dayStride(step: 4, rotationDegrees: 35),
                            actionPlacement: .metricsTrailing,
                            actionTitle: "查看 90 天",
                            onAction: { isShowingTrendDetail = true }
                        )
                        .equatable()
                        .frame(maxWidth: .infinity)

                        ListeningProfileRingPairCard(
                            summaryText: hotPresentation.combinedSummaryText,
                            genres: Array(hotPresentation.genres.prefix(profilePreviewCount)),
                            sources: Array(hotPresentation.sources.prefix(profilePreviewCount)),
                            accentKey: selectedAccent,
                            onSelect: { selectedProfileDetail = $0 }
                        )

                        AlbumShelfCard(
                            items: Array(showAllAlbums ? hotPresentation.albums.prefix(profileExpandedCount) : hotPresentation.albums.prefix(profilePreviewCount)),
                            artworkBaseURL: store.currentServer?.artworkBaseURL,
                            style: showAllAlbums ? .compactGrid : .rail,
                            collectionCount: hotPresentation.totalAlbumsCount,
                            accentKey: selectedAccent,
                            actionTitle: showAllAlbums ? "收起" : "查看全部",
                            onAction: { showAllAlbums.toggle() },
                            onAlbumTap: { selectedAlbumID = $0 }
                        )

                        ArtistLadderCard(
                            items: Array(showAllArtists ? hotPresentation.artists.prefix(profileExpandedCount) : hotPresentation.artists.prefix(profilePreviewCount)),
                            artworkBaseURL: store.currentServer?.artworkBaseURL,
                            collectionCount: hotPresentation.totalArtistsCount,
                            accentKey: selectedAccent,
                            actionTitle: showAllArtists ? "收起" : "查看全部",
                            onAction: { showAllArtists.toggle() }
                        )

                        TrackShelfCard(
                            items: Array(showAllTracks ? hotPresentation.tracks.prefix(profileExpandedCount) : hotPresentation.tracks.prefix(profilePreviewCount)),
                            artworkBaseURL: store.currentServer?.artworkBaseURL,
                            style: showAllTracks ? .compactGrid : .rail,
                            totalTracksCount: hotPresentation.totalTracksCount,
                            accentKey: selectedAccent,
                            actionTitle: showAllTracks ? "收起" : "查看全部",
                            onAction: { showAllTracks.toggle() },
                            onTrackTap: { selectedRecentTrack = $0.bridgeTrack }
                        )
                    }

                    RecentPlaysSection(
                        items: viewModel.recentPlays,
                        totalPlaysCount: hotPresentation.totalPlaysCount,
                        onTrackTap: { selectedRecentTrack = $0 }
                    )
                }
                .padding(.horizontal, 16)
                .padding(.top, 16)
                .padding(.bottom, 32)
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
        .navigationDestination(item: $selectedRecentTrack) { track in
            TrackDetailView(track: track)
        }
        .navigationDestination(item: $selectedAlbumID) { albumID in
            albumDetailDestination(albumID: albumID)
        }
        .navigationDestination(isPresented: $isShowingTrendDetail) {
            PhoneTrendDetailView()
        }
        .sheet(item: $selectedProfileDetail) { mode in
            ListeningProfileDetailSheet(
                mode: mode,
                summaryText: viewModel.hotModulePresentation.combinedSummaryText,
                footnoteText: viewModel.hotModulePresentation.profileFootnoteText,
                genres: Array(viewModel.hotModulePresentation.genres),
                sources: Array(viewModel.hotModulePresentation.sources),
                accentKey: viewModel.hotModulePresentation.primaryAccentKey
            )
            .presentationDetents([.medium, .large])
            .presentationDragIndicator(.visible)
        }
    }
}

private struct PhoneHomeHero: View {
    let nowPlaying: NowPlaying?
    let onOpenNowPlaying: () -> Void

    var body: some View {
        GlassPanel(cornerRadius: 24, padding: 0) {
            VStack(alignment: .leading, spacing: 16) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("SonicLens Bridge for iPhone")
                        .font(.system(size: 13, weight: .semibold, design: .rounded))
                        .foregroundStyle(TrendHeatmapView.levelFour)
                    Text("声之透镜 · 深度解析 · 聆听之印记")
                        .font(.system(size: 20, weight: .bold, design: .rounded))
                        .foregroundStyle(SonicTheme.textPrimary)
                    Text("音乐不仅是流动的空气，更是我们生命中不曾停歇的数字资产")
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                }

                if let nowPlaying {
                    Button(action: onOpenNowPlaying) {
                        HStack(spacing: 14) {
                            PlaybackArtworkView(artworkURL: nowPlaying.artwork)
                                .frame(width: 56, height: 56)

                            VStack(alignment: .leading, spacing: 3) {
                                Text("正在播放")
                                    .font(.caption.weight(.semibold))
                                    .foregroundStyle(SonicTheme.textSecondary)
                                Text(nowPlaying.track)
                                    .font(.headline.weight(.semibold))
                                    .foregroundStyle(SonicTheme.textPrimary)
                                    .lineLimit(1)
                                Text([nowPlaying.artist, nowPlaying.album].compactMap { $0 }.joined(separator: " · "))
                                    .font(.subheadline)
                                    .foregroundStyle(SonicTheme.textSecondary)
                                    .lineLimit(1)
                            }

                            Spacer()

                            Image(systemName: "arrow.up.left.and.arrow.down.right")
                                .font(.headline.weight(.semibold))
                                .foregroundStyle(TrendHeatmapView.levelFour)
                        }
                        .padding(14)
                        .background(
                            SonicTheme.dynamicColor(
                                light: .sonicRGBA(0.99, 0.98, 0.95, 0.82),
                                dark: .sonicWhite(1, alpha: 0.08)
                            ),
                            in: RoundedRectangle(cornerRadius: 18)
                        )
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(18)
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
                        endRadius: 260
                    )
                    RadialGradient(
                        colors: [SonicTheme.primary.opacity(0.035), Color.clear],
                        center: .bottomTrailing,
                        startRadius: 0,
                        endRadius: 200
                    )
                }
            }
        }
    }
}

private struct PhoneTrendDetailView: View {
    @EnvironmentObject private var store: AppStore
    @StateObject private var viewModel = TrendDetailViewModel(rangeDays: 90)

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                if let message = viewModel.errorMessage {
                    ErrorBanner(message: message)
                }

                DashboardTrendSection(
                    points: viewModel.trendPoints,
                    hourlyData: viewModel.hourlyData,
                    title: "聆听趋势",
                    subtitle: "横向滑动查看完整 90 天时间轴，深色表示更高频的聆听时段。",
                    heatmapHeight: 320,
                    heatmapLayout: .scrollable(minCellWidth: 8)
                )
                .equatable()

                GlassPanel(cornerRadius: 18, padding: 16) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("查看方式")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(SonicTheme.textPrimary)
                        Text("首页保留 30 天紧凑视图，便于 iPhone 快速扫读；90 天聆听趋势放到详情页横向展开，避免格子压扁和时间轴裁切。")
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(SonicTheme.textSecondary)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 16)
        }
        .navigationTitle("聆听趋势")
        .navigationBarTitleDisplayMode(.inline)
        .overlay {
            if viewModel.isLoading && viewModel.trendPoints.isEmpty {
                LoadingOverlay()
            }
        }
        .task {
            guard let server = store.currentServer else { return }
            await viewModel.load(using: server)
        }
        .onChange(of: store.currentServer) { _, server in
            guard let server else { return }
            Task { await viewModel.load(using: server, force: true) }
        }
    }
}

@MainActor
private final class TrendDetailViewModel: ObservableObject {
    @Published var trendPoints: [TrendPoint] = []
    @Published var hourlyData: [HourlyData] = []
    @Published var isLoading: Bool = false
    @Published var errorMessage: String?

    private let rangeDays: Int
    private var hasLoaded = false

    init(rangeDays: Int) {
        self.rangeDays = rangeDays
    }

    func load(using server: ServerConfig, force: Bool = false) async {
        if hasLoaded && !force {
            return
        }

        isLoading = true
        errorMessage = nil

        let client = APIClient(baseURL: server.baseURL)
        do {
            let response: TrendResponse = try await client.getJSON(
                path: APIPath.dashboardTrend,
                queryItems: [URLQueryItem(name: "range", value: "\(rangeDays)")]
            )
            let points = response.daily.map { TrendPoint(date: $0.key, count: $0.value) }
            let hourly = response.hourly.map { HourlyData(date: $0.key, hourly: $0.value.hourly) }

            trendPoints = points.sorted { $0.date < $1.date }
            hourlyData = hourly.sorted { $0.date < $1.date }
            hasLoaded = true
            isLoading = false
        } catch {
            errorMessage = "趋势数据加载失败"
            isLoading = false
        }
    }
}
#endif

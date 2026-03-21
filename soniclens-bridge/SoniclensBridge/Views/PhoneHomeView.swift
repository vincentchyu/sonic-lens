import SwiftUI

#if os(iOS)
struct PhoneHomeView: View {
    @EnvironmentObject private var store: AppStore
    @StateObject private var viewModel = HomeViewModel(defaultTrendRange: 30)
    @State private var selectedRecentTrack: Track?
    @State private var selectedAlbumID: Int64?
    @State private var isShowingTrendDetail = false

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
                ]
            )

            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    if let message = viewModel.errorMessage {
                        ErrorBanner(message: message)
                    }

                    PhoneHomeHero(nowPlaying: store.nowPlaying, onOpenNowPlaying: onOpenNowPlaying)

                    DashboardStatsSection(stats: viewModel.stats)

                    DashboardTrendSection(
                        points: viewModel.trendPoints,
                        hourlyData: viewModel.hourlyData,
                        title: "最近 30 天播放热力图",
                        subtitle: "iPhone 首页聚焦最近 30 天，保留更清晰的日期列和时段密度。",
                        heatmapHeight: 264,
                        heatmapLayout: .fitted(minCellWidth: 6),
                        axisLabelStyle: .dayStride(step: 4, rotationDegrees: 35),
                        actionPlacement: .metricsTrailing,
                        actionTitle: "查看 90 天",
                        onAction: { isShowingTrendDetail = true }
                    )
                    .equatable()
                    .frame(maxWidth: .infinity)

                    TopGenresCard(topGenres: viewModel.topGenres)

                    TopArtistsCard(topArtists: viewModel.topArtistsByPlays)

                    TopAlbumsCard(topAlbums: viewModel.topAlbums, onAlbumTap: { selectedAlbumID = $0 })

                    RecentPlaysSection(
                        items: viewModel.recentPlays,
                        onTrackTap: { selectedRecentTrack = $0 }
                    )

                    RankingsCard(
                        topArtistsByPlays: viewModel.topArtistsByPlays,
                        topTracks: viewModel.topTracks,
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
                        .foregroundStyle(Color.white.opacity(0.84))
                    Text("声之透镜 · 深度解析 · 聆听之印记")
                        .font(.system(size: 20, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)
                    Text("音乐不仅是流动的空气，更是我们生命中不曾停歇的数字资产")
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(Color.white.opacity(0.8))
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
                                    .foregroundStyle(Color.white.opacity(0.72))
                                Text(nowPlaying.track)
                                    .font(.headline.weight(.semibold))
                                    .foregroundStyle(.white)
                                    .lineLimit(1)
                                Text([nowPlaying.artist, nowPlaying.album].compactMap { $0 }.joined(separator: " · "))
                                    .font(.subheadline)
                                    .foregroundStyle(Color.white.opacity(0.76))
                                    .lineLimit(1)
                            }

                            Spacer()

                            Image(systemName: "arrow.up.left.and.arrow.down.right")
                                .font(.headline.weight(.semibold))
                                .foregroundStyle(.white)
                        }
                        .padding(14)
                        .background(Color.white.opacity(0.1), in: RoundedRectangle(cornerRadius: 18))
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(18)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(
                LinearGradient(
                    colors: [
                        SonicTheme.primary.opacity(0.88),
                        SonicTheme.secondaryAccent.opacity(0.54),
                        Color.black.opacity(0.14)
                    ],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
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
                    title: "最近 90 天播放热力图",
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
                        Text("首页保留 30 天紧凑视图，便于 iPhone 快速扫读；90 天时间轴放到详情页横向展开，避免格子压扁和时间轴裁切。")
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(SonicTheme.textSecondary)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 16)
        }
        .navigationTitle("播放热力图")
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

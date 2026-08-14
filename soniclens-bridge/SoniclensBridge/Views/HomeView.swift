import SwiftUI

struct HomeView: View {
    @EnvironmentObject var store: AppStore
    @StateObject private var viewModel = HomeViewModel()
    @State private var selectedRecentTrack: Track?
    @State private var selectedAlbumID: Int64?
    @State private var selectedGenreItem: HomeHotGenrePresentationItem?
    @State private var rankingSearchText: String = ""

    var body: some View {
        HomeContentView(
            viewModel: viewModel,
            selectedRecentTrack: $selectedRecentTrack,
            selectedAlbumID: $selectedAlbumID,
            selectedGenreItem: $selectedGenreItem,
            rankingSearchText: $rankingSearchText
        )
    }
}

struct HomeContentView: View {
    @EnvironmentObject private var store: AppStore
    @ObservedObject var viewModel: HomeViewModel
    @Binding var selectedRecentTrack: Track?
    @Binding var selectedAlbumID: Int64?
    @Binding var selectedGenreItem: HomeHotGenrePresentationItem?
    @Binding var rankingSearchText: String
    @State private var layoutModes = HomeLayoutModes()
    @State private var activeListeningProfileDetailMode: ListeningProfileDetailMode? = nil

    var body: some View {
        GeometryReader { proxy in
            ZStack {
                AmbientBackgroundView(
                    gradient: LinearGradient(
                        colors: [SonicTheme.background, SonicTheme.background],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    ),
                    orbs: [
                        AmbientOrb(color: SonicTheme.primary.opacity(0.35), size: 520, blur: 120, opacity: 0.7, offsetFrom: CGSize(width: -200, height: -260), offsetTo: CGSize(width: -120, height: -180), duration: 18),
                        AmbientOrb(color: SonicTheme.primary.opacity(0.25), size: 420, blur: 130, opacity: 0.7, offsetFrom: CGSize(width: 260, height: 120), offsetTo: CGSize(width: 200, height: 220), duration: 22),
                        AmbientOrb(color: SonicTheme.primary.opacity(0.18), size: 360, blur: 140, opacity: 0.6, offsetFrom: CGSize(width: 120, height: -40), offsetTo: CGSize(width: 80, height: 40), duration: 26)
                    ],
                    renderingStyle: .staticHome
                )

                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 28) {
                        if let message = viewModel.errorMessage {
                            ErrorBanner(message: message)
                        }

                        let hotPresentation = viewModel.hotModulePresentation
                        let trendSnapshot = viewModel.trendSnapshot
                        let selectedAccent = hotPresentation.primaryAccentKey

                        HomeHeroSection()

                        VStack(alignment: .leading, spacing: 14) {
                            if layoutModes.wideInsights {
                                WideListeningInsightsSection(
                                    points: trendSnapshot.points,
                                    hourlyData: trendSnapshot.hourlyData,
                                    trendSubtitle: "按日期展开 24 小时播放分布，保持时间轴比例，可横向浏览完整 90 天。",
                                    summaryText: hotPresentation.combinedSummaryText,
                                    footnoteText: hotPresentation.profileFootnoteText,
                                    genres: hotPresentation.genres,
                                    sources: hotPresentation.sources,
                                    accentKey: selectedAccent,
                                    onSelectGenre: { item in
                                        selectedGenreItem = item
                                    },
                                    onOpenDetailMode: { mode in
                                        activeListeningProfileDetailMode = mode
                                    }
                                )
                                .frame(maxWidth: .infinity, alignment: .topLeading)
                            } else {
                                VStack(alignment: .leading, spacing: 14) {
                                    DashboardTrendSection(
                                        points: trendSnapshot.points,
                                        hourlyData: trendSnapshot.hourlyData,
                                        subtitle: "按日期展开 24 小时播放分布，保持时间轴比例，可横向浏览完整 90 天。",
                                        heatmapLayout: .fixedWidthScrollable(cellWidth: 8),
                                        axisLabelStyle: .dayStride(step: 3, rotationDegrees: 42)
                                    )
                                    .equatable()
                                    .frame(maxWidth: .infinity)

                                    ListeningProfileCard(
                                        summaryText: hotPresentation.combinedSummaryText,
                                        footnoteText: hotPresentation.profileFootnoteText,
                                        genres: hotPresentation.genres,
                                        sources: hotPresentation.sources,
                                        accentKey: selectedAccent,
                                        layoutStyle: .split,
                                        onSelectGenre: { item in
                                            selectedGenreItem = item
                                        },
                                        onOpenDetailMode: { mode in
                                            activeListeningProfileDetailMode = mode
                                        }
                                    )
                                }
                            }

                            if layoutModes.wideRankings {
                                HStack(alignment: .top, spacing: 16) {
                                    AlbumShelfCard(
                                        items: Array(hotPresentation.albums.prefix(5)),
                                        artworkBaseURL: store.currentServer?.artworkBaseURL,
                                        style: .featureGrid,
                                        collectionCount: hotPresentation.totalAlbumsCount,
                                        accentKey: selectedAccent,
                                        onAlbumTap: { albumID in
                                            selectedAlbumID = albumID
                                        }
                                    )
                                    .frame(minWidth: AlbumShelfCard.homeMinimumWidth, maxWidth: .infinity, alignment: .topLeading)

                                    ArtistLadderCard(
                                        items: Array(hotPresentation.artists.prefix(6)),
                                        artworkBaseURL: store.currentServer?.artworkBaseURL,
                                        collectionCount: hotPresentation.totalArtistsCount,
                                        accentKey: selectedAccent
                                    )
                                    .frame(minWidth: ArtistLadderCard.homeMinimumWidth, maxWidth: .infinity, alignment: .topLeading)
                                }

                                HStack(alignment: .top, spacing: 16) {
                                    TrackShelfCard(
                                        items: Array(hotPresentation.tracks.prefix(10)),
                                        artworkBaseURL: store.currentServer?.artworkBaseURL,
                                        style: .featureGrid,
                                        visibleItemCount: 9,
                                        totalTracksCount: hotPresentation.totalTracksCount,
                                        accentKey: selectedAccent,
                                        onTrackTap: { selectedRecentTrack = $0.bridgeTrack }
                                    )
                                    .frame(minWidth: TrackShelfCard.homeMinimumWidth, maxWidth: .infinity, alignment: .topLeading)

                                    RecentPlaysSection(
                                        items: viewModel.recentPlays,
                                        artworkURLs: viewModel.recentPlayArtworkURLs,
                                        totalPlaysCount: hotPresentation.totalPlaysCount,
                                        accentKey: selectedAccent,
                                        onTrackTap: { track in
                                            selectedRecentTrack = track
                                        }
                                    )
                                    .frame(minWidth: RecentPlaysSection.homeMinimumWidth, maxWidth: .infinity, alignment: .topLeading)
                                }
                            } else {
                                VStack(alignment: .leading, spacing: 16) {
                                    AlbumShelfCard(
                                        items: Array(hotPresentation.albums.prefix(5)),
                                        artworkBaseURL: store.currentServer?.artworkBaseURL,
                                        style: .featureGrid,
                                        collectionCount: hotPresentation.totalAlbumsCount,
                                        accentKey: selectedAccent,
                                        onAlbumTap: { albumID in
                                            selectedAlbumID = albumID
                                        }
                                    )

                                    ArtistLadderCard(
                                        items: Array(hotPresentation.artists.prefix(6)),
                                        artworkBaseURL: store.currentServer?.artworkBaseURL,
                                        collectionCount: hotPresentation.totalArtistsCount,
                                        accentKey: selectedAccent
                                    )

                                    TrackShelfCard(
                                        items: Array(hotPresentation.tracks.prefix(5)),
                                        artworkBaseURL: store.currentServer?.artworkBaseURL,
                                        style: .featureGrid,
                                        visibleItemCount: 5,
                                        totalTracksCount: hotPresentation.totalTracksCount,
                                        accentKey: selectedAccent,
                                        onTrackTap: { selectedRecentTrack = $0.bridgeTrack }
                                    )

                                    RecentPlaysSection(
                                        items: viewModel.recentPlays,
                                        artworkURLs: viewModel.recentPlayArtworkURLs,
                                        totalPlaysCount: hotPresentation.totalPlaysCount,
                                        accentKey: selectedAccent,
                                        onTrackTap: { track in
                                            selectedRecentTrack = track
                                        }
                                    )
                                }
                            }
                        }
                    }
                    .padding(.horizontal, 32)
                    .padding(.top, 28)
                    .padding(.bottom, 108)
                }
            }
        }
        .background(
            GeometryReader { proxy in
                Color.clear
                    .onAppear {
                        layoutModes.update(for: max(proxy.size.width - 64, 0), force: true)
                    }
                    .onChange(of: proxy.size.width) { _, newValue in
                        layoutModes.update(for: max(newValue - 64, 0))
                    }
            }
        )
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
        .sheet(item: $selectedGenreItem) { genreItem in
            GenreAlbumsSheet(
                item: genreItem,
                onSelectAlbum: { albumID in
                    selectedAlbumID = albumID
                }
            )
        }
        .sheet(item: $activeListeningProfileDetailMode) { mode in
            ListeningProfileDetailSheet(
                mode: mode,
                summaryText: viewModel.hotModulePresentation.combinedSummaryText,
                footnoteText: viewModel.hotModulePresentation.profileFootnoteText,
                genres: viewModel.hotModulePresentation.genres,
                sources: viewModel.hotModulePresentation.sources,
                accentKey: viewModel.hotModulePresentation.primaryAccentKey,
                onSelectGenre: { genreItem in
                    activeListeningProfileDetailMode = nil
                    selectedGenreItem = genreItem
                }
            )
        }
    }
}

private struct HomeLayoutModes {
    var wideInsights = false
    var wideRankings = false
    private var didInitialize = false

    mutating func update(for width: CGFloat, force: Bool = false) {
        if force || !didInitialize {
            wideInsights = width >= 1120
            wideRankings = width >= 960
            didInitialize = true
            return
        }

        if wideInsights {
            if width < 980 {
                wideInsights = false
            }
        } else if width > 1120 {
            wideInsights = true
        }

        if wideRankings {
            if width < 840 {
                wideRankings = false
            }
        } else if width > 960 {
            wideRankings = true
        }
    }
}

struct HomeHeroSection: View {
    var body: some View {
        GlassPanel(cornerRadius: 22, padding: 0) {
            ZStack(alignment: .bottomLeading) {
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
                        colors: [
                            TrendHeatmapView.levelFour.opacity(0.14),
                            Color.clear
                        ],
                        center: .topLeading,
                        startRadius: 0,
                        endRadius: 280
                    )
                    RadialGradient(
                        colors: [
                            SonicTheme.primary.opacity(0.035),
                            Color.clear
                        ],
                        center: .bottomTrailing,
                        startRadius: 0,
                        endRadius: 220
                    )
                }
                .cornerRadius(22)

                VStack(alignment: .leading, spacing: 6) {
                    Text("SonicLens Bridge for Mac")
                       .font(.system(size: 12, weight: .semibold, design: .rounded))
                       .foregroundStyle(TrendHeatmapView.levelFour)
                    Text("声之透镜 · 深度解析 · 聆听之印记")
                        .font(.system(size: 26, weight: .bold, design: .rounded))
                        .foregroundStyle(SonicTheme.textPrimary)
                    Text("音乐不仅是流动的空气，更是我们生命中不曾停歇的数字资产")
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
                .padding(22)
            }
            .frame(height: 100)
        }
    }
}

struct DashboardStatsSection: View {
    let stats: DashboardStats?

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            SectionHeader(title: "仪表盘")

            let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]

            LazyVGrid(columns: columns, spacing: 14) {
                HomeStatCard(title: "播放", value: stats?.totalPlays ?? 0, tint: Color.red)
                HomeStatCard(title: "曲目", value: stats?.totalTracks ?? 0, tint: Color.blue)
                HomeStatCard(title: "艺术家", value: stats?.totalArtists ?? 0, tint: Color.green)
                HomeStatCard(title: "专辑", value: stats?.totalAlbums ?? 0, tint: Color.purple)
            }
        }
    }
}

struct HomeStatCard: View {
    let title: String
    let value: Int64
    let tint: Color

    var body: some View {
        GlassPanel(cornerRadius: 16, padding: 14) {
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    Text(title)
                        .font(.caption)
                        .foregroundColor(SonicTheme.textSecondary)
                    Spacer()
                    Circle()
                        .fill(tint.opacity(0.18))
                        .frame(width: 28, height: 28)
                        .overlay(Circle().stroke(tint.opacity(0.4), lineWidth: 1))
                }
                Text("\(value)")
                    .font(.system(size: 22, weight: .bold))
                    .foregroundColor(SonicTheme.textPrimary)
            }
            .frame(maxWidth: .infinity, minHeight: 80, alignment: .leading)
        }
    }
}

struct DashboardTrendSection: View, Equatable {
    let points: [TrendPoint]
    let hourlyData: [HourlyData]
    var title: String = "聆听趋势"
    var subtitle: String = "按日期展开 24 小时播放分布，深色表示更高频的聆听时段"
    var heatmapHeight: CGFloat = 276
    var heatmapLayout: TrendHeatmapLayout = .fitted(minCellWidth: 2)
    var axisLabelStyle: TrendAxisLabelStyle = .monthBoundaries
    var actionPlacement: TrendActionPlacement = .header
    var actionTitle: String? = nil
    var onAction: (() -> Void)? = nil

    static func == (lhs: DashboardTrendSection, rhs: DashboardTrendSection) -> Bool {
        lhs.points == rhs.points &&
        lhs.hourlyData == rhs.hourlyData &&
        lhs.title == rhs.title &&
        lhs.subtitle == rhs.subtitle &&
        lhs.heatmapHeight == rhs.heatmapHeight &&
        lhs.heatmapLayout == rhs.heatmapLayout &&
        lhs.axisLabelStyle == rhs.axisLabelStyle &&
        lhs.actionPlacement == rhs.actionPlacement &&
        lhs.actionTitle == rhs.actionTitle
    }

    var body: some View {
        GlassPanel(cornerRadius: 18, padding: 0) {
            DashboardTrendSectionBody(
                points: points,
                hourlyData: hourlyData,
                title: title,
                subtitle: subtitle,
                heatmapHeight: heatmapHeight,
                heatmapLayout: heatmapLayout,
                axisLabelStyle: axisLabelStyle,
                actionPlacement: actionPlacement,
                actionTitle: actionTitle,
                onAction: onAction
            )
        }
    }
}

private struct DashboardTrendSectionBody: View, Equatable {
    let points: [TrendPoint]
    let hourlyData: [HourlyData]
    let title: String
    let subtitle: String
    let heatmapHeight: CGFloat
    let heatmapLayout: TrendHeatmapLayout
    let axisLabelStyle: TrendAxisLabelStyle
    let actionPlacement: TrendActionPlacement
    let actionTitle: String?
    let onAction: (() -> Void)?
    private let metricsSnapshot: TrendMetricsSnapshot
    private let heatmapRenderModel: TrendHeatmapView.RenderModel

    init(
        points: [TrendPoint],
        hourlyData: [HourlyData],
        title: String,
        subtitle: String,
        heatmapHeight: CGFloat,
        heatmapLayout: TrendHeatmapLayout,
        axisLabelStyle: TrendAxisLabelStyle,
        actionPlacement: TrendActionPlacement,
        actionTitle: String?,
        onAction: (() -> Void)?
    ) {
        self.points = points
        self.hourlyData = hourlyData
        self.title = title
        self.subtitle = subtitle
        self.heatmapHeight = heatmapHeight
        self.heatmapLayout = heatmapLayout
        self.axisLabelStyle = axisLabelStyle
        self.actionPlacement = actionPlacement
        self.actionTitle = actionTitle
        self.onAction = onAction
        self.metricsSnapshot = TrendMetricsSnapshot(points: points, hourlyData: hourlyData)
        self.heatmapRenderModel = TrendHeatmapView.RenderModel(hourlyData: hourlyData, axisLabelStyle: axisLabelStyle)
    }

    static func == (lhs: DashboardTrendSectionBody, rhs: DashboardTrendSectionBody) -> Bool {
        lhs.points == rhs.points &&
        lhs.hourlyData == rhs.hourlyData &&
        lhs.title == rhs.title &&
        lhs.subtitle == rhs.subtitle &&
        lhs.heatmapHeight == rhs.heatmapHeight &&
        lhs.heatmapLayout == rhs.heatmapLayout &&
        lhs.axisLabelStyle == rhs.axisLabelStyle &&
        lhs.actionPlacement == rhs.actionPlacement &&
        lhs.actionTitle == rhs.actionTitle
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            VStack(alignment: .leading, spacing: 14) {
                headerContent

                if !points.isEmpty {
                    metricsContent
                }
            }
            .padding(.horizontal, 18)
            .padding(.top, 18)

            TrendHeatmapView(renderModel: heatmapRenderModel, layout: heatmapLayout, axisLabelStyle: axisLabelStyle)
            .frame(height: heatmapHeight)
            .padding(.horizontal, 18)
            .padding(.bottom, 18)
        }
        .padding(.bottom, 6)
    }

    private var headerContent: some View {
        ViewThatFits(in: .horizontal) {
            HStack(alignment: .top, spacing: 12) {
                headerText
                Spacer(minLength: 12)
                HStack(spacing: 10) {
                    TrendLegendView(maxCount: metricsSnapshot.maxHourlyCount)
                    if actionPlacement == .header {
                        headerAction
                    }
                }
            }

            VStack(alignment: .leading, spacing: 12) {
                headerText
                TrendLegendView(maxCount: metricsSnapshot.maxHourlyCount)
                if actionPlacement == .header {
                    headerAction
                }
            }
        }
    }

    private var metricsContent: some View {
        ViewThatFits(in: .horizontal) {
            HStack(spacing: 10) {
                TrendMetricPill(title: "总播放", value: "\(metricsSnapshot.totalPlays)")
                TrendMetricPill(title: "日均", value: "\(metricsSnapshot.averageDailyPlays)")
                TrendMetricPill(
                    title: "峰值",
                    value: metricsSnapshot.peakDay.map { "\($0.count) · \(TrendHeatmapView.compactDateLabel(for: $0.date))" } ?? "--",
                    valueColor: .orange
                )
                if actionPlacement == .metricsTrailing {
                    headerAction
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)

            VStack(alignment: .leading, spacing: 10) {
                HStack(spacing: 10) {
                    TrendMetricPill(title: "总播放", value: "\(metricsSnapshot.totalPlays)")
                    TrendMetricPill(title: "日均", value: "\(metricsSnapshot.averageDailyPlays)")
                    TrendMetricPill(
                        title: "峰值",
                        value: metricsSnapshot.peakDay.map { "\($0.count) · \(TrendHeatmapView.compactDateLabel(for: $0.date))" } ?? "--",
                        valueColor: .orange
                    )
                }
                if actionPlacement == .metricsTrailing {
                    headerAction
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var headerText: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 8) {
                Capsule()
                    .fill(
                        LinearGradient(
                            colors: [
                                TrendHeatmapView.levelTwo.opacity(0.92),
                                TrendHeatmapView.levelFour.opacity(0.82)
                            ],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                    )
                    .frame(width: 18, height: 6)
                Text(title)
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(SonicTheme.textPrimary)
            }
            Text(subtitle)
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(SonicTheme.textSecondary)
        }
    }

    @ViewBuilder
    private var headerAction: some View {
        if let actionTitle, let onAction {
            Button(action: onAction) {
                Label(actionTitle, systemImage: "arrow.right")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(SonicTheme.primary)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 7)
                    .background(
                        Capsule()
                            .fill(SonicTheme.primary.opacity(0.10))
                    )
            }
            .buttonStyle(.plain)
        }
    }
}

private struct TrendMetricsSnapshot: Equatable {
    let totalPlays: Int
    let averageDailyPlays: Int
    let peakDay: TrendPoint?
    let maxHourlyCount: Int

    init(points: [TrendPoint], hourlyData: [HourlyData]) {
        let totalPlays = points.map(\.count).reduce(0, +)
        self.totalPlays = totalPlays
        self.averageDailyPlays = points.isEmpty ? 0 : totalPlays / points.count
        self.peakDay = points.max(by: { lhs, rhs in
            if lhs.count == rhs.count {
                return lhs.date < rhs.date
            }
            return lhs.count < rhs.count
        })
        self.maxHourlyCount = hourlyData.reduce(0) { partialResult, day in
            max(partialResult, day.hourly.values.max() ?? 0)
        }
    }
}

private struct WideListeningInsightsSection: View {
    let points: [TrendPoint]
    let hourlyData: [HourlyData]
    let trendSubtitle: String
    let summaryText: String
    let footnoteText: String
    let genres: [HomeHotGenrePresentationItem]
    let sources: [HomeHotSourcePresentationItem]
    let accentKey: HomeHotAccentKey?
    var onSelectGenre: ((HomeHotGenrePresentationItem) -> Void)? = nil
    var onOpenDetailMode: ((ListeningProfileDetailMode) -> Void)? = nil

    var body: some View {
        GlassPanel(cornerRadius: 18, padding: 0) {
            HStack(alignment: .top, spacing: 0) {
                ListeningProfileSummarySidebar(
                    summaryText: summaryText,
                    footnoteText: footnoteText,
                    genres: genres,
                    sources: sources,
                    accentKey: accentKey,
                    onSelectGenre: onSelectGenre,
                    onOpenDetailMode: onOpenDetailMode
                )
                .padding(18)
                .frame(width: 380, alignment: .topLeading)
                
                
                Rectangle()
                    .fill(SonicTheme.glassBorder.opacity(0.9))
                    .frame(width: 1)
                    .padding(.vertical, 18)
                
                DashboardTrendSectionBody(
                    points: points,
                    hourlyData: hourlyData,
                    title: "聆听趋势",
                    subtitle: trendSubtitle,
                    heatmapHeight: 276,
                    heatmapLayout: .fixedWidthScrollable(cellWidth: 8),
                    axisLabelStyle: .dayStride(step: 3, rotationDegrees: 42),
                    actionPlacement: .header,
                    actionTitle: nil,
                    onAction: nil
                )
                .frame(minWidth: 720, maxWidth: .infinity, alignment: .topLeading)


               
            }
        }
    }
}

enum TrendHeatmapLayout: Equatable {
    case fitted(minCellWidth: CGFloat)
    case fixedWidthScrollable(cellWidth: CGFloat)
    case scrollable(minCellWidth: CGFloat)
}

enum TrendActionPlacement: Equatable {
    case header
    case metricsTrailing
}

enum TrendAxisLabelStyle: Equatable {
    case monthBoundaries
    case dayStride(step: Int, rotationDegrees: Double)

    var axisHeight: CGFloat {
        switch self {
        case .monthBoundaries:
            return 24
        case .dayStride:
            return 34
        }
    }

    var rotationDegrees: Double {
        switch self {
        case .monthBoundaries:
            return 0
        case let .dayStride(_, rotationDegrees):
            return rotationDegrees
        }
    }

    var axisToGridSpacing: CGFloat {
        switch self {
        case .monthBoundaries:
            return 8
        case .dayStride:
            return 2
        }
    }

    var trailingLabelInset: CGFloat {
        switch self {
        case .monthBoundaries:
            return 54
        case .dayStride:
            return 26
        }
    }

    var labelYOffset: CGFloat {
        switch self {
        case .monthBoundaries:
            return 2
        case .dayStride:
            return 6
        }
    }

    var yAxisColumnWidth: CGFloat {
        switch self {
        case .monthBoundaries:
            return 34
        case .dayStride:
            return 28
        }
    }

    var yAxisTextWidth: CGFloat {
        switch self {
        case .monthBoundaries:
            return 26
        case .dayStride:
            return 24
        }
    }

    var yAxisTextAlignment: Alignment {
        switch self {
        case .monthBoundaries:
            return .trailing
        case .dayStride:
            return .leading
        }
    }
}

struct TrendHeatmapView: View, Equatable {
    var layout: TrendHeatmapLayout = .fitted(minCellWidth: 2)
    var axisLabelStyle: TrendAxisLabelStyle = .monthBoundaries
    @State private var fixedScrollOffsetX: CGFloat = 0

    private let model: RenderModel
    private let cellSpacing: CGFloat = 2
    private let gridInset: CGFloat = 10

    init(
        hourlyData: [HourlyData],
        layout: TrendHeatmapLayout = .fitted(minCellWidth: 2),
        axisLabelStyle: TrendAxisLabelStyle = .monthBoundaries
    ) {
        self.layout = layout
        self.axisLabelStyle = axisLabelStyle
        self.model = RenderModel(hourlyData: hourlyData, axisLabelStyle: axisLabelStyle)
    }

    fileprivate init(
        renderModel: RenderModel,
        layout: TrendHeatmapLayout = .fitted(minCellWidth: 2),
        axisLabelStyle: TrendAxisLabelStyle = .monthBoundaries
    ) {
        self.layout = layout
        self.axisLabelStyle = axisLabelStyle
        self.model = renderModel
    }

    private var xAxisHeight: CGFloat {
        axisLabelStyle.axisHeight
    }

    private static let hours = Array(stride(from: 23, through: 0, by: -1))

    static func == (lhs: TrendHeatmapView, rhs: TrendHeatmapView) -> Bool {
        lhs.layout == rhs.layout &&
        lhs.axisLabelStyle == rhs.axisLabelStyle &&
        lhs.model == rhs.model
    }

    var body: some View {
        GeometryReader { geometry in
            let rowCount = CGFloat(Self.hours.count)
            let availableHeight = max(geometry.size.height - xAxisHeight, 0)
            let gridHeight = max(availableHeight - axisLabelStyle.axisToGridSpacing - gridInset * 2, 0)
            let cellHeight = max(
                4,
                (gridHeight - (rowCount - 1) * cellSpacing) / rowCount
            )

            HStack(alignment: .top, spacing: 0) {
                hourLabelsColumn(cellHeight: cellHeight)
                switch layout {
                case let .fitted(minCellWidth):
                    let availableWidth = max(geometry.size.width - axisLabelStyle.yAxisColumnWidth, 0)
                    let dayCount = max(model.dayColumns.count, 1)
                    let gridWidth = max(availableWidth - gridInset * 2, 0)
                    let cellWidth = max(
                        minCellWidth,
                        (gridWidth - CGFloat(dayCount - 1) * cellSpacing) / CGFloat(dayCount)
                    )
                    let contentWidth = CGFloat(dayCount) * cellWidth + CGFloat(dayCount - 1) * cellSpacing

                    heatmapColumns(
                        cellWidth: cellWidth,
                        cellHeight: cellHeight,
                        contentWidth: contentWidth
                    )
                case let .fixedWidthScrollable(cellWidth):
                    let availableWidth = max(geometry.size.width - axisLabelStyle.yAxisColumnWidth, 0)
                    let dayCount = max(model.dayColumns.count, 1)
                    let resolvedCellWidth = max(cellWidth, 2)
                    let resolvedCellHeight = resolvedCellWidth
                    let contentWidth = CGFloat(dayCount) * resolvedCellWidth + CGFloat(dayCount - 1) * cellSpacing
                    let maxScrollDistance = max(contentWidth + gridInset * 2 - availableWidth, 0)

                    ScrollView(.horizontal, showsIndicators: false) {
                        heatmapColumns(
                            cellWidth: resolvedCellWidth,
                            cellHeight: resolvedCellHeight,
                            contentWidth: contentWidth
                        )
                        .background(
                            GeometryReader { proxy in
                                Color.clear.preference(
                                    key: TrendFixedScrollOffsetPreferenceKey.self,
                                    value: proxy.frame(in: .named("trend-fixed-scroll")).minX
                                )
                            }
                        )
                    }
                    .coordinateSpace(name: "trend-fixed-scroll")
                    .onPreferenceChange(TrendFixedScrollOffsetPreferenceKey.self) { value in
                        fixedScrollOffsetX = value
                    }
                    .overlay(alignment: .topTrailing) {
                        if let hint = fixedScrollHint(maxScrollDistance: maxScrollDistance) {
                            Text(hint)
                                .font(.system(size: 10, weight: .semibold))
                                .foregroundStyle(SonicTheme.textSecondary)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 5)
                                .background(
                                    Capsule()
                                        .fill(SonicTheme.dynamicColor(
                                            light: .sonicWhite(1, alpha: 0.86),
                                            dark: .sonicWhite(0.14, alpha: 0.9)
                                        ))
                                )
                                .overlay(
                                    Capsule()
                                        .stroke(SonicTheme.glassBorder.opacity(0.75), lineWidth: 1)
                                )
                                .padding(.trailing, 8)
                        }
                    }
                case let .scrollable(minCellWidth):
                    let dayCount = max(model.dayColumns.count, 1)
                    let cellWidth = max(minCellWidth, 2)
                    let contentWidth = CGFloat(dayCount) * cellWidth + CGFloat(dayCount - 1) * cellSpacing

                    ScrollView(.horizontal, showsIndicators: false) {
                        heatmapColumns(
                            cellWidth: cellWidth,
                            cellHeight: cellHeight,
                            contentWidth: contentWidth
                        )
                    }
                }
            }
        }
    }

    private func hourLabelsColumn(cellHeight: CGFloat) -> some View {
        VStack(alignment: .trailing, spacing: cellSpacing) {
            ForEach(Self.hours, id: \.self) { hour in
                if hour == 23 || hour == 18 || hour == 12 || hour == 6 || hour == 0 {
                    Text(String(format: "%02d", hour))
                        .font(.system(size: 9, weight: .medium, design: .monospaced))
                        .foregroundStyle(SonicTheme.textSecondary)
                        .frame(
                            width: axisLabelStyle.yAxisTextWidth,
                            height: cellHeight,
                            alignment: axisLabelStyle.yAxisTextAlignment
                        )
                } else {
                    Spacer().frame(width: axisLabelStyle.yAxisTextWidth, height: cellHeight)
                }
            }
        }
        .padding(.top, xAxisHeight + axisLabelStyle.axisToGridSpacing + gridInset)
        .padding(.bottom, gridInset)
        .frame(width: axisLabelStyle.yAxisColumnWidth, alignment: .leading)
    }

    private func heatmapColumns(
        cellWidth: CGFloat,
        cellHeight: CGFloat,
        contentWidth: CGFloat,
        dayColumns: [DayColumn]? = nil,
        axisLabels: [AxisLabel]? = nil
    ) -> some View {
        let resolvedDayColumns = dayColumns ?? model.dayColumns
        let resolvedAxisLabels = axisLabels ?? model.axisLabels

        return VStack(alignment: .leading, spacing: axisLabelStyle.axisToGridSpacing) {
            xAxisLabels(contentWidth: contentWidth, cellWidth: cellWidth, axisLabels: resolvedAxisLabels)
            dayColumnsView(
                cellWidth: cellWidth,
                cellHeight: cellHeight,
                contentWidth: contentWidth,
                dayColumns: resolvedDayColumns
            )
        }
    }

    private func xAxisLabels(contentWidth: CGFloat, cellWidth: CGFloat, axisLabels: [AxisLabel]) -> some View {
        ZStack(alignment: .topLeading) {
            ForEach(axisLabels) { item in
                Text(item.text)
                    .font(.system(size: 10, weight: .medium, design: .monospaced))
                    .foregroundStyle(SonicTheme.textSecondary)
                    .fixedSize()
                    .rotationEffect(.degrees(axisLabelStyle.rotationDegrees), anchor: .topLeading)
                    .offset(
                        x: min(
                            max(0, labelX(at: item.index, cellWidth: cellWidth) - 12),
                            max(contentWidth - axisLabelStyle.trailingLabelInset, 0)
                        ),
                        y: axisLabelStyle.labelYOffset
                    )
            }
        }
        .frame(width: contentWidth, height: xAxisHeight, alignment: .topLeading)
        .padding(.horizontal, gridInset)
    }

    private func dayColumnsView(
        cellWidth: CGFloat,
        cellHeight: CGFloat,
        contentWidth: CGFloat,
        dayColumns: [DayColumn]
    ) -> some View {
        TrendHeatmapCanvas(
            dayColumns: dayColumns,
            cellWidth: cellWidth,
            cellHeight: cellHeight,
            cellSpacing: cellSpacing,
            maxCount: model.maxCount
        )
        .frame(
            width: contentWidth,
            height: CGFloat(Self.hours.count) * cellHeight + CGFloat(Self.hours.count - 1) * cellSpacing,
            alignment: .leading
        )
        .padding(gridInset)
        .background(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .fill(Self.gridBackground)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .stroke(Self.gridBorder, lineWidth: 1)
        )
    }

    private func fixedScrollHint(maxScrollDistance: CGFloat) -> String? {
        guard maxScrollDistance > 24 else { return nil }
        let offset = max(0, -fixedScrollOffsetX)
        if offset <= 12 {
            return "可向右查看更早日期"
        }
        if offset >= maxScrollDistance - 12 {
            return "可向左返回最近日期"
        }
        return nil
    }

    private func labelX(at index: Int, cellWidth: CGFloat) -> CGFloat {
        CGFloat(index) * (cellWidth + cellSpacing) + cellWidth / 2
    }

    static func compactDateLabel(for dateString: String) -> String {
        guard let date = dayFormatter.date(from: dateString) else { return dateString }
        return shortDayFormatter.string(from: date)
    }

    private static func makeAxisLabels(for dates: [String], axisLabelStyle: TrendAxisLabelStyle) -> [AxisLabel] {
        switch axisLabelStyle {
        case .monthBoundaries:
            return monthAxisLabels(for: dates)
        case let .dayStride(step, _):
            return strideAxisLabels(for: dates, step: step)
        }
    }

    private static func monthAxisLabels(for dates: [String]) -> [AxisLabel] {
        guard !dates.isEmpty else { return [] }

        var labels: [AxisLabel] = []
        var lastYearMonth: String?
        for (index, day) in dates.enumerated() {
            guard let date = dayFormatter.date(from: day) else { continue }
            let yearMonth = yearMonthFormatter.string(from: date)
            if index == 0 || yearMonth != lastYearMonth {
                labels.append(AxisLabel(index: index, text: yearMonth))
                lastYearMonth = yearMonth
            }
        }
        if labels.last?.index != dates.count - 1,
           let lastDate = dayFormatter.date(from: dates[dates.count - 1]) {
            labels.append(AxisLabel(index: dates.count - 1, text: yearMonthFormatter.string(from: lastDate)))
        }
        return dedupAxisLabels(labels)
    }

    private static func strideAxisLabels(for dates: [String], step: Int) -> [AxisLabel] {
        guard !dates.isEmpty else { return [] }

        let effectiveStep = max(step, 1)
        let lastIndex = dates.count - 1
        var labels = stride(from: 0, through: lastIndex, by: effectiveStep).map { index in
            AxisLabel(index: index, text: compactDateLabel(for: dates[index]))
        }

        if labels.last?.index != lastIndex {
            let lastLabel = AxisLabel(index: lastIndex, text: compactDateLabel(for: dates[lastIndex]))
            if let previousIndex = labels.indices.last, lastIndex - labels[previousIndex].index < effectiveStep {
                labels[previousIndex] = lastLabel
            } else {
                labels.append(lastLabel)
            }
        }

        return dedupAxisLabels(labels)
    }

    private static func dedupAxisLabels(_ labels: [AxisLabel]) -> [AxisLabel] {
        var unique: [AxisLabel] = []
        var seen = Set<String>()
        for label in labels where !seen.contains(label.text) {
            seen.insert(label.text)
            unique.append(label)
        }
        return unique.sorted { $0.index < $1.index }
    }

    fileprivate static func color(for count: Int, maxCount: Int) -> Color {
        if count == 0 { return emptyCell }
        let ratio = Double(count) / Double(max(maxCount, 1))
        switch ratio {
        case 0..<0.25: return levelOne
        case 0.25..<0.5: return levelTwo
        case 0.5..<0.75: return levelThree
        default: return levelFour
        }
    }

    static let emptyCell = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.98, 0.96, 0.91, 1),
        dark: .sonicRGBA(0.12, 0.10, 0.08, 1)
    )
    static let levelOne = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.96, 0.89, 0.70, 1),
        dark: .sonicRGBA(0.34, 0.24, 0.14, 1)
    )
    static let levelTwo = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.90, 0.75, 0.42, 1),
        dark: .sonicRGBA(0.48, 0.35, 0.16, 1)
    )
    static let levelThree = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.84, 0.62, 0.24, 1),
        dark: .sonicRGBA(0.64, 0.47, 0.18, 1)
    )
    static let levelFour = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.74, 0.49, 0.12, 1),
        dark: .sonicRGBA(0.82, 0.63, 0.26, 1)
    )
    private static let gridBackground = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.99, 0.97, 0.93, 0.82),
        dark: .sonicRGBA(0.10, 0.08, 0.06, 0.92)
    )
    private static let gridBorder = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.76, 0.56, 0.22, 0.10),
        dark: .sonicRGBA(0.88, 0.70, 0.34, 0.14)
    )

    private static let dayFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter
    }()

    private static let shortDayFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateFormat = "M/d"
        return formatter
    }()

    private static let yearMonthFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy/MM"
        return formatter
    }()

    fileprivate struct RenderModel: Equatable {
        let dayColumns: [DayColumn]
        let maxCount: Int
        let axisLabels: [AxisLabel]

        init(hourlyData: [HourlyData], axisLabelStyle: TrendAxisLabelStyle) {
            let sortedData = hourlyData.sorted { $0.date > $1.date }
            var maxCount = 1
            let dayColumns = sortedData.map { day in
                let counts = TrendHeatmapView.hours.map { hour -> Int in
                    let count = day.hourly[hour] ?? 0
                    maxCount = max(maxCount, count)
                    return count
                }
                return DayColumn(date: day.date, counts: counts)
            }

            self.dayColumns = dayColumns
            self.maxCount = maxCount
            self.axisLabels = TrendHeatmapView.makeAxisLabels(
                for: dayColumns.map(\.date),
                axisLabelStyle: axisLabelStyle
            )
        }
    }

    fileprivate struct DayColumn: Equatable {
        let date: String
        let counts: [Int]
    }
}

private struct TrendFixedScrollOffsetPreferenceKey: PreferenceKey {
    static var defaultValue: CGFloat = 0

    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = nextValue()
    }
}

private struct TrendHeatmapCanvas: View, Equatable {
    let dayColumns: [TrendHeatmapView.DayColumn]
    let cellWidth: CGFloat
    let cellHeight: CGFloat
    let cellSpacing: CGFloat
    let maxCount: Int

    var body: some View {
        Canvas(opaque: false, rendersAsynchronously: true) { context, _ in
            let cornerRadius = min(2, min(cellWidth, cellHeight) * 0.45)

            for (dayIndex, day) in dayColumns.enumerated() {
                let originX = CGFloat(dayIndex) * (cellWidth + cellSpacing)
                for (hourIndex, count) in day.counts.enumerated() {
                    let originY = CGFloat(hourIndex) * (cellHeight + cellSpacing)
                    let rect = CGRect(x: originX, y: originY, width: cellWidth, height: cellHeight)
                    if cornerRadius <= 0.5 {
                        context.fill(
                            Path(rect),
                            with: .color(TrendHeatmapView.color(for: count, maxCount: maxCount))
                        )
                    } else {
                        context.fill(
                            Path(roundedRect: rect, cornerRadius: cornerRadius),
                            with: .color(TrendHeatmapView.color(for: count, maxCount: maxCount))
                        )
                    }
                }
            }
        }
        .accessibilityLabel("聆听趋势")
    }
}

private struct AxisLabel: Identifiable, Equatable {
    let index: Int
    let text: String
    var id: Int { index }
}

struct TrendLegendView: View {
    let maxCount: Int

    var body: some View {
        HStack(spacing: 6) {
            Text("低")
                .font(.system(size: 11, weight: .medium))
                .foregroundStyle(SonicTheme.textSecondary)
                .frame(width: 12, alignment: .leading)

            HStack(spacing: 4) {
                ForEach(0..<5, id: \.self) { level in
                    RoundedRectangle(cornerRadius: 3, style: .continuous)
                        .fill(color(for: level))
                        .frame(width: 11, height: 11)
                }
            }

            Text(maxCount > 0 ? "高" : "暂无数据")
                .font(.system(size: 11, weight: .medium))
                .foregroundStyle(SonicTheme.textSecondary)
                .frame(width: 24, alignment: .leading)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 7)
        .background(
            Capsule()
                .fill(SonicTheme.dynamicColor(
                    light: .sonicWhite(1, alpha: 0.72),
                    dark: .sonicWhite(0.08, alpha: 0.72)
                ))
        )
        .overlay(
            Capsule()
                .stroke(SonicTheme.glassBorder, lineWidth: 1)
        )
    }

    private func color(for level: Int) -> Color {
        switch level {
        case 0: return TrendHeatmapView.emptyCell
        case 1: return TrendHeatmapView.levelOne
        case 2: return TrendHeatmapView.levelTwo
        case 3: return TrendHeatmapView.levelThree
        default: return TrendHeatmapView.levelFour
        }
    }
}

struct TrendMetricPill: View {
    let title: String
    let value: String
    var titleColor: Color = SonicTheme.textSecondary
    var valueColor: Color = SonicTheme.textPrimary

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
                .font(.system(size: 11, weight: .semibold, design: .rounded))
                .foregroundStyle(titleColor)
            Text(value)
                .font(.system(size: 14, weight: .bold))
                .foregroundStyle(valueColor)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 9)
        .background(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .fill(SonicTheme.dynamicColor(
                    light: .sonicRGBA(0.97, 0.98, 0.99, 0.92),
                    dark: .sonicRGBA(0.09, 0.12, 0.16, 0.88)
                ))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .stroke(TrendHeatmapView.levelTwo.opacity(0.18), lineWidth: 1)
        )
    }
}

struct TopArtistsCard: View {
    let topArtists: [TopArtist]

    var body: some View {
        GlassPanel(cornerRadius: 18, padding: 16) {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Image(systemName: "person.2")
                        .foregroundColor(SonicTheme.primary)
                    Text("热门艺术家")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundColor(SonicTheme.textPrimary)
                    Spacer()
                }

                ForEach(topArtists.prefix(6)) { artist in
                    HStack {
                        Text(artist.artist)
                            .font(.subheadline)
                            .foregroundColor(SonicTheme.textPrimary)
                            .lineLimit(1)
                        Spacer()
                        Text("\(artist.playCount ?? 0)")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    if artist.id != topArtists.prefix(6).last?.id { Divider() }
                }
            }
        }
    }
}

struct TopAlbumsCard: View {
    let topAlbums: [TopAlbum]
    let onAlbumTap: (Int64) -> Void

    var body: some View {
        GlassPanel(cornerRadius: 18, padding: 16) {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Image(systemName: "rectangle.stack")
                        .foregroundColor(SonicTheme.primary)
                    Text("热门专辑")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundColor(SonicTheme.textPrimary)
                    Spacer()
                }

                ForEach(topAlbums.prefix(6)) { album in
                    GlassPanel(cornerRadius: 12, padding: 10) {
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(album.album)
                                    .font(.subheadline)
                                    .foregroundColor(SonicTheme.textPrimary)
                                    .lineLimit(1)
                                    .help(album.album)
                                Text(album.artist)
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                    .lineLimit(1)
                                    .help(album.artist)
                            }
                            Spacer()
                            Text("\(album.playCount)")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                    .contentShape(Rectangle())
                    .onTapGesture {
                        onAlbumTap(album.albumID)
                    }
                }
            }
        }
    }
}

struct TopGenresCard: View {
    let topGenres: [TopGenre]

    var body: some View {
        GlassPanel(cornerRadius: 18, padding: 16) {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Image(systemName: "guitars")
                        .foregroundColor(SonicTheme.primary)
                    Text("热门流派")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundColor(SonicTheme.textPrimary)
                    Spacer()
                }

                ForEach(topGenres.prefix(6)) { genre in
                    let name = genre.genreNameZh.isEmpty ? genre.trackGenreName : genre.genreNameZh
                    HStack {
                        Text(name)
                            .font(.subheadline)
                            .foregroundColor(SonicTheme.textPrimary)
                        Spacer()
                        Text("\(genre.trackGenreCount)")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    if genre.id != topGenres.prefix(6).last?.id { Divider() }
                }
            }
        }
    }
}

struct RankingsCard: View {
    let topArtistsByPlays: [TopArtist]
    let topTracks: [Track]
    let onTrackTap: (Track) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("播放排行")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundColor(SonicTheme.textPrimary)
                Spacer()
            }

            VStack(spacing: 8) {
                ForEach(Array(topTracks.enumerated()), id: \.offset) { index, track in
                    GlassPanel(cornerRadius: 16, padding: 12) {
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(track.track)
                                    .font(.subheadline)
                                    .foregroundColor(SonicTheme.textPrimary)
                                    .lineLimit(1)
                                    .help(track.track)
                                Text("\(track.artist) · \(track.album)")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                    .lineLimit(1)
                                    .help("\(track.artist) · \(track.album)")
                            }
                            Spacer()
                            Text("\(track.playCount)")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                    .contentShape(Rectangle())
                    .onTapGesture {
                        onTrackTap(Track(
                            id: track.id,
                            artist: track.artist,
                            album: track.album,
                            track: track.track,
                            playCount: track.playCount,
                            trackNumber: track.trackNumber,
                            discNumber: track.discNumber,
                            duration: track.duration
                        ))
                    }
                }
            }
        }
    }
}

struct RecentPlaysSection: View {
    static let homeMinimumWidth: CGFloat = 340

    let items: [RecentPlayRecord]
    let artworkURLs: [Int64: String]
    let totalPlaysCount: Int64?
    let accentKey: HomeHotAccentKey?
    let onTrackTap: (Track) -> Void

    var body: some View {
        GlassPanel(cornerRadius: 20, padding: 18) {
            VStack(alignment: .leading, spacing: 14) {
                HotModuleSectionHeader(
                    title: "最近播放",
                    subtitle: "按时间倒序展示最近 10 次播放记录，优先突出封面和曲目信息。",
                    metricTag: totalPlaysCount.map { "总：\(compactCount($0))" },
                    accentKey: accentKey
                )

                if items.isEmpty {
                    HotModuleEmptyState(
                        title: "还没有最近播放",
                        message: "等播放记录同步进来，这里会变成一条更轻的播放流。"
                    )
                } else {
                    VStack(spacing: 0) {
                        ForEach(Array(items.prefix(10).enumerated()), id: \.element.id) { index, item in
                            recentPlayRow(
                                item: item,
                                rank: index + 1,
                                isLast: index == min(items.count, 10) - 1
                            )
                        }
                    }
                    .background(
                        RoundedRectangle(cornerRadius: 16, style: .continuous)
                            .fill(SonicTheme.dynamicColor(light: .sonicWhite(1, alpha: 0.55), dark: .sonicWhite(1, alpha: 0.04)))
                    )
                    .overlay(
                        RoundedRectangle(cornerRadius: 16, style: .continuous)
                            .stroke(SonicTheme.glassBorder, lineWidth: 1)
                    )
                }
            }
        }
    }

    @ViewBuilder
    private func recentPlayRow(item: RecentPlayRecord, rank: Int, isLast: Bool) -> some View {
        Button {
            onTrackTap(Track(
                id: 0,
                artist: item.artist,
                album: item.album,
                track: item.track,
                playCount: 0,
                trackNumber: nil,
                discNumber: nil,
                duration: nil
            ))
        } label: {
            HStack(alignment: .center, spacing: 14) {
                RecentPlayArtworkView(
                    item: item,
                    artworkURL: artworkURLs[item.id],
                    size: 42,
                    cornerRadius: 12
                )

                VStack(alignment: .leading, spacing: 4) {
                    Text(item.track)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(SonicTheme.textPrimary)
                        .lineLimit(1)
                        .help(item.track)

                    Text("\(item.artist) · \(item.album)")
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                        .lineLimit(1)
                        .help("\(item.artist) · \(item.album)")
                }

                Spacer(minLength: 12)

                Text(formatRecentPlayTime(item.playTime))
                    .font(.system(size: 12, weight: .semibold, design: .rounded))
                    .foregroundStyle(SonicTheme.textPrimary)
                    .lineLimit(1)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .background(
                        Capsule()
                            .fill(SonicTheme.dynamicColor(light: .sonicWhite(1, alpha: 0.58), dark: .sonicWhite(1, alpha: 0.06)))
                    )
                    .overlay(
                        Capsule()
                            .stroke(SonicTheme.glassBorder, lineWidth: 1)
                    )
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 12)
            .background(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .fill(rank == 1 ? SonicTheme.primary.opacity(0.06) : Color.clear)
            )
        }
        .buttonStyle(.plain)
        .contentShape(Rectangle())
        .overlay(alignment: .bottom) {
            if isLast == false {
                Rectangle()
                    .fill(SonicTheme.glassBorder)
                    .frame(height: 1)
                    .padding(.leading, 60)
            }
        }
    }

    private func formatRecentPlayTime(_ timeString: String) -> String {
        guard let date = Self.isoFormatter.date(from: timeString) else {
            return timeString
        }
        let interval = Date().timeIntervalSince(date)
        switch interval {
        case ..<60:
            return "刚刚"
        case ..<3600:
            return "\(max(1, Int(interval / 60)))m前"
        case ..<86400:
            return "\(max(1, Int(interval / 3600)))h前"
        case ..<604800:
            return "\(max(1, Int(interval / 86400)))d前"
        default:
            return Self.playTimeDisplayFormatter.string(from: date)
        }
    }

    private static let isoFormatter: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }()

    private static let playTimeDisplayFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateFormat = "MM/dd"
        return formatter
    }()
}

private struct RecentPlayArtworkView: View {
    let item: RecentPlayRecord
    let artworkURL: String?
    let size: CGFloat
    let cornerRadius: CGFloat

    var body: some View {
        ArtworkSquareView(
            artworkURL: artworkURL,
            fallbackTitle: item.track,
            size: size,
            cornerRadius: cornerRadius,
            style: .subtle
        )
    }
}

private func compactCount(_ value: Int64) -> String {
    switch value {
    case 1_000_000...:
        return String(format: "%.1fm", Double(value) / 1_000_000)
    case 1_000...:
        return String(format: "%.1fk", Double(value) / 1_000)
    default:
        return "\(value)"
    }
}

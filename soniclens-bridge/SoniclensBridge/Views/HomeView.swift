import SwiftUI
#if os(macOS)
import AppKit
#endif

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
    @EnvironmentObject private var navigationCoordinator: NavigationCoordinator
    @ObservedObject var viewModel: HomeViewModel
    @Binding var selectedRecentTrack: Track?
    @Binding var selectedAlbumID: Int64?
    @Binding var selectedGenreItem: HomeHotGenrePresentationItem?
    @Binding var rankingSearchText: String
    @State private var layoutModes = HomeLayoutModes()
    @State private var activeListeningProfileDetailMode: ListeningProfileDetailMode? = nil

    private static let homeAmbientOrbs: [AmbientOrb] = [
        AmbientOrb(color: SonicTheme.primary.opacity(0.35), size: 520, blur: 120, opacity: 0.7, offsetFrom: CGSize(width: -200, height: -260), offsetTo: CGSize(width: -120, height: -180), duration: 18),
        AmbientOrb(color: SonicTheme.primary.opacity(0.25), size: 420, blur: 130, opacity: 0.7, offsetFrom: CGSize(width: 260, height: 120), offsetTo: CGSize(width: 200, height: 220), duration: 22),
        AmbientOrb(color: SonicTheme.primary.opacity(0.18), size: 360, blur: 140, opacity: 0.6, offsetFrom: CGSize(width: 120, height: -40), offsetTo: CGSize(width: 80, height: 40), duration: 26)
    ]

    var body: some View {
        ZStack {
            AmbientBackgroundView(
                gradient: LinearGradient(
                    colors: [SonicTheme.background, SonicTheme.background],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                ),
                orbs: Self.homeAmbientOrbs,
                renderingStyle: .staticHome
            )

            ScrollView {
                LazyVStack(alignment: .leading, spacing: 16) {
                    if let message = viewModel.errorMessage {
                        ErrorBanner(message: message)
                    }

                    let hotPresentation = viewModel.hotModulePresentation
                    let trendSnapshot = viewModel.trendSnapshot
                    let selectedAccent = hotPresentation.primaryAccentKey

                    HomeHeroSection()
                        .padding(.bottom, 12)

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
                        .equatable()
                        .frame(maxWidth: .infinity, alignment: .topLeading)
                    } else {
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
                        .accessibilityIdentifier(UIAccessibilityID.homeProfileCard)
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
                                    navigationCoordinator.push(.albumDetail(albumID: albumID))
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
                                onTrackTap: { navigationCoordinator.push(.trackDetail(track: $0.bridgeTrack)) }
                            )
                            .frame(minWidth: TrackShelfCard.homeMinimumWidth, maxWidth: .infinity, alignment: .topLeading)

                            RecentPlaysSection(
                                items: viewModel.recentPlays,
                                artworkURLs: viewModel.recentPlayArtworkURLs,
                                totalPlaysCount: hotPresentation.totalPlaysCount,
                                accentKey: selectedAccent,
                                onTrackTap: { track in
                                    navigationCoordinator.push(.trackDetail(track: track))
                                }
                            )
                            .equatable()
                            .frame(minWidth: RecentPlaysSection.homeMinimumWidth, maxWidth: .infinity, alignment: .topLeading)
                        }
                    } else {
                        AlbumShelfCard(
                            items: Array(hotPresentation.albums.prefix(5)),
                            artworkBaseURL: store.currentServer?.artworkBaseURL,
                            style: .featureGrid,
                            collectionCount: hotPresentation.totalAlbumsCount,
                            accentKey: selectedAccent,
                            onAlbumTap: { albumID in
                                navigationCoordinator.push(.albumDetail(albumID: albumID))
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
                            onTrackTap: { navigationCoordinator.push(.trackDetail(track: $0.bridgeTrack)) }
                        )

                        RecentPlaysSection(
                            items: viewModel.recentPlays,
                            artworkURLs: viewModel.recentPlayArtworkURLs,
                            totalPlaysCount: hotPresentation.totalPlaysCount,
                            accentKey: selectedAccent,
                            onTrackTap: { track in
                                navigationCoordinator.push(.trackDetail(track: track))
                            }
                        )
                        .equatable()
                    }
                }
                .padding(.horizontal, 32)
                .padding(.top, 28)
                .padding(.bottom, 108)
            }
            .accessibilityIdentifier(UIAccessibilityID.homeScrollView)
        }
        .measureDebouncedWidth(delayMs: 120, threshold: 4) { width, isInitial in
            var next = layoutModes
            next.update(for: max(width - 64, 0), force: isInitial)
            if next != layoutModes {
                layoutModes = next
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
        .sheet(item: $selectedGenreItem) { genreItem in
            GenreAlbumsSheet(
                item: genreItem,
                onSelectAlbum: { albumID in
                    navigationCoordinator.push(.albumDetail(albumID: albumID))
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

private struct HomeLayoutModes: Equatable {
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
        .accessibilityIdentifier(UIAccessibilityID.homeHero)
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
    var previewHoveredCell: (dayIndex: Int, hourIndex: Int)? = nil

    static func == (lhs: DashboardTrendSection, rhs: DashboardTrendSection) -> Bool {
        lhs.points == rhs.points &&
        lhs.hourlyData == rhs.hourlyData &&
        lhs.title == rhs.title &&
        lhs.subtitle == rhs.subtitle &&
        lhs.heatmapHeight == rhs.heatmapHeight &&
        lhs.heatmapLayout == rhs.heatmapLayout &&
        lhs.axisLabelStyle == rhs.axisLabelStyle &&
        lhs.actionPlacement == rhs.actionPlacement &&
        lhs.actionTitle == rhs.actionTitle &&
        lhs.previewHoveredCell?.dayIndex == rhs.previewHoveredCell?.dayIndex &&
        lhs.previewHoveredCell?.hourIndex == rhs.previewHoveredCell?.hourIndex
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
                onAction: onAction,
                previewHoveredCell: previewHoveredCell
            )
        }
        .accessibilityIdentifier(UIAccessibilityID.homeTrendSection)
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
    let previewHoveredCell: TrendHeatmapHoveredCell?
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
        onAction: (() -> Void)?,
        previewHoveredCell: (dayIndex: Int, hourIndex: Int)? = nil
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
        self.previewHoveredCell = previewHoveredCell.map {
            TrendHeatmapHoveredCell(dayIndex: $0.dayIndex, hourIndex: $0.hourIndex)
        }
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
        lhs.actionTitle == rhs.actionTitle &&
        lhs.previewHoveredCell == rhs.previewHoveredCell
    }

    var body: some View {
        let resolvedHeatmapHeight = TrendHeatmapView.preferredHeight(
            fallbackHeight: heatmapHeight,
            layout: heatmapLayout,
            axisLabelStyle: axisLabelStyle
        )

        VStack(alignment: .leading, spacing: 18) {
            VStack(alignment: .leading, spacing: 14) {
                headerContent

                if !points.isEmpty {
                    metricsContent
                }
            }
            .padding(.horizontal, 18)
            .padding(.top, 18)

            TrendHeatmapView(
                renderModel: heatmapRenderModel,
                layout: heatmapLayout,
                axisLabelStyle: axisLabelStyle,
                previewHoveredCell: previewHoveredCell
            )
            .equatable()
            .accessibilityIdentifier(UIAccessibilityID.homeTrendHeatmap)
            .frame(height: resolvedHeatmapHeight)
            .padding(.horizontal, 18)
            .padding(.bottom, 18)
        }
        .padding(.bottom, 6)
    }

    private var headerContent: some View {
        #if os(macOS)
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
        #else
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
        #endif
    }

    private var metricsContent: some View {
        #if os(macOS)
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
        #else
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
        #endif
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
                .fixedSize(horizontal: false, vertical: true)
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

private struct WideListeningInsightsSection: View, Equatable {
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

    static func == (lhs: WideListeningInsightsSection, rhs: WideListeningInsightsSection) -> Bool {
        lhs.points == rhs.points &&
        lhs.hourlyData == rhs.hourlyData &&
        lhs.trendSubtitle == rhs.trendSubtitle &&
        lhs.summaryText == rhs.summaryText &&
        lhs.footnoteText == rhs.footnoteText &&
        lhs.genres == rhs.genres &&
        lhs.sources == rhs.sources &&
        lhs.accentKey == rhs.accentKey
    }

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
                .equatable()
                .accessibilityIdentifier(UIAccessibilityID.homeTrendSection)
                .frame(minWidth: 720, maxWidth: .infinity, alignment: .topLeading)
            }
        }
        .accessibilityIdentifier(UIAccessibilityID.homeProfileCard)
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
            return 38
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
            return 6
        case .dayStride:
            return 2
        }
    }

    var trailingAxisOverflow: CGFloat {
        switch self {
        case .monthBoundaries:
            return 0
        case .dayStride:
            return 16
        }
    }

    var trailingLabelInset: CGFloat {
        switch self {
        case .monthBoundaries:
            return 54
        case .dayStride:
            return 16
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
            return 20
        }
    }

    var yAxisTextAlignment: Alignment {
        .trailing
    }
}

private enum FixedScrollHintState: Equatable {
    case none
    case canScrollOlder
    case canReturnRecent
}

@MainActor
private final class TrendScrollOffsetTracker {
    var lastOffsetX: CGFloat = 0
}

struct TrendHeatmapView: View, Equatable {
    var layout: TrendHeatmapLayout = .fitted(minCellWidth: 2)
    var axisLabelStyle: TrendAxisLabelStyle = .monthBoundaries
    private let previewHoveredCell: TrendHeatmapHoveredCell?
    @State private var scrollHintState: FixedScrollHintState = .none
    @State private var fixedViewportWidth: CGFloat = 0
    @State private var scrollTracker = TrendScrollOffsetTracker()

    private let model: RenderModel
    private static let cellSpacingValue: CGFloat = 2
    private static let gridInsetValue: CGFloat = 10
    private static let fixedScrollHintBarHeight: CGFloat = 18
    private static let labeledHours: [(rowIndex: Int, text: String)] = [
        (0, "23"),
        (5, "18"),
        (11, "12"),
        (17, "06"),
        (23, "00")
    ]

    private var cellSpacing: CGFloat { Self.cellSpacingValue }
    private var gridInset: CGFloat { Self.gridInsetValue }

    init(
        hourlyData: [HourlyData],
        layout: TrendHeatmapLayout = .fitted(minCellWidth: 2),
        axisLabelStyle: TrendAxisLabelStyle = .monthBoundaries,
        previewHoveredCell: (dayIndex: Int, hourIndex: Int)? = nil
    ) {
        self.layout = layout
        self.axisLabelStyle = axisLabelStyle
        self.previewHoveredCell = previewHoveredCell.map {
            TrendHeatmapHoveredCell(dayIndex: $0.dayIndex, hourIndex: $0.hourIndex)
        }
        self.model = RenderModel(hourlyData: hourlyData, axisLabelStyle: axisLabelStyle)
    }

    fileprivate init(
        renderModel: RenderModel,
        layout: TrendHeatmapLayout = .fitted(minCellWidth: 2),
        axisLabelStyle: TrendAxisLabelStyle = .monthBoundaries,
        previewHoveredCell: TrendHeatmapHoveredCell? = nil
    ) {
        self.layout = layout
        self.axisLabelStyle = axisLabelStyle
        self.previewHoveredCell = previewHoveredCell
        self.model = renderModel
    }

    static func preferredHeight(
        fallbackHeight: CGFloat,
        layout: TrendHeatmapLayout,
        axisLabelStyle: TrendAxisLabelStyle
    ) -> CGFloat {
        switch layout {
        case let .fixedWidthScrollable(cellWidth):
            let cellSize = max(cellWidth, 2)
            let gridHeight = CGFloat(hours.count) * cellSize + CGFloat(hours.count - 1) * cellSpacingValue
            return axisLabelStyle.axisHeight
                + axisLabelStyle.axisToGridSpacing
                + gridInsetValue * 2
                + gridHeight
                + fixedScrollHintBarHeight
        case .fitted, .scrollable:
            return fallbackHeight
        }
    }

    private var xAxisHeight: CGFloat {
        axisLabelStyle.axisHeight
    }

    fileprivate static let hours = Array(stride(from: 23, through: 0, by: -1))

    static func == (lhs: TrendHeatmapView, rhs: TrendHeatmapView) -> Bool {
        lhs.layout == rhs.layout &&
        lhs.axisLabelStyle == rhs.axisLabelStyle &&
        lhs.previewHoveredCell == rhs.previewHoveredCell &&
        lhs.model == rhs.model
    }

    var body: some View {
        if model.dayColumns.isEmpty {
            emptyStateView
        } else {
            heatmapBody
        }
    }

    private var emptyStateView: some View {
        RoundedRectangle(cornerRadius: 14, style: .continuous)
            .fill(Self.gridBackground)
            .overlay(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .stroke(Self.gridBorder, lineWidth: 1)
            )
            .overlay {
                VStack(spacing: 8) {
                    Image(systemName: "chart.bar.xaxis")
                        .font(.system(size: 20, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                    Text("暂无播放趋势数据")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                }
                .padding(.vertical, 24)
            }
            .frame(maxWidth: .infinity)
            .frame(height: 220)
    }

    @ViewBuilder
    private var heatmapBody: some View {
        switch layout {
        case let .fixedWidthScrollable(cellWidth):
            let resolvedCellWidth = max(cellWidth, 2)
            let resolvedCellHeight = resolvedCellWidth
            let dayCount = max(model.dayColumns.count, 1)
            let contentWidth = CGFloat(dayCount) * resolvedCellWidth + CGFloat(dayCount - 1) * cellSpacing
            let totalTrackWidth = contentWidth + gridInset * 2 + axisLabelStyle.trailingAxisOverflow
            let availableViewportWidth = max(fixedViewportWidth - axisLabelStyle.yAxisColumnWidth, 0)
            let maxScrollDistance = max(totalTrackWidth - availableViewportWidth, 0)
            let tracker = scrollTracker
            let inset = gridInset

            VStack(alignment: .leading, spacing: 0) {
                HStack(alignment: .top, spacing: 0) {
                    hourLabelsColumn(cellHeight: resolvedCellHeight)

                    ScrollView(.horizontal, showsIndicators: false) {
                        heatmapColumns(
                            cellWidth: resolvedCellWidth,
                            cellHeight: resolvedCellHeight,
                            contentWidth: contentWidth,
                            visibleXRange: {
                                guard availableViewportWidth > 0 else { return 0...contentWidth }
                                let visibleMin = max(0, -tracker.lastOffsetX - inset)
                                let visibleMax = min(contentWidth, visibleMin + max(availableViewportWidth - inset * 2, 0))
                                return visibleMin <= visibleMax ? visibleMin...visibleMax : 0...contentWidth
                            }
                        )
                        .padding(.trailing, axisLabelStyle.trailingAxisOverflow)
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
                        guard abs(value - scrollTracker.lastOffsetX) >= 0.5 else { return }
                        scrollTracker.lastOffsetX = value
                        updateScrollHintState(offsetX: value, maxScrollDistance: maxScrollDistance)
                    }
                }

                fixedScrollHintBar(maxScrollDistance: maxScrollDistance)
            }
            .measureDebouncedWidth(delayMs: 120, threshold: 4) { width, _ in
                if abs(width - fixedViewportWidth) >= 1 {
                    fixedViewportWidth = width
                }
            }
            .onChange(of: maxScrollDistance) { _, newMax in
                updateScrollHintState(offsetX: scrollTracker.lastOffsetX, maxScrollDistance: newMax)
            }
            .onAppear {
                updateScrollHintState(offsetX: scrollTracker.lastOffsetX, maxScrollDistance: maxScrollDistance)
            }

        case let .fitted(minCellWidth):
            GeometryReader { geometry in
                let rowCount = CGFloat(Self.hours.count)
                let availableHeight = max(geometry.size.height - xAxisHeight, 0)
                let gridHeight = max(availableHeight - axisLabelStyle.axisToGridSpacing - gridInset * 2, 0)
                let resolvedCellHeight = max(
                    4,
                    (gridHeight - (rowCount - 1) * cellSpacing) / rowCount
                )
                let availableWidth = max(geometry.size.width - axisLabelStyle.yAxisColumnWidth, 0)
                let dayCount = max(model.dayColumns.count, 1)
                let gridWidth = max(availableWidth - gridInset * 2 - axisLabelStyle.trailingAxisOverflow, 0)
                let cellWidth = max(
                    minCellWidth,
                    (gridWidth - CGFloat(dayCount - 1) * cellSpacing) / CGFloat(dayCount)
                )
                let contentWidth = CGFloat(dayCount) * cellWidth + CGFloat(dayCount - 1) * cellSpacing

                HStack(alignment: .top, spacing: 0) {
                    hourLabelsColumn(cellHeight: resolvedCellHeight)
                    heatmapColumns(
                        cellWidth: cellWidth,
                        cellHeight: resolvedCellHeight,
                        contentWidth: contentWidth,
                        visibleXRange: { 0...contentWidth }
                    )
                    .padding(.trailing, axisLabelStyle.trailingAxisOverflow)
                }
            }

        case let .scrollable(minCellWidth):
            GeometryReader { geometry in
                let rowCount = CGFloat(Self.hours.count)
                let availableHeight = max(geometry.size.height - xAxisHeight, 0)
                let gridHeight = max(availableHeight - axisLabelStyle.axisToGridSpacing - gridInset * 2, 0)
                let resolvedCellHeight = max(
                    4,
                    (gridHeight - (rowCount - 1) * cellSpacing) / rowCount
                )
                let dayCount = max(model.dayColumns.count, 1)
                let cellWidth = max(minCellWidth, 2)
                let contentWidth = CGFloat(dayCount) * cellWidth + CGFloat(dayCount - 1) * cellSpacing

                HStack(alignment: .top, spacing: 0) {
                    hourLabelsColumn(cellHeight: resolvedCellHeight)
                    ScrollView(.horizontal, showsIndicators: false) {
                        heatmapColumns(
                            cellWidth: cellWidth,
                            cellHeight: resolvedCellHeight,
                            contentWidth: contentWidth,
                            visibleXRange: { 0...contentWidth }
                        )
                        .padding(.trailing, axisLabelStyle.trailingAxisOverflow)
                    }
                }
            }
        }
    }

    private func hourLabelsColumn(cellHeight: CGFloat) -> some View {
        let totalGridHeight = CGFloat(Self.hours.count) * cellHeight + CGFloat(Self.hours.count - 1) * cellSpacing

        return ZStack(alignment: .topLeading) {
            ForEach(Self.labeledHours, id: \.rowIndex) { item in
                let centerY = CGFloat(item.rowIndex) * (cellHeight + cellSpacing) + cellHeight / 2
                Text(item.text)
                    .font(.system(size: 9, weight: .medium, design: .monospaced))
                    .foregroundStyle(SonicTheme.textSecondary)
                    .frame(
                        width: axisLabelStyle.yAxisTextWidth,
                        height: max(cellHeight, 11),
                        alignment: axisLabelStyle.yAxisTextAlignment
                    )
                    .position(x: axisLabelStyle.yAxisTextWidth / 2, y: centerY)
            }
        }
        .frame(width: axisLabelStyle.yAxisColumnWidth, height: totalGridHeight, alignment: .topLeading)
        .padding(.top, xAxisHeight + axisLabelStyle.axisToGridSpacing + gridInset)
        .padding(.bottom, gridInset)
    }

    private func heatmapColumns(
        cellWidth: CGFloat,
        cellHeight: CGFloat,
        contentWidth: CGFloat,
        dayColumns: [DayColumn]? = nil,
        axisLabels: [AxisLabel]? = nil,
        visibleXRange: @escaping () -> ClosedRange<CGFloat>
    ) -> some View {
        let resolvedDayColumns = dayColumns ?? model.dayColumns
        let resolvedAxisLabels = axisLabels ?? model.axisLabels

        return VStack(alignment: .leading, spacing: axisLabelStyle.axisToGridSpacing) {
            xAxisLabels(contentWidth: contentWidth, cellWidth: cellWidth, axisLabels: resolvedAxisLabels)
            dayColumnsView(
                cellWidth: cellWidth,
                cellHeight: cellHeight,
                contentWidth: contentWidth,
                dayColumns: resolvedDayColumns,
                visibleXRange: visibleXRange
            )
        }
    }

    private func xAxisLabels(contentWidth: CGFloat, cellWidth: CGFloat, axisLabels: [AxisLabel]) -> some View {
        ZStack(alignment: .topLeading) {
            ForEach(axisLabels) { item in
                let centerX = labelX(at: item.index, cellWidth: cellWidth)

                Rectangle()
                    .fill(SonicTheme.textSecondary.opacity(0.35))
                    .frame(width: 1, height: 3)
                    .position(x: centerX, y: xAxisHeight - 1.5)

                switch axisLabelStyle {
                case .monthBoundaries:
                    let columnLeadingX = CGFloat(item.index) * (cellWidth + cellSpacing)
                    Text(item.text)
                        .font(.system(size: 10, weight: .medium, design: .monospaced))
                        .foregroundStyle(SonicTheme.textSecondary)
                        .fixedSize()
                        .offset(
                            x: min(columnLeadingX, max(contentWidth - axisLabelStyle.trailingLabelInset, 0)),
                            y: axisLabelStyle.labelYOffset
                        )
                case let .dayStride(_, rotationDegrees):
                    Color.clear
                        .frame(width: 1, height: 1)
                        .overlay(alignment: .bottomLeading) {
                            Text(item.text)
                                .font(.system(size: 10, weight: .medium, design: .monospaced))
                                .foregroundStyle(SonicTheme.textSecondary)
                                .fixedSize()
                                .rotationEffect(.degrees(-abs(rotationDegrees)), anchor: .bottomLeading)
                        }
                        .position(x: centerX, y: xAxisHeight - 5)
                }
            }
        }
        .frame(width: contentWidth, height: xAxisHeight, alignment: .topLeading)
        .padding(.horizontal, gridInset)
    }

    private func dayColumnsView(
        cellWidth: CGFloat,
        cellHeight: CGFloat,
        contentWidth: CGFloat,
        dayColumns: [DayColumn],
        visibleXRange: @escaping () -> ClosedRange<CGFloat>
    ) -> some View {
        TrendHeatmapInteractiveGrid(
            dayColumns: dayColumns,
            cellWidth: cellWidth,
            cellHeight: cellHeight,
            cellSpacing: cellSpacing,
            maxCount: model.maxCount,
            contentWidth: contentWidth,
            previewHoveredCell: previewHoveredCell,
            visibleXRange: visibleXRange
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

    private func fixedScrollHintBar(maxScrollDistance: CGFloat) -> some View {
        HStack(spacing: 4) {
            Spacer(minLength: 0)
            switch scrollHintState {
            case .none:
                EmptyView()
            case .canScrollOlder:
                HStack(spacing: 4) {
                    Text("可向右查看更早日期")
                    Image(systemName: "arrow.right")
                        .font(.system(size: 9, weight: .semibold))
                }
            case .canReturnRecent:
                HStack(spacing: 4) {
                    Image(systemName: "arrow.left")
                        .font(.system(size: 9, weight: .semibold))
                    Text("可向左返回最近日期")
                }
            }
        }
        .font(.system(size: 10, weight: .semibold))
        .foregroundStyle(SonicTheme.textSecondary.opacity(0.85))
        .padding(.trailing, axisLabelStyle.trailingAxisOverflow)
        .frame(height: Self.fixedScrollHintBarHeight, alignment: .trailing)
        .opacity(maxScrollDistance > 24 && scrollHintState != .none ? 1 : 0)
        .accessibilityIdentifier(UIAccessibilityID.homeTrendScrollHint)
    }

    private func updateScrollHintState(offsetX: CGFloat, maxScrollDistance: CGFloat) {
        let nextState = resolvedScrollHintState(offsetX: offsetX, maxScrollDistance: maxScrollDistance)
        if nextState != scrollHintState {
            scrollHintState = nextState
        }
    }

    private func resolvedScrollHintState(offsetX: CGFloat, maxScrollDistance: CGFloat) -> FixedScrollHintState {
        guard maxScrollDistance > 24 else { return .none }
        let offset = max(0, -offsetX)
        if offset <= 12 {
            return .canScrollOlder
        }
        if offset >= maxScrollDistance - 12 {
            return .canReturnRecent
        }
        return .none
    }

    private func labelX(at index: Int, cellWidth: CGFloat) -> CGFloat {
        CGFloat(index) * (cellWidth + cellSpacing) + cellWidth / 2
    }

    static func compactDateLabel(for dateString: String) -> String {
        guard let date = dayFormatter.date(from: dateString) else { return dateString }
        return shortDayFormatter.string(from: date)
    }

    static func tooltipDateLabel(for dateString: String) -> String {
        guard let date = dayFormatter.date(from: dateString) else { return dateString }
        return tooltipDayFormatter.string(from: date)
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

    fileprivate static func colorLevel(for count: Int, maxCount: Int) -> Int {
        if count <= 0 { return 0 }
        let ratio = Double(count) / Double(max(maxCount, 1))
        switch ratio {
        case 0..<0.25: return 1
        case 0.25..<0.5: return 2
        case 0.5..<0.75: return 3
        default: return 4
        }
    }

    fileprivate static func color(forLevel level: Int) -> Color {
        switch level {
        case 0: return emptyCell
        case 1: return levelOne
        case 2: return levelTwo
        case 3: return levelThree
        default: return levelFour
        }
    }

    fileprivate static func color(for count: Int, maxCount: Int) -> Color {
        color(forLevel: colorLevel(for: count, maxCount: maxCount))
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

    private static let tooltipDayFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "zh_CN")
        formatter.dateFormat = "M/d EEE"
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
                var totalCount = 0
                let counts = TrendHeatmapView.hours.map { hour -> Int in
                    let count = day.hourly[hour] ?? 0
                    maxCount = max(maxCount, count)
                    totalCount += count
                    return count
                }
                return DayColumn(
                    date: day.date,
                    tooltipDateLabel: TrendHeatmapView.tooltipDateLabel(for: day.date),
                    counts: counts,
                    totalCount: totalCount
                )
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
        let tooltipDateLabel: String
        let counts: [Int]
        let totalCount: Int
    }
}

private struct TrendFixedScrollOffsetPreferenceKey: PreferenceKey {
    static var defaultValue: CGFloat = 0

    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = nextValue()
    }
}

private struct TrendHeatmapHoveredCell: Equatable {
    let dayIndex: Int
    let hourIndex: Int
}

#if os(macOS)
@MainActor
private final class HeatmapScrollHoverGate {
    private(set) var isScrolling = false
    nonisolated(unsafe) private var monitor: Any?
    private var resetTask: Task<Void, Never>?

    func start(onScrollStart: @escaping @MainActor () -> Void) {
        guard monitor == nil else { return }
        monitor = NSEvent.addLocalMonitorForEvents(matching: .scrollWheel) { [weak self] event in
            if Thread.isMainThread {
                MainActor.assumeIsolated {
                    self?.handleScroll(onScrollStart: onScrollStart)
                }
            } else {
                Task { @MainActor [weak self] in
                    self?.handleScroll(onScrollStart: onScrollStart)
                }
            }
            return event
        }
    }

    func stop() {
        resetTask?.cancel()
        resetTask = nil
        if let monitor {
            NSEvent.removeMonitor(monitor)
            self.monitor = nil
        }
        isScrolling = false
    }

    private func handleScroll(onScrollStart: @escaping @MainActor () -> Void) {
        if !isScrolling {
            isScrolling = true
            onScrollStart()
        }
        resetTask?.cancel()
        resetTask = Task { @MainActor [weak self] in
            do {
                try await Task.sleep(nanoseconds: 180_000_000)
            } catch {
                return
            }
            guard !Task.isCancelled else { return }
            self?.isScrolling = false
        }
    }

    deinit {
        if let monitor {
            NSEvent.removeMonitor(monitor)
        }
    }
}
#endif

private struct TrendHeatmapInteractiveGrid: View {
    let dayColumns: [TrendHeatmapView.DayColumn]
    let cellWidth: CGFloat
    let cellHeight: CGFloat
    let cellSpacing: CGFloat
    let maxCount: Int
    let contentWidth: CGFloat
    let previewHoveredCell: TrendHeatmapHoveredCell?
    let visibleXRange: () -> ClosedRange<CGFloat>

    @State private var hoveredCell: TrendHeatmapHoveredCell?
    #if os(macOS)
    @State private var hoverGate = HeatmapScrollHoverGate()
    #endif

    private var gridHeight: CGFloat {
        CGFloat(TrendHeatmapView.hours.count) * cellHeight
            + CGFloat(max(TrendHeatmapView.hours.count - 1, 0)) * cellSpacing
    }

    var body: some View {
        let activeHoveredCell = previewHoveredCell ?? hoveredCell

        TrendHeatmapCanvas(
            dayColumns: dayColumns,
            cellWidth: cellWidth,
            cellHeight: cellHeight,
            cellSpacing: cellSpacing,
            maxCount: maxCount
        )
        .equatable()
        .accessibilityIdentifier(UIAccessibilityID.homeTrendCanvas)
        .frame(width: contentWidth, height: gridHeight, alignment: .leading)
        .contentShape(Rectangle())
        .onContinuousHover(coordinateSpace: .local) { phase in
            switch phase {
            case .active(let location):
                #if os(macOS)
                guard !hoverGate.isScrolling else { return }
                #endif
                let strideX = cellWidth + cellSpacing
                let strideY = cellHeight + cellSpacing
                guard !dayColumns.isEmpty,
                      strideX > 0, strideY > 0,
                      location.x >= 0, location.x <= contentWidth,
                      location.y >= 0, location.y <= gridHeight else {
                    if hoveredCell != nil {
                        hoveredCell = nil
                    }
                    return
                }
                let dayIndex = min(max(Int(location.x / strideX), 0), dayColumns.count - 1)
                let hourIndex = min(max(Int(location.y / strideY), 0), TrendHeatmapView.hours.count - 1)
                let next = TrendHeatmapHoveredCell(dayIndex: dayIndex, hourIndex: hourIndex)
                if next != hoveredCell {
                    hoveredCell = next
                }
            case .ended:
                if hoveredCell != nil {
                    hoveredCell = nil
                }
            }
        }
        #if os(macOS)
        .onAppear {
            hoverGate.start {
                if hoveredCell != nil {
                    hoveredCell = nil
                }
            }
        }
        .onDisappear {
            hoverGate.stop()
        }
        #endif
        .onChange(of: dayColumns.count) { _, newCount in
            if let current = hoveredCell, current.dayIndex >= newCount {
                hoveredCell = nil
            }
        }
        .overlay(alignment: .topLeading) {
            if let activeHoveredCell,
               dayColumns.indices.contains(activeHoveredCell.dayIndex),
               TrendHeatmapView.hours.indices.contains(activeHoveredCell.hourIndex),
               dayColumns[activeHoveredCell.dayIndex].counts.indices.contains(activeHoveredCell.hourIndex) {
                hoverOverlay(for: activeHoveredCell)
                    .allowsHitTesting(false)
            }
        }
    }

    private func hoverOverlay(for cell: TrendHeatmapHoveredCell) -> some View {
        let day = dayColumns[cell.dayIndex]
        let hour = TrendHeatmapView.hours[cell.hourIndex]
        let count = day.counts[cell.hourIndex]
        let cellOriginX = CGFloat(cell.dayIndex) * (cellWidth + cellSpacing)
        let cellOriginY = CGFloat(cell.hourIndex) * (cellHeight + cellSpacing)
        let cellCenterX = cellOriginX + cellWidth / 2
        let cellCenterY = cellOriginY + cellHeight / 2

        let estimatedHalfWidth: CGFloat = 86
        let estimatedHeight: CGFloat = 44
        let range = visibleXRange()
        let minCenterX = range.lowerBound + estimatedHalfWidth + 6
        let maxCenterX = range.upperBound - estimatedHalfWidth - 6
        let tooltipX: CGFloat = if minCenterX <= maxCenterX {
            min(max(cellCenterX, minCenterX), maxCenterX)
        } else {
            (range.lowerBound + range.upperBound) / 2
        }

        let placesBelow = cellOriginY < estimatedHeight + 14
        let tooltipY = placesBelow
            ? cellOriginY + cellHeight + 10 + estimatedHeight / 2
            : cellOriginY - 10 - estimatedHeight / 2

        return ZStack(alignment: .topLeading) {
            // 极淡十字引导线，帮助将格子映射到横纵轴刻度
            Rectangle()
                .fill(TrendHeatmapView.levelThree.opacity(0.10))
                .frame(width: cellWidth, height: gridHeight)
                .position(x: cellCenterX, y: gridHeight / 2)

            Rectangle()
                .fill(TrendHeatmapView.levelThree.opacity(0.10))
                .frame(width: contentWidth, height: cellHeight)
                .position(x: contentWidth / 2, y: cellCenterY)

            RoundedRectangle(
                cornerRadius: min(2.5, min(cellWidth, cellHeight) * 0.45 + 0.5),
                style: .continuous
            )
            .stroke(Self.highlightBorderColor, lineWidth: 1.5)
            .frame(width: cellWidth + 2, height: cellHeight + 2)
            .position(x: cellCenterX, y: cellCenterY)

            tooltipCard(day: day, hour: hour, count: count)
                .fixedSize()
                .accessibilityIdentifier(UIAccessibilityID.homeTrendTooltip)
                .position(x: tooltipX, y: tooltipY)
        }
        .frame(width: contentWidth, height: gridHeight, alignment: .topLeading)
    }

    private func tooltipCard(day: TrendHeatmapView.DayColumn, hour: Int, count: Int) -> some View {
        let shape = RoundedRectangle(cornerRadius: 8, style: .continuous)

        return VStack(alignment: .leading, spacing: 3) {
            Text("\(day.tooltipDateLabel) · \(String(format: "%02d:00–%02d:00", hour, hour + 1))")
                .font(.system(size: 10, weight: .medium, design: .monospaced))
                .foregroundStyle(SonicTheme.textSecondary)

            HStack(spacing: 6) {
                RoundedRectangle(cornerRadius: 2, style: .continuous)
                    .fill(TrendHeatmapView.color(for: count, maxCount: maxCount))
                    .frame(width: 7, height: 7)
                    .overlay(
                        RoundedRectangle(cornerRadius: 2, style: .continuous)
                            .stroke(Self.swatchBorderColor, lineWidth: 0.5)
                    )

                if count > 0 {
                    Text("\(count) 次播放")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(SonicTheme.textPrimary)
                } else {
                    Text("无播放记录")
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                }

                if day.totalCount > 0 {
                    Text("· 当日共 \(day.totalCount) 次")
                        .font(.system(size: 10, weight: .medium))
                        .foregroundStyle(SonicTheme.textSecondary)
                }
            }
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 7)
        .background {
            shape
                .fill(Self.tooltipBackgroundColor)
                .shadow(color: Color.black.opacity(0.16), radius: 8, x: 0, y: 4)
        }
        .overlay {
            shape
                .stroke(Self.tooltipBorderColor, lineWidth: 1)
        }
    }

    private static let highlightBorderColor = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.58, 0.36, 0.08, 0.95),
        dark: .sonicRGBA(1.0, 0.88, 0.62, 0.95)
    )

    private static let swatchBorderColor = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.60, 0.42, 0.16, 0.25),
        dark: .sonicRGBA(1.0, 0.90, 0.70, 0.25)
    )

    private static let tooltipBackgroundColor = SonicTheme.dynamicColor(
        light: .sonicRGBA(1.0, 0.99, 0.96, 0.97),
        dark: .sonicRGBA(0.14, 0.12, 0.10, 0.96)
    )

    private static let tooltipBorderColor = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.76, 0.56, 0.22, 0.22),
        dark: .sonicRGBA(0.88, 0.70, 0.34, 0.26)
    )
}

private struct TrendHeatmapCanvas: View, Equatable {
    let dayColumns: [TrendHeatmapView.DayColumn]
    let cellWidth: CGFloat
    let cellHeight: CGFloat
    let cellSpacing: CGFloat
    let maxCount: Int

    var body: some View {
        Canvas(opaque: false, rendersAsynchronously: false) { context, _ in
            let cornerRadius = min(2, min(cellWidth, cellHeight) * 0.45)
            let useRounded = cornerRadius > 0.5
            let cornerSize = CGSize(width: cornerRadius, height: cornerRadius)
            var bucketPaths = Array(repeating: Path(), count: 5)

            for (dayIndex, day) in dayColumns.enumerated() {
                let originX = CGFloat(dayIndex) * (cellWidth + cellSpacing)
                for (hourIndex, count) in day.counts.enumerated() {
                    let originY = CGFloat(hourIndex) * (cellHeight + cellSpacing)
                    let rect = CGRect(x: originX, y: originY, width: cellWidth, height: cellHeight)
                    let level = TrendHeatmapView.colorLevel(for: count, maxCount: maxCount)
                    if useRounded {
                        bucketPaths[level].addRoundedRect(in: rect, cornerSize: cornerSize, style: .continuous)
                    } else {
                        bucketPaths[level].addRect(rect)
                    }
                }
            }

            for level in 0..<5 where !bucketPaths[level].isEmpty {
                context.fill(
                    bucketPaths[level],
                    with: .color(TrendHeatmapView.color(forLevel: level))
                )
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
                .fixedSize(horizontal: true, vertical: false)
                .frame(minWidth: 16, alignment: .leading)
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
                                Text(album.displayAlbum)
                                    .font(.subheadline)
                                    .foregroundColor(SonicTheme.textPrimary)
                                    .lineLimit(1)
                                    .help(album.displayAlbum)
                                Text(album.displaySubtitle)
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                    .lineLimit(1)
                                    .help(album.displaySubtitle)
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

struct RecentPlaysSection: View, Equatable {
    static let homeMinimumWidth: CGFloat = 340

    let items: [RecentPlayRecord]
    let artworkURLs: [Int64: String]
    let totalPlaysCount: Int64?
    let accentKey: HomeHotAccentKey?
    let onTrackTap: (Track) -> Void

    static func == (lhs: RecentPlaysSection, rhs: RecentPlaysSection) -> Bool {
        lhs.totalPlaysCount == rhs.totalPlaysCount &&
        lhs.accentKey == rhs.accentKey &&
        lhs.artworkURLs == rhs.artworkURLs &&
        lhs.items.count == rhs.items.count &&
        zip(lhs.items, rhs.items).allSatisfy {
            $0.id == $1.id &&
            $0.playTime == $1.playTime &&
            $0.track == $1.track &&
            $0.artist == $1.artist &&
            $0.album == $1.album
        }
    }

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
            onTrackTap(item.bridgeTrack)
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

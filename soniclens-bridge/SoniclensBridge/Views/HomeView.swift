import SwiftUI

struct HomeView: View {
    @EnvironmentObject var store: AppStore
    @StateObject private var viewModel = HomeViewModel()
    @State private var selectedRecentTrack: Track?
    @State private var selectedAlbumID: Int64?
    @State private var rankingSearchText: String = ""

    var body: some View {
        HomeContentView(
            viewModel: viewModel,
            selectedRecentTrack: $selectedRecentTrack,
            selectedAlbumID: $selectedAlbumID,
            rankingSearchText: $rankingSearchText
        )
    }
}

struct HomeContentView: View {
    @EnvironmentObject private var store: AppStore
    @ObservedObject var viewModel: HomeViewModel
    @Binding var selectedRecentTrack: Track?
    @Binding var selectedAlbumID: Int64?
    @Binding var rankingSearchText: String

    var body: some View {
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
                ]
            )

            ScrollView {
                VStack(alignment: .leading, spacing: 28) {
                    if let message = viewModel.errorMessage {
                        ErrorBanner(message: message)
                    }

                    HomeHeroSection()

                    DashboardStatsSection(stats: viewModel.stats)

                    DashboardTrendSection(
                        points: viewModel.trendPoints,
                        hourlyData: viewModel.hourlyData
                    )
                    .frame(maxWidth: .infinity)

                    HStack(alignment: .top, spacing: 16) {
                        TopGenresCard(topGenres: viewModel.topGenres)
                            .frame(maxWidth: .infinity)
                        TopArtistsCard(topArtists: viewModel.topArtistsByPlays)
                            .frame(maxWidth: .infinity)
                    }

                    TopAlbumsCard(topAlbums: viewModel.topAlbums, onAlbumTap: { albumID in
                        selectedAlbumID = albumID
                    })
                        .frame(maxWidth: .infinity)

                    HStack(alignment: .top, spacing: 16) {
                        RecentPlaysSection(
                            items: viewModel.recentPlays,
                            onTrackTap: { track in
                                selectedRecentTrack = track
                            }
                        )
                        .frame(maxWidth: .infinity)
                        RankingsCard(
                            topArtistsByPlays: viewModel.topArtistsByPlays,
                            topTracks: viewModel.topTracks,
                            onTrackTap: { track in
                                selectedRecentTrack = track
                            }
                        )
                        .frame(maxWidth: .infinity)
                    }
                }
                .padding(.horizontal, 32)
                .padding(.top, 28)
                .padding(.bottom, 48)
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
    }
}

struct HomeHeroSection: View {
    var body: some View {
        GlassPanel(cornerRadius: 22, padding: 0) {
            ZStack(alignment: .bottomLeading) {
                LinearGradient(
                    colors: [
                        SonicTheme.primary.opacity(0.8),
                        SonicTheme.primary.opacity(0.35),
                        SonicTheme.secondaryAccent.opacity(0.35)
                    ],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
                .cornerRadius(22)

                VStack(alignment: .leading, spacing: 6) {
                    Text("SonicLens Bridge for Mac")
                       .font(.system(size: 12, weight: .semibold, design: .rounded))
                       .foregroundStyle(Color.white.opacity(0.84))
                    Text("声之透镜 · 深度解析 · 聆听之印记")
                        .font(.system(size: 26, weight: .bold, design: .rounded))
                        .foregroundStyle(.white)
                    Text("音乐不仅是流动的空气，更是我们生命中不曾停歇的数字资产")
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(Color.white.opacity(0.82))
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

struct DashboardTrendSection: View {
    let points: [TrendPoint]
    let hourlyData: [HourlyData]
    var title: String = "最近 90 天播放热力图"
    var subtitle: String = "按日期展开 24 小时播放分布，深色表示更高频的聆听时段"
    var heatmapHeight: CGFloat = 276
    var heatmapLayout: TrendHeatmapLayout = .fitted(minCellWidth: 2)
    var axisLabelStyle: TrendAxisLabelStyle = .monthBoundaries
    var actionPlacement: TrendActionPlacement = .header
    var actionTitle: String? = nil
    var onAction: (() -> Void)? = nil

    private var totalPlays: Int {
        points.map(\.count).reduce(0, +)
    }

    private var averageDailyPlays: Int {
        guard !points.isEmpty else { return 0 }
        return totalPlays / points.count
    }

    private var peakDay: TrendPoint? {
        points.max(by: { lhs, rhs in
            if lhs.count == rhs.count {
                return lhs.date < rhs.date
            }
            return lhs.count < rhs.count
        })
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            // SectionHeader(title: "趋势")

            GlassPanel(cornerRadius: 18, padding: 0) {
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
                        hourlyData: hourlyData,
                        layout: heatmapLayout,
                        axisLabelStyle: axisLabelStyle
                    )
                        .frame(height: heatmapHeight)
                        .padding(.horizontal, 18)
                        .padding(.bottom, 18)
                }
            }
        }
    }

    private var headerContent: some View {
        ViewThatFits(in: .horizontal) {
            HStack(alignment: .top, spacing: 12) {
                headerText
                Spacer(minLength: 12)
                HStack(spacing: 10) {
                    TrendLegendView(maxCount: hourlyData.flatMap { $0.hourly.values }.max() ?? 0)
                    if actionPlacement == .header {
                        headerAction
                    }
                }
            }

            VStack(alignment: .leading, spacing: 12) {
                headerText
                TrendLegendView(maxCount: hourlyData.flatMap { $0.hourly.values }.max() ?? 0)
                if actionPlacement == .header {
                    headerAction
                }
            }
        }
    }

    private var metricsContent: some View {
        ViewThatFits(in: .horizontal) {
            HStack(spacing: 10) {
                TrendMetricPill(title: "总播放", value: "\(totalPlays)")
                TrendMetricPill(title: "日均", value: "\(averageDailyPlays)")
                TrendMetricPill(
                    title: "峰值",
                    value: peakDay.map { "\($0.count) · \(TrendHeatmapView.compactDateLabel(for: $0.date))" } ?? "--"
                )
                if actionPlacement == .metricsTrailing {
                    headerAction
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)

            VStack(alignment: .leading, spacing: 10) {
                HStack(spacing: 10) {
                    TrendMetricPill(title: "总播放", value: "\(totalPlays)")
                    TrendMetricPill(title: "日均", value: "\(averageDailyPlays)")
                    TrendMetricPill(
                        title: "峰值",
                        value: peakDay.map { "\($0.count) · \(TrendHeatmapView.compactDateLabel(for: $0.date))" } ?? "--"
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
            Text(title)
                .font(.system(size: 16, weight: .semibold))
                .foregroundStyle(SonicTheme.textPrimary)
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

enum TrendHeatmapLayout {
    case fitted(minCellWidth: CGFloat)
    case scrollable(minCellWidth: CGFloat)
}

enum TrendActionPlacement {
    case header
    case metricsTrailing
}

enum TrendAxisLabelStyle {
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

struct TrendHeatmapView: View {
    let hourlyData: [HourlyData]
    var layout: TrendHeatmapLayout = .fitted(minCellWidth: 2)
    var axisLabelStyle: TrendAxisLabelStyle = .monthBoundaries

    private let hours = Array(stride(from: 23, through: 0, by: -1))
    private let cellSpacing: CGFloat = 2
    private let gridInset: CGFloat = 10

    private var sortedData: [HourlyData] {
        hourlyData.sorted { $0.date < $1.date }
    }

    private var xAxisHeight: CGFloat {
        axisLabelStyle.axisHeight
    }

    private var maxCount: Int {
        max(sortedData.flatMap { $0.hourly.values }.max() ?? 1, 1)
    }

    var body: some View {
        GeometryReader { geometry in
            let rowCount = CGFloat(hours.count)
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
                    let dayCount = max(sortedData.count, 1)
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
                case let .scrollable(minCellWidth):
                    let dayCount = max(sortedData.count, 1)
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
            ForEach(hours, id: \.self) { hour in
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

    private func heatmapColumns(cellWidth: CGFloat, cellHeight: CGFloat, contentWidth: CGFloat) -> some View {
        VStack(alignment: .leading, spacing: axisLabelStyle.axisToGridSpacing) {
            xAxisLabels(contentWidth: contentWidth, cellWidth: cellWidth)
            dayColumns(cellWidth: cellWidth, cellHeight: cellHeight, contentWidth: contentWidth)
        }
    }

    private func xAxisLabels(contentWidth: CGFloat, cellWidth: CGFloat) -> some View {
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

    private func dayColumns(cellWidth: CGFloat, cellHeight: CGFloat, contentWidth: CGFloat) -> some View {
        HStack(alignment: .top, spacing: cellSpacing) {
            ForEach(sortedData) { day in
                VStack(spacing: cellSpacing) {
                    let dayTotal = day.hourly.values.reduce(0, +)
                    ForEach(hours, id: \.self) { hour in
                        let count = day.countForHour(hour)
                        RoundedRectangle(cornerRadius: 2, style: .continuous)
                            .fill(color(for: count))
                            .frame(width: cellWidth, height: cellHeight)
                            .help(
                                """
                                日期：\(day.date)
                                当日总播放：\(dayTotal)
                                时段：\(String(format: "%02d:00-%02d:59", hour, hour))
                                当前时段播放：\(count)
                                """
                            )
                    }
                }
            }
        }
        .frame(width: contentWidth, alignment: .leading)
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

    private func labelX(at index: Int, cellWidth: CGFloat) -> CGFloat {
        CGFloat(index) * (cellWidth + cellSpacing) + cellWidth / 2
    }

    static func compactDateLabel(for dateString: String) -> String {
        guard let date = dayFormatter.date(from: dateString) else { return dateString }
        return shortDayFormatter.string(from: date)
    }

    private var axisLabels: [AxisLabel] {
        switch axisLabelStyle {
        case .monthBoundaries:
            return monthAxisLabels
        case let .dayStride(step, _):
            return strideAxisLabels(step: step)
        }
    }

    private var monthAxisLabels: [AxisLabel] {
        guard !sortedData.isEmpty else { return [] }

        var labels: [AxisLabel] = []
        for (index, day) in sortedData.enumerated() {
            guard let date = Self.dayFormatter.date(from: day.date) else { continue }
            let dayNumber = Calendar.current.component(.day, from: date)
            if index == 0 || dayNumber == 1 || index == sortedData.count - 1 {
                labels.append(AxisLabel(index: index, text: Self.yearMonthFormatter.string(from: date)))
            }
        }
        return dedupAxisLabels(labels)
    }

    private func strideAxisLabels(step: Int) -> [AxisLabel] {
        guard !sortedData.isEmpty else { return [] }

        let effectiveStep = max(step, 1)
        let lastIndex = sortedData.count - 1
        var labels = stride(from: 0, through: lastIndex, by: effectiveStep).map { index in
            AxisLabel(index: index, text: Self.compactDateLabel(for: sortedData[index].date))
        }

        if labels.last?.index != lastIndex {
            let lastLabel = AxisLabel(index: lastIndex, text: Self.compactDateLabel(for: sortedData[lastIndex].date))
            if let previousIndex = labels.indices.last, lastIndex - labels[previousIndex].index < effectiveStep {
                labels[previousIndex] = lastLabel
            } else {
                labels.append(lastLabel)
            }
        }

        return dedupAxisLabels(labels)
    }

    private func dedupAxisLabels(_ labels: [AxisLabel]) -> [AxisLabel] {
        var unique: [AxisLabel] = []
        var seen = Set<String>()
        for label in labels where !seen.contains(label.text) {
            seen.insert(label.text)
            unique.append(label)
        }
        return unique.sorted { $0.index < $1.index }
    }

    private func color(for count: Int) -> Color {
        if count == 0 { return Self.emptyCell }
        let ratio = Double(count) / Double(maxCount)
        switch ratio {
        case 0..<0.25: return Self.levelOne
        case 0.25..<0.5: return Self.levelTwo
        case 0.5..<0.75: return Self.levelThree
        default: return Self.levelFour
        }
    }

    static let emptyCell = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.93, 0.95, 0.93, 1),
        dark: .sonicRGBA(0.09, 0.12, 0.10, 1)
    )
    static let levelOne = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.61, 0.85, 0.64, 1),
        dark: .sonicRGBA(0.10, 0.33, 0.18, 1)
    )
    static let levelTwo = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.27, 0.72, 0.37, 1),
        dark: .sonicRGBA(0.15, 0.53, 0.27, 1)
    )
    static let levelThree = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.15, 0.57, 0.22, 1),
        dark: .sonicRGBA(0.23, 0.72, 0.35, 1)
    )
    static let levelFour = SonicTheme.dynamicColor(
        light: .sonicRGBA(0.08, 0.43, 0.16, 1),
        dark: .sonicRGBA(0.34, 0.86, 0.46, 1)
    )
    private static let gridBackground = SonicTheme.dynamicColor(
        light: .sonicWhite(1, alpha: 0.7),
        dark: .sonicWhite(0.08, alpha: 0.88)
    )
    private static let gridBorder = SonicTheme.dynamicColor(
        light: .sonicWhite(0, alpha: 0.08),
        dark: .sonicWhite(1, alpha: 0.08)
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
}

private struct AxisLabel: Identifiable {
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

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title.uppercased())
                .font(.system(size: 10, weight: .bold, design: .monospaced))
                .foregroundStyle(SonicTheme.textSecondary)
            Text(value)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(SonicTheme.textPrimary)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 9)
        .background(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .fill(SonicTheme.dynamicColor(
                    light: .sonicWhite(1, alpha: 0.76),
                    dark: .sonicWhite(0.1, alpha: 0.76)
                ))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .stroke(SonicTheme.glassBorder, lineWidth: 1)
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
    let items: [RecentPlayRecord]
    let onTrackTap: (Track) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionHeader(title: "最近播放")

            VStack(spacing: 8) {
                ForEach(items.prefix(10)) { item in
                    GlassPanel(cornerRadius: 16, padding: 12) {
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(item.track)
                                    .font(.subheadline)
                                    .foregroundColor(SonicTheme.textPrimary)
                                    .lineLimit(1)
                                    .help(item.track)
                                Text("\(item.artist) · \(item.album)")
                                    .font(.caption)
                                    .foregroundColor(SonicTheme.textSecondary)
                                    .lineLimit(1)
                                    .help("\(item.artist) · \(item.album)")
                            }
                            Spacer()
                            Text(formatPlayTime(item.playTime))
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                    .contentShape(Rectangle())
                    .onTapGesture {
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
                    }
                }
            }
        }
    }

    private func formatPlayTime(_ timeString: String) -> String {
        let isoFormatter = ISO8601DateFormatter()
        isoFormatter.formatOptions = [.withInternetDateTime]
        guard let date = isoFormatter.date(from: timeString) else { return timeString }
        let displayFormatter = DateFormatter()
        displayFormatter.dateFormat = "yyyy-MM-dd HH:mm"
        return displayFormatter.string(from: date)
    }
}

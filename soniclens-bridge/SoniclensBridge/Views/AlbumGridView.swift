import SwiftUI
import Foundation

struct AlbumGridView: View {
    @ObservedObject var viewModel: LibraryViewModel
    let sort: LibrarySort
    let query: String
    let artworkBaseURL: URL?
    let statusSummary: String?
    var prefersCompactLayout: Bool = false
    @State private var selectedAlbum: Album?

    init(
        viewModel: LibraryViewModel,
        sort: LibrarySort,
        query: String,
        artworkBaseURL: URL? = nil,
        statusSummary: String? = nil,
        prefersCompactLayout: Bool = false
    ) {
        self.viewModel = viewModel
        self.sort = sort
        self.query = query
        self.artworkBaseURL = artworkBaseURL
        self.statusSummary = statusSummary
        self.prefersCompactLayout = prefersCompactLayout
    }

    var body: some View {
        GeometryReader { proxy in
            let metrics = AlbumGridMetrics(
                availableWidth: proxy.size.width,
                prefersCompactLayout: prefersCompactLayout
            )

            ScrollView {
                VStack(alignment: .leading, spacing: metrics.sectionSpacing) {
                    if prefersCompactLayout, let statusSummary {
                        LibraryStatusSummaryChip(text: statusSummary)
                    }

                    if let loadingStatus = viewModel.albumLoadingStatusText {
                        LibraryCollectionLoadingHint(text: loadingStatus)
                    }

                    if viewModel.albums.isEmpty {
                        EmptyStateView(
                            title: "未找到专辑",
                            subtitle: "请尝试更换搜索关键词或清除筛选。"
                        )
                    } else {
                        LazyVGrid(columns: metrics.columns, spacing: metrics.gridSpacing) {
                            ForEach(Array(viewModel.albums.enumerated()), id: \.element.id) { index, album in
                                AlbumCardView(
                                    album: album,
                                    artworkBaseURL: artworkBaseURL,
                                    cardWidth: metrics.cardWidth,
                                    contentWidth: metrics.cardContentWidth,
                                    prefersCompactLayout: prefersCompactLayout,
                                    onSelect: {
                                        selectedAlbum = album
                                    }
                                )
                                .contentShape(RoundedRectangle(cornerRadius: 18))
                                .onAppear {
                                    if viewModel.shouldLoadMoreAlbums(at: index) {
                                        Task { await viewModel.loadMoreAlbums(sort: sort, query: query) }
                                    }
                                }
                            }
                        }
                    }
                }
                .padding(.horizontal, metrics.horizontalPadding)
                .padding(.vertical, prefersCompactLayout ? 20 : 32)
            }
        }
        .task(id: "\(sort.rawValue)|\(query)") {
            await viewModel.reloadAlbums(sort: sort, query: query)
        }
        .navigationDestination(item: $selectedAlbum) { album in
            albumDetailDestination(albumID: album.id)
        }
    }
}

private struct AlbumGridMetrics {
    let horizontalPadding: CGFloat
    let gridSpacing: CGFloat
    let sectionSpacing: CGFloat
    let cardWidth: CGFloat
    let cardContentWidth: CGFloat
    let columns: [GridItem]

    init(availableWidth: CGFloat, prefersCompactLayout: Bool) {
        horizontalPadding = prefersCompactLayout ? 16 : 32
        gridSpacing = prefersCompactLayout ? 18 : 24
        sectionSpacing = prefersCompactLayout ? 16 : 20

        let contentWidth = max(availableWidth - horizontalPadding * 2, 0)
        let cardInset: CGFloat = prefersCompactLayout ? 8 : 10
        let minWidth: CGFloat = prefersCompactLayout ? 162 : 196
        let maxWidth: CGFloat = prefersCompactLayout ? 188 : 236
        let minimumColumns = prefersCompactLayout ? 2 : 1

        let rawCount = Int((contentWidth + gridSpacing) / (minWidth + gridSpacing))
        let columnCount = max(rawCount, minimumColumns)
        let resolvedWidth = Self.maxWidthForColumnCount(
            contentWidth: contentWidth,
            spacing: gridSpacing,
            columnCount: columnCount,
            minWidth: minWidth,
            maxWidth: maxWidth
        )

        cardWidth = resolvedWidth
        cardContentWidth = max(resolvedWidth - cardInset * 2, 0)
        columns = Array(repeating: GridItem(.fixed(resolvedWidth), spacing: gridSpacing, alignment: .top), count: columnCount)
    }

    private static func maxWidthForColumnCount(
        contentWidth: CGFloat,
        spacing: CGFloat,
        columnCount: Int,
        minWidth: CGFloat,
        maxWidth: CGFloat
    ) -> CGFloat {
        guard columnCount > 0 else { return minWidth }
        let totalSpacing = CGFloat(max(columnCount - 1, 0)) * spacing
        let candidate = floor((contentWidth - totalSpacing) / CGFloat(columnCount))
        return min(max(candidate, minWidth), maxWidth)
    }
}

struct AlbumCardView: View {
    let album: Album
    let artworkBaseURL: URL?
    let cardWidth: CGFloat
    let contentWidth: CGFloat
    var prefersCompactLayout: Bool = false
    let onSelect: () -> Void

    @State private var isHovered = false
    @State private var isTapFeedbackActive = false

    var body: some View {
        let spacing: CGFloat = prefersCompactLayout ? 10 : 12
        let inset: CGFloat = prefersCompactLayout ? 8 : 10
        let metadataHeight: CGFloat = prefersCompactLayout ? 68 : 84
        let contentHeight = contentWidth + spacing + metadataHeight
        let resolvedArtworkURL = ArtworkURLResolver.resolveArtworkPath(album.coverArtURL, artworkBaseURL: artworkBaseURL)
        let isBadgeHighlighted = isHovered || isTapFeedbackActive

        VStack(alignment: .leading, spacing: spacing) {
            AlbumArtworkCard(
                size: contentWidth,
                artworkURL: resolvedArtworkURL,
                title: album.name,
                hasInsight: album.hasInsight,
                prefersCompactBadge: prefersCompactLayout,
                isHighlighted: isBadgeHighlighted,
                isTapFeedbackActive: isTapFeedbackActive
            )

            AlbumCardMetadata(
                album: album,
                showsReleaseDate: !prefersCompactLayout,
                titleFont: prefersCompactLayout ? .subheadline.weight(.semibold) : .body.weight(.semibold),
                subtitleFont: prefersCompactLayout ? .caption : .subheadline,
                fixedHeight: metadataHeight
            )
        }
        .frame(width: contentWidth, height: contentHeight, alignment: .topLeading)
        .padding(inset)
        .frame(width: cardWidth, height: contentHeight + inset * 2, alignment: .topLeading)
        .background(
            RoundedRectangle(cornerRadius: 18)
                .fill(SonicTheme.card)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 18)
                .stroke(SonicTheme.glassBorder, lineWidth: 1)
        )
        .accessibilityAddTraits(.isButton)
        .onHover { hovering in
            withAnimation(.easeOut(duration: 0.16)) {
                isHovered = hovering
            }
        }
        .onTapGesture {
            triggerSelection()
        }
    }

    private func triggerSelection() {
        guard !isTapFeedbackActive else {
            onSelect()
            return
        }

        withAnimation(.spring(response: 0.2, dampingFraction: 0.62)) {
            isTapFeedbackActive = true
        }

        Task {
            try? await Task.sleep(nanoseconds: 110_000_000)
            await MainActor.run {
                isTapFeedbackActive = false
                onSelect()
            }
        }
    }
}

private struct AlbumArtworkCard: View {
    let size: CGFloat
    let artworkURL: String?
    let title: String
    let hasInsight: Bool
    let prefersCompactBadge: Bool
    let isHighlighted: Bool
    let isTapFeedbackActive: Bool

    var body: some View {
        ArtworkSquareView(
            artworkURL: artworkURL,
            fallbackTitle: title,
            size: size,
            cornerRadius: 16,
            style: .vivid
        )
        .overlay(
            RoundedRectangle(cornerRadius: 16)
                .stroke(Color.white.opacity(0.22), lineWidth: 1)
        )
        .overlay(alignment: .topTrailing) {
            if hasInsight {
                AlbumInsightBadge(
                    compact: prefersCompactBadge,
                    isHighlighted: isHighlighted,
                    isTapFeedbackActive: isTapFeedbackActive
                )
                    .padding(prefersCompactBadge ? 8 : 10)
            }
        }
    }
}

private struct AlbumInsightBadge: View {
    let compact: Bool
    let isHighlighted: Bool
    let isTapFeedbackActive: Bool
    var style: AlbumInsightBadgeStyle = .brass //选型
    @Environment(\.sonicPerformanceModeEnabled) private var performanceModeEnabled

    private var badgeSize: CGFloat { compact ? 18 : 28 } //设置大小
    private var glyphColor: Color {
        switch style {
        case .brass:
            return Color(red: 0.27, green: 0.21, blue: 0.11).opacity(0.92)
        case .brassV2:
            return Color(red: 0.24, green: 0.18, blue: 0.09).opacity(0.95)
        default:
            return Color.black.opacity(0.74)
        }
    }
    private var borderColor: Color {
        switch style {
        case .brass:
            return Color.white.opacity(0.22)
        case .brassV2:
            return Color.white.opacity(0.18)
        default:
            return Color.white.opacity(0.45)
        }
    }
    private var fillGradient: LinearGradient {
        switch style {
        case .brass:
            return LinearGradient(
                colors: [
                    Color(red: 0.95, green: 0.83, blue: 0.39),
                    Color(red: 0.82, green: 0.63, blue: 0.17)
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        case .brassV2:
            return LinearGradient(
                colors: [
                    Color(red: 0.97, green: 0.85, blue: 0.45),
                    Color(red: 0.80, green: 0.60, blue: 0.16)
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        default:
            return LinearGradient(
                colors: [
                    Color(red: 1.0, green: 0.91, blue: 0.47),
                    Color(red: 0.95, green: 0.76, blue: 0.18)
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        }
    }
    private var badgeScale: CGFloat {
        if isTapFeedbackActive {
            return 1.12
        }
        return isHighlighted ? 1.05 : 1.0
    }
    private var ringOpacity: Double {
        isHighlighted ? 0.52 : 0.34
    }
    private var glyphRotation: Double {
        if isTapFeedbackActive {
            return 5
        }
        return isHighlighted ? 2 : 0
    }

    var body: some View {
        ZStack {
            Circle()
                .fill(fillGradient)

            if style == .brassV2 {
                Circle()
                    .fill(
                        RadialGradient(
                            colors: [
                                Color.white.opacity(0.30),
                                Color.white.opacity(0.05),
                                .clear
                            ],
                            center: UnitPoint(x: 0.34, y: 0.28),
                            startRadius: 0,
                            endRadius: badgeSize * 0.72
                        )
                    )

                Circle()
                    .fill(
                        LinearGradient(
                            colors: [
                                .clear,
                                Color.black.opacity(0.12)
                            ],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        )
                    )
            }

            Circle()
                .stroke(borderColor, lineWidth: style == .brass || style == .brassV2 ? 0.8 : 1)

            AlbumInsightBadgeGlyph(
                style: style,
                compact: compact,
                color: glyphColor,
                ringOpacity: ringOpacity,
                rotationDegrees: glyphRotation,
                isHighlighted: isHighlighted,
                isTapFeedbackActive: isTapFeedbackActive,
                simplified: performanceModeEnabled
            )
        }
        .frame(width: badgeSize, height: badgeSize)
        .scaleEffect(badgeScale)
        .shadow(color: Color.black.opacity(performanceModeEnabled ? 0.08 : 0.16), radius: performanceModeEnabled ? 4 : 8, x: 0, y: 3)
        .animation(.easeOut(duration: 0.16), value: isHighlighted)
        .animation(.spring(response: 0.22, dampingFraction: 0.58), value: isTapFeedbackActive)
        .accessibilityLabel("已生成音眸分析")
    }
}

private enum AlbumInsightBadgeStyle: String, CaseIterable {
    case lens
    case stamp
    case echo
    case original
    case brass
    case brassV2

    var previewTitle: String {
        switch self {
        case .lens:
            return "透镜"
        case .stamp:
            return "印章"
        case .echo:
            return "声波"
        case .original:
            return "原版"
        case .brass:
            return "黄铜章"
        case .brassV2:
            return "黄铜章 V2"
        }
    }
}

private struct AlbumInsightBadgeGlyph: View {
    let style: AlbumInsightBadgeStyle
    let compact: Bool
    let color: Color
    let ringOpacity: Double
    let rotationDegrees: Double
    let isHighlighted: Bool
    let isTapFeedbackActive: Bool
    let simplified: Bool

    var body: some View {
        Group {
            switch style {
            case .lens:
                LensFocusGlyph(
                    compact: compact,
                    color: color,
                    ringOpacity: ringOpacity,
                    rotationDegrees: rotationDegrees,
                    isTapFeedbackActive: isTapFeedbackActive,
                    simplified: simplified
                )
            case .stamp:
                ListeningStampGlyph(
                    compact: compact,
                    color: color,
                    ringOpacity: ringOpacity,
                    rotationDegrees: rotationDegrees,
                    isTapFeedbackActive: isTapFeedbackActive,
                    simplified: simplified
                )
            case .echo:
                EchoWaveGlyph(
                    compact: compact,
                    color: color,
                    ringOpacity: ringOpacity,
                    rotationDegrees: rotationDegrees,
                    isHighlighted: isHighlighted,
                    isTapFeedbackActive: isTapFeedbackActive,
                    simplified: simplified
                )
            case .original:
                OriginalLogoGlyph(
                    compact: compact,
                    color: color,
                    rotationDegrees: rotationDegrees,
                    isTapFeedbackActive: isTapFeedbackActive
                )
            case .brass:
                OriginalLogoGlyph(
                    compact: compact,
                    color: color,
                    rotationDegrees: rotationDegrees,
                    isTapFeedbackActive: isTapFeedbackActive,
                    lineCap: .butt
                )
            case .brassV2:
                OriginalLogoGlyph(
                    compact: compact,
                    color: color,
                    rotationDegrees: rotationDegrees,
                    isTapFeedbackActive: isTapFeedbackActive,
                    lineCap: .butt,
                    insetCore: true,
                    coreGlowOpacity: simplified ? 0.08 : 0.12
                )
            }
        }
    }
}

private struct OriginalLogoGlyph: View {
    let compact: Bool
    let color: Color
    let rotationDegrees: Double
    let isTapFeedbackActive: Bool
    var lineCap: CGLineCap = .round
    var insetCore: Bool = false
    var coreGlowOpacity: Double = 0

    private var lineWidth: CGFloat {
        compact ? 0.95 : 1.1
    }

    private var outerScale: CGFloat {
        compact ? 0.31 : 0.35
    }

    private var middleScale: CGFloat {
        compact ? 0.22 : 0.25
    }

    private var innerScale: CGFloat {
        compact ? 0.135 : 0.155
    }

    private var coreSize: CGFloat {
        compact ? 4.5 : 5.4
    }

    var body: some View {
        ZStack {
            Circle()
                .trim(from: 0, to: 0.972)
                .stroke(color, style: StrokeStyle(lineWidth: lineWidth, lineCap: lineCap))
                .frame(width: 70 * outerScale, height: 70 * outerScale)

            Circle()
                .trim(from: 0.01, to: 0.972)
                .stroke(color, style: StrokeStyle(lineWidth: lineWidth, lineCap: lineCap))
                .frame(width: 70 * middleScale, height: 70 * middleScale)
                .rotationEffect(.degrees(5))

            Circle()
                .trim(from: 0.015, to: 0.972)
                .stroke(color, style: StrokeStyle(lineWidth: lineWidth, lineCap: lineCap))
                .frame(width: 70 * innerScale, height: 70 * innerScale)
                .rotationEffect(.degrees(10))

            Circle()
                .stroke(color.opacity(coreGlowOpacity), lineWidth: compact ? 0.45 : 0.6)
                .frame(width: coreSize * 1.9, height: coreSize * 1.9)
                .blur(radius: compact ? 0.2 : 0.35)
                .opacity(insetCore ? 1 : 0)

            Circle()
                .fill(
                    insetCore
                        ? RadialGradient(
                            colors: [
                                Color.white.opacity(0.12),
                                color.opacity(0.82),
                                color.opacity(0.98)
                            ],
                            center: UnitPoint(x: 0.38, y: 0.34),
                            startRadius: 0,
                            endRadius: coreSize
                        )
                        : RadialGradient(
                            colors: [color, color],
                            center: .center,
                            startRadius: 0,
                            endRadius: coreSize
                        )
                )
                .frame(width: coreSize, height: coreSize)
                .overlay {
                    Circle()
                        .stroke(Color.white.opacity(insetCore ? 0.08 : 0), lineWidth: compact ? 0.24 : 0.32)
                }
                .overlay(alignment: .topLeading) {
                    if insetCore {
                        Circle()
                            .trim(from: 0.58, to: 0.92)
                            .stroke(Color.white.opacity(0.2), style: StrokeStyle(lineWidth: compact ? 0.28 : 0.36, lineCap: .round))
                            .frame(width: coreSize * 1.16, height: coreSize * 1.16)
                            .rotationEffect(.degrees(-24))
                            .offset(x: -coreSize * 0.03, y: -coreSize * 0.02)
                    }
                }
                .scaleEffect(isTapFeedbackActive ? 1.12 : 1.0)
        }
        .rotationEffect(.degrees(rotationDegrees * 0.6))
    }
}

private struct ListeningStampGlyph: View {
    let compact: Bool
    let color: Color
    let ringOpacity: Double
    let rotationDegrees: Double
    let isTapFeedbackActive: Bool
    let simplified: Bool

    private var ringSizes: [CGFloat] {
        compact ? [17, 11.8, 6.8] : [20, 14, 8]
    }

    private var pupilSize: CGFloat {
        compact ? 4.6 : 5.5
    }

    var body: some View {
        ZStack {
            ForEach(Array(ringSizes.enumerated()), id: \.offset) { index, size in
                Circle()
                    .trim(from: 0.06, to: 0.92 - Double(index) * 0.01)
                    .stroke(
                        color.opacity(ringOpacity - Double(index) * 0.06),
                        style: StrokeStyle(lineWidth: compact ? 1.05 : 1.2, lineCap: .round)
                    )
                    .frame(width: size, height: size)
                    .scaleEffect(x: 1.12, y: 1.0)
                    .rotationEffect(.degrees(2 + rotationDegrees + Double(index) * 1.6))
                    .opacity(simplified && index == 2 ? 0 : 1)
            }

            Circle()
                .fill(color)
                .frame(width: pupilSize, height: pupilSize)
                .scaleEffect(isTapFeedbackActive ? 1.14 : 1.0)
        }
    }
}

private struct LensFocusGlyph: View {
    let compact: Bool
    let color: Color
    let ringOpacity: Double
    let rotationDegrees: Double
    let isTapFeedbackActive: Bool
    let simplified: Bool

    private var ringSizes: [CGFloat] {
        compact ? [17, 12.1, 7.1] : [20, 14.2, 8.3]
    }

    private var pupilSize: CGFloat {
        compact ? 4.5 : 5.3
    }

    var body: some View {
        ZStack {
            ForEach(Array(ringSizes.enumerated()), id: \.offset) { index, size in
                Circle()
                    .trim(from: 0.08, to: 0.92)
                    .stroke(
                        color.opacity(ringOpacity - Double(index) * 0.05),
                        style: StrokeStyle(lineWidth: compact ? 1.0 : 1.15, lineCap: .round)
                    )
                    .frame(width: size, height: size)
                    .rotationEffect(.degrees(Double(index) * 5 + rotationDegrees))
                    .opacity(simplified && index == 2 ? 0 : 1)
            }

            Circle()
                .fill(color)
                .frame(width: pupilSize, height: pupilSize)
                .scaleEffect(isTapFeedbackActive ? 1.12 : 1.0)
        }
    }
}

private struct EchoWaveGlyph: View {
    let compact: Bool
    let color: Color
    let ringOpacity: Double
    let rotationDegrees: Double
    let isHighlighted: Bool
    let isTapFeedbackActive: Bool
    let simplified: Bool

    private var ringSizes: [CGSize] {
        compact
            ? [CGSize(width: 17.5, height: 11.2), CGSize(width: 12.7, height: 8.1), CGSize(width: 8.1, height: 5.2)]
            : [CGSize(width: 20.5, height: 13.2), CGSize(width: 14.8, height: 9.4), CGSize(width: 9.3, height: 6.1)]
    }

    private var pupilSize: CGFloat {
        compact ? 4.5 : 5.3
    }

    var body: some View {
        ZStack {
            ForEach(Array(ringSizes.enumerated()), id: \.offset) { index, size in
                Ellipse()
                    .trim(from: 0.1, to: 0.93 - Double(index) * 0.02)
                    .stroke(
                        color.opacity(ringOpacity - Double(index) * 0.06),
                        style: StrokeStyle(lineWidth: compact ? 1.0 : 1.15, lineCap: .round)
                    )
                    .frame(width: size.width, height: size.height)
                    .rotationEffect(.degrees(-4 + rotationDegrees + Double(index)))
                    .scaleEffect(x: isHighlighted ? 1.02 : 1.0, y: isTapFeedbackActive ? 0.94 : 1.0)
                    .opacity(simplified && index == 2 ? 0 : 1)
            }

            Circle()
                .fill(color)
                .frame(width: pupilSize, height: pupilSize)
                .scaleEffect(isTapFeedbackActive ? 1.1 : 1.0)
        }
    }
}

#if DEBUG
#Preview("Album Insight Badge Drafts") {
    HStack(spacing: 24) {
        ForEach(AlbumInsightBadgeStyle.allCases, id: \.rawValue) { style in
            VStack(spacing: 10) {
                AlbumInsightBadge(
                    compact: false,
                    isHighlighted: style == .stamp,
                    isTapFeedbackActive: false,
                    style: style
                )
                Text(style.previewTitle)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }
    .padding(28)
    .background(Color(red: 0.93, green: 0.98, blue: 0.91))
}
#endif

private struct AlbumCardMetadata: View {
    let album: Album
    let showsReleaseDate: Bool
    let titleFont: Font
    let subtitleFont: Font
    let fixedHeight: CGFloat

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(album.name)
                .font(titleFont)
                .foregroundStyle(SonicTheme.textPrimary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .lineLimit(2)

            Text(album.artist)
                .font(subtitleFont)
                .foregroundStyle(SonicTheme.textSecondary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .lineLimit(1)

            if showsReleaseDate || album.playCount != nil {
                HStack(spacing: 8) {
                    if showsReleaseDate, let releaseDate = album.releaseDate, !releaseDate.isEmpty {
                        AlbumGridMetaPill(systemImage: "calendar", value: releaseDate)
                    }
                    if let playCount = album.playCount, playCount > 0 {
                        AlbumGridMetaPill(systemImage: "play.fill", value: "\(playCount)")
                    }
                }
            }

            Spacer(minLength: 0)
        }
        .frame(maxWidth: .infinity, minHeight: fixedHeight, maxHeight: fixedHeight, alignment: .topLeading)
    }
}

private struct AlbumGridMetaPill: View {
    let systemImage: String
    let value: String

    var body: some View {
        Label(value, systemImage: systemImage)
            .font(.caption2)
            .foregroundStyle(SonicTheme.textSecondary)
            .lineLimit(1)
            .padding(.horizontal, 8)
            .padding(.vertical, 5)
            .background(Color.primary.opacity(0.06), in: Capsule())
    }
}

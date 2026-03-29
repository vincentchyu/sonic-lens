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
                                    prefersCompactLayout: prefersCompactLayout
                                )
                                .contentShape(RoundedRectangle(cornerRadius: 18))
                                .onTapGesture {
                                    selectedAlbum = album
                                }
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

    var body: some View {
        let spacing: CGFloat = prefersCompactLayout ? 10 : 12
        let inset: CGFloat = prefersCompactLayout ? 8 : 10
        let metadataHeight: CGFloat = prefersCompactLayout ? 68 : 84
        let contentHeight = contentWidth + spacing + metadataHeight
        let resolvedArtworkURL = ArtworkURLResolver.resolveArtworkPath(album.coverArtURL, artworkBaseURL: artworkBaseURL)

        VStack(alignment: .leading, spacing: spacing) {
            AlbumArtworkCard(
                size: contentWidth,
                artworkURL: resolvedArtworkURL,
                title: album.name
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
    }
}

private struct AlbumArtworkCard: View {
    let size: CGFloat
    let artworkURL: String?
    let title: String

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
    }
}

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

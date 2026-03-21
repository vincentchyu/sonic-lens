import SwiftUI

struct LibraryView: View {
    @EnvironmentObject private var store: AppStore
    @StateObject private var viewModel = LibraryViewModel()

    var body: some View {
        ZStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    LibraryHeroPanel(viewModel: viewModel)

                    if let message = viewModel.errorMessage {
                        ErrorBanner(message: message)
                    }

                    LibraryNavigationSection(viewModel: viewModel)
                }
                .padding(28)
            }

            if viewModel.isLoading && viewModel.albums.isEmpty && viewModel.tracks.isEmpty {
                LoadingOverlay()
            }
        }
        .navigationTitle("我的")
        .task {
            if let server = store.currentServer {
                await viewModel.load(using: server)
            }
        }
        .onChange(of: store.currentServer) { _, server in
            guard let server else { return }
            Task { await viewModel.load(using: server) }
        }
    }
}

private struct LibraryHeroPanel: View {
    @ObservedObject var viewModel: LibraryViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("本地资料库")
                        .font(.title2)
                        .fontWeight(.bold)
                        .foregroundStyle(SonicTheme.textPrimary)
                    Text("先展示本地索引，再在后台补偿同步。滚动与检索都保持轻量。")
                        .font(.subheadline)
                        .foregroundStyle(SonicTheme.textSecondary)
                }

                Spacer()

                LibrarySyncBadge(
                    title: viewModel.syncStatusText,
                    isSyncing: viewModel.isSyncing
                )
            }

            HStack(spacing: 14) {
                LibraryMetricCard(title: "专辑", value: viewModel.albumCountText, systemImage: "square.stack")
                LibraryMetricCard(title: "曲目", value: viewModel.trackCountText, systemImage: "music.note")
                LibraryMetricCard(title: "收藏", value: viewModel.favoriteCountText, systemImage: "heart")
                LibraryMetricCard(title: "待上报", value: viewModel.pendingScrobbleCountText, systemImage: "waveform.badge.exclamationmark")
            }
        }
        .padding(22)
        .background(
            LinearGradient(
                colors: [
                    Color.accentColor.opacity(0.20),
                    SonicTheme.card.opacity(0.95)
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        )
        .overlay(
            RoundedRectangle(cornerRadius: 22)
                .stroke(SonicTheme.glassBorder, lineWidth: 1)
        )
        .clipShape(RoundedRectangle(cornerRadius: 22))
    }
}

private struct LibraryMetricCard: View {
    let title: String
    let value: String
    let systemImage: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Label(title, systemImage: systemImage)
                .font(.caption)
                .foregroundStyle(SonicTheme.textSecondary)
            Text(value)
                .font(.title3)
                .fontWeight(.semibold)
                .foregroundStyle(SonicTheme.textPrimary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(14)
        .glassCard(cornerRadius: 16, isSimplified: true)
    }
}

private struct LibrarySyncBadge: View {
    let title: String
    let isSyncing: Bool

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: isSyncing ? "arrow.triangle.2.circlepath" : "checkmark.circle")
                .font(.caption.weight(.semibold))
            Text(title)
                .font(.caption)
                .foregroundStyle(SonicTheme.textPrimary)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(Color.white.opacity(0.16), in: Capsule())
        .overlay(
            Capsule()
                .stroke(Color.white.opacity(0.2), lineWidth: 1)
        )
    }
}

private struct LibraryNavigationSection: View {
    @ObservedObject var viewModel: LibraryViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            SectionHeader(title: "内容入口")

            VStack(spacing: 12) {
                NavigationLink(destination: AlbumListView(viewModel: viewModel)) {
                    LibraryEntryCard(
                        title: "专辑",
                        subtitle: "按本地索引分页浏览，支持搜索与排序。",
                        systemImage: "square.stack.3d.up"
                    )
                }
                .buttonStyle(.plain)

                NavigationLink(destination: TrackListView(viewModel: viewModel)) {
                    LibraryEntryCard(
                        title: "曲目",
                        subtitle: "更轻的表格式长列表，支持筛选、收藏与快速定位。",
                        systemImage: "music.note.list"
                    )
                }
                .buttonStyle(.plain)

                NavigationLink(destination: InsightListView(viewModel: viewModel)) {
                    LibraryEntryCard(
                        title: "音眸",
                        subtitle: "浏览歌曲洞察与历史生成结果。",
                        systemImage: "sparkles.rectangle.stack"
                    )
                }
                .buttonStyle(.plain)

                NavigationLink(destination: UnscrobbledListView(viewModel: viewModel)) {
                    LibraryEntryCard(
                        title: "未上报",
                        subtitle: "查看待同步到 Last.fm 的播放记录。",
                        systemImage: "clock.badge.exclamationmark"
                    )
                }
                .buttonStyle(.plain)
            }
        }
    }
}

private struct LibraryEntryCard: View {
    let title: String
    let subtitle: String
    let systemImage: String
    @State private var isHovered = false

    var body: some View {
        HStack(spacing: 16) {
            RoundedRectangle(cornerRadius: 14)
                .fill(
                    LinearGradient(
                        colors: [Color.accentColor.opacity(0.3), Color.accentColor.opacity(0.12)],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )
                .frame(width: 52, height: 52)
                .overlay(
                    Image(systemName: systemImage)
                        .font(.system(size: 18, weight: .semibold))
                        .foregroundStyle(SonicTheme.textPrimary)
                )

            VStack(alignment: .leading, spacing: 4) {
                Text(title)
                    .font(.headline)
                    .foregroundStyle(SonicTheme.textPrimary)
                Text(subtitle)
                    .font(.subheadline)
                    .foregroundStyle(SonicTheme.textSecondary)
            }

            Spacer()

            Image(systemName: "chevron.right")
                .font(.caption.weight(.semibold))
                .foregroundStyle(SonicTheme.textSecondary)
        }
        .padding(18)
        .background(
            RoundedRectangle(cornerRadius: 18)
                .fill(isHovered ? Color.primary.opacity(0.045) : Color.clear)
        )
        .glassCard(cornerRadius: 18, isSimplified: true)
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
    }
}

struct AlbumListView: View {
    @ObservedObject var viewModel: LibraryViewModel
    @State private var selectedSort: LibrarySort = .recent
    @State private var query = ""

    private let columns = [
        GridItem(.adaptive(minimum: 208), spacing: 18)
    ]

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                LibraryCollectionHeader(
                    title: "专辑",
                    subtitle: "本地分页索引，适合快速滚动浏览。",
                            trailing: {
                                HStack(spacing: 10) {
                                    ToolbarSearchField(text: $query)
                                    LibrarySortMenu(title: selectedSort.rawValue, selection: $selectedSort, options: LibrarySort.albumOptions)
                                }
                            }
                )

                if viewModel.albums.isEmpty {
                    EmptyStateView(
                        title: "未找到专辑",
                        subtitle: "请尝试更换搜索关键词或清除筛选。"
                    )
                } else {
                    LazyVGrid(columns: columns, spacing: 18) {
                        ForEach(Array(viewModel.albums.enumerated()), id: \.element.id) { index, album in
                            NavigationLink(destination: albumDetailDestination(albumID: album.id)) {
                                AlbumGridCard(album: album)
                            }
                            .buttonStyle(.plain)
                            .onAppear {
                                if viewModel.shouldLoadMoreAlbums(at: index) {
                                    Task {
                                        await viewModel.loadMoreAlbums(sort: selectedSort, query: query)
                                    }
                                }
                            }
                        }
                    }
                }
            }
            .padding(28)
        }
        .task(id: "\(selectedSort.rawValue)|\(query)") {
            await viewModel.reloadAlbums(sort: selectedSort, query: query)
        }
        .navigationTitle("专辑")
    }
}

struct AlbumGridCard: View {
    let album: Album
    @State private var isHovered = false

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            RoundedRectangle(cornerRadius: 16)
                .fill(
                    LinearGradient(
                        colors: [
                            Color.accentColor.opacity(0.48),
                            Color.accentColor.opacity(0.16)
                        ],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )
                .frame(height: 184)
                .overlay(alignment: .topLeading) {
                    Capsule()
                        .fill(Color.white.opacity(0.14))
                        .frame(width: 72, height: 24)
                        .overlay(
                            Text(album.releaseDate ?? "未知年份")
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(.white.opacity(0.86))
                        )
                        .padding(14)
                }
                .overlay(
                    Image(systemName: "music.note.list")
                        .font(.system(size: 28, weight: .semibold))
                        .foregroundStyle(.white.opacity(0.72))
                )

            VStack(alignment: .leading, spacing: 6) {
                Text(album.name)
                    .font(.headline)
                    .foregroundStyle(SonicTheme.textPrimary)
                    .lineLimit(2)
                Text(album.artist)
                    .font(.subheadline)
                    .foregroundStyle(SonicTheme.textSecondary)
                    .lineLimit(1)

                HStack(spacing: 10) {
                    LibraryMetaLabel(title: "播放", value: "\(album.playCount ?? 0)")
                    if let genre = album.genre, !genre.isEmpty {
                        LibraryMetaLabel(title: "流派", value: genre)
                    }
                }
            }
        }
        .padding(12)
        .background(
            RoundedRectangle(cornerRadius: 20)
                .fill(isHovered ? Color.primary.opacity(0.04) : Color.clear)
        )
        .glassCard(cornerRadius: 20, isSimplified: true)
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
    }
}

private struct LibraryMetaLabel: View {
    let title: String
    let value: String

    var body: some View {
        HStack(spacing: 4) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(SonicTheme.textSecondary)
            Text(value)
                .font(.caption)
                .foregroundStyle(SonicTheme.textPrimary)
                .lineLimit(1)
        }
    }
}

struct TrackListView: View {
    @EnvironmentObject private var store: AppStore
    @ObservedObject var viewModel: LibraryViewModel
    private let externalSort: LibrarySort
    private let externalFilter: TrackFilter
    private let externalQuery: String
    private let showsInlineControls: Bool
    private let prefersCompactLayout: Bool
    @State private var selectedSort: LibrarySort
    @State private var selectedFilter: TrackFilter
    @State private var query: String
    @State private var selectedTrack: Track?

    init(
        viewModel: LibraryViewModel,
        sort: LibrarySort = .recent,
        filter: TrackFilter = .all,
        query: String = "",
        showsInlineControls: Bool = true,
        prefersCompactLayout: Bool = false
    ) {
        self.viewModel = viewModel
        self.externalSort = sort
        self.externalFilter = filter
        self.externalQuery = query
        self.showsInlineControls = showsInlineControls
        self.prefersCompactLayout = prefersCompactLayout
        _selectedSort = State(initialValue: sort)
        _selectedFilter = State(initialValue: filter)
        _query = State(initialValue: query)
    }

    var body: some View {
        let nowPlaying = store.nowPlaying

        GeometryReader { proxy in
            let columns = TrackColumnLayout(totalWidth: proxy.size.width - 40)
            let usesCompactRows = prefersCompactLayout || proxy.size.width < 560

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if showsInlineControls {
                        LibraryCollectionHeader(
                            title: "曲目",
                            subtitle: "长列表使用轻量表面，避免在滚动时叠加高成本材质。",
                            trailing: {
                                if usesCompactRows {
                                    VStack(alignment: .trailing, spacing: 10) {
                                        ToolbarSearchField(text: $query)
                                        HStack(spacing: 10) {
                                            TrackFilterMenu(selection: $selectedFilter)
                                            LibrarySortMenu(title: selectedSort.rawValue, selection: $selectedSort, options: LibrarySort.trackOptions)
                                        }
                                    }
                                } else {
                                    HStack(spacing: 10) {
                                        ToolbarSearchField(text: $query)
                                        TrackFilterMenu(selection: $selectedFilter)
                                        LibrarySortMenu(title: selectedSort.rawValue, selection: $selectedSort, options: LibrarySort.trackOptions)
                                    }
                                }
                            }
                        )
                    }

                    if viewModel.tracks.isEmpty {
                        EmptyStateView(
                            title: emptyTitle,
                            subtitle: emptySubtitle
                        )
                    } else {
                        if !usesCompactRows {
                            TrackHeaderRow(columns: columns)
                        }

                        LazyVStack(spacing: 8) {
                            ForEach(Array(viewModel.tracks.enumerated()), id: \.element.id) { index, track in
                                Group {
                                    if usesCompactRows {
                                        CompactTrackRowView(
                                            track: track,
                                            isNowPlaying: nowPlaying?.track == track.track &&
                                                nowPlaying?.artist == track.artist &&
                                                nowPlaying?.album == track.album,
                                            isFavorite: track.isFavorited || store.isFavorite(
                                                artist: track.artist,
                                                album: track.album,
                                                track: track.track,
                                                trackNumber: track.trackNumber,
                                                discNumber: track.discNumber
                                            )
                                        )
                                    } else {
                                        TrackRowView(
                                            track: track,
                                            columns: columns,
                                            isNowPlaying: nowPlaying?.track == track.track &&
                                                nowPlaying?.artist == track.artist &&
                                                nowPlaying?.album == track.album,
                                            isFavorite: track.isFavorited || store.isFavorite(
                                                artist: track.artist,
                                                album: track.album,
                                                track: track.track,
                                                trackNumber: track.trackNumber,
                                                discNumber: track.discNumber
                                            )
                                        )
                                    }
                                }
                                .contentShape(Rectangle())
                                .onTapGesture {
                                    selectedTrack = track
                                }
                                .onAppear {
                                    if viewModel.shouldLoadMoreTracks(at: index) {
                                        Task {
                                            await viewModel.loadMoreTracks(
                                                sort: selectedSort,
                                                filter: selectedFilter,
                                                query: query
                                            )
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
                .padding(.horizontal, 20)
                .padding(.top, showsInlineControls ? 24 : 16)
                .padding(.bottom, 24)
            }
        }
        .task(id: "\(selectedSort.rawValue)|\(selectedFilter.rawValue)|\(query)") {
            await viewModel.reloadTracks(sort: selectedSort, filter: selectedFilter, query: query)
        }
        .onChange(of: externalSort) { _, value in
            selectedSort = value
        }
        .onChange(of: externalFilter) { _, value in
            selectedFilter = value
        }
        .onChange(of: externalQuery) { _, value in
            query = value
        }
        .navigationDestination(item: $selectedTrack) { track in
            TrackDetailView(track: track)
        }
    }

    private var emptyTitle: String {
        switch selectedFilter {
        case .favorites:
            return "暂无收藏"
        case .unreported:
            return "已全部完成"
        case .all:
            return "暂无曲目"
        }
    }

    private var emptySubtitle: String {
        switch selectedFilter {
        case .favorites:
            return "收藏曲目后会显示在这里。"
        case .unreported:
            return "暂无待上报曲目。"
        case .all:
            return "开始播放后资料库会逐步丰富。"
        }
    }
}

private struct LibraryCollectionHeader<Trailing: View>: View {
    let title: String
    let subtitle: String
    let trailing: Trailing

    init(
        title: String,
        subtitle: String,
        @ViewBuilder trailing: () -> Trailing
    ) {
        self.title = title
        self.subtitle = subtitle
        self.trailing = trailing()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .top, spacing: 16) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(title)
                        .font(.title3)
                        .fontWeight(.semibold)
                    Text(subtitle)
                        .font(.subheadline)
                        .foregroundStyle(SonicTheme.textSecondary)
                }
                Spacer(minLength: 12)
                trailing
            }
        }
    }
}

private struct TrackColumnLayout {
    let numberWidth: CGFloat = 36
    let durationWidth: CGFloat = 60
    let actionWidth: CGFloat = 32
    let trackWidth: CGFloat
    let artistWidth: CGFloat
    let albumWidth: CGFloat

    init(totalWidth: CGFloat) {
        let availableWidth = max(totalWidth, 320)
        let flexibleWidth = max(availableWidth - numberWidth - durationWidth - actionWidth - 88, 180)
        trackWidth = flexibleWidth * 0.42
        artistWidth = flexibleWidth * 0.24
        albumWidth = flexibleWidth * 0.34
    }
}

private struct TrackHeaderRow: View {
    let columns: TrackColumnLayout

    var body: some View {
        HStack(spacing: 16) {
            Text("#")
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: columns.numberWidth, alignment: .leading)
            Text("曲目")
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: columns.trackWidth, alignment: .leading)
            Text("艺术家")
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: columns.artistWidth, alignment: .leading)
            Text("专辑")
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: columns.albumWidth, alignment: .leading)
            Spacer()
            Text("时长")
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: columns.durationWidth, alignment: .trailing)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
        .glassCard(cornerRadius: 14, isSimplified: true)
    }
}

private struct TrackRowView: View {
    @EnvironmentObject private var store: AppStore
    let track: Track
    let columns: TrackColumnLayout
    let isNowPlaying: Bool
    let isFavorite: Bool
    @State private var isHovered = false

    var body: some View {
        HStack(spacing: 16) {
            Text(trackNumber)
                .font(.caption)
                .foregroundStyle(isNowPlaying ? Color.accentColor : Color.secondary)
                .frame(width: columns.numberWidth, alignment: .leading)

            Text(track.track)
                .font(.body)
                .fontWeight(isNowPlaying ? .semibold : .regular)
                .lineLimit(1)
                .frame(width: columns.trackWidth, alignment: .leading)

            Text(track.artist)
                .font(.body)
                .foregroundStyle(isNowPlaying ? SonicTheme.textPrimary : SonicTheme.textSecondary)
                .lineLimit(1)
                .frame(width: columns.artistWidth, alignment: .leading)

            Text(track.album)
                .font(.body)
                .foregroundStyle(SonicTheme.textSecondary)
                .lineLimit(1)
                .frame(width: columns.albumWidth, alignment: .leading)

            Spacer(minLength: 12)

            TrackRowActions(
                isVisible: isHovered,
                isFavorite: isFavorite,
                onFavorite: {
                    Task {
                        await store.toggleFavorite(
                            artist: track.artist,
                            album: track.album,
                            track: track.track,
                            trackNumber: track.trackNumber,
                            discNumber: track.discNumber
                        )
                    }
                }
            )

            Text(formatDuration(track.duration))
                .font(.caption)
                .foregroundStyle(isNowPlaying ? Color.accentColor : Color.secondary)
                .frame(width: columns.durationWidth, alignment: .trailing)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(
            RoundedRectangle(cornerRadius: 14)
                .fill(
                    isNowPlaying ? Color.accentColor.opacity(0.08) :
                    (isHovered ? Color.primary.opacity(0.045) : SonicTheme.card.opacity(0.55))
                )
        )
        .overlay(
            RoundedRectangle(cornerRadius: 14)
                .stroke(Color.white.opacity(isHovered || isNowPlaying ? 0.16 : 0.08), lineWidth: 1)
        )
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
    }

    private var trackNumber: String {
        if let trackNumber = track.trackNumber {
            return "\(trackNumber)"
        }
        return "—"
    }

    private func formatDuration(_ duration: Int64?) -> String {
        guard let duration else { return "--:--" }
        let totalSeconds = Int(duration)
        let minutes = totalSeconds / 60
        let seconds = totalSeconds % 60
        return String(format: "%02d:%02d", minutes, seconds)
    }
}

struct TrackRowActions: View {
    let isVisible: Bool
    let isFavorite: Bool
    let onFavorite: () -> Void

    var body: some View {
        HStack(spacing: 8) {
            TrackRowActionButton(symbol: isFavorite ? "heart.fill" : "heart", label: "收藏", action: onFavorite)
        }
        .opacity(isVisible ? 1 : 0)
        .offset(x: isVisible ? 0 : 4)
        .animation(.easeInOut(duration: 0.12), value: isVisible)
    }
}

struct CompactTrackRowView: View {
    @EnvironmentObject private var store: AppStore
    let track: Track
    let isNowPlaying: Bool
    let isFavorite: Bool
    @State private var isHovered = false

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 8) {
                    Text(track.track)
                        .font(.body.weight(isNowPlaying ? .semibold : .regular))
                        .foregroundStyle(isNowPlaying ? Color.accentColor : SonicTheme.textPrimary)
                        .lineLimit(1)

                    if isFavorite {
                        Image(systemName: "heart.fill")
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.pink)
                    }
                }

                Text("\(track.artist) · \(track.album)")
                    .font(.subheadline)
                    .foregroundStyle(SonicTheme.textSecondary)
                    .lineLimit(1)

                HStack(spacing: 10) {
                    if let trackNumber = track.trackNumber {
                        Text("Track \(trackNumber)")
                    }
                    if let discNumber = track.discNumber {
                        Text("Disc \(discNumber)")
                    }
                    Text(formatDuration(track.duration))
                }
                .font(.caption)
                .foregroundStyle(isNowPlaying ? Color.accentColor : Color.secondary)
            }

            Spacer(minLength: 10)

            TrackRowActions(
                isVisible: isHovered || isFavorite,
                isFavorite: isFavorite,
                onFavorite: {
                    Task {
                        await store.toggleFavorite(
                            artist: track.artist,
                            album: track.album,
                            track: track.track,
                            trackNumber: track.trackNumber,
                            discNumber: track.discNumber
                        )
                    }
                }
            )
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 14)
        .background(
            RoundedRectangle(cornerRadius: 16)
                .fill(
                    isNowPlaying ? Color.accentColor.opacity(0.08) :
                    (isHovered ? Color.primary.opacity(0.045) : SonicTheme.card.opacity(0.55))
                )
        )
        .overlay(
            RoundedRectangle(cornerRadius: 16)
                .stroke(Color.white.opacity(isHovered || isNowPlaying ? 0.16 : 0.08), lineWidth: 1)
        )
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
    }

    private func formatDuration(_ duration: Int64?) -> String {
        guard let duration else { return "--:--" }
        let totalSeconds = Int(duration)
        let minutes = totalSeconds / 60
        let seconds = totalSeconds % 60
        return String(format: "%02d:%02d", minutes, seconds)
    }
}

struct TrackRowActionButton: View {
    let symbol: String
    let label: String
    let action: () -> Void
    @State private var isHovered = false

    var body: some View {
        Button(action: action) {
            Image(systemName: symbol)
                .font(.system(size: 12, weight: .semibold))
                .frame(width: 26, height: 26)
                .background(
                    RoundedRectangle(cornerRadius: 8)
                        .fill(isHovered ? Color.primary.opacity(0.12) : Color.primary.opacity(0.06))
                )
        }
        .buttonStyle(.plain)
        .buttonStyle(PressableButtonStyle())
        .help(label)
        .onHover { hovering in
            withAnimation(.easeInOut(duration: 0.12)) {
                isHovered = hovering
            }
        }
    }
}

struct InsightListView: View {
    @EnvironmentObject private var store: AppStore
    @ObservedObject var viewModel: LibraryViewModel

    var body: some View {
        List {
            Section(header: Text("音眸")) {
                ForEach(viewModel.insights) { insight in
                    NavigationLink(destination: InsightDetailView(insight: insight)) {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(insight.track)
                                .font(.system(size: 14, weight: .semibold))
                            Text("\(insight.artist) · \(insight.album)")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                        .padding(.vertical, 4)
                    }
                    .onAppear {
                        if insight.id == viewModel.insights.last?.id, let server = store.currentServer {
                            Task { await viewModel.loadMoreInsights(using: server) }
                        }
                    }
                }
            }
        }
        .navigationTitle("音眸")
    }
}

struct UnscrobbledListView: View {
    @EnvironmentObject private var store: AppStore
    @ObservedObject var viewModel: LibraryViewModel

    var body: some View {
        List {
            Section(header: Text("未上报")) {
                ForEach(viewModel.unscrobbled) { record in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(record.track)
                            .font(.system(size: 14, weight: .semibold))
                        Text("\(record.artist) · \(record.album)")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    .padding(.vertical, 4)
                    .onAppear {
                        if record.id == viewModel.unscrobbled.last?.id, let server = store.currentServer {
                            Task { await viewModel.loadMoreUnscrobbled(using: server) }
                        }
                    }
                }
            }
        }
        .navigationTitle("未上报")
    }
}

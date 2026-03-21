import Foundation
import OSLog

@MainActor
final class LibraryViewModel: ObservableObject {
    enum SyncState {
        case idle
        case syncing
        case refreshed(Date)
    }

    @Published var albums: [Album] = []
    @Published var tracks: [Track] = []
    @Published var insights: [InsightSummary] = []
    @Published var albumInsights: [InsightSummary] = []
    @Published var unscrobbled: [UnscrobbledRecord] = []
    @Published var unscrobbledCount: Int?
    @Published var isLoading: Bool = false
    @Published var isRefreshing: Bool = false
    @Published var errorMessage: String?
    @Published var syncState: SyncState = .idle
    @Published private(set) var favoriteTrackCount: Int = 0

    private let pageSize = 30
    private let prefetchMargin = 8
    private let indexStore = LibraryIndexStore()
    private lazy var syncService = LibrarySyncService(indexStore: indexStore)
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "LibraryViewModel")

    private var insightOffset = 0
    private var insightTotal = 0
    private var albumInsightOffset = 0
    private var albumInsightTotal = 0
    private var isIndexStoreReady = false
    private var isLoadingMoreAlbums = false
    private var isLoadingMoreTracks = false
    private var isLoadingMoreInsights = false
    private var isLoadingMoreAlbumInsights = false
    private var albumTotal = 0
    private var trackTotal = 0
    private var lastAlbumPrefetchIndex = -1
    private var lastTrackPrefetchIndex = -1
    private var lastInsightPrefetchIndex = -1
    private var lastAlbumInsightPrefetchIndex = -1
    private var currentAlbumSort: LibrarySort = .recent
    private var currentAlbumQuery: String = ""
    private var currentTrackSort: LibrarySort = .recent
    private var currentTrackFilter: TrackFilter = .all
    private var currentTrackQuery: String = ""
    private var favoriteObserver: NSObjectProtocol?
    private var currentServerURL: URL?
    private var hasLoadedAlbums = false
    private var hasLoadedTracks = false
    private var hasLoadedTrackInsights = false
    private var hasLoadedAlbumInsights = false

    init() {
        favoriteObserver = NotificationCenter.default.addObserver(
            forName: .libraryFavoriteDidChange,
            object: nil,
            queue: .main
        ) { [weak self] notification in
            guard let change = notification.object as? LibraryFavoriteChange else { return }
            Task {
                await self?.handleFavoriteChange(change)
            }
        }
    }

    deinit {
        if let favoriteObserver {
            NotificationCenter.default.removeObserver(favoriteObserver)
        }
    }

    func load(using server: ServerConfig) async {
        let isServerChanged = currentServerURL != server.baseURL
        if isServerChanged {
            currentServerURL = server.baseURL
            resetSyncStateForServerChange()
            logger.info("切换服务端，重置资料库状态 \(server.displayName, privacy: .public)")
        }

        do {
            logger.info("开始加载本地资料库索引")
            try await ensureIndexStore()
            await reloadAlbums(sort: currentAlbumSort, query: currentAlbumQuery)
            await reloadTracks(sort: currentTrackSort, filter: currentTrackFilter, query: currentTrackQuery)
        } catch {
            logger.error("load failed: \(error.localizedDescription, privacy: .public)")
            errorMessage = "本地资料库初始化失败"
        }

        isLoading = albums.isEmpty && tracks.isEmpty
        errorMessage = nil
        insightOffset = 0
        albumInsightOffset = 0
        logger.info("本地资料库初始加载完成，准备后台同步")
        await runBackgroundRefresh(using: server, forceFullSync: isServerChanged)
        isLoading = false
    }

    func refresh(using server: ServerConfig) async {
        logger.debug("触发资料库刷新")
        await runBackgroundRefresh(using: server, forceFullSync: false)
    }

    func reloadAlbums(sort: LibrarySort, query: String, force: Bool = false) async {
        if !force, hasLoadedAlbums, sort == currentAlbumSort, query == currentAlbumQuery {
            logger.debug("专辑重载跳过，参数未变化")
            return
        }
        currentAlbumSort = sort
        currentAlbumQuery = query
        logger.info("开始重载专辑列表，排序 \(sort.rawValue, privacy: .public)，关键词长度 \(query.count, privacy: .public)")
        let nextAlbumTotal = (try? await indexStore.countAlbums(keyword: query)) ?? 0
        guard !Task.isCancelled else {
            logger.debug("专辑重载在计数后被取消")
            return
        }
        let nextAlbums = (try? await indexStore.queryAlbums(sort: sort, keyword: query, limit: pageSize, offset: 0)) ?? []
        guard !Task.isCancelled else {
            logger.debug("专辑重载在查询后被取消")
            return
        }
        albumTotal = nextAlbumTotal
        albums = nextAlbums
        lastAlbumPrefetchIndex = -1
        hasLoadedAlbums = true
        logger.info("专辑列表重载完成，总数 \(self.albumTotal, privacy: .public)，当前页数量 \(self.albums.count, privacy: .public)")
    }

    func loadMoreAlbums(sort: LibrarySort, query: String) async {
        guard !isLoadingMoreAlbums else { return }
        if sort != currentAlbumSort || query != currentAlbumQuery {
            logger.debug("专辑加载更多时发现参数变化，先回到重载")
            await reloadAlbums(sort: sort, query: query)
            return
        }
        guard albums.count < albumTotal else { return }

        isLoadingMoreAlbums = true
        let nextOffset = albums.count
        logger.debug("继续加载更多专辑，偏移 \(nextOffset, privacy: .public)")
        if let page = try? await indexStore.queryAlbums(sort: sort, keyword: query, limit: pageSize, offset: nextOffset) {
            guard !Task.isCancelled else {
                logger.debug("追加专辑页在查询后被取消")
                isLoadingMoreAlbums = false
                return
            }
            albums.append(contentsOf: page)
            logger.debug("追加专辑页完成，本次数量 \(page.count, privacy: .public)")
        }
        isLoadingMoreAlbums = false
    }

    func reloadTracks(sort: LibrarySort, filter: TrackFilter, query: String, force: Bool = false) async {
        if !force, hasLoadedTracks, sort == currentTrackSort, filter == currentTrackFilter, query == currentTrackQuery {
            logger.debug("曲目重载跳过，参数未变化")
            return
        }
        currentTrackSort = sort
        currentTrackFilter = filter
        currentTrackQuery = query
        logger.info("开始重载曲目列表，排序 \(sort.rawValue, privacy: .public)，筛选 \(filter.rawValue, privacy: .public)，关键词长度 \(query.count, privacy: .public)")
        let nextTrackTotal = (try? await indexStore.countTracks(filter: filter, keyword: query)) ?? 0
        guard !Task.isCancelled else {
            logger.debug("曲目重载在计数后被取消")
            return
        }
        let nextTracks = (try? await indexStore.queryTracks(sort: sort, filter: filter, keyword: query, limit: pageSize, offset: 0)) ?? []
        guard !Task.isCancelled else {
            logger.debug("曲目重载在查询后被取消")
            return
        }
        trackTotal = nextTrackTotal
        tracks = nextTracks
        favoriteTrackCount = nextTracks.filter(\.isFavorited).count
        lastTrackPrefetchIndex = -1
        hasLoadedTracks = true
        logger.info("曲目列表重载完成，总数 \(self.trackTotal, privacy: .public)，当前页数量 \(self.tracks.count, privacy: .public)")
    }

    func loadMoreTracks(sort: LibrarySort, filter: TrackFilter, query: String) async {
        guard !isLoadingMoreTracks else { return }
        if sort != currentTrackSort || filter != currentTrackFilter || query != currentTrackQuery {
            logger.debug("曲目加载更多时发现参数变化，先回到重载")
            await reloadTracks(sort: sort, filter: filter, query: query)
            return
        }
        guard tracks.count < trackTotal else { return }

        isLoadingMoreTracks = true
        let nextOffset = tracks.count
        logger.debug("继续加载更多曲目，偏移 \(nextOffset, privacy: .public)")
        if let page = try? await indexStore.queryTracks(
            sort: sort,
            filter: filter,
            keyword: query,
            limit: pageSize,
            offset: nextOffset
        ) {
            guard !Task.isCancelled else {
                logger.debug("追加曲目页在查询后被取消")
                isLoadingMoreTracks = false
                return
            }
            tracks.append(contentsOf: page)
            favoriteTrackCount = tracks.filter(\.isFavorited).count
            logger.debug("追加曲目页完成，本次数量 \(page.count, privacy: .public)")
        }
        isLoadingMoreTracks = false
    }

    var albumCountText: String {
        "\(albumTotal)"
    }

    var trackCountText: String {
        "\(trackTotal)"
    }

    var favoriteCountText: String {
        "\(favoriteTrackCount)"
    }

    var pendingScrobbleCountText: String {
        unscrobbledCount.map(String.init) ?? "—"
    }

    var syncStatusText: String {
        switch syncState {
        case .idle:
            return "等待同步"
        case .syncing:
            return "同步中"
        case .refreshed(let date):
            return "已更新 \(Self.syncFormatter.localizedString(for: date, relativeTo: Date()))"
        }
    }

    var isSyncing: Bool {
        if case .syncing = syncState {
            return true
        }
        return false
    }

    func ensureInsightsLoaded(using server: ServerConfig, targetType: InsightTargetType = .track) async {
        if targetType == .album {
            guard !hasLoadedAlbumInsights else { return }
        } else {
            guard !hasLoadedTrackInsights else { return }
        }
        await reloadInsights(using: server, targetType: targetType)
    }

    func reloadInsights(using server: ServerConfig, targetType: InsightTargetType = .track) async {
        guard !isLoadingMoreInsightsFlag(for: targetType) else { return }
        setLoadingMoreInsightsFlag(true, for: targetType)
        do {
            let page = try await fetchInsightsPage(using: server, offset: 0, targetType: targetType)
            guard !Task.isCancelled else {
                setLoadingMoreInsightsFlag(false, for: targetType)
                return
            }
            applyInsights(page.insights, total: Int(page.total), targetType: targetType, resetPagination: true)
        } catch {
            logger.error("reload insights failed: \(error.localizedDescription, privacy: .public)")
        }
        setLoadingMoreInsightsFlag(false, for: targetType)
    }

    func loadMoreInsights(using server: ServerConfig, targetType: InsightTargetType = .track) async {
        guard !isLoadingMoreInsightsFlag(for: targetType) else { return }
        let currentItems = insights(for: targetType)
        guard currentItems.count < insightTotalCount(for: targetType) else { return }
        setLoadingMoreInsightsFlag(true, for: targetType)
        let nextOffset = currentItems.count
        do {
            let page = try await fetchInsightsPage(using: server, offset: nextOffset, targetType: targetType)
            guard !Task.isCancelled else {
                setLoadingMoreInsightsFlag(false, for: targetType)
                return
            }
            if !page.insights.isEmpty {
                appendInsights(page.insights, targetType: targetType)
            }
        } catch {
            // ignore pagination errors
        }
        setLoadingMoreInsightsFlag(false, for: targetType)
    }

    func loadMoreUnscrobbled(using server: ServerConfig) async {
        await reloadUnscrobbled(using: server)
    }

    func shouldLoadMoreAlbums(at index: Int) -> Bool {
        guard index >= max(albums.count - prefetchMargin, 0) else { return false }
        guard index != lastAlbumPrefetchIndex else { return false }
        guard albums.count < albumTotal else { return false }
        lastAlbumPrefetchIndex = index
        return true
    }

    func shouldLoadMoreTracks(at index: Int) -> Bool {
        guard index >= max(tracks.count - prefetchMargin, 0) else { return false }
        guard index != lastTrackPrefetchIndex else { return false }
        guard tracks.count < trackTotal else { return false }
        lastTrackPrefetchIndex = index
        return true
    }

    func shouldLoadMoreInsights(at index: Int, targetType: InsightTargetType = .track) -> Bool {
        let currentItems = insights(for: targetType)
        guard index >= max(currentItems.count - prefetchMargin, 0) else { return false }
        guard index != lastInsightPrefetchIndexForTarget(for: targetType) else { return false }
        guard currentItems.count < insightTotalCount(for: targetType) else { return false }
        setLastInsightPrefetchIndex(index, for: targetType)
        return true
    }

    private func insights(for targetType: InsightTargetType) -> [InsightSummary] {
        targetType == .album ? albumInsights : insights
    }

    private func insightTotalCount(for targetType: InsightTargetType) -> Int {
        targetType == .album ? albumInsightTotal : insightTotal
    }

    private func isLoadingMoreInsightsFlag(for targetType: InsightTargetType) -> Bool {
        targetType == .album ? isLoadingMoreAlbumInsights : isLoadingMoreInsights
    }

    private func setLoadingMoreInsightsFlag(_ value: Bool, for targetType: InsightTargetType) {
        if targetType == .album {
            isLoadingMoreAlbumInsights = value
        } else {
            isLoadingMoreInsights = value
        }
    }

    private func lastInsightPrefetchIndexForTarget(for targetType: InsightTargetType) -> Int {
        targetType == .album ? lastAlbumInsightPrefetchIndex : lastInsightPrefetchIndex
    }

    private func setLastInsightPrefetchIndex(_ index: Int, for targetType: InsightTargetType) {
        if targetType == .album {
            lastAlbumInsightPrefetchIndex = index
        } else {
            lastInsightPrefetchIndex = index
        }
    }

    private func applyInsights(
        _ pageInsights: [InsightSummary],
        total: Int,
        targetType: InsightTargetType,
        resetPagination: Bool
    ) {
        if targetType == .album {
            albumInsightTotal = total
            albumInsightOffset = 0
            albumInsights = pageInsights
            hasLoadedAlbumInsights = true
            if resetPagination {
                lastAlbumInsightPrefetchIndex = -1
            }
        } else {
            insightTotal = total
            insightOffset = 0
            insights = pageInsights
            hasLoadedTrackInsights = true
            if resetPagination {
                lastInsightPrefetchIndex = -1
            }
        }
    }

    private func appendInsights(_ pageInsights: [InsightSummary], targetType: InsightTargetType) {
        if targetType == .album {
            albumInsightOffset = albumInsights.count
            albumInsights.append(contentsOf: pageInsights)
        } else {
            insightOffset = insights.count
            insights.append(contentsOf: pageInsights)
        }
    }

    private func ensureIndexStore() async throws {
        guard !isIndexStoreReady else { return }
        try await indexStore.setup()
        isIndexStoreReady = true
    }

    private func runBackgroundRefresh(using server: ServerConfig, forceFullSync: Bool) async {
        guard !isRefreshing else { return }
        isRefreshing = true
        syncState = .syncing
        logger.info("开始后台同步，强制全量 \(forceFullSync, privacy: .public)")

        do {
            try await ensureIndexStore()
            try await syncService.sync(using: server, forceFullSync: forceFullSync)
            await reloadAlbums(sort: currentAlbumSort, query: currentAlbumQuery, force: true)
            await reloadTracks(sort: currentTrackSort, filter: currentTrackFilter, query: currentTrackQuery, force: true)

            do {
                let count = try await fetchUnscrobbledCount(using: server)
                unscrobbledCount = count
            } catch {
                logger.error("refresh unscrobbled failed: \(error.localizedDescription, privacy: .public)")
                // 忽略附加面板刷新失败，避免阻断资料库主列表更新。
            }

            syncState = .refreshed(Date())
            errorMessage = nil
            logger.info("后台同步完成")
        } catch {
            logger.error("background refresh failed: \(error.localizedDescription, privacy: .public)")
            if albums.isEmpty && tracks.isEmpty {
                errorMessage = "资料库数据同步失败"
            }
            if case .refreshed = syncState {
                // 保留已有状态文案
            } else {
                syncState = .idle
            }
        }

        isRefreshing = false
    }

    private func resetSyncStateForServerChange() {
        syncState = .idle
        insights = []
        albumInsights = []
        insightTotal = 0
        albumInsightTotal = 0
        insightOffset = 0
        albumInsightOffset = 0
        lastInsightPrefetchIndex = -1
        lastAlbumInsightPrefetchIndex = -1
        hasLoadedTrackInsights = false
        hasLoadedAlbumInsights = false
        unscrobbled = []
        unscrobbledCount = nil
        hasLoadedAlbums = false
        hasLoadedTracks = false
    }

    func fetchInsightDetail(using server: ServerConfig, id: Int64) async throws -> Insight {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(path: APIPath.insightDetail(id: id))
    }

    private func fetchInsightsPage(
        using server: ServerConfig,
        offset: Int,
        targetType: InsightTargetType
    ) async throws -> PaginatedInsights {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(
            path: APIPath.insights,
            queryItems: [
                URLQueryItem(name: "limit", value: "\(pageSize)"),
                URLQueryItem(name: "offset", value: "\(offset)"),
                URLQueryItem(name: "analysis_target_type", value: targetType.rawValue)
            ]
        )
    }

    func reloadUnscrobbled(using server: ServerConfig) async {
        logger.debug("开始重载未上报曲目")
        do {
            let all = try await fetchAllUnscrobbled(using: server)
            try await indexStore.replaceReportedStatus(using: all)
            unscrobbled = all
            logger.info("未上报曲目重载完成，数量 \(all.count, privacy: .public)")
        } catch {
            logger.error("reload unscrobbled failed: \(error.localizedDescription, privacy: .public)")
        }
    }

    private func fetchUnscrobbledPage(using server: ServerConfig, offset: Int) async throws -> [UnscrobbledRecord] {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(
            path: APIPath.unscrobbled,
            queryItems: [
                URLQueryItem(name: "limit", value: "100"),
                URLQueryItem(name: "offset", value: "\(offset)")
            ]
        )
    }

    private func fetchAllUnscrobbled(using server: ServerConfig) async throws -> [UnscrobbledRecord] {
        var records: [UnscrobbledRecord] = []
        var offset = 0

        while true {
            let page = try await fetchUnscrobbledPage(using: server, offset: offset)
            if page.isEmpty {
                break
            }
            records.append(contentsOf: page)
            if page.count < 100 {
                break
            }
            offset += page.count
        }

        return records
    }

    private func fetchUnscrobbledCount(using server: ServerConfig) async throws -> Int {
        let client = APIClient(baseURL: server.baseURL)
        let response: UnscrobbledCountResponse = try await client.getJSON(path: APIPath.unscrobbledCount)
        return response.count
    }

    private func handleFavoriteChange(_ change: LibraryFavoriteChange) async {
        logger.debug("收到收藏变更，准备刷新本地索引")
        do {
            try await indexStore.updateTrackFavoriteStatus(
                artist: change.artist,
                album: change.album,
                track: change.track,
                trackNumber: change.trackNumber,
                discNumber: change.discNumber,
                appleMusic: change.appleMusic,
                lastFm: change.lastfm
            )
            await reloadTracks(sort: currentTrackSort, filter: currentTrackFilter, query: currentTrackQuery, force: true)
            logger.debug("收藏变更已回写到本地索引")
        } catch {
            // 忽略本地收藏状态回写失败，交给下一次同步兜底
        }
    }

    private static let syncFormatter: RelativeDateTimeFormatter = {
        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .short
        return formatter
    }()
}

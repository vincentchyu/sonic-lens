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
    @Published var insights: [Insight] = []
    @Published var unscrobbled: [UnscrobbledRecord] = []
    @Published var isLoading: Bool = false
    @Published var isRefreshing: Bool = false
    @Published var errorMessage: String?
    @Published var syncState: SyncState = .idle

    private let pageSize = 30
    private let prefetchMargin = 8
    private let indexStore = LibraryIndexStore()
    private lazy var syncService = LibrarySyncService(indexStore: indexStore)
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "LibraryViewModel")

    private var insightOffset = 0
    private var isIndexStoreReady = false
    private var isLoadingMoreAlbums = false
    private var isLoadingMoreTracks = false
    private var albumTotal = 0
    private var trackTotal = 0
    private var lastAlbumPrefetchIndex = -1
    private var lastTrackPrefetchIndex = -1
    private var currentAlbumSort: LibrarySort = .recent
    private var currentAlbumQuery: String = ""
    private var currentTrackSort: LibrarySort = .recent
    private var currentTrackFilter: TrackFilter = .all
    private var currentTrackQuery: String = ""
    private var favoriteObserver: NSObjectProtocol?
    private var currentServerURL: URL?

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
        }

        do {
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
        await runBackgroundRefresh(using: server, forceFullSync: isServerChanged)
        isLoading = false
    }

    func refresh(using server: ServerConfig) async {
        await runBackgroundRefresh(using: server, forceFullSync: false)
    }

    func reloadAlbums(sort: LibrarySort, query: String) async {
        currentAlbumSort = sort
        currentAlbumQuery = query
        albumTotal = (try? await indexStore.countAlbums(keyword: query)) ?? 0
        albums = (try? await indexStore.queryAlbums(sort: sort, keyword: query, limit: pageSize, offset: 0)) ?? []
        lastAlbumPrefetchIndex = -1
    }

    func loadMoreAlbums(sort: LibrarySort, query: String) async {
        guard !isLoadingMoreAlbums else { return }
        if sort != currentAlbumSort || query != currentAlbumQuery {
            await reloadAlbums(sort: sort, query: query)
            return
        }
        guard albums.count < albumTotal else { return }

        isLoadingMoreAlbums = true
        let nextOffset = albums.count
        if let page = try? await indexStore.queryAlbums(sort: sort, keyword: query, limit: pageSize, offset: nextOffset) {
            albums.append(contentsOf: page)
        }
        isLoadingMoreAlbums = false
    }

    func reloadTracks(sort: LibrarySort, filter: TrackFilter, query: String) async {
        currentTrackSort = sort
        currentTrackFilter = filter
        currentTrackQuery = query
        trackTotal = (try? await indexStore.countTracks(filter: filter, keyword: query)) ?? 0
        tracks = (try? await indexStore.queryTracks(sort: sort, filter: filter, keyword: query, limit: pageSize, offset: 0)) ?? []
        lastTrackPrefetchIndex = -1
    }

    func loadMoreTracks(sort: LibrarySort, filter: TrackFilter, query: String) async {
        guard !isLoadingMoreTracks else { return }
        if sort != currentTrackSort || filter != currentTrackFilter || query != currentTrackQuery {
            await reloadTracks(sort: sort, filter: filter, query: query)
            return
        }
        guard tracks.count < trackTotal else { return }

        isLoadingMoreTracks = true
        let nextOffset = tracks.count
        if let page = try? await indexStore.queryTracks(
            sort: sort,
            filter: filter,
            keyword: query,
            limit: pageSize,
            offset: nextOffset
        ) {
            tracks.append(contentsOf: page)
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
        "\(tracks.filter(\.isFavorited).count)"
    }

    var pendingScrobbleCountText: String {
        "\(unscrobbled.count)"
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

    func ensureInsightsLoaded(using server: ServerConfig) async {
        guard insights.isEmpty else { return }
        await reloadInsights(using: server)
    }

    func reloadInsights(using server: ServerConfig) async {
        insightOffset = 0
        do {
            insights = try await fetchInsights(using: server, offset: 0)
        } catch {
            logger.error("reload insights failed: \(error.localizedDescription, privacy: .public)")
        }
    }

    func loadMoreInsights(using server: ServerConfig) async {
        insightOffset += pageSize
        do {
            let page = try await fetchInsights(using: server, offset: insightOffset)
            if !page.isEmpty {
                insights.append(contentsOf: page)
            }
        } catch {
            // ignore pagination errors
        }
    }

    func loadMoreUnscrobbled(using server: ServerConfig) async {
        do {
            let all = try await fetchAllUnscrobbled(using: server)
            if all.count > unscrobbled.count {
                unscrobbled = all
            }
        } catch {
            // ignore refresh errors
        }
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

    private func ensureIndexStore() async throws {
        guard !isIndexStoreReady else { return }
        try await indexStore.setup()
        isIndexStoreReady = true
    }

    private func runBackgroundRefresh(using server: ServerConfig, forceFullSync: Bool) async {
        guard !isRefreshing else { return }
        isRefreshing = true
        syncState = .syncing

        do {
            try await ensureIndexStore()
            try await syncService.sync(using: server, forceFullSync: forceFullSync)
            await reloadAlbums(sort: currentAlbumSort, query: currentAlbumQuery)
            await reloadTracks(sort: currentTrackSort, filter: currentTrackFilter, query: currentTrackQuery)

            do {
                let allUnscrobbled = try await fetchAllUnscrobbled(using: server)
                try await indexStore.replaceReportedStatus(using: allUnscrobbled)
                unscrobbled = allUnscrobbled
            } catch {
                logger.error("refresh unscrobbled failed: \(error.localizedDescription, privacy: .public)")
                // 忽略附加面板刷新失败，避免阻断资料库主列表更新。
            }

            syncState = .refreshed(Date())
            errorMessage = nil
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
        unscrobbled = []
    }

    private func fetchInsights(using server: ServerConfig, offset: Int) async throws -> [Insight] {
        let client = APIClient(baseURL: server.baseURL)
        let page: PaginatedInsights = try await client.getJSON(
            path: APIPath.insights,
            queryItems: [
                URLQueryItem(name: "limit", value: "\(pageSize)"),
                URLQueryItem(name: "offset", value: "\(offset)")
            ]
        )
        return page.insights
    }

    private func fetchAllUnscrobbled(using server: ServerConfig) async throws -> [UnscrobbledRecord] {
        let client = APIClient(baseURL: server.baseURL)
        var records: [UnscrobbledRecord] = []
        var offset = 0

        while true {
            let page: [UnscrobbledRecord] = try await client.getJSON(
                path: APIPath.unscrobbled,
                queryItems: [
                    URLQueryItem(name: "limit", value: "100"),
                    URLQueryItem(name: "offset", value: "\(offset)")
                ]
            )
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

    private func handleFavoriteChange(_ change: LibraryFavoriteChange) async {
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
            await reloadTracks(sort: currentTrackSort, filter: currentTrackFilter, query: currentTrackQuery)
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

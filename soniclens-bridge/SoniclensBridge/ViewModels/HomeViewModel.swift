import Foundation

@MainActor
final class HomeViewModel: ObservableObject {
    @Published var stats: DashboardStats?
    @Published var topArtistsByPlays: [TopArtist] = []
    @Published var topArtistsByTracks: [TopArtist] = []
    @Published var playSourceCounts: [String: Int64] = [:]
    @Published var topAlbums: [TopAlbum] = []
    @Published var topGenres: [TopGenre] = []
    @Published var topTracks: [TopTrack] = []
    @Published var recentPlays: [RecentPlayRecord] = []
    @Published var trendPoints: [TrendPoint] = []
    @Published var hourlyData: [HourlyData] = []
    @Published private(set) var hotModulePresentation: HomeHotModulePresentation = .empty
    @Published private(set) var trendSnapshot: HomeTrendSnapshot = .empty
    
    // 过滤与搜索状态
    @Published var selectedTrendRange: Int
    @Published var topAlbumsDays: Int = 30 // Web 默认通常是 30 或 7
    @Published var rankingPeriod: String = "month" // 保留给旧排行页兼容，不再驱动首页热门曲目
    @Published var rankingSearchKeyword: String = ""
    
    @Published var isLoading: Bool = false
    @Published var errorMessage: String?

    private let cache = SnapshotCache()
    private let defaultTrendRange: Int
    private var loadRevision: Int = 0
    private var activeLoadRevision: Int = 0
    private var activeLoadServer: ServerConfig?
    private var queuedLoadServer: ServerConfig?

    init(defaultTrendRange: Int = 90) {
        self.defaultTrendRange = defaultTrendRange
        self.selectedTrendRange = defaultTrendRange
    }

    func load(using server: ServerConfig) async {
        if stats == nil && topAlbums.isEmpty {
            loadCache()
        }
        if isLoading {
            if activeLoadServer == server || queuedLoadServer == server {
                return
            }
            loadRevision &+= 1
            queuedLoadServer = server
            return
        }

        await performLoad(using: server)

        while let nextServer = queuedLoadServer {
            queuedLoadServer = nil
            await performLoad(using: nextServer)
        }
    }

    // 独立拉取函数，方便局部刷新
    func updateTopAlbums(using server: ServerConfig, days: Int) async {
        self.topAlbumsDays = days
        do {
            async let albumsReq = fetchTopAlbums(server: server)
            async let tracksReq = fetchRanking(server: server)
            self.topAlbums = try await albumsReq
            self.topTracks = try await tracksReq
            refreshDerivedState()
        } catch {
            print("Failed to update top albums: \(error)")
        }
    }

    func updateRanking(using server: ServerConfig, period: String, keyword: String = "") async {
        self.rankingPeriod = period
        self.rankingSearchKeyword = keyword
        do {
            self.topTracks = try await fetchRanking(server: server)
            refreshDerivedState()
        } catch {
            print("Failed to update ranking: \(error)")
        }
    }

    private func performLoad(using server: ServerConfig) async {
        loadRevision &+= 1
        let revision = loadRevision
        activeLoadRevision = revision
        activeLoadServer = server
        isLoading = true
        errorMessage = nil

        defer {
            if activeLoadRevision == revision {
                isLoading = false
                activeLoadRevision = 0
                activeLoadServer = nil
            }
        }

        do {
            async let statsReq = fetchStats(server: server)
            async let topArtistsPlaysReq = fetchTopArtistsPlays(server: server)
            async let topArtistsTracksReq = fetchTopArtistsTracks(server: server)
            async let playSourceReq = fetchPlaySourceCounts(server: server)
            async let topAlbumsReq = fetchTopAlbums(server: server)
            async let topGenresReq = fetchTopGenres(server: server)
            async let topTracksReq = fetchRanking(server: server)
            async let recentPlaysReq = fetchRecentPlays(server: server)
            async let trendReq = fetchTrendSnapshot(using: server, rangeDays: defaultTrendRange)

            let (s, tap, tat, sourceCounts, tal, tg, tt, rp, trend) = try await (
                statsReq,
                topArtistsPlaysReq,
                topArtistsTracksReq,
                playSourceReq,
                topAlbumsReq,
                topGenresReq,
                topTracksReq,
                recentPlaysReq,
                trendReq
            )

            guard activeLoadRevision == revision, loadRevision == revision else {
                return
            }

            self.stats = s
            self.topArtistsByPlays = tap
            self.topArtistsByTracks = tat
            self.playSourceCounts = sourceCounts
            self.topAlbums = tal
            self.topGenres = tg
            self.topTracks = tt
            self.recentPlays = rp

            if let trend {
                self.trendPoints = trend.points
                self.hourlyData = trend.hourlyData
                self.selectedTrendRange = trend.rangeDays
            }

            refreshDerivedState()
            saveCache()
        } catch {
            guard activeLoadRevision == revision, loadRevision == revision else {
                return
            }
            errorMessage = "主页数据加载失败"
        }
    }

    // 私有辅助拉取方法
    private func fetchStats(server: ServerConfig) async throws -> DashboardStats {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(path: APIPath.dashboardStats)
    }

    private func fetchTopArtistsPlays(server: ServerConfig) async throws -> [TopArtist] {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(path: APIPath.topArtistsPlays)
    }

    private func fetchTopArtistsTracks(server: ServerConfig) async throws -> [TopArtist] {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(path: APIPath.topArtistsTracks)
    }

    private func fetchPlaySourceCounts(server: ServerConfig) async throws -> [String: Int64] {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(path: APIPath.playCountsBySource)
    }

    private func fetchTopAlbums(server: ServerConfig) async throws -> [TopAlbum] {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(
            path: APIPath.topAlbums,
            queryItems: [URLQueryItem(name: "days", value: "\(topAlbumsDays)"), URLQueryItem(name: "limit", value: "10")]
        )
    }

    private func fetchTopGenres(server: ServerConfig) async throws -> [TopGenre] {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(path: APIPath.topGenres)
    }

    private func fetchRanking(server: ServerConfig) async throws -> [TopTrack] {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(
            path: APIPath.topTracks,
            queryItems: [
                URLQueryItem(name: "days", value: "\(topAlbumsDays)"),
                URLQueryItem(name: "limit", value: "10")
            ]
        )
    }

    private func fetchRecentPlays(server: ServerConfig) async throws -> [RecentPlayRecord] {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(path: APIPath.recentPlays, queryItems: [URLQueryItem(name: "limit", value: "20")])
    }

    private func fetchTrendSnapshot(using server: ServerConfig, rangeDays: Int) async -> HomeTrendSnapshot? {
        let client = APIClient(baseURL: server.baseURL)
        do {
            let response: TrendResponse = try await client.getJSON(
                path: APIPath.dashboardTrend,
                queryItems: [URLQueryItem(name: "range", value: "\(rangeDays)")]
            )
            let points = response.daily.map { TrendPoint(date: $0.key, count: $0.value) }
            let hourly = response.hourly.map { HourlyData(date: $0.key, hourly: $0.value.hourly) }
            return HomeTrendSnapshot(
                rangeDays: rangeDays,
                points: points.sorted { $0.date < $1.date },
                hourlyData: hourly.sorted { $0.date < $1.date }
            )
        } catch {
            return nil
        }
    }

    private func cacheKey() -> String {
        "soniclens.home.snapshot"
    }

    private func loadCache() {
        guard let snapshot = cache.load(HomeSnapshot.self, key: cacheKey()) else { return }
        stats = snapshot.stats
        topArtistsByPlays = snapshot.topArtistsByPlays
        topArtistsByTracks = snapshot.topArtistsByTracks
        playSourceCounts = snapshot.playSourceCounts ?? [:]
        topAlbums = snapshot.topAlbums
        topGenres = snapshot.topGenres
        topTracks = snapshot.topTracks
        recentPlays = snapshot.recentPlays
        trendPoints = snapshot.trendPoints
        hourlyData = snapshot.hourlyData
        selectedTrendRange = defaultTrendRange
        refreshDerivedState()
    }

    private func saveCache() {
        let snapshot = HomeSnapshot(
            stats: stats,
            topArtistsByPlays: topArtistsByPlays,
            topArtistsByTracks: topArtistsByTracks,
            playSourceCounts: playSourceCounts,
            topAlbums: topAlbums,
            topGenres: topGenres,
            topTracks: topTracks,
            recentPlays: recentPlays,
            trendPoints: trendPoints,
            hourlyData: hourlyData,
            selectedTrendRange: selectedTrendRange
        )
        cache.save(snapshot, key: cacheKey())
    }

    private func refreshDerivedState() {
        hotModulePresentation = HomeHotModulePresentation(
            topGenres: topGenres,
            topArtists: topArtistsByPlays,
            topAlbums: topAlbums,
            topTracks: topTracks,
            playSourceCounts: playSourceCounts,
            stats: stats
        )
        trendSnapshot = HomeTrendSnapshot(
            rangeDays: selectedTrendRange,
            points: trendPoints,
            hourlyData: hourlyData
        )
    }
}

struct HomeTrendSnapshot: Equatable {
    let rangeDays: Int
    let points: [TrendPoint]
    let hourlyData: [HourlyData]

    static let empty = HomeTrendSnapshot(rangeDays: 0, points: [], hourlyData: [])
}

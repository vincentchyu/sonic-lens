import Foundation

@MainActor
final class HomeViewModel: ObservableObject {
    @Published var stats: DashboardStats?
    @Published var topArtistsByPlays: [TopArtist] = []
    @Published var topArtistsByTracks: [TopArtist] = []
    @Published var topAlbums: [TopAlbum] = []
    @Published var topGenres: [TopGenre] = []
    @Published var topTracks: [Track] = []
    @Published var recentPlays: [RecentPlayRecord] = []
    @Published var trendPoints: [TrendPoint] = []
    @Published var hourlyData: [HourlyData] = []
    
    // 过滤与搜索状态
    @Published var selectedTrendRange: Int
    @Published var topAlbumsDays: Int = 30 // Web 默认通常是 30 或 7
    @Published var rankingPeriod: String = "month" // all, week, month
    @Published var rankingSearchKeyword: String = ""
    
    @Published var isLoading: Bool = false
    @Published var errorMessage: String?

    private let cache = SnapshotCache()
    private let defaultTrendRange: Int

    init(defaultTrendRange: Int = 90) {
        self.defaultTrendRange = defaultTrendRange
        self.selectedTrendRange = defaultTrendRange
    }

    func load(using server: ServerConfig) async {
        if stats == nil && topAlbums.isEmpty {
            loadCache()
        }
        isLoading = true
        errorMessage = nil
        
        do {
            async let statsReq = fetchStats(server: server)
            async let topArtistsPlaysReq = fetchTopArtistsPlays(server: server)
            async let topArtistsTracksReq = fetchTopArtistsTracks(server: server)
            async let topAlbumsReq = fetchTopAlbums(server: server)
            async let topGenresReq = fetchTopGenres(server: server)
            async let topTracksReq = fetchRanking(server: server)
            async let recentPlaysReq = fetchRecentPlays(server: server)
            async let trendReq: Void = loadTrend(using: server, rangeDays: defaultTrendRange)
            
            let (s, tap, tat, tal, tg, tt, rp, _) = try await (statsReq, topArtistsPlaysReq, topArtistsTracksReq, topAlbumsReq, topGenresReq, topTracksReq, recentPlaysReq, trendReq)
            
            self.stats = s
            self.topArtistsByPlays = tap
            self.topArtistsByTracks = tat
            self.topAlbums = tal
            self.topGenres = tg
            self.topTracks = tt
            self.recentPlays = rp
            
            saveCache()
            isLoading = false
        } catch {
            errorMessage = "主页数据加载失败"
            isLoading = false
        }
    }

    // 独立拉取函数，方便局部刷新
    func updateTopAlbums(using server: ServerConfig, days: Int) async {
        self.topAlbumsDays = days
        do {
            self.topAlbums = try await fetchTopAlbums(server: server)
        } catch {
            print("Failed to update top albums: \(error)")
        }
    }

    func updateRanking(using server: ServerConfig, period: String, keyword: String = "") async {
        self.rankingPeriod = period
        self.rankingSearchKeyword = keyword
        do {
            self.topTracks = try await fetchRanking(server: server)
        } catch {
            print("Failed to update ranking: \(error)")
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

    private func fetchRanking(server: ServerConfig) async throws -> [Track] {
        let client = APIClient(baseURL: server.baseURL)
        var queryItems = [URLQueryItem(name: "limit", value: "10")]
        
        let path: String
        if rankingPeriod == "all" {
            path = APIPath.topTracks
        } else {
            path = APIPath.topTracksPeriod
            queryItems.append(URLQueryItem(name: "period", value: rankingPeriod))
        }
        
        if !rankingSearchKeyword.isEmpty {
            queryItems.append(URLQueryItem(name: "keyword", value: rankingSearchKeyword))
        }
        
        return try await client.getJSON(path: path, queryItems: queryItems)
    }

    private func fetchRecentPlays(server: ServerConfig) async throws -> [RecentPlayRecord] {
        let client = APIClient(baseURL: server.baseURL)
        return try await client.getJSON(path: APIPath.recentPlays, queryItems: [URLQueryItem(name: "limit", value: "20")])
    }

    func loadTrend(using server: ServerConfig, rangeDays: Int) async {
        let client = APIClient(baseURL: server.baseURL)
        do {
            selectedTrendRange = rangeDays
            let response: TrendResponse = try await client.getJSON(
                path: APIPath.dashboardTrend,
                queryItems: [URLQueryItem(name: "range", value: "\(rangeDays)")]
            )
            let points = response.daily.map { TrendPoint(date: $0.key, count: $0.value) }
            trendPoints = points.sorted { $0.date < $1.date }
            
            // 处理 hourly 数据
            let hourly = response.hourly.map { HourlyData(date: $0.key, hourly: $0.value.hourly) }
            hourlyData = hourly.sorted { $0.date < $1.date }
        } catch {
            // TODO: handle error
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
        topAlbums = snapshot.topAlbums
        topGenres = snapshot.topGenres
        topTracks = snapshot.topTracks
        recentPlays = snapshot.recentPlays
        trendPoints = snapshot.trendPoints
        hourlyData = snapshot.hourlyData
        selectedTrendRange = defaultTrendRange
    }

    private func saveCache() {
        let snapshot = HomeSnapshot(
            stats: stats,
            topArtistsByPlays: topArtistsByPlays,
            topArtistsByTracks: topArtistsByTracks,
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
}

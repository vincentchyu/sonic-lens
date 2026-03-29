import Foundation
import Combine
import OSLog

extension Notification.Name {
    static let libraryFavoriteDidChange = Notification.Name("libraryFavoriteDidChange")
    static let librarySyncDidUpdate = Notification.Name("librarySyncDidUpdate")
}

struct LibraryFavoriteChange {
    let artist: String
    let album: String
    let track: String
    let trackNumber: Int?
    let discNumber: Int?
    let appleMusic: Bool
    let lastfm: Bool
    let appleMusicState: TrackFavoriteState
    let lastfmState: TrackFavoriteState
    let favoriteState: TrackFavoriteState
}

struct LibrarySyncUpdate {
    let version: Int64
}

final class AppStore: ObservableObject {
    @Published var currentServer: ServerConfig?
    @Published var connectionStatus: ConnectionStatus = .disconnected
    @Published var nowPlaying: NowPlaying?
    @Published var nowPlayingSource: String?
    @Published var favoriteKeys: Set<String> = []
    @Published private(set) var favoriteProjections: [String: TrackFavoriteProjection] = [:]
    @Published var recentServers: [ServerConfig] = []

    private var nowPlayingService: NowPlayingService?
    private var libraryUpdateWorkItem: DispatchWorkItem?
    private let recentStore = RecentServerStore()
    private let artworkResolveService = ArtworkResolveService.shared
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "AppStore")

    @MainActor
    func connect(_ server: ServerConfig) async {
        let startedAt = CFAbsoluteTimeGetCurrent()
        connectionStatus = .connecting
        logger.info("开始连接服务端 \(server.displayName, privacy: .public)")
        do {
            let client = APIClient(baseURL: server.baseURL)
            logger.debug("开始检查服务端健康状态 \(server.baseURL.absoluteString, privacy: .public)")
            let healthStartedAt = CFAbsoluteTimeGetCurrent()
            let health: HealthResponse = try await client.getJSON(path: APIPath.health)
            let healthElapsed = CFAbsoluteTimeGetCurrent() - healthStartedAt
            logger.info("服务端健康检查完成，耗时 \(String(format: "%.3f", healthElapsed), privacy: .public) 秒")
            guard health.status == "ok" else {
                connectionStatus = .failed("服务端状态异常")
                logger.error("服务端健康检查返回异常状态 \(health.status, privacy: .public)")
                return
            }
            currentServer = server
            connectionStatus = .connected
            recentStore.add(server)
            recentServers = recentStore.load()
            logger.info("服务端连接成功，开始启动播放态监听")
            startNowPlaying(server)
            let elapsed = CFAbsoluteTimeGetCurrent() - startedAt
            logger.info("连接服务端流程完成，耗时 \(String(format: "%.3f", elapsed), privacy: .public) 秒")
        } catch {
            connectionStatus = .failed("连接失败，请检查地址和端口")
            logger.error("连接服务端失败 \(error.localizedDescription, privacy: .public)")
        }
    }

    func disconnect() {
        logger.info("断开当前服务端连接")
        libraryUpdateWorkItem?.cancel()
        libraryUpdateWorkItem = nil
        nowPlayingService?.stop()
        nowPlayingService = nil
        currentServer = nil
        nowPlaying = nil
        nowPlayingSource = nil
        favoriteKeys = []
        favoriteProjections = [:]
        connectionStatus = .disconnected
    }

    func loadRecentServers() {
        let startedAt = CFAbsoluteTimeGetCurrent()
        recentServers = recentStore.load()
        let elapsed = CFAbsoluteTimeGetCurrent() - startedAt
        logger.debug("读取最近连接服务端数量 \(self.recentServers.count, privacy: .public)，耗时 \(String(format: "%.3f", elapsed), privacy: .public) 秒")
    }

    private func startNowPlaying(_ server: ServerConfig) {
        nowPlayingService?.stop()
        logger.debug("启动播放态 WebSocket 监听 \(server.webSocketURL.absoluteString, privacy: .public)")
        let service = NowPlayingService(server: server)
        service.onUpdate = { [weak self] nowPlaying, source in
            DispatchQueue.main.async {
                self?.nowPlaying = nowPlaying
                self?.nowPlayingSource = source
                self?.syncFavoriteProjection(with: nowPlaying)
                self?.resolveNowPlayingArtworkIfNeeded(for: nowPlaying)
            }
        }
        service.onLibraryUpdate = { [weak self] version in
            self?.scheduleLibraryUpdate(version)
        }
        nowPlayingService = service
        service.start()
    }

    func isFavorite(
        artist: String, album: String?, track: String, trackNumber: Int? = nil, discNumber: Int? = nil,
    ) -> Bool {
        favoriteProjection(
            artist: artist,
            album: album,
            track: track,
            trackNumber: trackNumber,
            discNumber: discNumber
        )?.isFavoritedEffective ?? false
    }

    func favoriteProjection(
        artist: String,
        album: String?,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil
    ) -> TrackFavoriteProjection? {
        guard let album else { return nil }
        return favoriteProjections[
            favoriteKey(
                artist: artist,
                album: album,
                track: track,
                trackNumber: trackNumber,
                discNumber: discNumber
            )
        ]
    }

    @MainActor
    func toggleFavorite(
        artist: String,
        album: String?,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil,
        source: String? = nil
    ) async {
        guard let album else { return }
        let key = favoriteKey(
            artist: artist,
            album: album,
            track: track,
            trackNumber: trackNumber,
            discNumber: discNumber
        )
        let nextValue = !(favoriteProjections[key]?.isFavoritedEffective ?? false)
        await setFavorite(
            artist: artist,
            album: album,
            track: track,
            trackNumber: trackNumber,
            discNumber: discNumber,
            favorite: nextValue,
            source: source
        )
    }

    @MainActor
    func setFavorite(
        artist: String,
        album: String?,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil,
        favorite: Bool,
        source: String? = nil
    ) async {
        guard let album, !album.isEmpty, let server = currentServer else { return }
        let resolvedSource = source ?? nowPlayingSource ?? "apple_music"
        let key = favoriteKey(
            artist: artist,
            album: album,
            track: track,
            trackNumber: trackNumber,
            discNumber: discNumber
        )
        let request = FavoriteRequest(
            artist: artist,
            album: album,
            track: track,
            trackNumber: trackNumber,
            discNumber: discNumber,
            source: resolvedSource,
            favorite: favorite
        )

        do {
            let client = APIClient(baseURL: server.baseURL)
            let response: FavoriteResponse = try await client.postJSON(path: APIPath.favorite, body: request)
            let projection = response.projection
            syncFavoriteProjection(key: key, projection: projection)
            NotificationCenter.default.post(
                name: .libraryFavoriteDidChange,
                object: LibraryFavoriteChange(
                    artist: artist,
                    album: album,
                    track: track,
                    trackNumber: trackNumber,
                    discNumber: discNumber,
                    appleMusic: projection.appleMusic,
                    lastfm: projection.lastfm,
                    appleMusicState: projection.appleMusicState,
                    lastfmState: projection.lastfmState,
                    favoriteState: projection.favoriteState
                )
            )
            patchNowPlayingFavoriteProjection(
                artist: artist,
                album: album,
                track: track,
                trackNumber: trackNumber,
                discNumber: discNumber,
                projection: projection
            )
        } catch {
            // ignore favorite errors
        }
    }

    private func syncFavoriteProjection(with nowPlaying: NowPlaying?) {
        guard let nowPlaying, let album = nowPlaying.album, !album.isEmpty else { return }
        let key = favoriteKey(
            artist: nowPlaying.artist,
            album: album,
            track: nowPlaying.track,
            trackNumber: nowPlaying.trackNumber,
            discNumber: nowPlaying.discNumber
        )
        syncFavoriteProjection(key: key, projection: nowPlaying.favoriteProjection)
    }

    private func syncFavoriteProjection(key: String, projection: TrackFavoriteProjection) {
        favoriteProjections[key] = projection
        if projection.isFavoritedEffective {
            favoriteKeys.insert(key)
        } else {
            favoriteKeys.remove(key)
        }
    }

    private func patchNowPlayingFavoriteProjection(
        artist: String,
        album: String,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil,
        projection: TrackFavoriteProjection
    ) {
        guard let nowPlaying else { return }
        guard nowPlaying.artist == artist,
              (nowPlaying.album ?? "") == album,
              nowPlaying.track == track,
              nowPlaying.trackNumber == trackNumber,
              nowPlaying.discNumber == discNumber else { return }

        self.nowPlaying = NowPlaying(
            artist: nowPlaying.artist,
            album: nowPlaying.album,
            track: nowPlaying.track,
            duration: nowPlaying.duration,
            position: nowPlaying.position,
            positionMs: nowPlaying.positionMs,
            artwork: nowPlaying.artwork,
            isAppleMusicFav: projection.appleMusic,
            isLastFmFav: projection.lastfm,
            appleMusicState: projection.appleMusicState,
            lastfmState: projection.lastfmState,
            favoriteState: projection.favoriteState,
            trackNumber: nowPlaying.trackNumber,
            discNumber: nowPlaying.discNumber
        )
    }

    private func favoriteKey(
        artist: String, album: String, track: String, trackNumber: Int? = nil, discNumber: Int? = nil,
    ) -> String {
        [artist, album, track, String(trackNumber ?? 0), String(discNumber ?? 0)].joined(separator: "•")
    }

    private func scheduleLibraryUpdate(_ version: Int64) {
        logger.debug("收到资料库更新版本 \(version, privacy: .public)，准备延迟通知")
        let workItem = DispatchWorkItem {
            self.logger.info("发布资料库更新通知 \(version, privacy: .public)")
            NotificationCenter.default.post(
                name: .librarySyncDidUpdate,
                object: LibrarySyncUpdate(version: version)
            )
        }
        libraryUpdateWorkItem?.cancel()
        libraryUpdateWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + 1, execute: workItem)
    }

    private func resolveNowPlayingArtworkIfNeeded(for nowPlaying: NowPlaying?) {
        guard let server = currentServer else { return }
        guard let nowPlaying else { return }
        guard (nowPlaying.artwork ?? "").isEmpty else { return }
        guard let album = nowPlaying.album, !album.isEmpty else { return }

        Task { [weak self] in
            guard let self else { return }
            guard let resolved = await self.artworkResolveService.resolveArtworkURL(
                using: server,
                albumID: nil,
                albumArtist: nowPlaying.artist,
                artist: nowPlaying.artist,
                album: album,
                artworkKey: nil
            ) else {
                return
            }

            await MainActor.run {
                guard let current = self.nowPlaying else { return }
                guard current.artist == nowPlaying.artist,
                      current.track == nowPlaying.track,
                      current.album == nowPlaying.album,
                      current.trackNumber == nowPlaying.trackNumber,
                      current.discNumber == nowPlaying.discNumber else { return }
                guard (current.artwork ?? "").isEmpty else { return }

                self.nowPlaying = NowPlaying(
                    artist: current.artist,
                    album: current.album,
                    track: current.track,
                    duration: current.duration,
                    position: current.position,
                    positionMs: current.positionMs,
                    artwork: resolved,
                    isAppleMusicFav: current.isAppleMusicFav,
                    isLastFmFav: current.isLastFmFav,
                    appleMusicState: current.appleMusicState,
                    lastfmState: current.lastfmState,
                    favoriteState: current.favoriteState,
                    trackNumber: current.trackNumber,
                    discNumber: current.discNumber
                )
            }
        }
    }
}

actor ArtworkResolveService {
    static let shared = ArtworkResolveService()

    private enum ResolveKey: Hashable {
        case albumID(Int64)
        case artistAlbum(String, String)
        case artworkKey(String)
    }

    private struct CacheEntry {
        let url: String?
        let expiresAt: Date
    }

    private var cache: [ResolveKey: CacheEntry] = [:]
    private var inFlight: [ResolveKey: Task<String?, Never>] = [:]

    func resolveArtworkURL(
        using server: ServerConfig,
        albumID: Int64?,
        albumArtist: String?,
        artist: String?,
        album: String?,
        artworkKey: String?
    ) async -> String? {
        guard let key = Self.resolveKey(albumID: albumID, albumArtist: albumArtist, artist: artist, album: album, artworkKey: artworkKey) else {
            return nil
        }

        let now = Date()
        if let cached = cache[key], cached.expiresAt > now {
            return cached.url
        }

        if let running = inFlight[key] {
            return await running.value
        }

        let task = Task<String?, Never> {
            let client = APIClient(baseURL: server.baseURL)
            let items = Self.queryItems(
                albumID: albumID,
                albumArtist: albumArtist,
                artist: artist,
                album: album,
                artworkKey: artworkKey
            )

            do {
                let response: ResolveArtworkResponse = try await client.getJSON(path: APIPath.artworkResolve, queryItems: items)
                guard response.exists else {
                    return nil
                }
                return ArtworkURLResolver.resolveArtworkPath(response.coverArtURL, artworkBaseURL: server.artworkBaseURL)
            } catch {
                return nil
            }
        }
        inFlight[key] = task

        let resolved = await task.value
        inFlight.removeValue(forKey: key)

        let ttl: TimeInterval = resolved == nil ? 45 : 1800
        cache[key] = CacheEntry(url: resolved, expiresAt: now.addingTimeInterval(ttl))
        return resolved
    }

    private static func resolveKey(
        albumID: Int64?,
        albumArtist: String?,
        artist: String?,
        album: String?,
        artworkKey: String?
    ) -> ResolveKey? {
        if let albumID, albumID > 0 {
            return .albumID(albumID)
        }

        let canonicalAlbum = canonical(album)
        let canonicalArtist = canonical(albumArtist) ?? canonical(artist)
        if let canonicalArtist, let canonicalAlbum {
            return .artistAlbum(canonicalArtist, canonicalAlbum)
        }

        if let key = canonical(artworkKey) {
            return .artworkKey(key)
        }
        return nil
    }

    private static func queryItems(
        albumID: Int64?,
        albumArtist: String?,
        artist: String?,
        album: String?,
        artworkKey: String?
    ) -> [URLQueryItem] {
        var items: [URLQueryItem] = []
        if let albumID, albumID > 0 {
            items.append(URLQueryItem(name: "album_id", value: String(albumID)))
        }
        if let albumArtist = nonEmpty(albumArtist) {
            items.append(URLQueryItem(name: "albumArtist", value: albumArtist))
        }
        if let artist = nonEmpty(artist) {
            items.append(URLQueryItem(name: "artist", value: artist))
        }
        if let album = nonEmpty(album) {
            items.append(URLQueryItem(name: "album", value: album))
        }
        if let artworkKey = nonEmpty(artworkKey) {
            items.append(URLQueryItem(name: "artworkKey", value: artworkKey))
        }
        return items
    }

    private static func canonical(_ value: String?) -> String? {
        nonEmpty(value)?.folding(options: [.caseInsensitive, .diacriticInsensitive], locale: .current)
    }

    private static func nonEmpty(_ value: String?) -> String? {
        guard let value else { return nil }
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}

enum ConnectionStatus {
    case disconnected
    case connecting
    case connected
    case failed(String)
}

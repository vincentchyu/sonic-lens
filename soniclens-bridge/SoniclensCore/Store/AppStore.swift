import Foundation
import Observation
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

@MainActor
@Observable
final class PlaybackStore {
    var nowPlaying: NowPlaying?
    var nowPlayingSource: String?

    func update(nowPlaying: NowPlaying?, source: String?) {
        self.nowPlaying = nowPlaying
        self.nowPlayingSource = source
    }

    func reset() {
        nowPlaying = nil
        nowPlayingSource = nil
    }
}

@MainActor
@Observable
final class FavoriteStore {
    private(set) var favoriteKeys: Set<String> = []
    private(set) var favoriteProjections: [String: TrackFavoriteProjection] = [:]

    func syncProjection(key: String, projection: TrackFavoriteProjection) {
        favoriteProjections[key] = projection
        if projection.isFavoritedEffective {
            favoriteKeys.insert(key)
        } else {
            favoriteKeys.remove(key)
        }
    }

    func projection(for key: String) -> TrackFavoriteProjection? {
        favoriteProjections[key]
    }

    func reset() {
        favoriteKeys = []
        favoriteProjections = [:]
    }
}

@MainActor
final class AppStore: ObservableObject {
    @Published var currentServer: ServerConfig?
    @Published var connectionStatus: ConnectionStatus = .idle
    @Published var recentServers: [ServerConfig] = []
    @Published private(set) var activeConnectionTargetKey: String?

    let playbackStore = PlaybackStore()
    let favoriteStore = FavoriteStore()
    let insightCoordinator = InsightAnalysisCoordinator()

    private var nowPlayingService: NowPlayingService?
    private var libraryUpdateWorkItem: DispatchWorkItem?
    private let recentStore = RecentServerStore()
    private let artworkResolveService = ArtworkResolveService.shared
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "AppStore")
    private var connectionTask: Task<Void, Never>?
    private var currentConnectionAttemptID: UUID?
    private let preferredResolvedHealthCheckTimeout: TimeInterval = 1.5
    private let fallbackHealthCheckTimeout: TimeInterval = 3

    @MainActor
    func connect(_ server: ServerConfig) async {
        let targetKey = connectionTargetKey(for: server)
        if let activeConnectionTargetKey {
            if activeConnectionTargetKey == targetKey {
                logger.info("点击同一连接目标，取消当前连接 \(server.displayName, privacy: .public)")
                cancelConnection()
                return
            }
            logger.info("切换连接目标，取消旧连接并切到 \(server.displayName, privacy: .public)")
            cancelActiveConnection(announceCancellation: false)
        }

        let attemptID = UUID()
        currentConnectionAttemptID = attemptID
        activeConnectionTargetKey = targetKey
        setConnectionStatus(
            phase: .resolving,
            message: "正在准备连接...",
            detail: server.displayName
        )
        let task = Task { [weak self] in
            guard let self else { return }
            await self.runConnectionFlow(server: server, attemptID: attemptID)
        }
        connectionTask = task
        await task.value
    }

    func cancelConnection() {
        cancelActiveConnection(announceCancellation: true)
    }

    private func runConnectionFlow(server: ServerConfig, attemptID: UUID) async {
        guard isCurrentConnectionAttempt(attemptID) else { return }

        let startedAt = CFAbsoluteTimeGetCurrent()
        let connectionDetail = describeConnectionTarget(server)
        setConnectionStatus(
            phase: .resolving,
            message: "正在解析连接目标...",
            detail: connectionDetail
        )
        logger.info("开始连接服务端 \(server.displayName, privacy: .public)")

        do {
            try Task.checkCancellation()

            let healthStartedAt = CFAbsoluteTimeGetCurrent()
            let (effectiveServer, health) = try await resolveReachableServer(from: server)
            try Task.checkCancellation()
            let healthElapsed = CFAbsoluteTimeGetCurrent() - healthStartedAt
            logger.info("服务端健康检查完成，耗时 \(String(format: "%.3f", healthElapsed), privacy: .public) 秒")
            guard health.status == "ok" else {
                finishConnectionFailure(
                    attemptID: attemptID,
                    message: "服务端状态异常",
                    detail: "health=\(health.status)"
                )
                logger.error("服务端健康检查返回异常状态 \(health.status, privacy: .public)")
                return
            }

            setConnectionStatus(
                phase: .establishingRealtime,
                message: "正在建立实时连接...",
                detail: effectiveServer.webSocketURL.absoluteString
            )
            let realtimeStartedAt = CFAbsoluteTimeGetCurrent()
            currentServer = effectiveServer
            recentStore.add(effectiveServer)
            recentServers = recentStore.load()
            logger.info("服务端连接成功，开始启动播放态监听")
            startNowPlaying(effectiveServer)
            let realtimeElapsed = CFAbsoluteTimeGetCurrent() - realtimeStartedAt
            logger.info("实时连接启动完成，耗时 \(String(format: "%.3f", realtimeElapsed), privacy: .public) 秒")

            let elapsed = CFAbsoluteTimeGetCurrent() - startedAt
            finishConnectionSuccess(attemptID: attemptID, server: effectiveServer, elapsed: elapsed)
            logger.info("连接服务端流程完成，耗时 \(String(format: "%.3f", elapsed), privacy: .public) 秒")
        } catch is CancellationError {
            logger.info("连接流程已取消 \(server.displayName, privacy: .public)")
            finalizeConnectionAttempt(attemptID: attemptID, resetStatus: false)
        } catch {
            finishConnectionFailure(
                attemptID: attemptID,
                message: resolveConnectionErrorMessage(error),
                detail: error.localizedDescription
            )
            logger.error("连接服务端失败 \(error.localizedDescription, privacy: .public)")
        }
    }

    func disconnect() {
        logger.info("断开当前服务端连接")
        cancelActiveConnection(announceCancellation: false)
        libraryUpdateWorkItem?.cancel()
        libraryUpdateWorkItem = nil
        nowPlayingService?.stop()
        nowPlayingService = nil
        currentServer = nil
        activeConnectionTargetKey = nil
        playbackStore.reset()
        favoriteStore.reset()
        connectionStatus = .idle
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
                self?.playbackStore.update(nowPlaying: nowPlaying, source: source)
                self?.syncFavoriteProjection(with: nowPlaying)
                self?.resolveNowPlayingArtworkIfNeeded(for: nowPlaying)
            }
        }
        service.onLibraryUpdate = { [weak self] version in
            self?.scheduleLibraryUpdate(version)
        }
        service.onInsightJobUpdate = { [weak self] job in
            DispatchQueue.main.async {
                guard let self else { return }
                self.insightCoordinator.handleWebSocketJobUpdate(job, using: self.currentServer)
            }
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
        return favoriteStore.projection(for:
            favoriteKey(
                artist: artist,
                album: album,
                track: track,
                trackNumber: trackNumber,
                discNumber: discNumber
            )
        )
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
        let nextValue = !(favoriteStore.projection(for: key)?.isFavoritedEffective ?? false)
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
        let resolvedSource = source ?? playbackStore.nowPlayingSource ?? "apple_music"
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
        favoriteStore.syncProjection(key: key, projection: projection)
    }

    private func patchNowPlayingFavoriteProjection(
        artist: String,
        album: String,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil,
        projection: TrackFavoriteProjection
    ) {
        guard let nowPlaying = playbackStore.nowPlaying else { return }
        guard nowPlaying.artist == artist,
              (nowPlaying.album ?? "") == album,
              nowPlaying.track == track,
              nowPlaying.trackNumber == trackNumber,
              nowPlaying.discNumber == discNumber else { return }

        playbackStore.nowPlaying = NowPlaying(
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
                guard let current = self.playbackStore.nowPlaying else { return }
                guard current.artist == nowPlaying.artist,
                      current.track == nowPlaying.track,
                      current.album == nowPlaying.album,
                      current.trackNumber == nowPlaying.trackNumber,
                      current.discNumber == nowPlaying.discNumber else { return }
                guard (current.artwork ?? "").isEmpty else { return }

                self.playbackStore.nowPlaying = NowPlaying(
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

    func isConnecting(to server: ServerConfig) -> Bool {
        activeConnectionTargetKey == connectionTargetKey(for: server)
    }

    var isConnecting: Bool {
        activeConnectionTargetKey != nil
    }

    private func resolveReachableServer(from server: ServerConfig) async throws -> (ServerConfig, HealthResponse) {
        let attempts = connectionAttempts(for: server)
        var lastError: Error?

        for attempt in attempts {
            try Task.checkCancellation()
            let client = APIClient(baseURL: attempt.server.baseURL)
            logger.debug("开始检查服务端健康状态 \(attempt.server.baseURL.absoluteString, privacy: .public)")
            setConnectionStatus(
                phase: .healthCheck,
                message: "正在检查服务端健康状态...",
                detail: attempt.detail
            )

            do {
                let health: HealthResponse = try await client.getJSON(
                    path: APIPath.health,
                    timeout: attempt.timeout
                )
                if attempt.usesResolvedHost {
                    logger.info("服务端健康检查命中局域网直连地址 \(attempt.server.baseURL.absoluteString, privacy: .public)")
                } else {
                    logger.info("服务端健康检查回退到主机名地址 \(attempt.server.baseURL.absoluteString, privacy: .public)")
                }
                return (attempt.server, health)
            } catch {
                lastError = error
                logger.warning("服务端健康检查失败，准备尝试下一个候选地址 \(attempt.server.baseURL.absoluteString, privacy: .public)，错误 \(error.localizedDescription, privacy: .public)")
            }
        }

        throw lastError ?? URLError(.cannotFindHost)
    }

    private func connectionTargetKey(for server: ServerConfig) -> String {
        "\(server.scheme.lowercased())://\(server.host.lowercased()):\(server.port)"
    }

    private func describeConnectionTarget(_ server: ServerConfig) -> String {
        guard let resolvedHost = server.resolvedHost, !resolvedHost.isEmpty else {
            return server.displayName
        }
        return "\(server.displayName) · 优先直连 \(resolvedHost)"
    }

    private func connectionAttempts(for server: ServerConfig) -> [ConnectionAttempt] {
        var attempts: [ConnectionAttempt] = []

        if let resolvedHost = server.resolvedHost, !resolvedHost.isEmpty {
            let directServer = server.withResolvedHost(resolvedHost)
            attempts.append(
                ConnectionAttempt(
                    server: directServer,
                    timeout: preferredResolvedHealthCheckTimeout,
                    detail: "\(directServer.baseURL.absoluteString) · 局域网直连",
                    usesResolvedHost: true
                )
            )
        }

        let hostnameServer = server.withResolvedHost(nil)
        if attempts.isEmpty || hostnameServer.baseURL != attempts[0].server.baseURL {
            attempts.append(
                ConnectionAttempt(
                    server: hostnameServer,
                    timeout: fallbackHealthCheckTimeout,
                    detail: "\(hostnameServer.baseURL.absoluteString) · 主机名回退",
                    usesResolvedHost: false
                )
            )
        }

        return attempts
    }

    private struct ConnectionAttempt {
        let server: ServerConfig
        let timeout: TimeInterval
        let detail: String
        let usesResolvedHost: Bool
    }

    private func cancelActiveConnection(announceCancellation: Bool) {
        connectionTask?.cancel()
        connectionTask = nil
        activeConnectionTargetKey = nil
        currentConnectionAttemptID = nil
        if announceCancellation {
            connectionStatus = .cancelled(message: "已取消当前连接", detail: nil)
        }
    }

    private func isCurrentConnectionAttempt(_ attemptID: UUID) -> Bool {
        currentConnectionAttemptID == attemptID
    }

    private func setConnectionStatus(phase: ConnectionPhase, message: String, detail: String?) {
        connectionStatus = ConnectionStatus(phase: phase, message: message, detail: detail)
    }

    private func finishConnectionSuccess(attemptID: UUID, server: ServerConfig, elapsed: CFAbsoluteTime) {
        guard isCurrentConnectionAttempt(attemptID) else { return }
        connectionStatus = .connected(message: "已连接", detail: "\(server.displayName) · \(String(format: "%.2f", elapsed)) 秒")
        finalizeConnectionAttempt(attemptID: attemptID, resetStatus: false)
    }

    private func finishConnectionFailure(attemptID: UUID, message: String, detail: String?) {
        guard isCurrentConnectionAttempt(attemptID) else { return }
        connectionStatus = .failed(message: message, detail: detail)
        finalizeConnectionAttempt(attemptID: attemptID, resetStatus: false)
    }

    private func finalizeConnectionAttempt(attemptID: UUID, resetStatus: Bool) {
        guard isCurrentConnectionAttempt(attemptID) else { return }
        activeConnectionTargetKey = nil
        connectionTask = nil
        currentConnectionAttemptID = nil
        if resetStatus {
            connectionStatus = .idle
        }
    }

    private func resolveConnectionErrorMessage(_ error: Error) -> String {
        if let urlError = error as? URLError {
            switch urlError.code {
            case .timedOut, .cannotFindHost, .cannotConnectToHost, .dnsLookupFailed, .networkConnectionLost:
                return "服务端未响应"
            case .badServerResponse:
                return "服务端响应异常"
            default:
                return "连接失败，请检查地址和端口"
            }
        }
        return "连接失败，请检查地址和端口"
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
        let resource: ResolvedArtworkResource?
        let expiresAt: Date
    }

    private var cache: [ResolveKey: CacheEntry] = [:]
    private var inFlight: [ResolveKey: Task<ResolvedArtworkResource?, Never>] = [:]

    func resolveArtworkResource(
        using server: ServerConfig,
        albumID: Int64?,
        albumArtist: String?,
        artist: String?,
        album: String?,
        artworkKey: String?
    ) async -> ResolvedArtworkResource? {
        guard let key = Self.resolveKey(albumID: albumID, albumArtist: albumArtist, artist: artist, album: album, artworkKey: artworkKey) else {
            return nil
        }

        let now = Date()
        if let cached = cache[key], cached.expiresAt > now {
            return cached.resource
        }

        if let running = inFlight[key] {
            return await running.value
        }

        let task = Task<ResolvedArtworkResource?, Never> {
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
                let remoteURL = ArtworkURLResolver.resolveArtworkPath(response.coverArtURL, artworkBaseURL: server.artworkBaseURL)
                return ResolvedArtworkResource(remoteURL: remoteURL, coverArtObjectKey: response.coverArtObjectKey)
            } catch {
                return nil
            }
        }
        inFlight[key] = task

        let resolved = await task.value
        inFlight.removeValue(forKey: key)

        let ttl: TimeInterval = resolved == nil ? 45 : 1800
        cache[key] = CacheEntry(resource: resolved, expiresAt: now.addingTimeInterval(ttl))
        return resolved
    }

    func resolveArtworkURL(
        using server: ServerConfig,
        albumID: Int64?,
        albumArtist: String?,
        artist: String?,
        album: String?,
        artworkKey: String?
    ) async -> String? {
        let resource = await resolveArtworkResource(
            using: server,
            albumID: albumID,
            albumArtist: albumArtist,
            artist: artist,
            album: album,
            artworkKey: artworkKey
        )
        return resource?.remoteURL
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

enum ConnectionPhase: String {
    case idle
    case resolving
    case healthCheck
    case establishingRealtime
    case connected
    case failed
    case cancelled

    var isInFlight: Bool {
        switch self {
        case .resolving, .healthCheck, .establishingRealtime:
            return true
        case .idle, .connected, .failed, .cancelled:
            return false
        }
    }

    var inlineStatusTitle: String {
        switch self {
        case .resolving:
            return "解析中"
        case .healthCheck:
            return "检查中"
        case .establishingRealtime:
            return "建立中"
        case .connected:
            return "已连接"
        case .failed:
            return "连接失败"
        case .cancelled:
            return "已取消"
        case .idle:
            return "未连接"
        }
    }
}

struct ConnectionStatus: Equatable {
    let phase: ConnectionPhase
    let message: String
    let detail: String?

    static let idle = ConnectionStatus(phase: .idle, message: "未连接", detail: nil)

    static func connected(message: String, detail: String?) -> ConnectionStatus {
        ConnectionStatus(phase: .connected, message: message, detail: detail)
    }

    static func failed(message: String, detail: String?) -> ConnectionStatus {
        ConnectionStatus(phase: .failed, message: message, detail: detail)
    }

    static func cancelled(message: String, detail: String?) -> ConnectionStatus {
        ConnectionStatus(phase: .cancelled, message: message, detail: detail)
    }
}

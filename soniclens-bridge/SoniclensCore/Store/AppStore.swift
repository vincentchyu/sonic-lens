import Foundation
import Combine

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
    @Published var recentServers: [ServerConfig] = []

    private var nowPlayingService: NowPlayingService?
    private var libraryUpdateWorkItem: DispatchWorkItem?
    private let recentStore = RecentServerStore()

    @MainActor
    func connect(_ server: ServerConfig) async {
        connectionStatus = .connecting
        do {
            let client = APIClient(baseURL: server.baseURL)
            let health: HealthResponse = try await client.getJSON(path: APIPath.health)
            guard health.status == "ok" else {
                connectionStatus = .failed("服务端状态异常")
                return
            }
            currentServer = server
            connectionStatus = .connected
            recentStore.add(server)
            recentServers = recentStore.load()
            startNowPlaying(server)
        } catch {
            connectionStatus = .failed("连接失败，请检查地址和端口")
        }
    }

    func disconnect() {
        libraryUpdateWorkItem?.cancel()
        libraryUpdateWorkItem = nil
        nowPlayingService?.stop()
        nowPlayingService = nil
        currentServer = nil
        nowPlaying = nil
        nowPlayingSource = nil
        connectionStatus = .disconnected
    }

    func loadRecentServers() {
        recentServers = recentStore.load()
    }

    private func startNowPlaying(_ server: ServerConfig) {
        nowPlayingService?.stop()
        let service = NowPlayingService(server: server)
        service.onUpdate = { [weak self] nowPlaying, source in
            DispatchQueue.main.async {
                self?.nowPlaying = nowPlaying
                self?.nowPlayingSource = source
                self?.syncFavoriteKey(with: nowPlaying)
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
        guard let album else { return false }
        return favoriteKeys.contains(
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
        let nextValue = !favoriteKeys.contains(key)
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
            if response.appleMusic || response.lastfm {
                favoriteKeys.insert(key)
            } else {
                favoriteKeys.remove(key)
            }
            NotificationCenter.default.post(
                name: .libraryFavoriteDidChange,
                object: LibraryFavoriteChange(
                    artist: artist,
                    album: album,
                    track: track,
                    trackNumber: trackNumber,
                    discNumber: discNumber,
                    appleMusic: response.appleMusic,
                    lastfm: response.lastfm
                )
            )
            patchNowPlayingFavoriteFlags(
                artist: artist,
                album: album,
                track: track,
                trackNumber: trackNumber,
                discNumber: discNumber,
                appleMusic: response.appleMusic,
                lastfm: response.lastfm
            )
        } catch {
            // ignore favorite errors
        }
    }

    private func syncFavoriteKey(with nowPlaying: NowPlaying?) {
        guard let nowPlaying, let album = nowPlaying.album, !album.isEmpty else { return }
        let key = favoriteKey(
            artist: nowPlaying.artist,
            album: album,
            track: nowPlaying.track,
            trackNumber: nowPlaying.trackNumber,
            discNumber: nowPlaying.discNumber
        )
        let isAnyFavorited = (nowPlaying.isAppleMusicFav ?? false) || (nowPlaying.isLastFmFav ?? false)
        if isAnyFavorited {
            favoriteKeys.insert(key)
        } else {
            favoriteKeys.remove(key)
        }
    }

    private func patchNowPlayingFavoriteFlags(
        artist: String,
        album: String,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil,
        appleMusic: Bool,
        lastfm: Bool
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
            isAppleMusicFav: appleMusic,
            isLastFmFav: lastfm,
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
        let workItem = DispatchWorkItem {
            NotificationCenter.default.post(
                name: .librarySyncDidUpdate,
                object: LibrarySyncUpdate(version: version)
            )
        }
        libraryUpdateWorkItem?.cancel()
        libraryUpdateWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + 1, execute: workItem)
    }
}

enum ConnectionStatus {
    case disconnected
    case connecting
    case connected
    case failed(String)
}

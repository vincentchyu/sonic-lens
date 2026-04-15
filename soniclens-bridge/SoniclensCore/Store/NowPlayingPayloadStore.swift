import Foundation

struct NowPlayingPayloadRequest: Hashable {
    let serverBaseURL: String
    let artist: String
    let album: String?
    let track: String
    let trackNumber: Int?
    let discNumber: Int?

    init(server: ServerConfig, nowPlaying: NowPlaying) {
        self.init(
            serverBaseURL: server.baseURL.absoluteString,
            artist: nowPlaying.artist,
            album: nowPlaying.album,
            track: nowPlaying.track,
            trackNumber: nowPlaying.trackNumber,
            discNumber: nowPlaying.discNumber
        )
    }

    init(
        serverBaseURL: String,
        artist: String,
        album: String?,
        track: String,
        trackNumber: Int?,
        discNumber: Int?
    ) {
        self.serverBaseURL = serverBaseURL
        self.artist = artist
        self.album = album
        self.track = track
        self.trackNumber = trackNumber
        self.discNumber = discNumber
    }

    var requestKey: String {
        [
            serverBaseURL,
            artist,
            album ?? "",
            track,
            String(trackNumber ?? 0),
            String(discNumber ?? 0)
        ].joined(separator: "•")
    }
}

struct NowPlayingPayloadSnapshot {
    let lyricsResponse: TrackLyricsResponse
    let lyricLines: [LyricLine]
    let insightResponse: TrackInsightResponse
}

actor NowPlayingPayloadStore {
    static let shared = NowPlayingPayloadStore()

    private var cache: [String: NowPlayingPayloadSnapshot] = [:]
    private var inFlight: [String: Task<NowPlayingPayloadSnapshot?, Never>] = [:]

    func snapshot(
        using server: ServerConfig,
        request: NowPlayingPayloadRequest
    ) async -> NowPlayingPayloadSnapshot? {
        let key = request.requestKey
        if let cached = cache[key] {
            return cached
        }
        if let running = inFlight[key] {
            return await running.value
        }

        let task = Task<NowPlayingPayloadSnapshot?, Never> {
            await Self.loadSnapshot(using: server, request: request)
        }
        inFlight[key] = task

        let resolved = await task.value
        inFlight.removeValue(forKey: key)
        if let resolved {
            cache[key] = resolved
        }
        return resolved
    }

    func prefetch(
        using server: ServerConfig,
        request: NowPlayingPayloadRequest
    ) async {
        _ = await snapshot(using: server, request: request)
    }

    func reset() {
        for task in inFlight.values {
            task.cancel()
        }
        inFlight = [:]
        cache = [:]
    }

    private static func loadSnapshot(
        using server: ServerConfig,
        request: NowPlayingPayloadRequest
    ) async -> NowPlayingPayloadSnapshot? {
        let client = APIClient(baseURL: server.baseURL)
        do {
            async let lyricsResponse: TrackLyricsResponse = client.getJSON(
                path: APIPath.trackLyrics,
                queryItems: [
                    URLQueryItem(name: "artist", value: request.artist),
                    URLQueryItem(name: "album", value: request.album ?? ""),
                    URLQueryItem(name: "track", value: request.track),
                    URLQueryItem(name: "trackNumber", value: request.trackNumber.map(String.init)),
                    URLQueryItem(name: "discNumber", value: request.discNumber.map(String.init))
                ]
            )
            async let insightResponse: TrackInsightResponse = client.getJSON(
                path: APIPath.trackInsight,
                queryItems: [
                    URLQueryItem(name: "artist", value: request.artist),
                    URLQueryItem(name: "album", value: request.album ?? ""),
                    URLQueryItem(name: "track", value: request.track),
                    URLQueryItem(name: "trackNumber", value: request.trackNumber.map(String.init)),
                    URLQueryItem(name: "discNumber", value: request.discNumber.map(String.init))
                ]
            )

            let lyrics = try await lyricsResponse
            let insight = try await insightResponse
            let lyricLines = await Task.detached(priority: .userInitiated) {
                LRCParser.parseLyrics(lyrics.lyrics, hasLRC: lyrics.hasLRC)
            }.value

            return NowPlayingPayloadSnapshot(
                lyricsResponse: lyrics,
                lyricLines: lyricLines,
                insightResponse: insight
            )
        } catch {
            return nil
        }
    }
}

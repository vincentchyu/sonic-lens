import Foundation

struct NowPlayingMessage: Decodable {
    let type: String
    let source: String?
    let data: NowPlaying?
}

struct NowPlaying: Decodable {
    static let pauseStaleTimeout: TimeInterval = 5
    static let inactiveTimeout: TimeInterval = 10

    let artist: String
    let album: String?
    let albumSubtitle: String?
    let track: String
    let duration: Int?
    let position: Int?
    let positionMs: Int?
    let sampleRate: Int?
    let artwork: String?
    let isAppleMusicFav: Bool?
    let isLastFmFav: Bool?
    let appleMusicState: TrackFavoriteState?
    let lastfmState: TrackFavoriteState?
    let favoriteState: TrackFavoriteState?
    let trackNumber: Int?
    let discNumber: Int?
    let genre: String?
    var receivedAt: Date = Date()

    enum CodingKeys: String, CodingKey {
        case artist
        case album
        case albumSubtitle = "album_subtitle"
        case track = "title"
        case duration
        case position
        case positionMs = "position_ms"
        case sampleRate = "sample_rate"
        case artwork = "cover_art_url"
        case isAppleMusicFav = "apple_music"
        case isLastFmFav = "lastfm"
        case appleMusicState = "apple_music_state"
        case lastfmState = "lastfm_state"
        case favoriteState = "favorite_state"
        case trackNumber = "track_number"
        case discNumber = "disc_number"
        case genre
    }

    var favoriteProjection: TrackFavoriteProjection {
        TrackFavoriteProjection(
            appleMusic: isAppleMusicFav ?? false,
            lastfm: isLastFmFav ?? false,
            appleMusicState: appleMusicState,
            lastfmState: lastfmState,
            favoriteState: favoriteState
        )
    }

    var displayAlbumTitle: String? {
        let title = album?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let subtitle = albumSubtitle?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !title.isEmpty else { return nil }
        guard !subtitle.isEmpty else { return title }
        return "\(title) (\(subtitle))"
    }

    var sampleRateDisplayText: String? {
        guard let sampleRate, sampleRate > 0 else { return nil }
        let kilohertz = Double(sampleRate) / 1000
        let rounded = (kilohertz * 10).rounded() / 10
        if rounded == floor(rounded) {
            return "\(Int(rounded))kHz"
        }
        return String(format: "%.1fkHz", rounded)
    }

    func playbackActivityState(at date: Date = Date()) -> PlaybackActivityState {
        let silence = date.timeIntervalSince(receivedAt)
        if silence >= Self.inactiveTimeout {
            return .inactive
        }
        if silence >= Self.pauseStaleTimeout {
            return .pausedStale
        }
        return .active
    }

    init(
        artist: String,
        album: String?,
        albumSubtitle: String? = nil,
        track: String,
        duration: Int?,
        position: Int?,
        positionMs: Int?,
        sampleRate: Int?,
        artwork: String?,
        isAppleMusicFav: Bool?,
        isLastFmFav: Bool?,
        appleMusicState: TrackFavoriteState?,
        lastfmState: TrackFavoriteState?,
        favoriteState: TrackFavoriteState?,
        trackNumber: Int?,
        discNumber: Int?,
        genre: String? = nil,
        receivedAt: Date = Date()
    ) {
        self.artist = artist
        self.album = album
        self.albumSubtitle = albumSubtitle
        self.track = track
        self.duration = duration
        self.position = position
        self.positionMs = positionMs
        self.sampleRate = sampleRate
        self.artwork = artwork
        self.isAppleMusicFav = isAppleMusicFav
        self.isLastFmFav = isLastFmFav
        self.appleMusicState = appleMusicState
        self.lastfmState = lastfmState
        self.favoriteState = favoriteState
        self.trackNumber = trackNumber
        self.discNumber = discNumber
        self.genre = genre
        self.receivedAt = receivedAt
    }
}

import Foundation

struct NowPlayingMessage: Decodable {
    let type: String
    let source: String?
    let data: NowPlaying?
}

struct NowPlaying: Decodable {
    let artist: String
    let album: String?
    let track: String
    let duration: Int?
    let position: Int?
    let positionMs: Int?
    let artwork: String?
    let isAppleMusicFav: Bool?
    let isLastFmFav: Bool?
    let trackNumber: Int?
    let discNumber: Int?

    enum CodingKeys: String, CodingKey {
        case artist
        case album
        case track = "title"
        case duration
        case position
        case positionMs = "position_ms"
        case artwork = "cover_art_url"
        case isAppleMusicFav = "apple_music"
        case isLastFmFav = "lastfm"
        case trackNumber = "track_number"
        case discNumber = "disc_number"
    }
}

import Foundation

struct FavoriteRequest: Encodable {
    let artist: String
    let album: String
    let track: String
    let trackNumber: Int?
    let discNumber: Int?
    let source: String
    let favorite: Bool

    enum CodingKeys: String, CodingKey {
        case artist
        case album
        case track
        case trackNumber = "track_number"
        case discNumber = "disc_number"
        case source
        case favorite
    }
}

struct FavoriteResponse: Decodable {
    let appleMusic: Bool
    let lastfm: Bool

    enum CodingKeys: String, CodingKey {
        case appleMusic = "apple_music"
        case lastfm
    }
}

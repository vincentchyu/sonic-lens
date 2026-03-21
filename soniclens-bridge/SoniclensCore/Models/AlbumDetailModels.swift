import Foundation

struct ReleaseCandidate: Codable, Identifiable {
    let id: Int64
    let mbid: String
    let albumID: Int64
    let name: String?
    let jsonData: String?

    enum CodingKeys: String, CodingKey {
        case id
        case mbid
        case albumID = "album_id"
        case name
        case jsonData = "json_data"
    }
}

struct AlbumReleaseMBLink: Codable {
    let id: Int64
    let albumID: Int64
    let releaseMBID: Int64
    let mbid: String
    let confirmed: Bool

    enum CodingKeys: String, CodingKey {
        case id
        case albumID = "album_id"
        case releaseMBID = "release_mb_id"
        case mbid
        case confirmed
    }
}

struct LinkAlbumRequest: Encodable {
    let albumID: Int64
    let releaseMBID: Int64
    let mbid: String

    enum CodingKeys: String, CodingKey {
        case albumID = "album_id"
        case releaseMBID = "release_mb_id"
        case mbid
    }
}

struct AlbumDetail: Codable, Identifiable {
    let id: Int64
    let name: String
    let artist: String
    let releaseDate: String?
    let genre: String?
    let totalDiscs: Int?
    let tracks: [Track]
    let releaseMB: AlbumReleaseMBLink?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case artist
        case releaseDate = "release_date"
        case genre
        case totalDiscs = "total_discs"
        case tracks
        case releaseMB = "release_mb"
    }
}

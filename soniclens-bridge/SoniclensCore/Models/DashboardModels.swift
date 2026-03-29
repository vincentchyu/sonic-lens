import Foundation

struct DashboardStats: Codable {
    let totalPlays: Int64
    let totalTracks: Int64
    let totalArtists: Int64
    let totalAlbums: Int64
}

struct TrendResponse: Codable {
    let daily: [String: Int]
    let hourly: [String: HourlyTrend]
}

struct HourlyTrend: Codable {
    let date: String
    let total: Int
    let hourly: [Int: Int]
}

struct TopArtist: Codable, Identifiable {
    var id: String { artist }
    let artist: String
    let playCount: Int?
    let trackCount: Int?

    enum CodingKeys: String, CodingKey {
        case artist
        case playCount = "play_count"
        case trackCount = "track_count"
    }
}

struct TopAlbum: Codable, Identifiable {
    let albumID: Int64
    let album: String
    let artist: String
    let playCount: Int

    var id: Int64 { albumID }

    enum CodingKeys: String, CodingKey {
        case albumID = "album_id"
        case album
        case artist
        case playCount = "play_count"
    }
}

struct TopGenre: Codable, Identifiable {
    let trackGenreName: String
    let trackGenreCount: Int64
    let genreNameZh: String
    let genreCount: Int64

    var id: String { trackGenreName }

    enum CodingKeys: String, CodingKey {
        case trackGenreName = "track_genre_name"
        case trackGenreCount = "track_genre_count"
        case genreNameZh = "genre_name_zh"
        case genreCount = "genre_count"
    }
}

struct RecentPlayRecord: Codable, Identifiable {
    let id: Int64
    let artist: String
    let album: String
    let track: String
    let playTime: String

    enum CodingKeys: String, CodingKey {
        case id
        case artist
        case album
        case track
        case playTime = "play_time"
    }
}

struct TrendPoint: Codable, Identifiable, Equatable {
    let date: String
    let count: Int

    var id: String { date }
}

struct HourlyData: Codable, Identifiable, Equatable {
    let date: String
    let hourly: [Int: Int]

    var id: String { date }

    func countForHour(_ hour: Int) -> Int {
        hourly[hour] ?? 0
    }
}

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
    let rank: Int?
    let avatarURL: String?
    let avatarMime: String?
    let avatarObjectKey: String?

    enum CodingKeys: String, CodingKey {
        case artist
        case playCount = "play_count"
        case trackCount = "track_count"
        case rank
        case avatarURL = "avatar_url"
        case avatarMime = "avatar_mime"
        case avatarObjectKey = "avatar_object_key"
    }
}

struct TopAlbum: Codable, Identifiable {
    let albumID: Int64
    let album: String
    let albumSubtitle: String?
    let artist: String
    let playCount: Int
    let coverArtURL: String?
    let coverArtMime: String?
    let coverArtObjectKey: String?

    var id: Int64 { albumID }

    enum CodingKeys: String, CodingKey {
        case albumID = "album_id"
        case album
        case albumSubtitle = "album_subtitle"
        case artist
        case playCount = "play_count"
        case coverArtURL = "cover_art_url"
        case coverArtMime = "cover_art_mime"
        case coverArtObjectKey = "cover_art_object_key"
    }

    var displayAlbum: String {
        let subtitle = albumSubtitle?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !subtitle.isEmpty else { return album }
        return "\(album) (\(subtitle))"
    }
}

struct TopTrack: Codable, Identifiable, Hashable {
    let trackID: Int64
    let track: String
    let album: String
    let artist: String
    let playCount: Int
    let rank: Int
    let coverArtURL: String?
    let coverArtMime: String?
    let coverArtObjectKey: String?

    var id: String { "\(trackID)-\(artist)-\(album)-\(track)" }

    enum CodingKeys: String, CodingKey {
        case trackID = "track_id"
        case track
        case album
        case artist
        case playCount = "play_count"
        case rank
        case coverArtURL = "cover_art_url"
        case coverArtMime = "cover_art_mime"
        case coverArtObjectKey = "cover_art_object_key"
    }

    var bridgeTrack: Track {
        Track(
            id: trackID,
            artist: artist,
            album: album,
            track: track,
            playCount: playCount,
            trackNumber: nil,
            discNumber: nil,
            duration: nil
        )
    }
}

struct TopGenre: Codable, Identifiable {
    let trackGenreName: String
    let trackGenreCount: Int64
    let genreNameZh: String
    let genreCount: Int64
    let rank: Int?

    var id: String { trackGenreName }

    enum CodingKeys: String, CodingKey {
        case trackGenreName = "track_genre_name"
        case trackGenreCount = "track_genre_count"
        case genreNameZh = "genre_name_zh"
        case genreCount = "genre_count"
        case rank
    }
}

struct RecentPlayRecord: Codable, Identifiable {
    let id: Int64
    let artist: String
    let album: String
    let albumSubtitle: String?
    let track: String
    let playTime: String
    let coverArtPath: String?

    enum CodingKeys: String, CodingKey {
        case id
        case artist
        case album
        case albumSubtitle = "album_subtitle"
        case track
        case playTime = "play_time"
        case coverArtPath = "cover_art_path"
    }

    var displayAlbum: String {
        let subtitle = albumSubtitle?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !subtitle.isEmpty else { return album }
        return "\(album) (\(subtitle))"
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

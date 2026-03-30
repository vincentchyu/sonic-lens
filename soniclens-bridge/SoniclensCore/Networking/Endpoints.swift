import Foundation

enum APIPath {
    static let dashboardStats = "/api/dashboard/stats"
    static let dashboardTrend = "/api/dashboard/trend"
    static let topArtistsPlays = "/api/dashboard/top-artists/plays"
    static let topArtistsTracks = "/api/dashboard/top-artists/tracks"
    static let playCountsBySource = "/api/dashboard/play-counts-by-source"
    static let topAlbums = "/api/dashboard/top-albums"
    static let topTracks = "/api/dashboard/top-tracks"
    static let topGenres = "/api/dashboard/top-genres"
    static let topTracksPeriod = "/api/track-play-counts/period"
    static let recentPlays = "/api/recent-plays"

    static let albums = "/api/albums"
    static let tracks = "/api/tracks"
    static let librarySync = "/api/library/sync"
    static let insights = "/api/insights/all"
    static func insightDetail(id: Int64) -> String { "/api/insights/\(id)" }
    static func insightLogs(id: Int64) -> String { "/api/insights/\(id)/logs" }
    static let unscrobbled = "/api/unscrobbled-records"
    static let unscrobbledCount = "/api/unscrobbled-records/count"

    static let trackLyrics = "/api/track-lyrics"
    static let aiModels = "/api/ai-models"
    static func aiPlatformModels(platformID: String) -> String { "/api/ai-models/\(platformID)/models" }
    static let trackInsight = "/api/track-insight"
    static let albumInsight = "/api/album-insight"
    static let trackInsightStream = "/api/track-insight-stream"
    static let insightJobs = "/api/insight-jobs"
    static func insightJob(id: String) -> String { "/api/insight-jobs/\(id)" }
    static func insightJobLiveActivityToken(id: String) -> String { "/api/insight-jobs/\(id)/live-activity-token" }
    static let favorite = "/api/favorite"
    static let artworkResolve = "/api/artwork/resolve"

    static let musicBrainzSearchReleases = "/api/musicbrainz/search-releases"
    static let musicBrainzCandidates = "/api/musicbrainz/candidates"
    static let musicBrainzLinkAlbum = "/api/musicbrainz/link-album"
    static let musicBrainzDeepMaintenance = "/api/musicbrainz/deep-maintenance"

    static let health = "/health"
}

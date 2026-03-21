import Foundation

enum APIPath {
    static let dashboardStats = "/api/dashboard/stats"
    static let dashboardTrend = "/api/dashboard/trend"
    static let topArtistsPlays = "/api/dashboard/top-artists/plays"
    static let topArtistsTracks = "/api/dashboard/top-artists/tracks"
    static let topAlbums = "/api/dashboard/top-albums"
    static let topGenres = "/api/dashboard/top-genres"
    static let topTracks = "/api/track-play-counts"
    static let topTracksPeriod = "/api/track-play-counts/period"
    static let recentPlays = "/api/recent-plays"

    static let albums = "/api/albums"
    static let tracks = "/api/tracks"
    static let librarySync = "/api/library/sync"
    static let insights = "/api/insights/all"
    static let unscrobbled = "/api/unscrobbled-records"

    static let trackLyrics = "/api/track-lyrics"
    static let trackInsight = "/api/track-insight"
    static let trackInsightStream = "/api/track-insight-stream"
    static let favorite = "/api/favorite"

    static let musicBrainzSearchReleases = "/api/musicbrainz/search-releases"
    static let musicBrainzCandidates = "/api/musicbrainz/candidates"
    static let musicBrainzLinkAlbum = "/api/musicbrainz/link-album"
    static let musicBrainzDeepMaintenance = "/api/musicbrainz/deep-maintenance"

    static let health = "/health"
}

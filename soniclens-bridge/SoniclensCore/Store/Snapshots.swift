import Foundation

struct HomeSnapshot: Codable {
    let stats: DashboardStats?
    let topArtistsByPlays: [TopArtist]
    let topArtistsByTracks: [TopArtist]
    let topAlbums: [TopAlbum]
    let topGenres: [TopGenre]
    let topTracks: [Track]
    let recentPlays: [RecentPlayRecord]
    let trendPoints: [TrendPoint]
    let hourlyData: [HourlyData]
    let selectedTrendRange: Int
}

struct LibrarySnapshot: Codable {
    let albums: [Album]
    let tracks: [Track]
    let insights: [Insight]
    let unscrobbled: [UnscrobbledRecord]
}

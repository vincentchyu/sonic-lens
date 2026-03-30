import Foundation

struct TrackDetailRoute: Identifiable, Hashable {
    let track: Track
    let selectedTab: TrackDetailTab

    var id: String {
        [
            track.artist,
            track.album,
            track.track,
            String(track.trackNumber ?? 0),
            String(track.discNumber ?? 0),
            selectedTab.rawValue
        ].joined(separator: "::")
    }
}

struct AlbumDetailRoute: Identifiable, Hashable {
    let albumID: Int64
    let selectedTab: AlbumDetailTab

    var id: String {
        "\(albumID)::\(selectedTab.rawValue)"
    }
}

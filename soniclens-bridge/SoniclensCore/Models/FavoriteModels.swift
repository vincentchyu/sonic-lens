import Foundation

enum TrackFavoriteState: String, Codable, Equatable {
    case notFavorited = "not_favorited"
    case favorited
    case favoritePending = "favorite_pending"
    case unfavoritePending = "unfavorite_pending"

    static func effective(_ isFavorited: Bool) -> TrackFavoriteState {
        isFavorited ? .favorited : .notFavorited
    }

    var isFavoritedEffective: Bool {
        switch self {
        case .favorited, .favoritePending:
            return true
        case .notFavorited, .unfavoritePending:
            return false
        }
    }

    var isPending: Bool {
        switch self {
        case .favoritePending, .unfavoritePending:
            return true
        case .notFavorited, .favorited:
            return false
        }
    }
}

struct TrackFavoriteProjection: Equatable {
    let appleMusic: Bool
    let lastfm: Bool
    let appleMusicState: TrackFavoriteState
    let lastfmState: TrackFavoriteState
    let favoriteState: TrackFavoriteState

    init(
        appleMusic: Bool,
        lastfm: Bool,
        appleMusicState: TrackFavoriteState? = nil,
        lastfmState: TrackFavoriteState? = nil,
        favoriteState: TrackFavoriteState? = nil
    ) {
        let resolvedAppleMusicState = appleMusicState ?? .effective(appleMusic)
        let resolvedLastfmState = lastfmState ?? .effective(lastfm)

        self.appleMusicState = resolvedAppleMusicState
        self.lastfmState = resolvedLastfmState
        self.appleMusic = resolvedAppleMusicState.isFavoritedEffective
        self.lastfm = resolvedLastfmState.isFavoritedEffective
        self.favoriteState = favoriteState ?? Self.aggregate(
            appleMusicState: resolvedAppleMusicState,
            lastfmState: resolvedLastfmState
        )
    }

    var isFavoritedEffective: Bool {
        favoriteState.isFavoritedEffective
    }

    var isPending: Bool {
        favoriteState.isPending || appleMusicState.isPending || lastfmState.isPending
    }

    private static func aggregate(
        appleMusicState: TrackFavoriteState,
        lastfmState: TrackFavoriteState
    ) -> TrackFavoriteState {
        let states = [appleMusicState, lastfmState]
        if states.contains(.unfavoritePending) {
            return .unfavoritePending
        }
        if states.contains(.favoritePending) {
            return .favoritePending
        }
        if states.contains(.favorited) {
            return .favorited
        }
        return .notFavorited
    }
}

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
    let appleMusicState: TrackFavoriteState?
    let lastfmState: TrackFavoriteState?
    let favoriteState: TrackFavoriteState?

    enum CodingKeys: String, CodingKey {
        case appleMusic = "apple_music"
        case lastfm
        case appleMusicState = "apple_music_state"
        case lastfmState = "lastfm_state"
        case favoriteState = "favorite_state"
    }

    var projection: TrackFavoriteProjection {
        TrackFavoriteProjection(
            appleMusic: appleMusic,
            lastfm: lastfm,
            appleMusicState: appleMusicState,
            lastfmState: lastfmState,
            favoriteState: favoriteState
        )
    }
}

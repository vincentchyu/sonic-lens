import Foundation
import Observation

struct FavoriteActionContext: Equatable {
    let artist: String
    let album: String
    let track: String
    let trackNumber: Int?
    let discNumber: Int?
    let source: String
    let favorite: Bool

    var identityKey: String {
        [
            artist,
            album,
            track,
            trackNumber.map(String.init) ?? "",
            discNumber.map(String.init) ?? "",
            source,
            favorite ? "1" : "0"
        ].joined(separator: "::")
    }
}

enum FavoriteActionState: Equatable {
    case idle
    case loading(FavoriteActionContext)
    case success(FavoriteActionContext, message: String)
    case failure(FavoriteActionContext, message: String)

    var isLoading: Bool {
        if case .loading = self {
            return true
        }
        return false
    }

    var notice: FavoriteActionNotice? {
        switch self {
        case .success(let context, let message):
            return FavoriteActionNotice(context: context, message: message, style: .success)
        case .failure(let context, let message):
            return FavoriteActionNotice(context: context, message: message, style: .failure)
        case .loading, .idle:
            return nil
        }
    }
}

struct FavoriteActionNotice: Equatable {
    enum Style {
        case success
        case failure
    }

    let context: FavoriteActionContext
    let message: String
    let style: Style
}

extension FavoriteActionContext {
    func matches(nowPlaying: NowPlaying) -> Bool {
        artist == nowPlaying.artist
            && album == (nowPlaying.album ?? "")
            && track == nowPlaying.track
            && trackNumber == nowPlaying.trackNumber
            && discNumber == nowPlaying.discNumber
    }
}

extension FavoriteActionState {
    func isLoading(matching nowPlaying: NowPlaying?) -> Bool {
        guard let nowPlaying else { return false }
        if case let .loading(context) = self {
            return context.matches(nowPlaying: nowPlaying)
        }
        return false
    }

    func notice(matching nowPlaying: NowPlaying?) -> FavoriteActionNotice? {
        guard let nowPlaying else { return nil }
        guard let notice else { return nil }
        guard notice.context.matches(nowPlaying: nowPlaying) else { return nil }
        return notice
    }
}

@MainActor
@Observable
final class FavoriteActionStore {
    private(set) var state: FavoriteActionState = .idle

    func setLoading(_ context: FavoriteActionContext) {
        state = .loading(context)
    }

    func setSuccess(_ context: FavoriteActionContext, message: String) {
        state = .success(context, message: message)
    }

    func setFailure(_ context: FavoriteActionContext, message: String) {
        state = .failure(context, message: message)
    }

    func clear() {
        state = .idle
    }
}

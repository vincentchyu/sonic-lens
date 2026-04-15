import Foundation
import XCTest
@testable import SoniclensBridgePhone

@MainActor
final class AppStoreFavoriteActionTests: XCTestCase {
    func testSetFavoritePublishesLoadingWhileRequestIsInFlight() async {
        let requestStarted = expectation(description: "request started")
        var resumeContinuation: CheckedContinuation<Void, Never>?
        let store = makeStore { _, _ in
            requestStarted.fulfill()
            await withCheckedContinuation { continuation in
                resumeContinuation = continuation
            }
            return makeSuccessResponse()
        }
        store.currentServer = makeServer()

        let task = Task {
            await store.setFavorite(
                artist: "Yes",
                album: "90125 (Deluxe Edition)",
                track: "Owner of a Lonely Heart",
                favorite: true,
                source: "Apple Music"
            )
        }

        await fulfillment(of: [requestStarted], timeout: 1.0)

        guard case let .loading(context) = store.favoriteActionStore.state else {
            XCTFail("应该在请求进行中进入 loading 状态")
            resumeContinuation?.resume()
            _ = await task.value
            return
        }
        XCTAssertEqual(context.artist, "Yes")
        XCTAssertEqual(context.album, "90125 (Deluxe Edition)")
        XCTAssertEqual(context.track, "Owner of a Lonely Heart")
        XCTAssertTrue(context.favorite)

        resumeContinuation?.resume()
        await task.value
    }

    func testSetFavoriteSuccessUpdatesProjectionAndPublishesSuccess() async {
        var capturedRequest: FavoriteRequest?
        let store = makeStore { _, request in
            capturedRequest = request
            return makeSuccessResponse()
        }
        store.currentServer = makeServer()

        await store.setFavorite(
            artist: "Yes",
            album: "90125 (Deluxe Edition)",
            track: "Owner of a Lonely Heart",
            trackNumber: 1,
            discNumber: 1,
            favorite: true,
            source: "Apple Music"
        )

        XCTAssertEqual(capturedRequest?.artist, "Yes")
        XCTAssertEqual(capturedRequest?.album, "90125 (Deluxe Edition)")
        XCTAssertEqual(capturedRequest?.track, "Owner of a Lonely Heart")
        XCTAssertEqual(capturedRequest?.trackNumber, 1)
        XCTAssertEqual(capturedRequest?.discNumber, 1)
        XCTAssertEqual(capturedRequest?.source, "Apple Music")
        XCTAssertEqual(capturedRequest?.favorite, true)

        guard case let .success(context, message) = store.favoriteActionStore.state else {
            XCTFail("应该在请求成功后进入 success 状态")
            return
        }
        XCTAssertEqual(context.artist, "Yes")
        XCTAssertEqual(context.album, "90125 (Deluxe Edition)")
        XCTAssertEqual(context.track, "Owner of a Lonely Heart")
        XCTAssertEqual(message, "已同步收藏到 Apple Music 和 Last.fm")
        XCTAssertTrue(store.isFavorite(
            artist: "Yes",
            album: "90125 (Deluxe Edition)",
            track: "Owner of a Lonely Heart",
            trackNumber: 1,
            discNumber: 1
        ))
    }

    func testSetFavoriteFailurePublishesFailureAndKeepsProjectionStable() async {
        let store = makeStore { _, _ in
            throw URLError(.notConnectedToInternet)
        }
        store.currentServer = makeServer()

        await store.setFavorite(
            artist: "Yes",
            album: "90125 (Deluxe Edition)",
            track: "Owner of a Lonely Heart",
            trackNumber: 1,
            discNumber: 1,
            favorite: true,
            source: "Apple Music"
        )

        guard case let .failure(context, message) = store.favoriteActionStore.state else {
            XCTFail("应该在请求失败后进入 failure 状态")
            return
        }
        XCTAssertEqual(context.artist, "Yes")
        XCTAssertEqual(context.album, "90125 (Deluxe Edition)")
        XCTAssertEqual(context.track, "Owner of a Lonely Heart")
        XCTAssertTrue(message.hasPrefix("收藏失败"))
        XCTAssertFalse(store.isFavorite(
            artist: "Yes",
            album: "90125 (Deluxe Edition)",
            track: "Owner of a Lonely Heart",
            trackNumber: 1,
            discNumber: 1
        ))
    }

    private func makeStore(
        favoriteRequestHandler: @escaping (URL, FavoriteRequest) async throws -> FavoriteResponse
    ) -> AppStore {
        AppStore(favoriteRequestHandler: favoriteRequestHandler)
    }

    private func makeServer() -> ServerConfig {
        ServerConfig(
            name: "Local",
            host: "127.0.0.1",
            port: 8080
        )
    }

    private func makeSuccessResponse() -> FavoriteResponse {
        FavoriteResponse(
            appleMusic: true,
            lastfm: true,
            appleMusicState: .favorited,
            lastfmState: .favorited,
            favoriteState: .favorited
        )
    }
}

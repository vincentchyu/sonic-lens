import Foundation
import XCTest
@testable import SoniclensBridgePhone

final class HomeHotModulePresentationTests: XCTestCase {
    func testTopAlbumDecodesWithReleaseDate() throws {
        let json = """
        {
            "album_id": 101,
            "album": "The Dark Side of the Moon",
            "album_subtitle": "50th Anniversary",
            "artist": "Pink Floyd",
            "release_date": "2023-03-24",
            "play_count": 88,
            "cover_art_url": "https://example.com/cover.jpg"
        }
        """
        let album = try JSONDecoder().decode(TopAlbum.self, from: Data(json.utf8))
        XCTAssertEqual(album.albumID, 101)
        XCTAssertEqual(album.album, "The Dark Side of the Moon")
        XCTAssertEqual(album.albumSubtitle, "50th Anniversary")
        XCTAssertEqual(album.releaseDate, "2023-03-24")
        XCTAssertEqual(album.releaseYear, "2023")
        XCTAssertEqual(album.displayAlbum, "The Dark Side of the Moon (50th Anniversary)")
        XCTAssertEqual(album.displaySubtitle, "Pink Floyd · 2023")
    }

    func testTopAlbumDecodesWithoutReleaseDateFallback() throws {
        let json = """
        {
            "album_id": 102,
            "album": "Abbey Road",
            "artist": "The Beatles",
            "play_count": 66
        }
        """
        let album = try JSONDecoder().decode(TopAlbum.self, from: Data(json.utf8))
        XCTAssertEqual(album.releaseDate, nil)
        XCTAssertEqual(album.releaseYear, nil)
        XCTAssertEqual(album.displayAlbum, "Abbey Road")
        XCTAssertEqual(album.displaySubtitle, "The Beatles")
    }

    func testMakeTracksDoesNotCrashOnDuplicateOrIdenticalAlbums() {
        // 同名专辑，但子标题、年份不同或即使完全重复
        let albumA = TopAlbum(
            albumID: 1,
            album: "Weezer",
            albumSubtitle: "Blue Album",
            artist: "Weezer",
            releaseDate: "1994-05-10",
            playCount: 100,
            coverArtURL: "https://example.com/blue.jpg"
        )
        let albumB = TopAlbum(
            albumID: 2,
            album: "Weezer",
            albumSubtitle: "Green Album",
            artist: "Weezer",
            releaseDate: "2001-05-15",
            playCount: 90,
            coverArtURL: "https://example.com/green.jpg"
        )
        // 包含完全相同 artist + album 的重复项
        let albumC = TopAlbum(
            albumID: 3,
            album: "Weezer",
            albumSubtitle: nil,
            artist: "Weezer",
            releaseDate: nil,
            playCount: 50,
            coverArtURL: "https://example.com/weezer.jpg"
        )

        let trackA = TopTrack(
            trackID: 10,
            track: "Buddy Holly",
            album: "Weezer",
            artist: "Weezer",
            playCount: 40,
            rank: 1,
            coverArtURL: nil,
            coverArtMime: nil,
            coverArtObjectKey: nil
        )

        // 核心断言：在执行 HomeHotModulePresentation 初始化时绝对不能因为 Dictionary(uniqueKeysWithValues:) 崩溃
        let presentation = HomeHotModulePresentation(
            topGenres: [],
            topArtists: [],
            topAlbums: [albumA, albumB, albumC],
            topTracks: [trackA],
            playSourceCounts: [:],
            stats: nil
        )

        XCTAssertEqual(presentation.albums.count, 3)
        XCTAssertEqual(presentation.albums[0].title, "Weezer (Blue Album)")
        XCTAssertEqual(presentation.albums[0].subtitle, "Weezer · 1994")
        XCTAssertEqual(presentation.albums[1].title, "Weezer (Green Album)")
        XCTAssertEqual(presentation.albums[1].subtitle, "Weezer · 2001")
        XCTAssertEqual(presentation.albums[2].title, "Weezer")
        XCTAssertEqual(presentation.albums[2].subtitle, "Weezer")

        XCTAssertEqual(presentation.tracks.count, 1)
        XCTAssertNotNil(presentation.tracks[0].artworkPath)
    }
}

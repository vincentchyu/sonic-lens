import XCTest
@testable import SoniclensBridgePhone

final class SharePayloadBuilderTests: XCTestCase {
    func testTrackInsightPayloadBuildsOrderedCards() throws {
        let payload = SharePayloadBuilder.build(
            scene: .trackInsight,
            track: makeTrack(duration: 187, discNumber: 2, trackNumber: 8),
            resolvedArtworkURL: "https://example.com/artwork.png",
            lyrics: nil,
            insight: try makeTrackInsight(),
            isFavorite: true
        )

        guard case let .insight(insightPayload) = payload else {
            XCTFail("应该生成曲目音眸 payload")
            return
        }

        XCTAssertEqual(insightPayload.header.trackName, "The Track")
        XCTAssertEqual(insightPayload.header.artworkURL, "https://example.com/artwork.png")
        XCTAssertEqual(insightPayload.document.cards.map(\.id), [
            "lyrics_translation",
            "analysis_summary",
            "analysis_by_section",
            "background_info",
            "era_context"
        ])
    }

    func testTrackInfoPayloadUsesStructuredFields() {
        let payload = SharePayloadBuilder.build(
            scene: .trackInfo,
            track: makeTrack(duration: 187, discNumber: 2, trackNumber: 8),
            resolvedArtworkURL: "https://example.com/artwork.png",
            lyrics: nil,
            insight: nil,
            isFavorite: true
        )

        guard case let .info(infoPayload) = payload else {
            XCTFail("应该生成曲目信息 payload")
            return
        }

        XCTAssertEqual(infoPayload.meta.items.map(\.id), ["play_count", "duration"])
        XCTAssertEqual(infoPayload.fields.map(\.id), ["analysis_summary", "literary_analysis"])
        XCTAssertEqual(infoPayload.fields.last?.maxCharacterCount, 200)
        XCTAssertNotNil(infoPayload.fields.last?.note)
    }

    func testAlbumInfoPayloadBuildsHeaderAndTeaserFields() throws {
        let payload = SharePayloadBuilder.buildAlbum(
            scene: .albumInfo,
            albumDetail: makeAlbumDetail(),
            albumInsight: try makeAlbumInsight(),
            resolvedArtworkURL: "https://example.com/album.png"
        )

        guard case let .albumInfo(infoPayload) = payload else {
            XCTFail("应该生成专辑基础信息 payload")
            return
        }

        XCTAssertEqual(infoPayload.header.trackName, "The Album")
        XCTAssertEqual(infoPayload.header.subtitleText, "The Artist")
        XCTAssertFalse(infoPayload.header.showsFavoriteBadge)
        XCTAssertEqual(infoPayload.meta.items.map(\.id), ["track_count", "duration", "release_date"])
        XCTAssertEqual(infoPayload.fields.map(\.id), [
            "analysis_summary",
            "key_tracks",
            "philosophical_reflection",
            "listening_guide"
        ])
        XCTAssertEqual(infoPayload.fields.dropFirst().compactMap(\.maxCharacterCount), [120, 120, 120])
        XCTAssertEqual(infoPayload.fields.dropFirst().compactMap(\.note).count, 3)
    }

    func testAlbumInsightPayloadIncludesFullSchemaWithoutDebugMetadata() throws {
        let payload = SharePayloadBuilder.buildAlbum(
            scene: .albumInsight,
            albumDetail: makeAlbumDetail(),
            albumInsight: try makeAlbumInsight(),
            resolvedArtworkURL: nil
        )

        guard case let .albumInsight(insightPayload) = payload else {
            XCTFail("应该生成专辑音眸 payload")
            return
        }

        XCTAssertEqual(insightPayload.document.cards.map(\.id), [
            "analysis_summary",
            "analysis_by_section",
            "background_info",
            "era_context"
        ])
    }

    func testAlbumDetailDecodesOriginalReleaseDate() throws {
        let json = """
        {
          "id": 42,
          "name": "The Album",
          "artist": "The Artist",
          "release_date": "2024-01-01",
          "original_release_date": "2023-11-17",
          "cover_art_url": null,
          "cover_art_mime": null,
          "cover_art_object_key": null,
          "genre": "Art Pop",
          "total_discs": 1,
          "tracks": [],
          "release_mb": null
        }
        """

        let detail = try JSONDecoder().decode(AlbumDetail.self, from: Data(json.utf8))

        XCTAssertEqual(detail.releaseDate, "2024-01-01")
        XCTAssertEqual(detail.originalReleaseDate, "2023-11-17")
    }

    func testPrimaryInsightSelectionRemainsFirstForShareContract() throws {
        let trackPrimary = try makeTrackInsight()
        let trackAlternate = try makeTrackInsight(id: 2, provider: "AltProvider", createdAt: "2026-03-27 09:00")
        XCTAssertEqual([trackPrimary, trackAlternate].primaryInsight?.id, trackPrimary.id)

        let albumPrimary = try makeAlbumInsight()
        let albumAlternate = try makeAlbumInsight(id: 11, provider: "AltProvider", createdAt: "2026-03-27 11:00")
        XCTAssertEqual([albumPrimary, albumAlternate].primaryInsight?.id, albumPrimary.id)
    }

    private func makeTrack(duration: Int64?, discNumber: Int?, trackNumber: Int?) -> Track {
        Track(
            id: 1,
            artist: "The Artist",
            album: "The Album",
            track: "The Track",
            playCount: 12,
            trackNumber: trackNumber,
            discNumber: discNumber,
            duration: duration
        )
    }

    private func makeAlbumDetail() -> AlbumDetail {
        AlbumDetail(
            id: 42,
            name: "The Album",
            artist: "The Artist",
            releaseDate: "2024-01-01",
            coverArtURL: nil,
            coverArtMime: nil,
            coverArtObjectKey: nil,
            genre: "Art Pop",
            totalDiscs: 1,
            tracks: [
                Track(id: 1, artist: "The Artist", album: "The Album", track: "Track One", playCount: 4, trackNumber: 1, discNumber: 1, duration: 180),
                Track(id: 2, artist: "The Artist", album: "The Album", track: "Track Two", playCount: 5, trackNumber: 2, discNumber: 1, duration: 205)
            ],
            releaseMB: nil
        )
    }

    private func makeTrackInsight(
        id: Int64 = 1,
        provider: String = "MockProvider",
        createdAt: String = "2026-03-24 12:00"
    ) throws -> Insight {
        let json = """
        {
          "id": \(id),
          "artist": "The Artist",
          "album": "The Album",
          "track": "The Track",
          "analysis_summary": "这是一首在编曲和文本之间彼此照亮的作品。",
          "lyrics_translation": "<original>Hello world</original><translation>你好，世界</translation>",
          "analysis_by_section": {
            "literary_analysis": "文学层面重点关注意象。",
            "musical_analysis": "编曲从稀薄到饱满推进。"
          },
          "background_info": "录制于巡演结束后的间歇期。",
          "era_context": "发布时正值风格转型期。",
          "llm_provider": "\(provider)",
          "metadata": null,
          "created_at": "\(createdAt)",
          "total_score": 95
        }
        """

        return try JSONDecoder().decode(Insight.self, from: Data(json.utf8))
    }

    private func makeAlbumInsight(
        id: Int64 = 10,
        provider: String = "MockProvider",
        createdAt: String = "2026-03-26 10:00"
    ) throws -> AlbumInsight {
        let json = """
        {
          "id": \(id),
          "album_id": 42,
          "artist": "The Artist",
          "album": "The Album",
          "analysis_summary": "这是对艺术家创作母题的一次完整展开。",
          "analysis_by_section": {
            "album_positioning": "处在创作生涯转向期。",
            "theme_and_narrative": "从克制开场到最终释怀。",
            "literary_analysis": "文本中反复出现海与城市的意象。",
            "musical_analysis": "曲序从极简质感逐步推向宏大声场。",
            "author_motivation": "创作者试图回应身份与关系的摇摆。",
            "philosophical_reflection": "专辑持续追问自由与归属的边界。",
            "key_tracks": "Track One 和 Track Two 分别承担命题提出与回响。",
            "listening_guide": "建议从首尾曲的呼应关系切入。"
          },
          "background_info": "制作期横跨两个城市。",
          "era_context": "它发布在流媒体审美趋同最明显的几年里。",
          "llm_provider": "\(provider)",
          "metadata": {
            "album_id": 42,
            "analyzed_tracks": 2,
            "total_tracks": 2
          },
          "created_at": "\(createdAt)"
        }
        """

        return try JSONDecoder().decode(AlbumInsight.self, from: Data(json.utf8))
    }
}

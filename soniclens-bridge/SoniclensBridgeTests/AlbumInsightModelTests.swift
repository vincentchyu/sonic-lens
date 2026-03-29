import XCTest
@testable import SoniclensBridgePhone

final class AlbumInsightModelTests: XCTestCase {
    func testAlbumInsightDecodesStringifiedSectionsInSchemaOrder() throws {
        let json = """
        {
          "id": 1,
          "album_id": 8,
          "artist": "The Artist",
          "album": "The Album",
          "analysis_summary": "summary",
          "analysis_by_section": "{\\"key_tracks\\":\\"tracks\\",\\"album_positioning\\":\\"position\\",\\"listening_guide\\":\\"guide\\",\\"theme_and_narrative\\":\\"theme\\",\\"appendix\\":\\"extra\\"}",
          "background_info": "",
          "era_context": null,
          "llm_provider": "Mock",
          "metadata": "{\\"album_id\\":8,\\"method\\":\\"aggregate\\"}",
          "created_at": "2026-03-26 10:00"
        }
        """

        let insight = try JSONDecoder().decode(AlbumInsight.self, from: Data(json.utf8))

        XCTAssertEqual(insight.orderedSections.map(\.id), [
            "album_positioning",
            "theme_and_narrative",
            "key_tracks",
            "listening_guide",
            "appendix"
        ])
        XCTAssertEqual(insight.metadata?.displayText?.contains("\"method\""), true)
    }

    func testAlbumInsightHandlesEmptySectionStringAndMissingMetadata() throws {
        let json = """
        {
          "id": 2,
          "album_id": 9,
          "artist": "The Artist",
          "album": "The Album",
          "analysis_summary": "summary",
          "analysis_by_section": "",
          "background_info": null,
          "era_context": null,
          "llm_provider": "Mock",
          "created_at": "2026-03-26 10:00"
        }
        """

        let insight = try JSONDecoder().decode(AlbumInsight.self, from: Data(json.utf8))

        XCTAssertTrue(insight.orderedSections.isEmpty)
        XCTAssertNil(insight.metadata)
    }
}

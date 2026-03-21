import Foundation

struct Album: Codable, Identifiable, Hashable {
    let id: Int64
    let name: String
    let artist: String
    let releaseDate: String?
    let genre: String?
    let totalDiscs: Int?
    let playCount: Int?
    let createdAt: String?
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case artist
        case releaseDate = "release_date"
        case genre
        case totalDiscs = "total_discs"
        case playCount = "play_count"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    init(
        id: Int64,
        name: String,
        artist: String,
        releaseDate: String?,
        genre: String?,
        totalDiscs: Int?,
        playCount: Int? = nil,
        createdAt: String? = nil,
        updatedAt: String? = nil
    ) {
        self.id = id
        self.name = name
        self.artist = artist
        self.releaseDate = releaseDate
        self.genre = genre
        self.totalDiscs = totalDiscs
        self.playCount = playCount
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

struct Track: Codable, Identifiable, Hashable {
    let id: Int64
    let artist: String
    let album: String
    let track: String
    let playCount: Int
    let trackNumber: Int?
    let discNumber: Int?
    let duration: Int64?
    let isAppleMusicFav: Bool?
    let isLastFmFav: Bool?
    let createdAt: String?
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case artist
        case album
        case track
        case playCount = "play_count"
        case trackNumber = "track_number"
        case discNumber = "disc_number"
        case duration
        case isAppleMusicFav = "is_apple_music_fav"
        case isLastFmFav = "is_last_fm_fav"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    init(
        id: Int64,
        artist: String,
        album: String,
        track: String,
        playCount: Int,
        trackNumber: Int?,
        discNumber: Int?,
        duration: Int64?,
        isAppleMusicFav: Bool? = nil,
        isLastFmFav: Bool? = nil,
        createdAt: String? = nil,
        updatedAt: String? = nil
    ) {
        self.id = id
        self.artist = artist
        self.album = album
        self.track = track
        self.playCount = playCount
        self.trackNumber = trackNumber
        self.discNumber = discNumber
        self.duration = duration
        self.isAppleMusicFav = isAppleMusicFav
        self.isLastFmFav = isLastFmFav
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }

    var isFavorited: Bool {
        (isAppleMusicFav ?? false) || (isLastFmFav ?? false)
    }
}

struct PaginatedAlbums: Codable {
    let albums: [Album]
    let total: Int64
    let limit: Int
    let offset: Int
}

struct PaginatedTracks: Codable {
    let tracks: [Track]
    let total: Int64
    let limit: Int
    let offset: Int
}

struct LibrarySyncResponse: Codable {
    let syncVersion: Int64
    let generatedAt: String
    let albums: [Album]
    let tracks: [Track]
    let deletedAlbumIDs: [Int64]
    let deletedTrackIDs: [Int64]

    enum CodingKeys: String, CodingKey {
        case syncVersion = "sync_version"
        case generatedAt = "generated_at"
        case albums
        case tracks
        case deletedAlbumIDs = "deleted_album_ids"
        case deletedTrackIDs = "deleted_track_ids"
    }
}

struct Insight: Codable, Identifiable, Hashable {
    let id: Int64
    let artist: String
    let album: String
    let track: String
    let analysisSummary: String?
    let lyricsTranslation: String?
    let analysisBySection: InsightAnalysisSections
    let backgroundInfo: String?
    let eraContext: String?
    let llmProvider: String?
    let metadata: String?
    let createdAt: String?
    let totalScore: Int?

    enum CodingKeys: String, CodingKey {
        case id
        case artist
        case album
        case track
        case analysisSummary = "analysis_summary"
        case lyricsTranslation = "lyrics_translation"
        case analysisBySection = "analysis_by_section"
        case backgroundInfo = "background_info"
        case eraContext = "era_context"
        case llmProvider = "llm_provider"
        case metadata
        case createdAt = "created_at"
        case totalScore = "total_score"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(Int64.self, forKey: .id)
        artist = try container.decode(String.self, forKey: .artist)
        album = try container.decode(String.self, forKey: .album)
        track = try container.decode(String.self, forKey: .track)
        analysisSummary = try container.decodeIfPresent(String.self, forKey: .analysisSummary)
        lyricsTranslation = try container.decodeIfPresent(String.self, forKey: .lyricsTranslation)
        analysisBySection = try container.decodeIfPresent(InsightAnalysisSections.self, forKey: .analysisBySection) ?? .empty
        backgroundInfo = try container.decodeIfPresent(String.self, forKey: .backgroundInfo)
        eraContext = try container.decodeIfPresent(String.self, forKey: .eraContext)
        llmProvider = try container.decodeIfPresent(String.self, forKey: .llmProvider)
        metadata = try container.decodeIfPresent(String.self, forKey: .metadata)
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt)
        totalScore = try container.decodeIfPresent(Int.self, forKey: .totalScore)
    }

    var displayBlocks: [InsightDisplayBlock] {
        var blocks: [InsightDisplayBlock] = []

        if let summary = analysisSummary?.trimmedOrNil {
            blocks.append(.text(id: "summary", title: "曲目解读", text: summary))
        }
        if let translation = lyricsTranslation?.trimmedOrNil {
            blocks.append(.text(id: "lyrics_translation", title: "歌词对照", text: translation))
        }
        let sectionBlocks = analysisBySection.orderedBlocks
        if !sectionBlocks.isEmpty {
            blocks.append(.sections(id: "analysis_by_section", title: "分段解析", sections: sectionBlocks))
        }
        if let background = backgroundInfo?.trimmedOrNil {
            blocks.append(.text(id: "background_info", title: "背景信息", text: background))
        }
        if let era = eraContext?.trimmedOrNil {
            blocks.append(.text(id: "era_context", title: "时代语境", text: era))
        }

        return blocks
    }

    var hasDisplayContent: Bool {
        !displayBlocks.isEmpty
    }

    var teaserText: String? {
        analysisSummary?.trimmedOrNil ?? eraContext?.trimmedOrNil
    }

    var providerLine: String? {
        let provider = llmProvider?.trimmedOrNil
        let createdAt = createdAt?.trimmedOrNil
        switch (provider, createdAt) {
        case let (.some(provider), .some(createdAt)):
            return "\(provider) · \(createdAt)"
        case let (.some(provider), nil):
            return provider
        case let (nil, .some(createdAt)):
            return createdAt
        default:
            return nil
        }
    }
}

struct PaginatedInsights: Codable {
    let insights: [Insight]
    let total: Int64
    let limit: Int
    let offset: Int
}

struct UnscrobbledRecord: Codable, Identifiable {
    let id: Int64
    let artist: String
    let album: String
    let track: String
    let playTime: String
    let source: String

    enum CodingKeys: String, CodingKey {
        case id
        case artist
        case album
        case track
        case playTime = "play_time"
        case source
    }
}

struct InsightAnalysisSections: Codable, Hashable {
    fileprivate static let empty = InsightAnalysisSections(values: [:])

    let values: [String: String]

    init(values: [String: String]) {
        self.values = values
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()

        if let values = try? container.decode([String: String].self) {
            self.values = values
            return
        }

        if let rawString = try? container.decode(String.self) {
            let trimmed = rawString.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmed.isEmpty else {
                values = [:]
                return
            }

            if let data = trimmed.data(using: .utf8),
               let decoded = try? JSONDecoder().decode([String: String].self, from: data) {
                values = decoded
            } else {
                values = ["appreciate_analysis": trimmed]
            }
            return
        }

        values = [:]
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        try container.encode(values)
    }

    var orderedBlocks: [InsightSectionBlock] {
        let titleMap: [String: String] = [
            "literary_analysis": "文学解读",
            "musical_analysis": "乐评分析",
            "cultural_context": "文化背景",
            "translation_notes": "翻译说明",
            "appreciate_analysis": "分句赏析"
        ]
        let orderedKeys = [
            "literary_analysis",
            "musical_analysis",
            "cultural_context",
            "translation_notes",
            "appreciate_analysis"
        ]

        var blocks: [InsightSectionBlock] = orderedKeys.compactMap { key in
            guard let value = values[key]?.trimmedOrNil else { return nil }
            return InsightSectionBlock(id: key, title: titleMap[key] ?? key, content: value)
        }

        let unknownKeys = values.keys
            .filter { !orderedKeys.contains($0) }
            .sorted()

        blocks.append(
            contentsOf: unknownKeys.compactMap { key in
                guard let value = values[key]?.trimmedOrNil else { return nil }
                return InsightSectionBlock(id: key, title: titleMap[key] ?? key, content: value)
            }
        )

        return blocks
    }
}

struct InsightSectionBlock: Hashable, Identifiable {
    let id: String
    let title: String
    let content: String
}

enum InsightDisplayBlock: Hashable, Identifiable {
    case text(id: String, title: String, text: String)
    case sections(id: String, title: String, sections: [InsightSectionBlock])

    var id: String {
        switch self {
        case let .text(id, _, _):
            return id
        case let .sections(id, _, _):
            return id
        }
    }

    var title: String {
        switch self {
        case let .text(_, title, _):
            return title
        case let .sections(_, title, _):
            return title
        }
    }
}

enum InsightTaggedSegment: Hashable {
    case text(String)
    case rows([InsightTaggedRow])
    case explain(String)
}

struct InsightTaggedRow: Hashable {
    let original: String
    let translation: String
}

enum InsightTaggedContentParser {
    private static let tagPattern = #"<(original|translation|explain)>([\s\S]*?)<(?:/)?\1>"#

    static func parse(_ text: String) -> [InsightTaggedSegment]? {
        let cleaned = text.replacingOccurrences(of: "\\n", with: "\n")
        guard let regex = try? NSRegularExpression(pattern: tagPattern, options: [.caseInsensitive]) else {
            return nil
        }

        let nsText = cleaned as NSString
        let matches = regex.matches(in: cleaned, options: [], range: NSRange(location: 0, length: nsText.length))
        guard !matches.isEmpty else { return nil }

        var segments: [InsightTaggedSegment] = []
        var rows: [InsightTaggedRow] = []
        var lastIndex = 0

        func flushRows() {
            guard !rows.isEmpty else { return }
            segments.append(.rows(rows))
            rows.removeAll(keepingCapacity: true)
        }

        for match in matches {
            let beforeRange = NSRange(location: lastIndex, length: match.range.location - lastIndex)
            if beforeRange.length > 0 {
                let before = nsText.substring(with: beforeRange).trimmingCharacters(in: .whitespacesAndNewlines)
                if !before.isEmpty {
                    flushRows()
                    segments.append(.text(before))
                }
            }

            let tag = nsText.substring(with: match.range(at: 1)).lowercased()
            let content = nsText.substring(with: match.range(at: 2)).trimmingCharacters(in: .whitespacesAndNewlines)

            switch tag {
            case "original":
                rows.append(InsightTaggedRow(original: content, translation: ""))
            case "translation":
                if rows.isEmpty || !rows[rows.count - 1].translation.isEmpty {
                    rows.append(InsightTaggedRow(original: "", translation: content))
                } else {
                    let lastRow = rows.removeLast()
                    rows.append(InsightTaggedRow(original: lastRow.original, translation: content))
                }
            case "explain":
                flushRows()
                if !content.isEmpty {
                    segments.append(.explain(content))
                }
            default:
                break
            }

            lastIndex = match.range.location + match.range.length
        }

        let tailRange = NSRange(location: lastIndex, length: nsText.length - lastIndex)
        if tailRange.length > 0 {
            let tail = nsText.substring(with: tailRange).trimmingCharacters(in: .whitespacesAndNewlines)
            if !tail.isEmpty {
                flushRows()
                segments.append(.text(tail))
            } else {
                flushRows()
            }
        } else {
            flushRows()
        }

        return segments
    }
}

extension Collection where Element == Insight {
    var primaryInsight: Insight? {
        first
    }
}

private extension String {
    var trimmedOrNil: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}

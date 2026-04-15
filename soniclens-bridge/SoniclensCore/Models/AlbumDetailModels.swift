import Foundation

struct ReleaseCandidate: Codable, Identifiable {
    let id: Int64
    let mbid: String
    let albumID: Int64
    let name: String?
    let jsonData: String?

    enum CodingKeys: String, CodingKey {
        case id
        case mbid
        case albumID = "album_id"
        case name
        case jsonData = "json_data"
    }
}

struct AlbumReleaseMBLink: Codable {
    let id: Int64
    let albumID: Int64
    let releaseMBID: Int64
    let mbid: String
    let confirmed: Bool

    enum CodingKeys: String, CodingKey {
        case id
        case albumID = "album_id"
        case releaseMBID = "release_mb_id"
        case mbid
        case confirmed
    }
}

struct LinkAlbumRequest: Encodable {
    let albumID: Int64
    let releaseMBID: Int64
    let mbid: String

    enum CodingKeys: String, CodingKey {
        case albumID = "album_id"
        case releaseMBID = "release_mb_id"
        case mbid
    }
}

struct AlbumDetail: Codable, Identifiable {
    let id: Int64
    let name: String
    let nameSubtitle: String?
    let artist: String
    let releaseDate: String?
    let originalReleaseDate: String?
    let coverArtURL: String?
    let coverArtMime: String?
    let coverArtObjectKey: String?
    let genre: String?
    let totalDiscs: Int?
    let tracks: [Track]
    let releaseMB: AlbumReleaseMBLink?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case nameSubtitle = "name_subtitle"
        case artist
        case releaseDate = "release_date"
        case originalReleaseDate = "original_release_date"
        case coverArtURL = "cover_art_url"
        case coverArtMime = "cover_art_mime"
        case coverArtObjectKey = "cover_art_object_key"
        case genre
        case totalDiscs = "total_discs"
        case tracks
        case releaseMB = "release_mb"
    }

    var displayName: String {
        let subtitle = nameSubtitle?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !subtitle.isEmpty else { return name }
        return "\(name) (\(subtitle))"
    }

    init(
        id: Int64,
        name: String,
        nameSubtitle: String? = nil,
        artist: String,
        releaseDate: String?,
        originalReleaseDate: String? = nil,
        coverArtURL: String?,
        coverArtMime: String?,
        coverArtObjectKey: String?,
        genre: String?,
        totalDiscs: Int?,
        tracks: [Track],
        releaseMB: AlbumReleaseMBLink?
    ) {
        self.id = id
        self.name = name
        self.nameSubtitle = nameSubtitle
        self.artist = artist
        self.releaseDate = releaseDate
        self.originalReleaseDate = originalReleaseDate
        self.coverArtURL = coverArtURL
        self.coverArtMime = coverArtMime
        self.coverArtObjectKey = coverArtObjectKey
        self.genre = genre
        self.totalDiscs = totalDiscs
        self.tracks = tracks
        self.releaseMB = releaseMB
    }
}

struct AlbumInsight: Codable, Identifiable, Hashable {
    let id: Int64
    let albumID: Int64?
    let artist: String
    let album: String
    let analysisSummary: String?
    let analysisBySection: InsightAnalysisSections
    let backgroundInfo: String?
    let eraContext: String?
    let llmProvider: String?
    let metadata: AlbumInsightMetadata?
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case albumID = "album_id"
        case artist
        case album
        case analysisSummary = "analysis_summary"
        case analysisBySection = "analysis_by_section"
        case backgroundInfo = "background_info"
        case eraContext = "era_context"
        case llmProvider = "llm_provider"
        case metadata
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(Int64.self, forKey: .id)
        albumID = try container.decodeIfPresent(Int64.self, forKey: .albumID)
        artist = try container.decode(String.self, forKey: .artist)
        album = try container.decode(String.self, forKey: .album)
        analysisSummary = try container.decodeIfPresent(String.self, forKey: .analysisSummary)
        analysisBySection = try container.decodeIfPresent(InsightAnalysisSections.self, forKey: .analysisBySection) ?? .empty
        backgroundInfo = try container.decodeIfPresent(String.self, forKey: .backgroundInfo)
        eraContext = try container.decodeIfPresent(String.self, forKey: .eraContext)
        llmProvider = try container.decodeIfPresent(String.self, forKey: .llmProvider)
        metadata = try container.decodeIfPresent(AlbumInsightMetadata.self, forKey: .metadata)
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt)
    }

    var orderedSections: [InsightSectionBlock] {
        analysisBySection.albumOrderedBlocks
    }

    var providerLine: String? {
        let provider = llmProvider?.trimmedNonEmpty
        let createdAt = createdAt?.trimmedNonEmpty
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

    var hasDisplayContent: Bool {
        analysisSummary?.trimmedNonEmpty != nil
            || !orderedSections.isEmpty
            || backgroundInfo?.trimmedNonEmpty != nil
            || eraContext?.trimmedNonEmpty != nil
    }
}

indirect enum JSONValue: Codable, Hashable {
    case string(String)
    case number(Double)
    case bool(Bool)
    case object([String: JSONValue])
    case array([JSONValue])
    case null

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()

        if container.decodeNil() {
            self = .null
        } else if let bool = try? container.decode(Bool.self) {
            self = .bool(bool)
        } else if let intValue = try? container.decode(Int.self) {
            self = .number(Double(intValue))
        } else if let doubleValue = try? container.decode(Double.self) {
            self = .number(doubleValue)
        } else if let stringValue = try? container.decode(String.self) {
            self = .string(stringValue)
        } else if let objectValue = try? container.decode([String: JSONValue].self) {
            self = .object(objectValue)
        } else if let arrayValue = try? container.decode([JSONValue].self) {
            self = .array(arrayValue)
        } else {
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Unsupported JSON value")
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case let .string(value):
            try container.encode(value)
        case let .number(value):
            try container.encode(value)
        case let .bool(value):
            try container.encode(value)
        case let .object(value):
            try container.encode(value)
        case let .array(value):
            try container.encode(value)
        case .null:
            try container.encodeNil()
        }
    }

    fileprivate var foundationObject: Any {
        switch self {
        case let .string(value):
            return value
        case let .number(value):
            return value
        case let .bool(value):
            return value
        case let .object(value):
            return value.mapValues(\.foundationObject)
        case let .array(value):
            return value.map(\.foundationObject)
        case .null:
            return NSNull()
        }
    }
}

enum AlbumInsightMetadata: Codable, Hashable {
    case text(String)
    case json(JSONValue)

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()

        if container.decodeNil() {
            self = .text("")
            return
        }

        if let rawString = try? container.decode(String.self) {
            let trimmed = rawString.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmed.isEmpty else {
                self = .text("")
                return
            }

            if let data = trimmed.data(using: .utf8),
               let decoded = try? JSONDecoder().decode(JSONValue.self, from: data) {
                self = .json(decoded)
            } else {
                self = .text(trimmed)
            }
            return
        }

        let value = try JSONValue(from: decoder)
        self = .json(value)
    }

    func encode(to encoder: Encoder) throws {
        switch self {
        case let .text(value):
            var container = encoder.singleValueContainer()
            try container.encode(value)
        case let .json(value):
            try value.encode(to: encoder)
        }
    }

    var displayText: String? {
        switch self {
        case let .text(value):
            return value.trimmedNonEmpty
        case let .json(value):
            if case .null = value {
                return nil
            }
            guard JSONSerialization.isValidJSONObject(value.foundationObject),
                  let data = try? JSONSerialization.data(withJSONObject: value.foundationObject, options: [.prettyPrinted, .sortedKeys]),
                  let string = String(data: data, encoding: .utf8) else {
                return nil
            }
            return string.trimmedNonEmpty
        }
    }
}

private extension String {
    var trimmedNonEmpty: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}

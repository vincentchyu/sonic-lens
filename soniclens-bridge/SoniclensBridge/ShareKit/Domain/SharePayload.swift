import CoreGraphics
import Foundation

enum ShareScene: String, CaseIterable, Hashable {
    case trackInsight
    case trackInfo
    case trackLyrics
    case albumInfo
    case albumInsight

    var title: String {
        switch self {
        case .trackInsight:
            return "音眸解析"
        case .trackInfo:
            return "曲目信息"
        case .trackLyrics:
            return "歌词长图"
        case .albumInfo:
            return "专辑信息"
        case .albumInsight:
            return "专辑音眸"
        }
    }

    var exportFilenameSuffix: String {
        switch self {
        case .trackInsight:
            return "音眸"
        case .trackInfo:
            return "信息"
        case .trackLyrics:
            return "歌词"
        case .albumInfo:
            return "专辑信息"
        case .albumInsight:
            return "专辑音眸"
        }
    }
}

struct ShareHeaderPayload: Hashable {
    let artworkURL: String?
    let trackName: String
    let artistName: String
    let albumName: String
    let sceneTitle: String
    let positionTag: String?
    let isFavorite: Bool
    let showsFavoriteBadge: Bool
    let subtitleText: String?
    let artworkFallbackTitle: String?
    let metricTags: [ShareMetaItem]
}

struct ShareMetaItem: Identifiable, Hashable {
    let id: String
    let title: String
    let value: String
    let systemImage: String
}

struct ShareMetaPayload: Hashable {
    let items: [ShareMetaItem]
}

struct ShareFooterPayload: Hashable {
    let brandText: String
    let sloganText: String
    let authorText: String
    let timestampText: String?
}

struct InsightShareRow: Identifiable, Hashable {
    let id: UUID
    let original: String
    let translation: String?

    init(id: UUID = UUID(), original: String, translation: String?) {
        self.id = id
        self.original = original
        self.translation = translation
    }
}

struct InsightShareGroup: Identifiable, Hashable {
    let id: UUID
    let rows: [InsightShareRow]
    let explain: String?

    init(id: UUID = UUID(), rows: [InsightShareRow], explain: String?) {
        self.id = id
        self.rows = rows
        self.explain = explain
    }
}

struct InsightShareSection: Identifiable, Hashable {
    let id: String
    let title: String
    let text: String?
    let groups: [InsightShareGroup]
}

enum InsightShareCard: Identifiable, Hashable {
    case text(id: String, title: String, text: String)
    case tagged(id: String, title: String, groups: [InsightShareGroup], text: String?)
    case section(id: String, title: String, sections: [InsightShareSection])

    var id: String {
        switch self {
        case let .text(id, _, _):
            return id
        case let .tagged(id, _, _, _):
            return id
        case let .section(id, _, _):
            return id
        }
    }
}

struct InsightShareDocument: Hashable {
    let cards: [InsightShareCard]

    var isEmpty: Bool {
        cards.isEmpty
    }
}

struct ShareTextBlock: Identifiable, Hashable {
    let id: String
    let title: String?
    let text: String
}

struct ShareInfoField: Identifiable, Hashable {
    let id: String
    let title: String
    let value: String
    let note: String?
    let maxCharacterCount: Int?
}

struct TrackInsightSharePayload: Hashable {
    let header: ShareHeaderPayload
    let meta: ShareMetaPayload
    let document: InsightShareDocument
    let footer: ShareFooterPayload
}

struct TrackLyricsSharePayload: Hashable {
    let header: ShareHeaderPayload
    let meta: ShareMetaPayload
    let blocks: [ShareTextBlock]
    let footer: ShareFooterPayload
}

struct TrackInfoSharePayload: Hashable {
    let header: ShareHeaderPayload
    let meta: ShareMetaPayload
    let fields: [ShareInfoField]
    let footer: ShareFooterPayload
}

struct AlbumInfoSharePayload: Hashable {
    let header: ShareHeaderPayload
    let meta: ShareMetaPayload
    let fields: [ShareInfoField]
    let footer: ShareFooterPayload
}

struct AlbumInsightSharePayload: Hashable {
    let header: ShareHeaderPayload
    let meta: ShareMetaPayload
    let document: InsightShareDocument
    let footer: ShareFooterPayload
}

enum SharePayload: Hashable {
    case insight(TrackInsightSharePayload)
    case lyrics(TrackLyricsSharePayload)
    case info(TrackInfoSharePayload)
    case albumInfo(AlbumInfoSharePayload)
    case albumInsight(AlbumInsightSharePayload)

    var scene: ShareScene {
        switch self {
        case .insight:
            return .trackInsight
        case .lyrics:
            return .trackLyrics
        case .info:
            return .trackInfo
        case .albumInfo:
            return .albumInfo
        case .albumInsight:
            return .albumInsight
        }
    }

    var header: ShareHeaderPayload {
        switch self {
        case let .insight(payload):
            return payload.header
        case let .lyrics(payload):
            return payload.header
        case let .info(payload):
            return payload.header
        case let .albumInfo(payload):
            return payload.header
        case let .albumInsight(payload):
            return payload.header
        }
    }

    var filename: String {
        let artist = header.artistName.replacingOccurrences(of: "/", with: "-")
        let track = header.trackName.replacingOccurrences(of: "/", with: "-")
        return "\(artist)-\(track)-\(scene.exportFilenameSuffix)"
    }
}

struct ShareRenderResult {
    let fileURLs: [URL]
    let logicalSize: CGSize

    var pageCount: Int {
        fileURLs.count
    }
}

struct SharePreviewRequest: Identifiable {
    let id = UUID()
    let payload: SharePayload
}

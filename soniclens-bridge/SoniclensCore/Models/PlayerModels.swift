import Foundation

struct TrackLyricsResponse: Decodable {
    let lyrics: String
    let hasLRC: Bool

    enum CodingKeys: String, CodingKey {
        case lyrics
        case hasLRC = "has_lrc"
    }
}

struct LyricLine: Identifiable {
    let id = UUID()
    let time: TimeInterval?
    let text: String
    let isSectionLabel: Bool
}

/// 播放静默状态，用于区分“仍在更新”“已暂停/已停止更新”和“无活动播放”。
enum PlaybackActivityState: Equatable {
    case active
    case pausedStale
    case inactive

    var isActive: Bool {
        self == .active
    }

    var bannerText: String? {
        switch self {
        case .active:
            return nil
        case .pausedStale:
            return "已暂停/已停止更新"
        case .inactive:
            return nil
        }
    }

    var isInactive: Bool {
        self == .inactive
    }
}

enum LRCParser {
    static func parseLyrics(_ raw: String, hasLRC: Bool) -> [LyricLine] {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return [] }

        let shouldParseTimedLyrics = hasLRC || containsTimedLyrics(trimmed)
        if !shouldParseTimedLyrics {
            return trimmed
                .components(separatedBy: .newlines)
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .filter { !$0.isEmpty }
                .map { LyricLine(time: nil, text: $0, isSectionLabel: false) }
        }

        let pattern = #"\[(\d{1,2}):(\d{2})(?:\.(\d{1,3}))?\]"#
        guard let regex = try? NSRegularExpression(pattern: pattern, options: []) else {
            return []
        }

        let metadataRegex = try? NSRegularExpression(pattern: #"^\[[a-z]{1,8}:[^\]]*\]$"#, options: [.caseInsensitive])
        let sectionRegex = try? NSRegularExpression(
            pattern: #"^\[((?:verse|chorus|bridge|intro|outro|pre-chorus|post-chorus|hook|refrain|interlude))\]$"#,
            options: [.caseInsensitive]
        )
        let lines = trimmed.components(separatedBy: .newlines)
        var result: [LyricLine] = []

        for line in lines {
            let nsLine = line as NSString
            let range = NSRange(location: 0, length: nsLine.length)
            let matches = regex.matches(in: line, options: [], range: range)
            let cleaned = regex.stringByReplacingMatches(in: line, options: [], range: range, withTemplate: "")
                .trimmingCharacters(in: .whitespacesAndNewlines)

            if matches.isEmpty {
                guard !cleaned.isEmpty else { continue }
                if let metadataRegex,
                   metadataRegex.firstMatch(in: cleaned, options: [], range: NSRange(location: 0, length: (cleaned as NSString).length)) != nil {
                    continue
                }
                if let sectionRegex,
                   let match = sectionRegex.firstMatch(
                    in: cleaned,
                    options: [],
                    range: NSRange(location: 0, length: (cleaned as NSString).length)
                   ) {
                    let section = (cleaned as NSString).substring(with: match.range(at: 1))
                    result.append(LyricLine(time: nil, text: section, isSectionLabel: true))
                } else {
                    result.append(LyricLine(time: nil, text: cleaned, isSectionLabel: false))
                }
                continue
            }

            guard !cleaned.isEmpty else { continue }
            for match in matches {
                let minuteText = nsLine.substring(with: match.range(at: 1))
                let secondText = nsLine.substring(with: match.range(at: 2))
                let fractionText = match.range(at: 3).location != NSNotFound
                    ? nsLine.substring(with: match.range(at: 3))
                    : nil
                guard let time = parseTime(minuteText: minuteText, secondText: secondText, fractionText: fractionText) else {
                    continue
                }
                result.append(LyricLine(time: time, text: cleaned, isSectionLabel: false))
            }
        }

        return result
    }

    private static func containsTimedLyrics(_ raw: String) -> Bool {
        let pattern = #"\[\d{1,2}:\d{2}(?:\.\d{1,3})?\]"#
        guard let regex = try? NSRegularExpression(pattern: pattern, options: []) else {
            return false
        }
        let range = NSRange(location: 0, length: (raw as NSString).length)
        return regex.firstMatch(in: raw, options: [], range: range) != nil
    }

    private static func parseTime(minuteText: String, secondText: String, fractionText: String?) -> TimeInterval? {
        guard let minutes = TimeInterval(minuteText),
              let seconds = TimeInterval(secondText),
              seconds >= 0, seconds < 60 else {
            return nil
        }

        let milliseconds: TimeInterval
        if let fractionText, !fractionText.isEmpty {
            let normalized = fractionText.padding(toLength: 3, withPad: "0", startingAt: 0)
            milliseconds = (TimeInterval(normalized) ?? 0) / 1000
        } else {
            milliseconds = 0
        }

        return minutes * 60 + seconds + milliseconds
    }
}

struct TrackInsightResponse: Decodable {
    let insights: [Insight]
}

struct AlbumInsightResponse: Decodable {
    let insights: [AlbumInsight]
}

struct AIPlatformOption: Decodable, Identifiable, Hashable {
    let id: String
    let displayName: String
    let defaultModel: String?

    enum CodingKeys: String, CodingKey {
        case id
        case displayName = "display_name"
        case defaultModel = "default_model"
    }
}

struct AIModelOption: Decodable, Identifiable, Hashable {
    let id: String
    let displayName: String
    let isDefault: Bool

    enum CodingKeys: String, CodingKey {
        case id
        case displayName = "display_name"
        case isDefault = "is_default"
    }
}

struct AIPlatformListResponse: Decodable {
    let platforms: [AIPlatformOption]
}

struct AIModelListResponse: Decodable {
    let models: [AIModelOption]
}

struct TrackInsightGenerateRequest: Encodable {
    let artist: String
    let album: String
    let track: String
    let trackNumber: Int?
    let discNumber: Int?
    let provider: String
    let model: String

    enum CodingKeys: String, CodingKey {
        case artist
        case album
        case track
        case trackNumber = "track_number"
        case discNumber = "disc_number"
        case provider
        case model
    }
}

struct TrackInsightGenerateResponse: Decodable {
    let insights: [Insight]
    let cached: Bool?
}

struct AlbumInsightGenerateRequest: Encodable {
    let albumID: Int64
    let provider: String
    let model: String

    enum CodingKeys: String, CodingKey {
        case albumID = "album_id"
        case provider
        case model
    }
}

struct AlbumInsightGenerateResponse: Decodable {
    let insights: [AlbumInsight]
    let cached: Bool?
}

struct HealthResponse: Decodable {
    let status: String
}

struct WebSocketEnvelope: Decodable {
    let type: String
}

struct LibraryUpdatedMessage: Decodable {
    let type: String
    let version: Int64
}

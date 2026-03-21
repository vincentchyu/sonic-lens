import Foundation

enum ShareMetadataFormatter {
    private static let footerFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "zh_CN")
        formatter.dateFormat = "yyyy-MM-dd HH:mm"
        return formatter
    }()

    static func makeMetaItems(track: Track, isFavorite: Bool) -> [ShareMetaItem] {
        var items: [ShareMetaItem] = [
            ShareMetaItem(
                id: "play_count",
                title: "播放次数",
                value: "\(track.playCount)",
                systemImage: "waveform"
            )
        ]

        if let duration = track.duration {
            items.append(
                ShareMetaItem(
                    id: "duration",
                    title: "时长",
                    value: formatDuration(duration),
                    systemImage: "clock"
                )
            )
        }

        return items
    }

    static func makeAlbumMetaItems(albumDetail: AlbumDetail, totalDuration: Int64) -> [ShareMetaItem] {
        var items: [ShareMetaItem] = [
            ShareMetaItem(
                id: "track_count",
                title: "曲目数",
                value: "\(albumDetail.tracks.count)",
                systemImage: "music.note.list"
            ),
            ShareMetaItem(
                id: "duration",
                title: "总时长",
                value: formatDuration(totalDuration),
                systemImage: "clock"
            )
        ]

        if let releaseDate = trimmed(albumDetail.releaseDate) {
            items.append(
                ShareMetaItem(
                    id: "release_date",
                    title: "发行时间",
                    value: releaseDate,
                    systemImage: "calendar"
                )
            )
        }

        return items
    }

    static func makeFooter() -> ShareFooterPayload {
        ShareFooterPayload(
            brandText: "音眸轨迹",
            sloganText: "声之透镜 · 深度解析 · 聆听之印记",
            authorText: "vincentchyu",
            timestampText: footerFormatter.string(from: Date())
        )
    }

    static func makeLyricsBlocks(from lyrics: TrackLyricsResponse?) -> [ShareTextBlock] {
        guard let lyrics else { return [] }

        let lines = LRCParser.parseLyrics(lyrics.lyrics, hasLRC: lyrics.hasLRC)
        guard !lines.isEmpty else {
            let trimmed = trimmed(lyrics.lyrics)
            guard let trimmed else { return [] }
            return [ShareTextBlock(id: "lyrics_full", title: nil, text: trimmed)]
        }

        let lineTexts = lines
            .map(\.text)
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }

        guard !lineTexts.isEmpty else { return [] }
        return [
            ShareTextBlock(
                id: "lyrics_full",
                title: nil,
                text: lineTexts.joined(separator: "\n")
            )
        ]
    }

    static func formatDuration(_ duration: Int64) -> String {
        let totalSeconds = Int(duration)
        guard totalSeconds > 0 else { return "—" }
        let hours = totalSeconds / 3600
        let minutes = totalSeconds / 60
        let seconds = totalSeconds % 60
        if hours > 0 {
            let remainingMinutes = (totalSeconds % 3600) / 60
            return String(format: "%d:%02d:%02d", hours, remainingMinutes, seconds)
        }
        return String(format: "%02d:%02d", minutes, seconds)
    }

    static func formatDiscTrack(trackNumber: Int?, discNumber: Int?) -> String? {
        switch (discNumber, trackNumber) {
        case let (.some(disc), .some(track)):
            return "DISC \(disc) · TRACK \(track)"
        case let (.some(disc), nil):
            return "DISC \(disc)"
        case let (nil, .some(track)):
            return "TRACK \(track)"
        default:
            return nil
        }
    }

    static func trimmed(_ text: String?) -> String? {
        guard let text else { return nil }
        let value = text.trimmingCharacters(in: .whitespacesAndNewlines)
        return value.isEmpty ? nil : value
    }
}

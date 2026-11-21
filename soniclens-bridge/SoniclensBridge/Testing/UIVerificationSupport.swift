import Foundation
import SwiftUI

/// UI 验证确定性数据档位
enum UIVerificationFixtureKind: String {
    case dense
    case empty
    case longText
}

/// UI 自动化与快照验证启动参数解析（仅在 DEBUG 生效，Release 恒为 disabled）
struct UIVerificationLaunchConfig: Equatable {
    let isEnabled: Bool
    let fixtureKind: UIVerificationFixtureKind
    let initialTabRawValue: String?
    let preferredColorScheme: ColorScheme?
    let performanceModeOverride: Bool?

    static let disabled = UIVerificationLaunchConfig(
        isEnabled: false,
        fixtureKind: .dense,
        initialTabRawValue: nil,
        preferredColorScheme: nil,
        performanceModeOverride: nil
    )

    static let current: UIVerificationLaunchConfig = {
        #if DEBUG
        return parse(arguments: ProcessInfo.processInfo.arguments)
        #else
        return .disabled
        #endif
    }()

    static func parse(arguments: [String]) -> UIVerificationLaunchConfig {
        guard arguments.contains("-ui-mock") else {
            return .disabled
        }

        let fixtureRaw = argumentValue(after: "-ui-fixture", in: arguments)
        let fixtureKind = fixtureRaw.flatMap(UIVerificationFixtureKind.init(rawValue:)) ?? .dense
        let initialTabRawValue = argumentValue(after: "-ui-tab", in: arguments)

        let appearanceRaw = argumentValue(after: "-ui-appearance", in: arguments)?.lowercased()
        let preferredColorScheme: ColorScheme?
        switch appearanceRaw {
        case "light":
            preferredColorScheme = .light
        case "dark":
            preferredColorScheme = .dark
        default:
            preferredColorScheme = nil
        }

        let performanceRaw = argumentValue(after: "-ui-performance-mode", in: arguments)?.lowercased()
        let performanceModeOverride: Bool?
        switch performanceRaw {
        case "on", "true", "1":
            performanceModeOverride = true
        case "off", "false", "0":
            performanceModeOverride = false
        default:
            performanceModeOverride = nil
        }

        return UIVerificationLaunchConfig(
            isEnabled: true,
            fixtureKind: fixtureKind,
            initialTabRawValue: initialTabRawValue,
            preferredColorScheme: preferredColorScheme,
            performanceModeOverride: performanceModeOverride
        )
    }

    private static func argumentValue(after flag: String, in arguments: [String]) -> String? {
        guard let index = arguments.firstIndex(of: flag), index + 1 < arguments.count else {
            return nil
        }
        return arguments[index + 1]
    }
}

/// 100% 确定性的 UI 夹具数据源（固定基准日期 2026-11-20），供快照、XCUITest 与 Preview 共用
enum UIVerificationFixtures {
    static let mockServer = ServerConfig(name: "UI Fixture", host: "127.0.0.1", port: 8082)

    static let referenceDateString = "2026-11-20"
    static let peakDateString = "2026-10-24"

    private static let calendar: Calendar = {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 8 * 3600) ?? .current
        return calendar
    }()

    private static let dayFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.calendar = calendar
        formatter.timeZone = calendar.timeZone
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter
    }()

    /// 生成截至 2026-11-20 的最近 days 天趋势与分时分布（不含未来日期）
    static func trendData(
        days: Int = 90,
        kind: UIVerificationFixtureKind = .dense
    ) -> (points: [TrendPoint], hourlyData: [HourlyData]) {
        guard kind != .empty, days > 0 else {
            return ([], [])
        }
        guard let endDate = dayFormatter.date(from: referenceDateString) else {
            return ([], [])
        }

        var points: [TrendPoint] = []
        var hourlyList: [HourlyData] = []
        points.reserveCapacity(days)
        hourlyList.reserveCapacity(days)

        for offset in stride(from: -(days - 1), through: 0, by: 1) {
            guard let date = calendar.date(byAdding: .day, value: offset, to: endDate) else {
                continue
            }
            let dateKey = dayFormatter.string(from: date)
            let hourly: [Int: Int]

            if dateKey == peakDateString {
                // 2026-10-24 峰值日：共 144 次播放，在 00..08 点形成高亮长柱
                hourly = [
                    0: 18, 1: 16, 2: 14, 3: 15, 4: 16,
                    5: 17, 6: 16, 7: 16, 8: 16
                ]
            } else {
                let dayIndex = offset + days
                switch dayIndex % 7 {
                case 0:
                    hourly = [12: 3, 13: 5, 14: 4, 17: 6, 18: 4, 21: 5, 22: 3]
                case 2:
                    hourly = [9: 2, 10: 4, 15: 6, 16: 8, 22: 4, 23: 5]
                case 4:
                    hourly = [11: 3, 12: 7, 13: 5, 19: 4, 20: 6]
                case 5:
                    hourly = [14: 2, 15: 4, 16: 5, 23: 3]
                default:
                    hourly = [:]
                }
            }

            let total = hourly.values.reduce(0, +)
            points.append(TrendPoint(date: dateKey, count: total))
            hourlyList.append(HourlyData(date: dateKey, hourly: hourly))
        }

        return (points, hourlyList)
    }

    static func homeSnapshot(
        days: Int = 90,
        kind: UIVerificationFixtureKind = .dense
    ) -> HomeSnapshot {
        if kind == .empty {
            return HomeSnapshot(
                stats: DashboardStats(totalPlays: 0, totalTracks: 0, totalArtists: 0, totalAlbums: 0),
                topArtistsByPlays: [],
                topArtistsByTracks: [],
                playSourceCounts: [:],
                topAlbums: [],
                topGenres: [],
                topTracks: [],
                recentPlays: [],
                trendPoints: [],
                hourlyData: [],
                selectedTrendRange: days
            )
        }

        let trend = trendData(days: days, kind: kind)
        let totalPlaysInWindow = Int64(trend.points.map(\.count).reduce(0, +))
        let isLong = kind == .longText

        let albums = [
            TopAlbum(
                albumID: 1,
                album: isLong ? "The Dark Side of the Moon (Super Extended Collector Archival Mastering)" : "The Dark Side of the Moon",
                albumSubtitle: "50th Anniversary Remaster",
                artist: isLong ? "Pink Floyd & London Philharmonic Studio Ensemble" : "Pink Floyd",
                releaseDate: "1973-03-01",
                playCount: 148,
                coverArtURL: nil
            ),
            TopAlbum(
                albumID: 2,
                album: isLong ? "万能青年旅店 — 冀西南林路行（十周年黑胶母带纪念典藏版）" : "冀西南林路行",
                albumSubtitle: nil,
                artist: "万能青年旅店",
                releaseDate: "2020-12-22",
                playCount: 112,
                coverArtURL: nil
            ),
            TopAlbum(
                albumID: 3,
                album: "In Rainbows",
                albumSubtitle: "Discbox Edition",
                artist: "Radiohead",
                releaseDate: "2007-10-10",
                playCount: 94,
                coverArtURL: nil
            ),
            TopAlbum(
                albumID: 4,
                album: "Kid A",
                albumSubtitle: nil,
                artist: "Radiohead",
                releaseDate: "2000-10-02",
                playCount: 79,
                coverArtURL: nil
            ),
            TopAlbum(
                albumID: 5,
                album: "Abbey Road",
                albumSubtitle: "2019 Mix",
                artist: "The Beatles",
                releaseDate: "1969-09-26",
                playCount: 63,
                coverArtURL: nil
            )
        ]

        let artists = [
            TopArtist(artist: isLong ? "Pink Floyd & London Philharmonic Studio Ensemble" : "Pink Floyd", playCount: 312, trackCount: 42, rank: 1, avatarURL: nil, avatarMime: nil, avatarObjectKey: nil),
            TopArtist(artist: "Radiohead", playCount: 248, trackCount: 36, rank: 2, avatarURL: nil, avatarMime: nil, avatarObjectKey: nil),
            TopArtist(artist: "万能青年旅店", playCount: 186, trackCount: 18, rank: 3, avatarURL: nil, avatarMime: nil, avatarObjectKey: nil),
            TopArtist(artist: "The Beatles", playCount: 132, trackCount: 27, rank: 4, avatarURL: nil, avatarMime: nil, avatarObjectKey: nil),
            TopArtist(artist: "声音玩具", playCount: 96, trackCount: 14, rank: 5, avatarURL: nil, avatarMime: nil, avatarObjectKey: nil),
            TopArtist(artist: "Miles Davis", playCount: 84, trackCount: 19, rank: 6, avatarURL: nil, avatarMime: nil, avatarObjectKey: nil)
        ]

        let genres = [
            TopGenre(trackGenreName: "Progressive Rock", trackGenreCount: 412, genreNameZh: "前卫摇滚", genreCount: 1840, rank: 1),
            TopGenre(trackGenreName: "Alternative Rock", trackGenreCount: 289, genreNameZh: "另类摇滚", genreCount: 1320, rank: 2),
            TopGenre(trackGenreName: "Art Rock", trackGenreCount: 175, genreNameZh: "艺术摇滚", genreCount: 860, rank: 3),
            TopGenre(trackGenreName: "Jazz Fusion", trackGenreCount: 118, genreNameZh: "爵士融合", genreCount: 540, rank: 4)
        ]

        let tracks = [
            TopTrack(trackID: 101, track: isLong ? "Shine On You Crazy Diamond (Parts I–V) — Live at Wembley Empire Pool 1974" : "Time", album: "The Dark Side of the Moon", artist: "Pink Floyd", playCount: 46, rank: 1, coverArtURL: nil, coverArtMime: nil, coverArtObjectKey: nil),
            TopTrack(trackID: 102, track: "山雀", album: "冀西南林路行", artist: "万能青年旅店", playCount: 39, rank: 2, coverArtURL: nil, coverArtMime: nil, coverArtObjectKey: nil),
            TopTrack(trackID: 103, track: "Weird Fishes / Arpeggi", album: "In Rainbows", artist: "Radiohead", playCount: 34, rank: 3, coverArtURL: nil, coverArtMime: nil, coverArtObjectKey: nil),
            TopTrack(trackID: 104, track: "Everything In Its Right Place", album: "Kid A", artist: "Radiohead", playCount: 28, rank: 4, coverArtURL: nil, coverArtMime: nil, coverArtObjectKey: nil),
            TopTrack(trackID: 105, track: "Here Comes the Sun", album: "Abbey Road", artist: "The Beatles", playCount: 24, rank: 5, coverArtURL: nil, coverArtMime: nil, coverArtObjectKey: nil)
        ]

        let recentPlays = [
            RecentPlayRecord(id: 501, artist: "Pink Floyd", album: "The Dark Side of the Moon", albumSubtitle: "50th Anniversary", track: isLong ? "The Great Gig in the Sky (2023 Remastered High-Resolution Studio Master)" : "Time", genre: "Progressive Rock", duration: 413, trackNumber: 4, discNumber: 1, resolvedTrackID: 101, playTime: "2026-11-20 10:30:00", coverArtPath: nil),
            RecentPlayRecord(id: 502, artist: "万能青年旅店", album: "冀西南林路行", albumSubtitle: nil, track: "采石", genre: "Alternative Rock", duration: 365, trackNumber: 5, discNumber: 1, resolvedTrackID: 102, playTime: "2026-11-20 09:48:00", coverArtPath: nil),
            RecentPlayRecord(id: 503, artist: "Radiohead", album: "In Rainbows", albumSubtitle: nil, track: "Nude", genre: "Art Rock", duration: 255, trackNumber: 3, discNumber: 1, resolvedTrackID: 103, playTime: "2026-11-19 22:14:00", coverArtPath: nil)
        ]

        return HomeSnapshot(
            stats: DashboardStats(totalPlays: max(totalPlaysInWindow, 1271), totalTracks: 1842, totalArtists: 318, totalAlbums: 246),
            topArtistsByPlays: artists,
            topArtistsByTracks: artists,
            playSourceCounts: ["Apple Music": 740, "Audirvana": 385, "Roon": 146],
            topAlbums: albums,
            topGenres: genres,
            topTracks: tracks,
            recentPlays: recentPlays,
            trendPoints: trend.points,
            hourlyData: trend.hourlyData,
            selectedTrendRange: days
        )
    }

    static func libraryAlbums(kind: UIVerificationFixtureKind = .dense) -> [Album] {
        guard kind != .empty else { return [] }
        let isLong = kind == .longText
        return (1...12).map { idx in
            let name = isLong
                ? "Archival Studio Collection Volume \(idx) — Special Extended Edition"
                : "Sample Album \(idx)"
            let subtitle: String? = idx.isMultiple(of: 2) ? "Deluxe Remaster" : nil
            let artist = isLong
                ? "Various Orchestral Performers & Guest Soloists \(idx)"
                : "Artist \(idx)"
            let genre = idx.isMultiple(of: 2) ? "Progressive Rock" : "Jazz Fusion"
            let releaseDate = "202\(idx % 6)-05-1\(idx % 9)"
            let totalDiscs = idx.isMultiple(of: 3) ? 2 : 1
            return Album(
                id: Int64(idx),
                name: name,
                nameSubtitle: subtitle,
                artist: artist,
                releaseDate: releaseDate,
                hasInsight: idx <= 4,
                genre: genre,
                totalDiscs: totalDiscs,
                playCount: 160 - idx * 9
            )
        }
    }

    static func libraryTracks(kind: UIVerificationFixtureKind = .dense) -> [Track] {
        guard kind != .empty else { return [] }
        let isLong = kind == .longText
        return (1...20).map { idx in
            let artistIndex = (idx - 1) % 6 + 1
            let albumIndex = (idx - 1) % 5 + 1
            let artist = isLong
                ? "Ensemble Performer & Guest Collaborator \(idx)"
                : "Artist \(artistIndex)"
            let album = isLong
                ? "Extended Double Disc Studio Session Volume \(idx)"
                : "Album \(albumIndex)"
            let trackTitle = isLong
                ? "Symphonic Movement No. \(idx) in D Minor (2026 Remastered Master)"
                : "Track Title \(idx)"
            let genre = idx.isMultiple(of: 2) ? "Progressive Rock" : "Art Rock"
            let duration = Int64(180 + idx * 11)
            return Track(
                id: Int64(idx),
                artist: artist,
                album: album,
                track: trackTitle,
                playCount: 60 - idx * 2,
                trackNumber: idx,
                discNumber: 1,
                duration: duration,
                genre: genre,
                isAppleMusicFav: idx.isMultiple(of: 3),
                isLastFmFav: idx.isMultiple(of: 4)
            )
        }
    }

    static func unscrobbledRecords(kind: UIVerificationFixtureKind = .dense) -> [UnscrobbledRecord] {
        guard kind != .empty else { return [] }
        return [
            UnscrobbledRecord(id: 1, artist: "Pink Floyd", album: "Animals", albumSubtitle: "2018 Remix", track: "Dogs", playTime: "2026-11-20 08:15:00", source: "Audirvana"),
            UnscrobbledRecord(id: 2, artist: "万能青年旅店", album: "万能青年旅店", albumSubtitle: nil, track: "杀死那个石家庄人", playTime: "2026-11-19 23:40:00", source: "Apple Music")
        ]
    }
}

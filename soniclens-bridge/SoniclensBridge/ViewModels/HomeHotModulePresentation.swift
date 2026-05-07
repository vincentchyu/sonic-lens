import Foundation

enum HomeHotAccentKey: String, CaseIterable {
    case tide
    case amber
    case mint
    case coral
    case indigo
    case slate

    static func accent(forRank rank: Int) -> HomeHotAccentKey {
        switch rank {
        case ..<1:
            return .slate
        case 1:
            return .amber
        case 2:
            return .coral
        case 3:
            return .mint
        case 4:
            return .tide
        case 5:
            return .indigo
        default:
            return .slate
        }
    }
}

enum HomeHotModuleNavigationTarget: Equatable {
    case none
    case album(Int64)
    case artist(String)
    case genre(String)
    case track(Int64)
}

struct HomeHotGenrePresentationItem: Identifiable, Equatable {
    let id: String
    let title: String
    let count: Int
    let secondaryCount: Int
    let rank: Int
    let relativeWeight: Double
    let accentKey: HomeHotAccentKey
    let navigationTarget: HomeHotModuleNavigationTarget
}

struct HomeHotArtistPresentationItem: Identifiable, Equatable {
    let id: String
    let title: String
    let count: Int
    let secondaryCount: Int?
    let rank: Int
    let relativeWeight: Double
    let accentKey: HomeHotAccentKey
    let artworkPath: String?
    let navigationTarget: HomeHotModuleNavigationTarget
}

struct HomeHotAlbumPresentationItem: Identifiable, Equatable {
    let id: Int64
    let title: String
    let subtitle: String
    let count: Int
    let rank: Int
    let relativeWeight: Double
    let accentKey: HomeHotAccentKey
    let artworkPath: String?
    let navigationTarget: HomeHotModuleNavigationTarget
}

struct HomeHotTrackPresentationItem: Identifiable, Equatable {
    let id: String
    let title: String
    let subtitle: String
    let tertiaryText: String?
    let count: Int
    let rank: Int
    let relativeWeight: Double
    let accentKey: HomeHotAccentKey
    let artworkPath: String?
    let sourceTrack: TopTrack
    let navigationTarget: HomeHotModuleNavigationTarget
}

struct HomeHotSourcePresentationItem: Identifiable, Equatable {
    let id: String
    let title: String
    let count: Int
    let share: Double
    let rank: Int
    let relativeWeight: Double
    let accentKey: HomeHotAccentKey
    let symbolName: String
}

struct HomeHotModulePresentation: Equatable {
    static let empty = HomeHotModulePresentation(
        topGenres: [],
        topArtists: [],
        topAlbums: [],
        topTracks: [],
        playSourceCounts: [:],
        stats: nil
    )

    let genres: [HomeHotGenrePresentationItem]
    let artists: [HomeHotArtistPresentationItem]
    let albums: [HomeHotAlbumPresentationItem]
    let tracks: [HomeHotTrackPresentationItem]
    let sources: [HomeHotSourcePresentationItem]
    let totalArtistsCount: Int64?
    let totalAlbumsCount: Int64?
    let totalPlaysCount: Int64?
    let totalTracksCount: Int64?
    let primaryAccentKey: HomeHotAccentKey?
    let combinedSummaryText: String
    let tasteSummaryText: String
    let sourceSummaryText: String
    let profileFootnoteText: String

    init(
        topGenres: [TopGenre],
        topArtists: [TopArtist],
        topAlbums: [TopAlbum],
        topTracks: [TopTrack],
        playSourceCounts: [String: Int64],
        stats: DashboardStats?
    ) {
        let genres = Self.makeGenres(topGenres)
        let artists = Self.makeArtists(topArtists)
        let albums = Self.makeAlbums(topAlbums)
        let tracks = Self.makeTracks(topTracks, albums: topAlbums)
        let sources = Self.makeSources(playSourceCounts)
        let primaryAccentKey = genres.first?.accentKey
            ?? artists.first?.accentKey
            ?? albums.first?.accentKey
            ?? sources.first?.accentKey
        let tasteSummaryText = Self.makeTasteSummary(genres: genres)
        let sourceSummaryText = Self.makeSourceSummary(sources: sources)
        let combinedSummaryText = Self.makeCombinedSummary(genres: genres, sources: sources)
        let profileFootnoteText = Self.makeProfileFootnote(genres: genres, sources: sources)

        self.genres = genres
        self.artists = artists
        self.albums = albums
        self.tracks = tracks
        self.sources = sources
        self.totalArtistsCount = stats?.totalArtists
        self.totalAlbumsCount = stats?.totalAlbums
        self.totalPlaysCount = stats?.totalPlays
        self.totalTracksCount = stats?.totalTracks
        self.primaryAccentKey = primaryAccentKey
        self.tasteSummaryText = tasteSummaryText
        self.sourceSummaryText = sourceSummaryText
        self.combinedSummaryText = combinedSummaryText
        self.profileFootnoteText = profileFootnoteText
    }

    private static func makeGenres(_ source: [TopGenre]) -> [HomeHotGenrePresentationItem] {
        let maxCount = source.map { Int($0.trackGenreCount) }.max() ?? 0
        return source.enumerated().map { index, genre in
            let rank = genre.rank ?? (index + 1)
            return HomeHotGenrePresentationItem(
                id: genre.id,
                title: genre.displayName,
                count: Int(genre.trackGenreCount),
                secondaryCount: Int(genre.genreCount),
                rank: rank,
                relativeWeight: Self.relativeWeight(for: Int(genre.trackGenreCount), maxCount: maxCount),
                accentKey: .accent(forRank: rank),
                navigationTarget: .genre(genre.trackGenreName)
            )
        }
    }

    private static func makeArtists(_ source: [TopArtist]) -> [HomeHotArtistPresentationItem] {
        let maxCount = source.map { $0.playCount ?? $0.trackCount ?? 0 }.max() ?? 0
        return source.enumerated().map { index, artist in
            let count = artist.playCount ?? artist.trackCount ?? 0
            let rank = artist.rank ?? (index + 1)
            return HomeHotArtistPresentationItem(
                id: artist.id,
                title: artist.artist,
                count: count,
                secondaryCount: artist.trackCount,
                rank: rank,
                relativeWeight: Self.relativeWeight(for: count, maxCount: maxCount),
                accentKey: .accent(forRank: rank),
                artworkPath: Self.firstNonEmpty(artist.avatarURL, artist.avatarObjectKey),
                navigationTarget: .artist(artist.artist)
            )
        }
    }

    private static func makeAlbums(_ source: [TopAlbum]) -> [HomeHotAlbumPresentationItem] {
        let maxCount = source.map(\.playCount).max() ?? 0
        return source.enumerated().map { index, album in
            let rank = index + 1
            return HomeHotAlbumPresentationItem(
                id: album.id,
                title: album.album,
                subtitle: album.artist,
                count: album.playCount,
                rank: rank,
                relativeWeight: Self.relativeWeight(for: album.playCount, maxCount: maxCount),
                accentKey: .accent(forRank: rank),
                artworkPath: album.coverArtURL,
                navigationTarget: .album(album.albumID),
            )
        }
    }

    private static func makeTracks(_ source: [TopTrack], albums: [TopAlbum]) -> [HomeHotTrackPresentationItem] {
        let maxCount = source.map(\.playCount).max() ?? 0
        let artworkByAlbumKey = Dictionary(
            uniqueKeysWithValues: albums.map { album in
                (albumLookupKey(artist: album.artist, album: album.album), album.coverArtURL)
            }
        )

        return source.enumerated().map { index, track in
            let rank = track.rank
            return HomeHotTrackPresentationItem(
                id: track.id,
                title: track.track,
                subtitle: track.artist,
                tertiaryText: track.album,
                count: track.playCount,
                rank: rank,
                relativeWeight: Self.relativeWeight(for: track.playCount, maxCount: maxCount),
                accentKey: .accent(forRank: rank),
                artworkPath: track.coverArtURL ?? artworkByAlbumKey[albumLookupKey(artist: track.artist, album: track.album)] ?? nil,
                sourceTrack: track,
                navigationTarget: track.trackID > 0 ? .track(track.trackID) : .none
            )
        }
    }

    private static func makeSources(_ sourceCounts: [String: Int64]) -> [HomeHotSourcePresentationItem] {
        guard sourceCounts.isEmpty == false else { return [] }

        var aggregated: [String: Int64] = [:]
        for (rawSource, count) in sourceCounts {
            guard count > 0 else { continue }
            let normalizedTitle = normalizedSourceTitle(for: rawSource)
            aggregated[normalizedTitle, default: 0] += count
        }

        let sorted = aggregated
            .map { (title: $0.key, count: $0.value) }
            .sorted { lhs, rhs in
                if lhs.count == rhs.count {
                    return lhs.title < rhs.title
                }
                return lhs.count > rhs.count
            }

        let totalCount = sorted.reduce(Int64(0)) { $0 + $1.count }
        let maxCount = sorted.first?.count ?? 0

        return sorted.enumerated().map { index, item in
            let rank = index + 1
            return HomeHotSourcePresentationItem(
                id: item.title,
                title: item.title,
                count: Int(item.count),
                share: totalCount > 0 ? Double(item.count) / Double(totalCount) : 0,
                rank: rank,
                relativeWeight: Self.relativeWeight(for: Int(item.count), maxCount: Int(maxCount)),
                accentKey: .accent(forRank: rank),
                symbolName: sourceSymbolName(for: item.title)
            )
        }
    }

    private static func relativeWeight(for count: Int, maxCount: Int) -> Double {
        guard count > 0, maxCount > 0 else { return 0.18 }
        let normalized = Double(count) / Double(maxCount)
        return min(max(normalized, 0.18), 1)
    }

    private static func makeTasteSummary(genres: [HomeHotGenrePresentationItem]) -> String {
        guard let topGenre = genres.first else { return "口味画像还在形成中" }
        return "当前更偏\(topGenre.title)"
    }

    private static func makeSourceSummary(sources: [HomeHotSourcePresentationItem]) -> String {
        guard let topSource = sources.first else { return "播放渠道还在汇总中" }
        return "播放主要来自\(topSource.title)"
    }

    private static func makeCombinedSummary(
        genres: [HomeHotGenrePresentationItem],
        sources: [HomeHotSourcePresentationItem]
    ) -> String {
        switch (genres.first, sources.first) {
        case let (.some(topGenre), .some(topSource)):
            return "当前更偏\(topGenre.title)，播放主要来自\(topSource.title)。"
        case let (.some(topGenre), nil):
            return "当前更偏\(topGenre.title)，播放渠道会随着更多记录逐渐清晰。"
        case let (nil, .some(topSource)):
            return "播放主要来自\(topSource.title)，口味画像会随着更多记录逐渐清晰。"
        case (nil, nil):
            return "播放记录再积累一点，这里会把口味与渠道一起总结出来。"
        }
    }

    private static func makeProfileFootnote(
        genres: [HomeHotGenrePresentationItem],
        sources: [HomeHotSourcePresentationItem]
    ) -> String {
        switch (genres.isEmpty, sources.isEmpty) {
        case (false, false):
            return "流派画像与当前资料库累计播放来源分布会一起更新。"
        case (false, true):
            return "先展示口味画像，播放来源分布会在更多记录进入后补齐。"
        case (true, false):
            return "先展示播放来源分布，口味画像会在更多记录进入后补齐。"
        case (true, true):
            return "等待更多播放记录同步进来。"
        }
    }

    private static func albumLookupKey(artist: String, album: String) -> String {
        "\(artist.trimmingCharacters(in: .whitespacesAndNewlines).lowercased())::\(album.trimmingCharacters(in: .whitespacesAndNewlines).lowercased())"
    }

    private static func normalizedSourceTitle(for rawValue: String) -> String {
        let trimmed = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
        let lowered = trimmed.lowercased()

        if lowered.contains("apple") {
            return "Apple Music"
        }
        if lowered.contains("last.fm") || lowered.contains("lastfm") {
            return "Last.fm"
        }
        if lowered.contains("roon") {
            return "Roon"
        }
        if lowered.contains("audirvana") {
            return "Audirvana"
        }
        if lowered.contains("foobar") {
            return "Foobar2000"
        }
        if lowered.contains("netease") || lowered.contains("163music") || lowered == "163" {
            return "NetEase Music"
        }
        if lowered.contains("spotify") {
            return "Spotify"
        }
        return trimmed.isEmpty ? "其他来源" : trimmed
    }

    private static func sourceSymbolName(for title: String) -> String {
        switch title {
        case "Apple Music":
            return "music.note"
        case "Last.fm":
            return "waveform.path.ecg"
        case "Roon":
            return "dot.radiowaves.left.and.right"
        case "Audirvana":
            return "hifispeaker.fill"
        case "Foobar2000":
            return "waveform"
        case "NetEase Music":
            return "music.note.list"
        case "Spotify":
            return "headphones"
        default:
            return "square.stack.3d.up.fill"
        }
    }

    private static func firstNonEmpty(_ values: String?...) -> String? {
        for value in values {
            if let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines), trimmed.isEmpty == false {
                return trimmed
            }
        }
        return nil
    }
}

private extension TopGenre {
    var displayName: String {
        let trimmedZh = genreNameZh.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmedZh.isEmpty {
            return trimmedZh
        }
        let trimmedOriginal = trackGenreName.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmedOriginal.isEmpty ? "未知流派" : trimmedOriginal
    }
}

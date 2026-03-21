import Foundation

@MainActor
final class TrackDetailViewModel: ObservableObject {
    @Published var lyrics: TrackLyricsResponse?
    @Published var lyricLines: [LyricLine] = []
    @Published var insights: [Insight] = []
    @Published var isLoading: Bool = false
    @Published var errorMessage: String?

    func load(
        using server: ServerConfig,
        artist: String,
        album: String?,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil
    ) async {
        isLoading = true
        errorMessage = nil
        let client = APIClient(baseURL: server.baseURL)
        do {
            lyrics = try await client.getJSON(
                path: APIPath.trackLyrics,
                queryItems: [
                    URLQueryItem(name: "artist", value: artist),
                    URLQueryItem(name: "album", value: album ?? ""),
                    URLQueryItem(name: "track", value: track),
                    URLQueryItem(name: "trackNumber", value: trackNumber.map(String.init)),
                    URLQueryItem(name: "discNumber", value: discNumber.map(String.init))
                ]
            )
            lyricLines = LRCParser.parseLyrics(lyrics?.lyrics ?? "", hasLRC: lyrics?.hasLRC ?? false)

            let insightResponse: TrackInsightResponse = try await client.getJSON(
                path: APIPath.trackInsight,
                queryItems: [
                    URLQueryItem(name: "artist", value: artist),
                    URLQueryItem(name: "album", value: album ?? ""),
                    URLQueryItem(name: "track", value: track),
                    URLQueryItem(name: "trackNumber", value: trackNumber.map(String.init)),
                    URLQueryItem(name: "discNumber", value: discNumber.map(String.init))
                ]
            )
            insights = insightResponse.insights
            isLoading = false
        } catch {
            errorMessage = "曲目详情加载失败"
            isLoading = false
        }
    }

    func currentLineID(forPreviewTime previewTime: TimeInterval) -> UUID? {
        guard lyricLines.contains(where: { $0.time != nil }) else { return nil }
        var current: LyricLine?
        for line in lyricLines {
            guard let time = line.time else { continue }
            if time <= previewTime {
                current = line
            } else {
                break
            }
        }
        return current?.id
    }
}

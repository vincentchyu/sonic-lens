import Foundation

@MainActor
final class PlayerViewModel: ObservableObject {
    @Published var lyrics: TrackLyricsResponse?
    @Published var insights: [Insight] = []
    @Published var lyricLines: [LyricLine] = []
    @Published var currentLineID: UUID?
    @Published var currentTime: TimeInterval = 0

    private var timer: Timer?
    private var progressAnchorDate: Date?
    private var progressAnchorTime: TimeInterval = 0

    func load(
        using server: ServerConfig,
        artist: String,
        album: String?,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil
    ) async {
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
            applyPlaybackState(time: currentTime, forceTimeUpdate: true)

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
        } catch {
            // TODO: expose error state
        }
    }

    func startProgress(position: Int?, positionMs: Int?) {
        stopProgress()
        let startTime = resolvedTime(position: position, positionMs: positionMs)
        progressAnchorTime = startTime
        progressAnchorDate = Date()
        applyPlaybackState(time: startTime, forceTimeUpdate: true)
        timer = Timer.scheduledTimer(withTimeInterval: 0.35, repeats: true) { [weak self] _ in
            Task { @MainActor in
                guard let self else { return }
                self.refreshCurrentTime()
            }
        }
        timer?.tolerance = 0.12
    }

    func stopProgress() {
        timer?.invalidate()
        timer = nil
        progressAnchorDate = nil
    }

    func syncProgress(position: Int?, positionMs: Int?) {
        let incoming = resolvedTime(position: position, positionMs: positionMs)
        if abs(incoming - currentTime) > 0.35 {
            progressAnchorTime = incoming
            progressAnchorDate = Date()
            applyPlaybackState(time: incoming, forceTimeUpdate: true)
        }
    }

    private func applyPlaybackState(time: TimeInterval, forceTimeUpdate: Bool) {
        let nextLineID = currentLineID(for: time)
        let secondChanged = Int(time.rounded(.down)) != Int(currentTime.rounded(.down))
        if forceTimeUpdate || secondChanged || nextLineID != currentLineID {
            currentTime = time
        }
        if nextLineID != currentLineID {
            currentLineID = nextLineID
        }
    }

    private func refreshCurrentTime() {
        guard let progressAnchorDate else { return }
        let nextTime = progressAnchorTime + Date().timeIntervalSince(progressAnchorDate)
        applyPlaybackState(time: nextTime, forceTimeUpdate: false)
    }

    private func resolvedTime(position: Int?, positionMs: Int?) -> TimeInterval {
        if let positionMs {
            return TimeInterval(positionMs) / 1000
        }
        return TimeInterval(position ?? 0)
    }

    private func currentLineID(for time: TimeInterval) -> UUID? {
        guard lyricLines.contains(where: { $0.time != nil }) else {
            return nil
        }
        var current: LyricLine?
        for line in lyricLines {
            guard let lineTime = line.time else { continue }
            if lineTime <= time {
                current = line
            } else {
                break
            }
        }
        return current?.id
    }
}

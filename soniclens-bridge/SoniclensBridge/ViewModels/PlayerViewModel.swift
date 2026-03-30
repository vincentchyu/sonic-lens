import Foundation

@MainActor
final class PlayerViewModel: ObservableObject {
    @Published var lyrics: TrackLyricsResponse?
    @Published var insights: [Insight] = []
    @Published var lyricLines: [LyricLine] = []
    @Published var currentLineID: UUID?
    @Published var currentTime: TimeInterval = 0
    @Published var playbackState: PlaybackActivityState = .inactive

    private static let pauseStaleTimeout: TimeInterval = 5
    private static let inactiveTimeout: TimeInterval = 10
    private var timer: Timer?
    private var progressAnchorDate: Date?
    private var progressAnchorTime: TimeInterval = 0
    private var lastSyncDate: Date?
    private var loadSequence: UInt64 = 0

    func load(
        using server: ServerConfig,
        artist: String,
        album: String?,
        track: String,
        trackNumber: Int? = nil,
        discNumber: Int? = nil
    ) async {
        loadSequence &+= 1
        let requestToken = loadSequence
        let client = APIClient(baseURL: server.baseURL)
        do {
            async let lyricsResponse: TrackLyricsResponse = client.getJSON(
                path: APIPath.trackLyrics,
                queryItems: [
                    URLQueryItem(name: "artist", value: artist),
                    URLQueryItem(name: "album", value: album ?? ""),
                    URLQueryItem(name: "track", value: track),
                    URLQueryItem(name: "trackNumber", value: trackNumber.map(String.init)),
                    URLQueryItem(name: "discNumber", value: discNumber.map(String.init))
                ]
            )
            async let insightResponse: TrackInsightResponse = client.getJSON(
                path: APIPath.trackInsight,
                queryItems: [
                    URLQueryItem(name: "artist", value: artist),
                    URLQueryItem(name: "album", value: album ?? ""),
                    URLQueryItem(name: "track", value: track),
                    URLQueryItem(name: "trackNumber", value: trackNumber.map(String.init)),
                    URLQueryItem(name: "discNumber", value: discNumber.map(String.init))
                ]
            )
            let lyrics = try await lyricsResponse
            let insight = try await insightResponse
            guard !Task.isCancelled, requestToken == loadSequence else { return }

            self.lyrics = lyrics
            lyricLines = LRCParser.parseLyrics(lyrics.lyrics, hasLRC: lyrics.hasLRC)
            applyPlaybackState(time: currentTime, forceTimeUpdate: true)
            insights = insight.insights
        } catch {
            // TODO: expose error state
        }
    }

    func startProgress(position: Int?, positionMs: Int?) {
        stopProgress()
        let startTime = resolvedTime(position: position, positionMs: positionMs)
        progressAnchorTime = startTime
        progressAnchorDate = Date()
        lastSyncDate = Date()
        playbackState = .active
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
        lastSyncDate = nil
        playbackState = .inactive
        currentTime = 0
        currentLineID = nil
    }

    func syncProgress(position: Int?, positionMs: Int?) {
        let incoming = resolvedTime(position: position, positionMs: positionMs)
        lastSyncDate = Date()
        if timer == nil || playbackState.isInactive {
            startProgress(position: position, positionMs: positionMs)
            return
        }
        playbackState = .active
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
        guard let progressAnchorDate, let lastSyncDate else { return }
        let silence = Date().timeIntervalSince(lastSyncDate)
        if silence >= Self.inactiveTimeout {
            playbackState = .inactive
            stopProgress()
            return
        }
        if silence >= Self.pauseStaleTimeout {
            playbackState = .pausedStale
            return
        }
        playbackState = .active
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

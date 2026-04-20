import Foundation

@MainActor
final class PlayerViewModel: ObservableObject {
    @Published var lyrics: TrackLyricsResponse?
    @Published var insights: [Insight] = []
    @Published var lyricLines: [LyricLine] = []
    @Published var currentLineID: UUID?
    @Published var currentLineIndex: Int?
    @Published var currentTime: TimeInterval = 0
    @Published var playbackState: PlaybackActivityState = .inactive
    @Published var selectedInsightIndex: Int = 0
    @Published var insightViewMode: InsightViewMode = .current

    private var timer: Timer?
    private var progressAnchorDate: Date?
    private var progressAnchorTime: TimeInterval = 0
    private var lastSyncDate: Date?
    private var loadSequence: UInt64 = 0
    private var timedLyricMoments: [TimedLyricMoment] = []

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
        let request = NowPlayingPayloadRequest(
            serverBaseURL: server.baseURL.absoluteString,
            artist: artist,
            album: album,
            track: track,
            trackNumber: trackNumber,
            discNumber: discNumber
        )

        guard let snapshot = await NowPlayingPayloadStore.shared.snapshot(using: server, request: request) else {
            return
        }
        guard !Task.isCancelled, requestToken == loadSequence else { return }

        self.lyrics = snapshot.lyricsResponse
        lyricLines = snapshot.lyricLines
        timedLyricMoments = Self.timedLyricMoments(from: snapshot.lyricLines)
        applyPlaybackState(time: currentTime, forceTimeUpdate: true)
        insights = snapshot.insightResponse.insights
        selectedInsightIndex = Self.recommendedInsightIndex(
            in: insights,
            recommendedInsightID: snapshot.insightResponse.recommendedInsightID
        )
        insightViewMode = .current
    }

    private static func recommendedInsightIndex(in insights: [Insight], recommendedInsightID: Int64?) -> Int {
        guard let recommendedInsightID,
              let index = insights.firstIndex(where: { $0.id == recommendedInsightID }) else {
            return 0
        }
        return index
    }

    func startProgress(position: Int?, positionMs: Int?, receivedAt: Date = Date()) {
        stopProgress()
        let startTime = resolvedTime(position: position, positionMs: positionMs)
        progressAnchorTime = startTime
        progressAnchorDate = Date()
        lastSyncDate = receivedAt
        playbackState = activityState(for: receivedAt)
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
        currentLineIndex = nil
    }

    func syncProgress(position: Int?, positionMs: Int?, receivedAt: Date = Date()) {
        let incoming = resolvedTime(position: position, positionMs: positionMs)
        lastSyncDate = receivedAt
        if timer == nil || playbackState.isInactive {
            startProgress(position: position, positionMs: positionMs, receivedAt: receivedAt)
            return
        }
        playbackState = activityState(for: receivedAt)
        if abs(incoming - currentTime) > 0.35 {
            progressAnchorTime = incoming
            progressAnchorDate = Date()
            applyPlaybackState(time: incoming, forceTimeUpdate: true)
        }
    }

    private func applyPlaybackState(time: TimeInterval, forceTimeUpdate: Bool) {
        let nextLineIndex = currentLineIndex(for: time)
        let nextLineID = nextLineIndex.flatMap { lyricLines.indices.contains($0) ? lyricLines[$0].id : nil }
        let secondChanged = Int(time.rounded(.down)) != Int(currentTime.rounded(.down))
        if forceTimeUpdate || secondChanged || nextLineID != currentLineID {
            currentTime = time
        }
        if nextLineIndex != currentLineIndex {
            currentLineIndex = nextLineIndex
        }
        if nextLineID != currentLineID {
            currentLineID = nextLineID
        }
    }

    private func refreshCurrentTime() {
        guard let progressAnchorDate, let lastSyncDate else { return }
        let silence = Date().timeIntervalSince(lastSyncDate)
        if silence >= NowPlaying.inactiveTimeout {
            playbackState = .inactive
            stopProgress()
            return
        }
        if silence >= NowPlaying.pauseStaleTimeout {
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

    private func activityState(for receivedAt: Date) -> PlaybackActivityState {
        let silence = Date().timeIntervalSince(receivedAt)
        if silence >= NowPlaying.inactiveTimeout {
            return .inactive
        }
        if silence >= NowPlaying.pauseStaleTimeout {
            return .pausedStale
        }
        return .active
    }

    private func currentLineIndex(for time: TimeInterval) -> Int? {
        guard !timedLyricMoments.isEmpty else { return nil }

        var low = 0
        var high = timedLyricMoments.count - 1
        var candidate: Int?

        while low <= high {
            let mid = (low + high) / 2
            let moment = timedLyricMoments[mid]
            if moment.time <= time {
                candidate = moment.index
                low = mid + 1
            } else {
                high = mid - 1
            }
        }

        return candidate
    }

    private static func timedLyricMoments(from lines: [LyricLine]) -> [TimedLyricMoment] {
        lines.enumerated().compactMap { index, line in
            guard let time = line.time else { return nil }
            return TimedLyricMoment(time: time, index: index)
        }
    }
}

private struct TimedLyricMoment {
    let time: TimeInterval
    let index: Int
    }

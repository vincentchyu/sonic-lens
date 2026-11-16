import Foundation
import Observation
import OSLog
#if os(iOS)
import MediaPlayer
import UIKit
#endif

@MainActor
@Observable
final class AppleMusicScrobbleCoordinator {
    private(set) var authorizationState: MediaLibraryAuthorizationState = .notDetermined
    private(set) var pendingCount: Int = 0
    private(set) var baselineTrackCount: Int = 0
    private(set) var pendingItems: [PendingScrobbleItem] = []
    private(set) var syncHistory: [MobileScrobbleHistoryItem] = []
    private(set) var isBaselineInitialized: Bool = false
    private(set) var isScanning: Bool = false
    private(set) var isSyncing: Bool = false
    private(set) var lastSyncedCount: Int = 0
    private(set) var lastSyncDate: Date?
    private(set) var statusMessage: String?

    var onPendingItemsEnqueued: (@MainActor () -> Void)?

    private let store: MobileScrobbleStore
    private let logger = Logger(subsystem: "com.vincentchyu.soniclens-bridge", category: "AppleMusicScrobble")

    #if os(iOS)
    private var hasStartedObserving = false
    private var notificationObservers: [NSObjectProtocol] = []
    private var playbackPollTask: Task<Void, Never>?
    private var activeTrackKey: String?
    private var activeTrackStartedAt: TimeInterval = 0
    private var activeTrackLastPosition: Double = 0
    private var activeTrackLastCapturedAt: Int64 = 0
    private var activeTrackThresholdReached: Bool = false
    private var hasLoadedCheckpoint: Bool = false
    private var isEvaluatingNowPlaying: Bool = false
    private var needsReevaluateNowPlaying: Bool = false
    #endif

    init(store: MobileScrobbleStore = MobileScrobbleStore()) {
        self.store = store
        #if !os(iOS)
        self.authorizationState = .unsupported
        #endif
    }

    func startIfNeeded() async {
        await refreshLocalState()
        #if os(iOS)
        refreshAuthorizationState()
        startSystemPlayerMonitoringIfNeeded()
        if authorizationState == .authorized {
            await scanMediaLibraryAndEnqueue()
        }
        #endif
    }

    func requestAuthorizationAndScan(using server: ServerConfig?) async {
        #if os(iOS)
        let status = await withCheckedContinuation { continuation in
            MPMediaLibrary.requestAuthorization { authStatus in
                continuation.resume(returning: authStatus)
            }
        }
        authorizationState = Self.mapAuthorizationStatus(status)
        if authorizationState == .authorized {
            statusMessage = "已授权媒体库，正在建立听歌基线..."
            await scanMediaLibraryAndEnqueue()
            await syncPendingIfNeeded(using: server)
        } else if authorizationState == .denied || authorizationState == .restricted {
            statusMessage = "请在系统设置中允许访问“媒体与 Apple Music”"
        }
        #else
        _ = server
        #endif
    }

    func refreshAndSync(using server: ServerConfig?) async {
        await startIfNeeded()
        #if os(iOS)
        await evaluateSystemPlayerNowPlaying()
        #endif
        await syncPendingIfNeeded(using: server)
    }

    func scanMediaLibraryAndEnqueue() async {
        #if os(iOS)
        refreshAuthorizationState()
        guard authorizationState == .authorized else { return }
        guard !isScanning else { return }
        isScanning = true
        defer { isScanning = false }

        let wasInitialized = (try? await store.isBaselineInitialized()) ?? false
        let snapshots = await Task.detached(priority: .utility) {
            Self.collectMediaLibrarySnapshots()
        }.value

        do {
            let added = try await store.reconcileMediaLibrarySnapshots(snapshots)
            await refreshLocalState()
            if !wasInitialized {
                statusMessage = "已静默建立 \(snapshots.count) 首曲目基线，后续户外听歌将自动补录"
            } else if added > 0 {
                statusMessage = "捕获到 \(added) 条户外 Apple Music 听歌记录"
            } else if pendingCount == 0 {
                statusMessage = syncHistory.isEmpty ? "媒体库监控中 · 暂无新增播放" : "监控中 · 累计已同步 \(syncHistory.count) 条"
            }
            logger.info("媒体库扫描完成 total=\(snapshots.count, privacy: .public) newly_enqueued=\(added, privacy: .public) pending=\(self.pendingCount, privacy: .public)")
        } catch {
            logger.error("媒体库差量对账失败 error=\(error.localizedDescription, privacy: .public)")
        }
        #endif
    }

    func syncPendingIfNeeded(using server: ServerConfig?) async {
        guard let server else { return }
        guard !isSyncing else { return }

        await refreshLocalState()
        guard pendingCount > 0 else { return }

        isSyncing = true
        defer { isSyncing = false }

        var totalAccepted = 0
        var totalDuplicate = 0
        var remainingBatches = 10

        while remainingBatches > 0 {
            remainingBatches -= 1
            let batch: [PendingScrobbleItem]
            do {
                batch = try await store.fetchPendingItems(limit: 100)
            } catch {
                logger.error("读取待补录队列失败 error=\(error.localizedDescription, privacy: .public)")
                break
            }
            guard !batch.isEmpty else { break }

            let request = MobileScrobbleBatchRequest(
                deviceID: Self.deviceIdentifier,
                deviceName: Self.deviceDisplayName,
                records: batch.map { $0.toUploadPayload() }
            )

            do {
                let client = APIClient(baseURL: server.baseURL)
                let response: MobileScrobbleBatchResponse = try await client.postJSON(
                    path: APIPath.mobileScrobbleBatch,
                    body: request,
                    timeout: 15
                )

                if !response.acceptedKeys.isEmpty {
                    try await store.recordSyncedBatch(
                        batch: batch,
                        acceptedKeys: response.acceptedKeys,
                        itemResults: response.results ?? []
                    )
                    totalAccepted += response.acceptedCount
                    totalDuplicate += response.duplicateCount
                }
                if response.acceptedKeys.isEmpty {
                    logger.warning("移动端补录批次未返回任何已接收键，暂停本轮同步 failed=\(response.failedCount, privacy: .public)")
                    break
                }
            } catch {
                logger.error("移动端补录同步请求失败 error=\(error.localizedDescription, privacy: .public)")
                statusMessage = "同步暂时失败，将在下次连接时自动重试"
                break
            }
        }

        await refreshLocalState()
        if totalAccepted > 0 || totalDuplicate > 0 {
            lastSyncedCount = totalAccepted
            lastSyncDate = Date()
            if totalAccepted > 0 {
                statusMessage = pendingCount == 0 ? "已同步 \(totalAccepted) 首 · 待同步队列已清空" : "已同步 \(totalAccepted) 首 · 剩余 \(pendingCount) 条待同步"
                NotificationCenter.default.post(name: .recentPlaysDidUpdate, object: RecentPlaysUpdate())
            } else {
                statusMessage = "已完成去重校验（\(totalDuplicate) 条已存在）"
            }
            logger.info("移动端听歌补录同步完成 accepted=\(totalAccepted, privacy: .public) duplicate=\(totalDuplicate, privacy: .public) remaining=\(self.pendingCount, privacy: .public)")
        }
    }

    func reloadDetailSnapshot() async {
        await refreshLocalState()
    }

    private func refreshLocalState() async {
        isBaselineInitialized = (try? await store.isBaselineInitialized()) ?? false
        pendingCount = (try? await store.pendingCount()) ?? 0
        baselineTrackCount = (try? await store.baselineTrackCount()) ?? 0
        let fetchedPending = (try? await store.fetchPendingItems(limit: 200)) ?? []
        pendingItems = fetchedPending.sorted { $0.playedAtTimestamp > $1.playedAtTimestamp }
        syncHistory = (try? await store.fetchSyncHistory(limit: 100)) ?? []
        if lastSyncDate == nil, let ts = try? await store.lastSyncTimestamp(), ts > 0 {
            lastSyncDate = Date(timeIntervalSince1970: TimeInterval(ts))
        }
    }

    private static var deviceIdentifier: String {
        #if os(iOS)
        return UIDevice.current.identifierForVendor?.uuidString ?? "ios-device"
        #else
        return "mac-device"
        #endif
    }

    private static var deviceDisplayName: String {
        #if os(iOS)
        return UIDevice.current.name
        #else
        return Host.current().localizedName ?? "Mac"
        #endif
    }

    #if os(iOS)
    private func refreshAuthorizationState() {
        authorizationState = Self.mapAuthorizationStatus(MPMediaLibrary.authorizationStatus())
    }

    private static func mapAuthorizationStatus(_ status: MPMediaLibraryAuthorizationStatus) -> MediaLibraryAuthorizationState {
        switch status {
        case .notDetermined:
            return .notDetermined
        case .authorized:
            return .authorized
        case .denied:
            return .denied
        case .restricted:
            return .restricted
        @unknown default:
            return .unsupported
        }
    }

    private func startSystemPlayerMonitoringIfNeeded() {
        guard !hasStartedObserving else { return }
        hasStartedObserving = true

        let player = MPMusicPlayerController.systemMusicPlayer
        player.beginGeneratingPlaybackNotifications()

        let center = NotificationCenter.default
        let itemObserver = center.addObserver(
            forName: .MPMusicPlayerControllerNowPlayingItemDidChange,
            object: player,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor [weak self] in
                await self?.evaluateSystemPlayerNowPlaying()
            }
        }
        let stateObserver = center.addObserver(
            forName: .MPMusicPlayerControllerPlaybackStateDidChange,
            object: player,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor [weak self] in
                await self?.evaluateSystemPlayerNowPlaying()
            }
        }
        notificationObservers = [itemObserver, stateObserver]
    }

    private func loadRealtimeCheckpointIfNeeded() async {
        guard !hasLoadedCheckpoint else { return }
        let checkpoint = try? await store.loadRealtimeCheckpoint()
        hasLoadedCheckpoint = true
        guard let checkpoint else { return }
        activeTrackKey = checkpoint.trackKey
        activeTrackStartedAt = TimeInterval(checkpoint.startedAt)
        activeTrackLastPosition = checkpoint.lastPosition
        activeTrackLastCapturedAt = checkpoint.capturedAt
        activeTrackThresholdReached = checkpoint.thresholdReached
    }

    private func evaluateSystemPlayerNowPlaying() async {
        if isEvaluatingNowPlaying {
            needsReevaluateNowPlaying = true
            return
        }
        isEvaluatingNowPlaying = true
        defer { isEvaluatingNowPlaying = false }

        repeat {
            needsReevaluateNowPlaying = false
            await evaluateSystemPlayerNowPlayingOnce()
        } while needsReevaluateNowPlaying
    }

    private func evaluateSystemPlayerNowPlayingOnce() async {
        let player = MPMusicPlayerController.systemMusicPlayer
        guard player.playbackState == .playing,
              let item = player.nowPlayingItem,
              item.mediaType.contains(.music) else {
            stopPlaybackPolling()
            return
        }

        await loadRealtimeCheckpointIfNeeded()

        let title = (item.title ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let artist = (item.artist ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let album = (item.albumTitle ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        guard !title.isEmpty, !artist.isEmpty, !album.isEmpty else {
            return
        }

        let duration = Int64(item.playbackDuration.rounded())
        let trackKey = "\(item.persistentID)|\(artist)|\(album)|\(title)"
        let currentPosition = max(player.currentPlaybackTime, 0)
        let now = Date().timeIntervalSince1970

        if activeTrackKey != trackKey {
            activeTrackKey = trackKey
            activeTrackStartedAt = max(now - min(currentPosition, TimeInterval(max(duration, 0))), 1)
            activeTrackLastPosition = currentPosition
            activeTrackLastCapturedAt = 0
            activeTrackThresholdReached = false
            let checkpoint = RealtimeCaptureCheckpoint(
                trackKey: trackKey,
                startedAt: Int64(activeTrackStartedAt),
                lastPosition: currentPosition,
                capturedAt: 0,
                thresholdReached: false
            )
            try? await store.saveRealtimeCheckpoint(checkpoint)
        } else if activeTrackThresholdReached {
            let restartUpperBound = min(Double(max(duration, 0)) * 0.35, 30.0)
            if currentPosition < restartUpperBound &&
                (currentPosition + 15.0 < activeTrackLastPosition || Int64(now) - activeTrackLastCapturedAt >= 15) {
                activeTrackStartedAt = max(now - currentPosition, 1)
                activeTrackLastPosition = currentPosition
                activeTrackThresholdReached = false
                let checkpoint = RealtimeCaptureCheckpoint(
                    trackKey: trackKey,
                    startedAt: Int64(activeTrackStartedAt),
                    lastPosition: currentPosition,
                    capturedAt: activeTrackLastCapturedAt,
                    thresholdReached: false
                )
                try? await store.saveRealtimeCheckpoint(checkpoint)
            } else {
                activeTrackLastPosition = currentPosition
            }
        } else {
            activeTrackLastPosition = currentPosition
        }

        if duration >= 30 && !activeTrackThresholdReached {
            let threshold = min(Double(duration) * 0.5, 240.0)
            if currentPosition >= threshold {
                activeTrackThresholdReached = true
                activeTrackLastCapturedAt = Int64(now)
                let checkpoint = RealtimeCaptureCheckpoint(
                    trackKey: trackKey,
                    startedAt: Int64(activeTrackStartedAt),
                    lastPosition: currentPosition,
                    capturedAt: activeTrackLastCapturedAt,
                    thresholdReached: true
                )
                try? await store.saveRealtimeCheckpoint(checkpoint)
                await enqueueRealtimeMediaItem(item, startedAt: activeTrackStartedAt)
            }
        }

        if player.playbackState == .playing && duration >= 30 {
            ensurePlaybackPollingRunning()
        } else {
            stopPlaybackPolling()
        }
    }

    private func ensurePlaybackPollingRunning() {
        guard playbackPollTask == nil else { return }
        playbackPollTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 5_000_000_000)
                guard !Task.isCancelled else { break }
                guard let self else { break }
                await self.evaluateSystemPlayerNowPlaying()
            }
        }
    }

    private func stopPlaybackPolling() {
        playbackPollTask?.cancel()
        playbackPollTask = nil
    }

    private func enqueueRealtimeMediaItem(_ item: MPMediaItem, startedAt: TimeInterval) async {
        guard let pendingItem = Self.buildPendingItem(from: item, startedAtTimestamp: Int64(startedAt)) else {
            return
        }
        do {
            let added = try await store.enqueueRealtimeItem(pendingItem)
            await refreshLocalState()
            if added {
                statusMessage = "已记录正在播放：\(pendingItem.title)"
                logger.info("系统播放器实时捕获入队 title=\(pendingItem.title, privacy: .public) pending=\(self.pendingCount, privacy: .public)")
                onPendingItemsEnqueued?()
            }
        } catch {
            logger.error("系统播放器实时记录入队失败 error=\(error.localizedDescription, privacy: .public)")
        }
    }

    private nonisolated static func collectMediaLibrarySnapshots() -> [MediaLibraryTrackSnapshot] {
        guard let items = MPMediaQuery.songs().items else {
            return []
        }
        var snapshots: [MediaLibraryTrackSnapshot] = []
        snapshots.reserveCapacity(items.count)

        let dateFormatter = DateFormatter()
        dateFormatter.locale = Locale(identifier: "en_US_POSIX")
        dateFormatter.timeZone = TimeZone(secondsFromGMT: 0)
        dateFormatter.dateFormat = "yyyy-MM-dd"
        let calendar = Calendar(identifier: .gregorian)

        for item in items {
            guard item.mediaType.contains(.music) else { continue }
            let persistentID = item.persistentID != 0 ? String(item.persistentID) : ""
            guard !persistentID.isEmpty else { continue }

            let title = (item.title ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
            let artist = (item.artist ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
            let album = (item.albumTitle ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
            guard !title.isEmpty, !artist.isEmpty, !album.isEmpty else { continue }

            let releaseDateString: String
            let releaseYear: Int
            if let releaseDate = item.releaseDate {
                releaseDateString = dateFormatter.string(from: releaseDate)
                releaseYear = calendar.component(.year, from: releaseDate)
            } else {
                releaseDateString = ""
                releaseYear = 0
            }

            let lastPlayedAt: Int64
            if let lastPlayedDate = item.lastPlayedDate {
                lastPlayedAt = Int64(lastPlayedDate.timeIntervalSince1970)
            } else {
                lastPlayedAt = 0
            }

            snapshots.append(
                MediaLibraryTrackSnapshot(
                    persistentID: persistentID,
                    title: title,
                    artist: artist,
                    albumArtist: (item.albumArtist ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
                    album: album,
                    genre: (item.genre ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
                    composer: (item.composer ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
                    trackNumber: item.albumTrackNumber,
                    discNumber: item.discNumber,
                    duration: Int64(item.playbackDuration.rounded()),
                    releaseDate: releaseDateString,
                    releaseYear: releaseYear,
                    playCount: item.playCount,
                    lastPlayedAt: lastPlayedAt
                )
            )
        }
        return snapshots
    }

    private nonisolated static func buildPendingItem(from item: MPMediaItem, startedAtTimestamp: Int64) -> PendingScrobbleItem? {
        let title = (item.title ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let artist = (item.artist ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let album = (item.albumTitle ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        guard !title.isEmpty, !artist.isEmpty, !album.isEmpty else {
            return nil
        }

        let duration = Int64(item.playbackDuration.rounded())
        let persistentID = item.persistentID != 0 ? String(item.persistentID) : stableFNV1aHex("\(artist)|\(album)|\(title)")
        let key = "rt:\(persistentID):\(startedAtTimestamp)"

        let dateFormatter = DateFormatter()
        dateFormatter.locale = Locale(identifier: "en_US_POSIX")
        dateFormatter.timeZone = TimeZone(secondsFromGMT: 0)
        dateFormatter.dateFormat = "yyyy-MM-dd"
        let calendar = Calendar(identifier: .gregorian)

        let releaseDateString: String
        let releaseYear: Int
        if let releaseDate = item.releaseDate {
            releaseDateString = dateFormatter.string(from: releaseDate)
            releaseYear = calendar.component(.year, from: releaseDate)
        } else {
            releaseDateString = ""
            releaseYear = 0
        }

        let trackNumber = item.albumTrackNumber
        let confidence = (trackNumber > 0 && duration > 0) ? 2 : 1

        return PendingScrobbleItem(
            idempotencyKey: key,
            title: title,
            artist: artist,
            albumArtist: (item.albumArtist ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
            album: album,
            genre: (item.genre ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
            composer: (item.composer ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
            trackNumber: trackNumber,
            discNumber: item.discNumber > 0 ? item.discNumber : 1,
            duration: duration,
            releaseDate: releaseDateString,
            releaseYear: releaseYear,
            playedAtTimestamp: startedAtTimestamp,
            uniqueID: item.persistentID != 0 ? String(item.persistentID) : "",
            bundleID: "com.apple.Music",
            confidence: confidence
        )
    }

    private nonisolated static func stableFNV1aHex(_ input: String) -> String {
        var hash: UInt64 = 0xcbf29ce484222325
        for byte in input.utf8 {
            hash ^= UInt64(byte)
            hash &*= 0x100000001b3
        }
        return String(hash, radix: 16)
    }
    #endif
}

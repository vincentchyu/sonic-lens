import Foundation
import SQLite3

private let mobileScrobbleSQLiteTransient = unsafeBitCast(-1, to: sqlite3_destructor_type.self)

actor MobileScrobbleStore {
    private static let databaseFileName = "soniclens-mobile-scrobble.sqlite"
    private static let appSupportDirectoryName = "SonicLens"
    private static let dataDirectoryName = "data"
    private static let databaseDirectoryName = "db"
    private static let maxDeltaPerTrack = 20

    enum StoreError: Error {
        case openDatabase
        case prepareStatement(String)
        case executeStatement(String)
    }

    private var db: OpaquePointer?
    private let databaseURL: URL
    private let fileManager: FileManager

    init(fileManager: FileManager = .default, databaseURL: URL? = nil) {
        self.fileManager = fileManager
        if let databaseURL {
            self.databaseURL = databaseURL
        } else {
            let supportDirectory = try? fileManager.url(
                for: .applicationSupportDirectory,
                in: .userDomainMask,
                appropriateFor: nil,
                create: true
            )
            let baseDirectory = supportDirectory ?? fileManager.temporaryDirectory
            let databaseDirectory = baseDirectory
                .appendingPathComponent(Self.appSupportDirectoryName, isDirectory: true)
                .appendingPathComponent(Self.dataDirectoryName, isDirectory: true)
                .appendingPathComponent(Self.databaseDirectoryName, isDirectory: true)
            self.databaseURL = databaseDirectory.appendingPathComponent(Self.databaseFileName)
        }
    }

    deinit {
        if let db {
            sqlite3_close(db)
        }
    }

    func setup() throws {
        try openIfNeeded()
        try execute(
            """
            CREATE TABLE IF NOT EXISTS scrobble_meta (
                key TEXT PRIMARY KEY,
                value TEXT NOT NULL
            );
            """
        )
        try execute(
            """
            CREATE TABLE IF NOT EXISTS media_library_baseline (
                persistent_id TEXT PRIMARY KEY,
                play_count INTEGER NOT NULL,
                last_played_at INTEGER NOT NULL,
                updated_at INTEGER NOT NULL
            );
            """
        )
        try execute(
            """
            CREATE TABLE IF NOT EXISTS pending_scrobble_queue (
                idempotency_key TEXT PRIMARY KEY,
                payload_json TEXT NOT NULL,
                artist TEXT NOT NULL,
                track TEXT NOT NULL,
                played_at INTEGER NOT NULL,
                created_at INTEGER NOT NULL
            );
            """
        )
        try execute("CREATE INDEX IF NOT EXISTS idx_pending_scrobble_dedupe ON pending_scrobble_queue(artist, track, played_at);")
        try execute("CREATE INDEX IF NOT EXISTS idx_pending_scrobble_order ON pending_scrobble_queue(played_at ASC, created_at ASC);")
        try execute(
            """
            CREATE TABLE IF NOT EXISTS scrobble_sync_history (
                idempotency_key TEXT PRIMARY KEY,
                title TEXT NOT NULL,
                artist TEXT NOT NULL,
                album TEXT NOT NULL,
                played_at INTEGER NOT NULL,
                synced_at INTEGER NOT NULL,
                status TEXT NOT NULL,
                scrobbled INTEGER NOT NULL DEFAULT 0
            );
            """
        )
        try execute("CREATE INDEX IF NOT EXISTS idx_scrobble_sync_history_order ON scrobble_sync_history(synced_at DESC, played_at DESC);")
        try execute("CREATE INDEX IF NOT EXISTS idx_scrobble_sync_history_dedupe ON scrobble_sync_history(artist, title, played_at);")
    }

    func isBaselineInitialized() throws -> Bool {
        try setup()
        return try metaValue(for: "baseline_initialized") == "1"
    }

    func lastSyncTimestamp() throws -> Int64? {
        try setup()
        guard let raw = try metaValue(for: "last_sync_at"), let ts = Int64(raw), ts > 0 else {
            return nil
        }
        return ts
    }

    func loadRealtimeCheckpoint() throws -> RealtimeCaptureCheckpoint? {
        try setup()
        guard let jsonText = try metaValue(for: "realtime_capture_checkpoint"),
              let data = jsonText.data(using: .utf8) else {
            return nil
        }
        return try? JSONDecoders.defaultDecoder().decode(RealtimeCaptureCheckpoint.self, from: data)
    }

    func saveRealtimeCheckpoint(_ checkpoint: RealtimeCaptureCheckpoint) throws {
        try setup()
        let data = try JSONEncoder().encode(checkpoint)
        let jsonText = String(decoding: data, as: UTF8.self)
        try setMetaValue(jsonText, for: "realtime_capture_checkpoint")
    }

    func pendingCount() throws -> Int {
        try setup()
        guard let statement = try prepare("SELECT COUNT(*) FROM pending_scrobble_queue;") else {
            return 0
        }
        defer { sqlite3_finalize(statement) }
        if sqlite3_step(statement) == SQLITE_ROW {
            return Int(sqlite3_column_int64(statement, 0))
        }
        return 0
    }

    func baselineTrackCount() throws -> Int {
        try setup()
        guard let statement = try prepare("SELECT COUNT(*) FROM media_library_baseline;") else {
            return 0
        }
        defer { sqlite3_finalize(statement) }
        if sqlite3_step(statement) == SQLITE_ROW {
            return Int(sqlite3_column_int64(statement, 0))
        }
        return 0
    }

    func fetchSyncHistory(limit: Int = 100) throws -> [MobileScrobbleHistoryItem] {
        try setup()
        let safeLimit = max(1, min(limit, 200))
        let sql = """
        SELECT idempotency_key, title, artist, album, played_at, synced_at, status, scrobbled
        FROM scrobble_sync_history
        ORDER BY synced_at DESC, played_at DESC
        LIMIT ?;
        """
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }
        sqlite3_bind_int(statement, 1, Int32(safeLimit))

        var items: [MobileScrobbleHistoryItem] = []
        while sqlite3_step(statement) == SQLITE_ROW {
            items.append(
                MobileScrobbleHistoryItem(
                    idempotencyKey: string(at: 0, in: statement),
                    title: string(at: 1, in: statement),
                    artist: string(at: 2, in: statement),
                    album: string(at: 3, in: statement),
                    playedAtTimestamp: sqlite3_column_int64(statement, 4),
                    syncedAtTimestamp: sqlite3_column_int64(statement, 5),
                    status: string(at: 6, in: statement),
                    scrobbled: sqlite3_column_int64(statement, 7) != 0
                )
            )
        }
        return items
    }

    func fetchPendingItems(limit: Int = 100) throws -> [PendingScrobbleItem] {
        try setup()
        let safeLimit = max(1, min(limit, 200))
        let sql = """
        SELECT payload_json
        FROM pending_scrobble_queue
        ORDER BY played_at ASC, created_at ASC
        LIMIT ?;
        """
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }
        sqlite3_bind_int(statement, 1, Int32(safeLimit))

        let decoder = JSONDecoders.defaultDecoder()
        var items: [PendingScrobbleItem] = []
        while sqlite3_step(statement) == SQLITE_ROW {
            let jsonText = string(at: 0, in: statement)
            if let data = jsonText.data(using: .utf8),
               let decoded = try? decoder.decode(PendingScrobbleItem.self, from: data) {
                items.append(decoded)
            }
        }
        return items
    }

    func removePendingItems(keys: [String]) throws {
        guard !keys.isEmpty else { return }
        try setup()
        try execute("BEGIN IMMEDIATE TRANSACTION;")
        do {
            try deletePendingItemsInTx(keys: keys)
            try setMetaValue(String(Int64(Date().timeIntervalSince1970)), for: "last_sync_at")
            try execute("COMMIT;")
        } catch {
            _ = try? execute("ROLLBACK;")
            throw error
        }
    }

    func recordSyncedBatch(
        batch: [PendingScrobbleItem],
        acceptedKeys: [String],
        itemResults: [MobileScrobbleItemResultPayload],
        syncedAt: Int64 = Int64(Date().timeIntervalSince1970)
    ) throws {
        guard !acceptedKeys.isEmpty else { return }
        try setup()

        let acceptedSet = Set(acceptedKeys.map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }.filter { !$0.isEmpty })
        let batchMap = Dictionary(uniqueKeysWithValues: batch.map { ($0.idempotencyKey, $0) })
        let resultMap = Dictionary(uniqueKeysWithValues: itemResults.map { ($0.idempotencyKey, $0) })

        try execute("BEGIN IMMEDIATE TRANSACTION;")
        do {
            let insertSQL = """
            INSERT INTO scrobble_sync_history (
                idempotency_key, title, artist, album, played_at, synced_at, status, scrobbled
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(idempotency_key) DO UPDATE SET
                synced_at = excluded.synced_at,
                status = excluded.status,
                scrobbled = excluded.scrobbled;
            """
            guard let insertStmt = try prepare(insertSQL) else {
                throw StoreError.prepareStatement(insertSQL)
            }
            defer { sqlite3_finalize(insertStmt) }

            for key in acceptedSet {
                guard let item = batchMap[key] else { continue }
                let itemResult = resultMap[key]
                let status = itemResult?.status ?? "accepted"
                let scrobbled = itemResult?.scrobbled ?? (status == "accepted")

                sqlite3_reset(insertStmt)
                sqlite3_clear_bindings(insertStmt)
                bindText(item.idempotencyKey, to: 1, in: insertStmt)
                bindText(item.title, to: 2, in: insertStmt)
                bindText(item.artist, to: 3, in: insertStmt)
                bindText(item.album, to: 4, in: insertStmt)
                sqlite3_bind_int64(insertStmt, 5, item.playedAtTimestamp)
                sqlite3_bind_int64(insertStmt, 6, syncedAt)
                bindText(status, to: 7, in: insertStmt)
                sqlite3_bind_int64(insertStmt, 8, scrobbled ? 1 : 0)
                if sqlite3_step(insertStmt) != SQLITE_DONE {
                    throw StoreError.executeStatement(insertSQL)
                }
            }

            try deletePendingItemsInTx(keys: acceptedKeys)
            try execute(
                """
                DELETE FROM scrobble_sync_history
                WHERE idempotency_key NOT IN (
                    SELECT idempotency_key FROM scrobble_sync_history
                    ORDER BY synced_at DESC, played_at DESC
                    LIMIT 200
                );
                """
            )
            try setMetaValue(String(syncedAt), for: "last_sync_at")
            try execute("COMMIT;")
        } catch {
            _ = try? execute("ROLLBACK;")
            throw error
        }
    }

    private func deletePendingItemsInTx(keys: [String]) throws {
        let sql = "DELETE FROM pending_scrobble_queue WHERE idempotency_key = ?;"
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        for key in keys {
            let trimmed = key.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmed.isEmpty else { continue }
            sqlite3_reset(statement)
            sqlite3_clear_bindings(statement)
            bindText(trimmed, to: 1, in: statement)
            if sqlite3_step(statement) != SQLITE_DONE {
                throw StoreError.executeStatement(sql)
            }
        }
    }

    @discardableResult
    func enqueueRealtimeItem(_ item: PendingScrobbleItem, nowTimestamp: Int64 = Int64(Date().timeIntervalSince1970)) throws -> Bool {
        let trimmedArtist = item.artist.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedAlbum = item.album.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedTitle = item.title.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedArtist.isEmpty, !trimmedAlbum.isEmpty, !trimmedTitle.isEmpty, item.playedAtTimestamp > 0 else {
            return false
        }

        try setup()
        try execute("BEGIN IMMEDIATE TRANSACTION;")
        do {
            let dedupeWindow = Self.dedupeWindowSeconds(forDuration: item.duration)
            if try hasRecentRecordDuplicate(
                artist: trimmedArtist,
                track: trimmedTitle,
                around: item.playedAtTimestamp,
                window: dedupeWindow
            ) {
                try execute("COMMIT;")
                return false
            }

            try insertPendingItem(item, createdAt: nowTimestamp)
            try execute("COMMIT;")
            return true
        } catch {
            _ = try? execute("ROLLBACK;")
            throw error
        }
    }

    @discardableResult
    func reconcileMediaLibrarySnapshots(
        _ snapshots: [MediaLibraryTrackSnapshot],
        nowTimestamp: Int64 = Int64(Date().timeIntervalSince1970)
    ) throws -> Int {
        try setup()
        let initialized = try metaValue(for: "baseline_initialized") == "1"
        let baselineInitializedAt = Int64(try metaValue(for: "baseline_initialized_at") ?? "0") ?? 0

        try execute("BEGIN IMMEDIATE TRANSACTION;")
        do {
            if !initialized {
                try upsertBaselineBatch(snapshots, updatedAt: nowTimestamp)
                try setMetaValue("1", for: "baseline_initialized")
                try setMetaValue(String(nowTimestamp), for: "baseline_initialized_at")
                try setMetaValue(String(nowTimestamp), for: "last_scan_at")
                try execute("COMMIT;")
                return 0
            }

            let existingBaseline = try loadBaselineMap()
            var enqueuedCount = 0
            var changedSnapshots: [MediaLibraryTrackSnapshot] = []

            for snapshot in snapshots {
                guard !snapshot.persistentID.isEmpty else { continue }

                if let baseline = existingBaseline[snapshot.persistentID] {
                    if snapshot.playCount > baseline.playCount {
                        let delta = min(snapshot.playCount - baseline.playCount, Self.maxDeltaPerTrack)
                        enqueuedCount += try enqueueMediaLibraryDelta(
                            snapshot: snapshot,
                            delta: delta,
                            nowTimestamp: nowTimestamp
                        )
                        changedSnapshots.append(snapshot)
                    } else if snapshot.playCount != baseline.playCount || snapshot.lastPlayedAt != baseline.lastPlayedAt {
                        changedSnapshots.append(snapshot)
                    }
                } else {
                    if snapshot.playCount > 0 && baselineInitializedAt > 0 && snapshot.lastPlayedAt >= baselineInitializedAt {
                        let delta = min(snapshot.playCount, Self.maxDeltaPerTrack)
                        enqueuedCount += try enqueueMediaLibraryDelta(
                            snapshot: snapshot,
                            delta: delta,
                            nowTimestamp: nowTimestamp
                        )
                    }
                    changedSnapshots.append(snapshot)
                }
            }

            if !changedSnapshots.isEmpty {
                try upsertBaselineBatch(changedSnapshots, updatedAt: nowTimestamp)
            }
            try setMetaValue(String(nowTimestamp), for: "last_scan_at")
            try execute("COMMIT;")
            return enqueuedCount
        } catch {
            _ = try? execute("ROLLBACK;")
            throw error
        }
    }

    // MARK: - Private Helpers

    private func enqueueMediaLibraryDelta(
        snapshot: MediaLibraryTrackSnapshot,
        delta: Int,
        nowTimestamp: Int64
    ) throws -> Int {
        let trimmedTitle = snapshot.title.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedArtist = snapshot.artist.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedAlbum = snapshot.album.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedTitle.isEmpty, !trimmedArtist.isEmpty, !trimmedAlbum.isEmpty else {
            return 0
        }

        let anchorEnd = snapshot.lastPlayedAt > 0 ? snapshot.lastPlayedAt : nowTimestamp
        let stepDuration = max(snapshot.duration, 180)
        let dedupeWindow = Self.dedupeWindowSeconds(forDuration: snapshot.duration)
        let confidence = (snapshot.trackNumber > 0 && snapshot.duration > 0) ? 2 : 1
        var added = 0

        for offsetIndex in stride(from: delta - 1, through: 0, by: -1) {
            let estimatedStart = max(anchorEnd - snapshot.duration - Int64(offsetIndex) * stepDuration, 1)
            if try hasRecentRecordDuplicate(
                artist: trimmedArtist,
                track: trimmedTitle,
                around: estimatedStart,
                window: dedupeWindow
            ) {
                continue
            }

            let key = "ml:\(snapshot.persistentID):\(snapshot.playCount):\(offsetIndex)"
            let item = PendingScrobbleItem(
                idempotencyKey: key,
                title: trimmedTitle,
                artist: trimmedArtist,
                albumArtist: snapshot.albumArtist.trimmingCharacters(in: .whitespacesAndNewlines),
                album: trimmedAlbum,
                genre: snapshot.genre,
                composer: snapshot.composer,
                trackNumber: snapshot.trackNumber,
                discNumber: snapshot.discNumber > 0 ? snapshot.discNumber : 1,
                duration: snapshot.duration,
                releaseDate: snapshot.releaseDate,
                releaseYear: snapshot.releaseYear,
                playedAtTimestamp: estimatedStart,
                uniqueID: snapshot.persistentID,
                bundleID: "com.apple.Music",
                confidence: confidence
            )
            try insertPendingItem(item, createdAt: nowTimestamp)
            added += 1
        }
        return added
    }

    private static func dedupeWindowSeconds(forDuration duration: Int64) -> Int64 {
        guard duration > 0 else { return 60 }
        return min(90, max(duration / 3, min(45, max(duration - 15, 15))))
    }

    private func hasRecentRecordDuplicate(
        artist: String,
        track: String,
        around playedAt: Int64,
        window: Int64
    ) throws -> Bool {
        let pendingSQL = """
        SELECT 1
        FROM pending_scrobble_queue
        WHERE LOWER(artist) = LOWER(?)
          AND LOWER(track) = LOWER(?)
          AND played_at >= ?
          AND played_at <= ?
        LIMIT 1;
        """
        guard let pendingStmt = try prepare(pendingSQL) else {
            throw StoreError.prepareStatement(pendingSQL)
        }
        defer { sqlite3_finalize(pendingStmt) }
        bindText(artist, to: 1, in: pendingStmt)
        bindText(track, to: 2, in: pendingStmt)
        sqlite3_bind_int64(pendingStmt, 3, playedAt - window)
        sqlite3_bind_int64(pendingStmt, 4, playedAt + window)
        if sqlite3_step(pendingStmt) == SQLITE_ROW {
            return true
        }

        let historySQL = """
        SELECT 1
        FROM scrobble_sync_history
        WHERE LOWER(artist) = LOWER(?)
          AND LOWER(title) = LOWER(?)
          AND played_at >= ?
          AND played_at <= ?
        LIMIT 1;
        """
        guard let historyStmt = try prepare(historySQL) else {
            throw StoreError.prepareStatement(historySQL)
        }
        defer { sqlite3_finalize(historyStmt) }
        bindText(artist, to: 1, in: historyStmt)
        bindText(track, to: 2, in: historyStmt)
        sqlite3_bind_int64(historyStmt, 3, playedAt - window)
        sqlite3_bind_int64(historyStmt, 4, playedAt + window)
        return sqlite3_step(historyStmt) == SQLITE_ROW
    }

    private func insertPendingItem(_ item: PendingScrobbleItem, createdAt: Int64) throws {
        let data = try JSONEncoder().encode(item)
        let jsonText = String(decoding: data, as: UTF8.self)
        let sql = """
        INSERT INTO pending_scrobble_queue (
            idempotency_key, payload_json, artist, track, played_at, created_at
        ) VALUES (?, ?, ?, ?, ?, ?)
        ON CONFLICT(idempotency_key) DO NOTHING;
        """
        try execute(sql) { statement in
            self.bindText(item.idempotencyKey, to: 1, in: statement)
            self.bindText(jsonText, to: 2, in: statement)
            self.bindText(item.artist, to: 3, in: statement)
            self.bindText(item.title, to: 4, in: statement)
            sqlite3_bind_int64(statement, 5, item.playedAtTimestamp)
            sqlite3_bind_int64(statement, 6, createdAt)
        }
    }

    private func loadBaselineMap() throws -> [String: (playCount: Int, lastPlayedAt: Int64)] {
        let sql = "SELECT persistent_id, play_count, last_played_at FROM media_library_baseline;"
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        var map: [String: (playCount: Int, lastPlayedAt: Int64)] = [:]
        while sqlite3_step(statement) == SQLITE_ROW {
            let persistentID = string(at: 0, in: statement)
            let playCount = Int(sqlite3_column_int64(statement, 1))
            let lastPlayedAt = sqlite3_column_int64(statement, 2)
            map[persistentID] = (playCount, lastPlayedAt)
        }
        return map
    }

    private func upsertBaselineBatch(_ snapshots: [MediaLibraryTrackSnapshot], updatedAt: Int64) throws {
        let sql = """
        INSERT INTO media_library_baseline (persistent_id, play_count, last_played_at, updated_at)
        VALUES (?, ?, ?, ?)
        ON CONFLICT(persistent_id) DO UPDATE SET
            play_count = excluded.play_count,
            last_played_at = excluded.last_played_at,
            updated_at = excluded.updated_at;
        """
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        for snapshot in snapshots {
            guard !snapshot.persistentID.isEmpty else { continue }
            sqlite3_reset(statement)
            sqlite3_clear_bindings(statement)
            bindText(snapshot.persistentID, to: 1, in: statement)
            sqlite3_bind_int64(statement, 2, Int64(snapshot.playCount))
            sqlite3_bind_int64(statement, 3, snapshot.lastPlayedAt)
            sqlite3_bind_int64(statement, 4, updatedAt)
            if sqlite3_step(statement) != SQLITE_DONE {
                throw StoreError.executeStatement(sql)
            }
        }
    }

    private func metaValue(for key: String) throws -> String? {
        let sql = "SELECT value FROM scrobble_meta WHERE key = ? LIMIT 1;"
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }
        bindText(key, to: 1, in: statement)
        if sqlite3_step(statement) == SQLITE_ROW {
            return string(at: 0, in: statement)
        }
        return nil
    }

    private func setMetaValue(_ value: String, for key: String) throws {
        let sql = """
        INSERT INTO scrobble_meta (key, value) VALUES (?, ?)
        ON CONFLICT(key) DO UPDATE SET value = excluded.value;
        """
        try execute(sql) { statement in
            self.bindText(key, to: 1, in: statement)
            self.bindText(value, to: 2, in: statement)
        }
    }

    private func openIfNeeded() throws {
        if db != nil { return }
        let directory = databaseURL.deletingLastPathComponent()
        try fileManager.createDirectory(at: directory, withIntermediateDirectories: true, attributes: nil)

        var handle: OpaquePointer?
        let flags = SQLITE_OPEN_CREATE | SQLITE_OPEN_READWRITE | SQLITE_OPEN_FULLMUTEX
        if sqlite3_open_v2(databaseURL.path, &handle, flags, nil) != SQLITE_OK {
            if let handle {
                sqlite3_close(handle)
            }
            throw StoreError.openDatabase
        }
        db = handle
        _ = try? execute("PRAGMA journal_mode=WAL;")
        _ = try? execute("PRAGMA busy_timeout=3000;")
    }

    @discardableResult
    private func execute(_ sql: String, bind: ((OpaquePointer) -> Void)? = nil) throws -> Int32 {
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }
        bind?(statement)
        let result = sqlite3_step(statement)
        guard result == SQLITE_DONE || result == SQLITE_ROW else {
            throw StoreError.executeStatement(sql)
        }
        return result
    }

    private func prepare(_ sql: String) throws -> OpaquePointer? {
        try openIfNeeded()
        var statement: OpaquePointer?
        if sqlite3_prepare_v2(db, sql, -1, &statement, nil) != SQLITE_OK {
            throw StoreError.prepareStatement(sql)
        }
        return statement
    }

    private func bindText(_ value: String, to index: Int32, in statement: OpaquePointer?) {
        let nsValue = value as NSString
        sqlite3_bind_text(statement, index, nsValue.utf8String, -1, mobileScrobbleSQLiteTransient)
    }

    private func string(at index: Int32, in statement: OpaquePointer?) -> String {
        guard let value = sqlite3_column_text(statement, index) else {
            return ""
        }
        return String(cString: value)
    }
}

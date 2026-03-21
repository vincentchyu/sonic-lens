import Foundation
import SQLite3

private let sqliteTransient = unsafeBitCast(-1, to: sqlite3_destructor_type.self)

actor LibraryIndexStore {
    static let syncSchemaVersion = 5

    enum StoreError: Error {
        case openDatabase
        case prepareStatement(String)
        case executeStatement(String)
    }

    private var db: OpaquePointer?
    private let databaseURL: URL

    init(fileManager: FileManager = .default) {
        let supportDirectory = try? fileManager.url(
            for: .applicationSupportDirectory,
            in: .userDomainMask,
            appropriateFor: nil,
            create: true
        )
        let baseDirectory = supportDirectory ?? fileManager.temporaryDirectory
        self.databaseURL = baseDirectory.appendingPathComponent("soniclens-library-index.sqlite")
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
            CREATE TABLE IF NOT EXISTS album_index (
                id INTEGER PRIMARY KEY,
                name TEXT NOT NULL,
                artist TEXT NOT NULL,
                release_date TEXT,
                cover_art_url TEXT,
                cover_art_mime TEXT,
                cover_art_object_key TEXT,
                play_count INTEGER NOT NULL DEFAULT 0,
                created_at TEXT,
                updated_at TEXT
            );
            """
        )
        try execute("CREATE INDEX IF NOT EXISTS idx_album_index_name ON album_index(name);")
        try execute("CREATE INDEX IF NOT EXISTS idx_album_index_artist ON album_index(artist);")
        try execute("CREATE INDEX IF NOT EXISTS idx_album_index_updated_at ON album_index(updated_at);")
        try createFTS5Table(
            named: "album_index_fts",
            columns: "name, artist",
            preferredTokenizer: "trigram"
        )

        try execute(
            """
            CREATE TABLE IF NOT EXISTS track_index (
                id INTEGER PRIMARY KEY,
                artist TEXT NOT NULL,
                album TEXT NOT NULL,
                track TEXT NOT NULL,
                play_count INTEGER NOT NULL DEFAULT 0,
                track_number INTEGER,
                disc_number INTEGER,
                duration INTEGER,
                is_apple_music_fav INTEGER NOT NULL DEFAULT 0,
                is_last_fm_fav INTEGER NOT NULL DEFAULT 0,
                is_reported INTEGER NOT NULL DEFAULT 1,
                created_at TEXT,
                updated_at TEXT
            );
            """
        )
        try execute("CREATE INDEX IF NOT EXISTS idx_track_index_track ON track_index(track);")
        try execute("CREATE INDEX IF NOT EXISTS idx_track_index_artist ON track_index(artist);")
        try execute("CREATE INDEX IF NOT EXISTS idx_track_index_album ON track_index(album);")
        try execute("CREATE INDEX IF NOT EXISTS idx_track_index_updated_at ON track_index(updated_at);")
        try execute("CREATE INDEX IF NOT EXISTS idx_track_index_reported ON track_index(is_reported);")
        try createFTS5Table(
            named: "track_index_fts",
            columns: "track, artist, album",
            preferredTokenizer: "trigram"
        )

        try execute(
            """
            CREATE TABLE IF NOT EXISTS sync_meta (
                key TEXT PRIMARY KEY,
                value TEXT NOT NULL
            );
            """
        )
    }

    func resetForFullResync() throws {
        try openIfNeeded()
        try execute("BEGIN IMMEDIATE TRANSACTION;")
        do {
            try execute("DROP TABLE IF EXISTS album_index_fts;")
            try execute("DROP TABLE IF EXISTS track_index_fts;")
            try execute("DROP TABLE IF EXISTS album_index;")
            try execute("DROP TABLE IF EXISTS track_index;")
            try execute("DROP TABLE IF EXISTS sync_meta;")
            try execute("COMMIT;")
        } catch {
            _ = try? execute("ROLLBACK;")
            throw error
        }

        try setup()
    }

    func currentSyncVersion() throws -> Int64 {
        try openIfNeeded()
        guard let statement = try prepare("SELECT value FROM sync_meta WHERE key = 'library_sync_version' LIMIT 1;") else {
            return 0
        }
        defer { sqlite3_finalize(statement) }

        if sqlite3_step(statement) == SQLITE_ROW, let text = sqlite3_column_text(statement, 0) {
            return Int64(String(cString: text)) ?? 0
        }
        return 0
    }

    func currentSyncSchemaVersion() throws -> Int {
        try openIfNeeded()
        guard let statement = try prepare("SELECT value FROM sync_meta WHERE key = 'library_sync_schema_version' LIMIT 1;") else {
            return 0
        }
        defer { sqlite3_finalize(statement) }

        if sqlite3_step(statement) == SQLITE_ROW, let text = sqlite3_column_text(statement, 0) {
            return Int(String(cString: text)) ?? 0
        }
        return 0
    }

    func updateSyncVersion(_ version: Int64) throws {
        try execute(
            "INSERT INTO sync_meta(key, value) VALUES('library_sync_version', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value;",
            bind: { statement in
                sqlite3_bind_int64(statement, 1, version)
            }
        )
    }

    func updateSyncSchemaVersion(_ version: Int) throws {
        try execute(
            "INSERT INTO sync_meta(key, value) VALUES('library_sync_schema_version', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value;",
            bind: { statement in
                sqlite3_bind_int64(statement, 1, Int64(version))
            }
        )
    }

    func apply(_ payload: LibrarySyncResponse) throws {
        try openIfNeeded()
        try execute("BEGIN IMMEDIATE TRANSACTION;")
        do {
            try deleteAlbums(ids: payload.deletedAlbumIDs)
            try deleteTracks(ids: payload.deletedTrackIDs)
            try upsertAlbums(payload.albums)
            try upsertTracks(payload.tracks)
            try updateSyncVersion(payload.syncVersion)
            try updateSyncSchemaVersion(Self.syncSchemaVersion)
            try execute("COMMIT;")
        } catch {
            _ = try? execute("ROLLBACK;")
            throw error
        }
    }

    func replaceAll(with payload: LibrarySyncResponse) throws {
        try openIfNeeded()
        try execute("BEGIN IMMEDIATE TRANSACTION;")
        do {
            try execute("DELETE FROM album_index;")
            try execute("DELETE FROM track_index;")
            try execute("DELETE FROM album_index_fts;")
            try execute("DELETE FROM track_index_fts;")
            try upsertAlbums(payload.albums)
            try upsertTracks(payload.tracks)
            try updateSyncVersion(payload.syncVersion)
            try updateSyncSchemaVersion(Self.syncSchemaVersion)
            try execute("COMMIT;")
        } catch {
            _ = try? execute("ROLLBACK;")
            throw error
        }
    }

    func requiresFullResync() throws -> Bool {
        try currentSyncSchemaVersion() != Self.syncSchemaVersion
    }

    func replaceReportedStatus(using records: [UnscrobbledRecord]) throws {
        try openIfNeeded()
        try execute("BEGIN IMMEDIATE TRANSACTION;")
        do {
            try execute("UPDATE track_index SET is_reported = 1;")
            let sql = "UPDATE track_index SET is_reported = 0 WHERE artist = ? AND album = ? AND track = ?;"
            guard let statement = try prepare(sql) else {
                throw StoreError.prepareStatement(sql)
            }
            defer { sqlite3_finalize(statement) }

            for record in records {
                sqlite3_reset(statement)
                sqlite3_clear_bindings(statement)
                bindText(record.artist, to: 1, in: statement)
                bindText(record.album, to: 2, in: statement)
                bindText(record.track, to: 3, in: statement)
                if sqlite3_step(statement) != SQLITE_DONE {
                    throw StoreError.executeStatement(sql)
                }
            }
            try execute("COMMIT;")
        } catch {
            _ = try? execute("ROLLBACK;")
            throw error
        }
    }

    func updateTrackFavoriteStatus(
        artist: String,
        album: String,
        track: String,
        trackNumber: Int?,
        discNumber: Int?,
        appleMusic: Bool,
        lastFm: Bool
    ) throws {
        let hasPhysicalPosition = trackNumber != nil || discNumber != nil
        let sql: String
        if hasPhysicalPosition {
            sql = """
            UPDATE track_index
            SET is_apple_music_fav = ?, is_last_fm_fav = ?
            WHERE artist = ? AND album = ? AND track = ?
              AND track_number = COALESCE(?, track_number)
              AND disc_number = COALESCE(?, disc_number);
            """
        } else {
            sql = """
            UPDATE track_index
            SET is_apple_music_fav = ?, is_last_fm_fav = ?
            WHERE artist = ? AND album = ? AND track = ?;
            """
        }

        try execute(sql) { statement in
            sqlite3_bind_int(statement, 1, appleMusic ? 1 : 0)
            sqlite3_bind_int(statement, 2, lastFm ? 1 : 0)
            self.bindText(artist, to: 3, in: statement)
            self.bindText(album, to: 4, in: statement)
            self.bindText(track, to: 5, in: statement)
            if hasPhysicalPosition {
                self.bindOptionalInt(trackNumber, to: 6, in: statement)
                self.bindOptionalInt(discNumber, to: 7, in: statement)
            }
        }
    }

    func queryAlbums(sort: LibrarySort, keyword: String, limit: Int, offset: Int) throws -> [Album] {
        let trimmed = keyword.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            let ftsRows = try queryAlbumsUsingFTS(sort: sort, keyword: trimmed, limit: limit, offset: offset)
            if !ftsRows.isEmpty {
                return ftsRows
            }
        }
        return try queryAlbumsUsingLIKE(sort: sort, keyword: trimmed, limit: limit, offset: offset)
    }

    func countAlbums(keyword: String) throws -> Int {
        let trimmed = keyword.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            let ftsCount = try countAlbumsUsingFTS(keyword: trimmed)
            if ftsCount > 0 {
                return ftsCount
            }
        }
        return try countAlbumsUsingLIKE(keyword: trimmed)
    }

    func queryTracks(sort: LibrarySort, filter: TrackFilter, keyword: String, limit: Int, offset: Int) throws -> [Track] {
        let trimmed = keyword.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            let ftsRows = try queryTracksUsingFTS(sort: sort, filter: filter, keyword: trimmed, limit: limit, offset: offset)
            if !ftsRows.isEmpty {
                return ftsRows
            }
        }
        return try queryTracksUsingLIKE(sort: sort, filter: filter, keyword: trimmed, limit: limit, offset: offset)
    }

    func countTracks(filter: TrackFilter, keyword: String) throws -> Int {
        let trimmed = keyword.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            let ftsCount = try countTracksUsingFTS(filter: filter, keyword: trimmed)
            if ftsCount > 0 {
                return ftsCount
            }
        }
        return try countTracksUsingLIKE(filter: filter, keyword: trimmed)
    }

    private func upsertAlbums(_ albums: [Album]) throws {
        let sql = """
        INSERT INTO album_index(
            id, name, artist, release_date, cover_art_url, cover_art_mime, cover_art_object_key, play_count, created_at, updated_at
        )
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            name = excluded.name,
            artist = excluded.artist,
            release_date = excluded.release_date,
            cover_art_url = excluded.cover_art_url,
            cover_art_mime = excluded.cover_art_mime,
            cover_art_object_key = excluded.cover_art_object_key,
            play_count = excluded.play_count,
            created_at = excluded.created_at,
            updated_at = excluded.updated_at;
        """
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        for album in albums {
            sqlite3_reset(statement)
            sqlite3_clear_bindings(statement)
            sqlite3_bind_int64(statement, 1, album.id)
            bindText(album.name, to: 2, in: statement)
            bindText(album.artist, to: 3, in: statement)
            bindOptionalText(album.releaseDate, to: 4, in: statement)
            bindOptionalText(album.coverArtURL, to: 5, in: statement)
            bindOptionalText(album.coverArtMime, to: 6, in: statement)
            bindOptionalText(album.coverArtObjectKey, to: 7, in: statement)
            sqlite3_bind_int64(statement, 8, Int64(album.playCount ?? 0))
            bindOptionalText(album.createdAt, to: 9, in: statement)
            bindOptionalText(album.updatedAt, to: 10, in: statement)
            if sqlite3_step(statement) != SQLITE_DONE {
                throw StoreError.executeStatement(sql)
            }
        }
        try upsertAlbumFTS(albums)
    }

    private func deleteAlbums(ids: [Int64]) throws {
        guard !ids.isEmpty else { return }
        let sql = "DELETE FROM album_index WHERE id = ?;"
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        for id in ids {
            sqlite3_reset(statement)
            sqlite3_clear_bindings(statement)
            sqlite3_bind_int64(statement, 1, id)
            if sqlite3_step(statement) != SQLITE_DONE {
                throw StoreError.executeStatement(sql)
            }
        }
        try deleteAlbumFTS(ids: ids)
    }

    private func upsertTracks(_ tracks: [Track]) throws {
        let sql = """
        INSERT INTO track_index(
            id, artist, album, track, play_count, track_number, disc_number, duration,
            is_apple_music_fav, is_last_fm_fav, created_at, updated_at
        )
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            artist = excluded.artist,
            album = excluded.album,
            track = excluded.track,
            play_count = excluded.play_count,
            track_number = excluded.track_number,
            disc_number = excluded.disc_number,
            duration = excluded.duration,
            is_apple_music_fav = excluded.is_apple_music_fav,
            is_last_fm_fav = excluded.is_last_fm_fav,
            created_at = excluded.created_at,
            updated_at = excluded.updated_at;
        """
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        for track in tracks {
            sqlite3_reset(statement)
            sqlite3_clear_bindings(statement)
            sqlite3_bind_int64(statement, 1, track.id)
            bindText(track.artist, to: 2, in: statement)
            bindText(track.album, to: 3, in: statement)
            bindText(track.track, to: 4, in: statement)
            sqlite3_bind_int64(statement, 5, Int64(track.playCount))
            bindOptionalInt(track.trackNumber, to: 6, in: statement)
            bindOptionalInt(track.discNumber, to: 7, in: statement)
            bindOptionalInt64(track.duration, to: 8, in: statement)
            sqlite3_bind_int(statement, 9, (track.isAppleMusicFav ?? false) ? 1 : 0)
            sqlite3_bind_int(statement, 10, (track.isLastFmFav ?? false) ? 1 : 0)
            bindOptionalText(track.createdAt, to: 11, in: statement)
            bindOptionalText(track.updatedAt, to: 12, in: statement)
            if sqlite3_step(statement) != SQLITE_DONE {
                throw StoreError.executeStatement(sql)
            }
        }
        try upsertTrackFTS(tracks)
    }

    private func deleteTracks(ids: [Int64]) throws {
        guard !ids.isEmpty else { return }
        let sql = "DELETE FROM track_index WHERE id = ?;"
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        for id in ids {
            sqlite3_reset(statement)
            sqlite3_clear_bindings(statement)
            sqlite3_bind_int64(statement, 1, id)
            if sqlite3_step(statement) != SQLITE_DONE {
                throw StoreError.executeStatement(sql)
            }
        }
        try deleteTrackFTS(ids: ids)
    }

    private func upsertAlbumFTS(_ albums: [Album]) throws {
        guard !albums.isEmpty else { return }
        let sql = "INSERT OR REPLACE INTO album_index_fts(rowid, name, artist) VALUES(?, ?, ?);"
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        for album in albums {
            sqlite3_reset(statement)
            sqlite3_clear_bindings(statement)
            sqlite3_bind_int64(statement, 1, album.id)
            bindText(album.name, to: 2, in: statement)
            bindText(album.artist, to: 3, in: statement)
            if sqlite3_step(statement) != SQLITE_DONE {
                throw StoreError.executeStatement(sql)
            }
        }
    }

    private func deleteAlbumFTS(ids: [Int64]) throws {
        guard !ids.isEmpty else { return }
        let sql = "DELETE FROM album_index_fts WHERE rowid = ?;"
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        for id in ids {
            sqlite3_reset(statement)
            sqlite3_clear_bindings(statement)
            sqlite3_bind_int64(statement, 1, id)
            if sqlite3_step(statement) != SQLITE_DONE {
                throw StoreError.executeStatement(sql)
            }
        }
    }

    private func upsertTrackFTS(_ tracks: [Track]) throws {
        guard !tracks.isEmpty else { return }
        let sql = "INSERT OR REPLACE INTO track_index_fts(rowid, track, artist, album) VALUES(?, ?, ?, ?);"
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        for track in tracks {
            sqlite3_reset(statement)
            sqlite3_clear_bindings(statement)
            sqlite3_bind_int64(statement, 1, track.id)
            bindText(track.track, to: 2, in: statement)
            bindText(track.artist, to: 3, in: statement)
            bindText(track.album, to: 4, in: statement)
            if sqlite3_step(statement) != SQLITE_DONE {
                throw StoreError.executeStatement(sql)
            }
        }
    }

    private func deleteTrackFTS(ids: [Int64]) throws {
        guard !ids.isEmpty else { return }
        let sql = "DELETE FROM track_index_fts WHERE rowid = ?;"
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        for id in ids {
            sqlite3_reset(statement)
            sqlite3_clear_bindings(statement)
            sqlite3_bind_int64(statement, 1, id)
            if sqlite3_step(statement) != SQLITE_DONE {
                throw StoreError.executeStatement(sql)
            }
        }
    }

    private func trackWhereClauses(filter: TrackFilter, keyword: String) -> (sql: String, binds: [String]) {
        var clauses: [String] = []
        var binds: [String] = []

        switch filter {
        case .all:
            break
        case .favorites:
            clauses.append("(is_apple_music_fav = 1 OR is_last_fm_fav = 1)")
        case .unreported:
            clauses.append("is_reported = 0")
        }

        let trimmed = keyword.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            clauses.append("(track LIKE ? OR artist LIKE ? OR album LIKE ?)")
            let term = "%\(trimmed)%"
            binds.append(contentsOf: [term, term, term])
        }

        return (clauses.joined(separator: " AND "), binds)
    }

    private func createFTS5Table(named name: String, columns: String, preferredTokenizer: String) throws {
        do {
            try execute("CREATE VIRTUAL TABLE IF NOT EXISTS \(name) USING fts5(\(columns), tokenize = '\(preferredTokenizer)');")
        } catch {
            try execute("CREATE VIRTUAL TABLE IF NOT EXISTS \(name) USING fts5(\(columns), tokenize = 'unicode61 remove_diacritics 0');")
        }
    }

    private func queryAlbumsUsingFTS(sort: LibrarySort, keyword: String, limit: Int, offset: Int) throws -> [Album] {
        let sql = """
        SELECT id, name, artist, release_date, cover_art_url, cover_art_mime, cover_art_object_key, play_count, created_at, updated_at
        FROM album_index
        WHERE id IN (
            SELECT rowid FROM album_index_fts WHERE album_index_fts MATCH ?
        )
        ORDER BY \(albumOrder(sort))
        LIMIT ? OFFSET ?;
        """
        return try fetchAlbums(sql: sql, binds: [ftsQuery(for: keyword)], limit: limit, offset: offset)
    }

    private func queryAlbumsUsingLIKE(sort: LibrarySort, keyword: String, limit: Int, offset: Int) throws -> [Album] {
        let hasKeyword = !keyword.isEmpty
        let sql = """
        SELECT id, name, artist, release_date, cover_art_url, cover_art_mime, cover_art_object_key, play_count, created_at, updated_at
        FROM album_index
        \(hasKeyword ? "WHERE name LIKE ? OR artist LIKE ?" : "")
        ORDER BY \(albumOrder(sort))
        LIMIT ? OFFSET ?;
        """
        let binds = hasKeyword ? ["%\(keyword)%", "%\(keyword)%"] : []
        return try fetchAlbums(sql: sql, binds: binds, limit: limit, offset: offset)
    }

    private func countAlbumsUsingFTS(keyword: String) throws -> Int {
        let sql = """
        SELECT COUNT(*)
        FROM album_index
        WHERE id IN (
            SELECT rowid FROM album_index_fts WHERE album_index_fts MATCH ?
        );
        """
        return try scalarCount(sql: sql, binds: [ftsQuery(for: keyword)])
    }

    private func countAlbumsUsingLIKE(keyword: String) throws -> Int {
        let hasKeyword = !keyword.isEmpty
        let sql = hasKeyword
            ? "SELECT COUNT(*) FROM album_index WHERE name LIKE ? OR artist LIKE ?;"
            : "SELECT COUNT(*) FROM album_index;"
        let binds = hasKeyword ? ["%\(keyword)%", "%\(keyword)%"] : []
        return try scalarCount(sql: sql, binds: binds)
    }

    private func queryTracksUsingFTS(sort: LibrarySort, filter: TrackFilter, keyword: String, limit: Int, offset: Int) throws -> [Track] {
        var clauses = trackWhereClauses(filter: filter, keyword: "")
        clauses.sql = clauses.sql.isEmpty
            ? "id IN (SELECT rowid FROM track_index_fts WHERE track_index_fts MATCH ?)"
            : clauses.sql + " AND id IN (SELECT rowid FROM track_index_fts WHERE track_index_fts MATCH ?)"
        clauses.binds.append(ftsQuery(for: keyword))
        return try fetchTracks(sort: sort, clauses: clauses, limit: limit, offset: offset)
    }

    private func queryTracksUsingLIKE(sort: LibrarySort, filter: TrackFilter, keyword: String, limit: Int, offset: Int) throws -> [Track] {
        try fetchTracks(sort: sort, clauses: trackWhereClauses(filter: filter, keyword: keyword), limit: limit, offset: offset)
    }

    private func countTracksUsingFTS(filter: TrackFilter, keyword: String) throws -> Int {
        var clauses = trackWhereClauses(filter: filter, keyword: "")
        clauses.sql = clauses.sql.isEmpty
            ? "id IN (SELECT rowid FROM track_index_fts WHERE track_index_fts MATCH ?)"
            : clauses.sql + " AND id IN (SELECT rowid FROM track_index_fts WHERE track_index_fts MATCH ?)"
        clauses.binds.append(ftsQuery(for: keyword))
        return try scalarCount(
            sql: "SELECT COUNT(*) FROM track_index \(clauses.sql.isEmpty ? "" : "WHERE \(clauses.sql)");",
            binds: clauses.binds
        )
    }

    private func countTracksUsingLIKE(filter: TrackFilter, keyword: String) throws -> Int {
        let clauses = trackWhereClauses(filter: filter, keyword: keyword)
        return try scalarCount(
            sql: "SELECT COUNT(*) FROM track_index \(clauses.sql.isEmpty ? "" : "WHERE \(clauses.sql)");",
            binds: clauses.binds
        )
    }

    private func fetchAlbums(sql: String, binds: [String], limit: Int, offset: Int) throws -> [Album] {
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        var bindIndex: Int32 = 1
        for value in binds {
            bindText(value, to: bindIndex, in: statement)
            bindIndex += 1
        }
        sqlite3_bind_int64(statement, bindIndex, Int64(limit))
        sqlite3_bind_int64(statement, bindIndex + 1, Int64(offset))

        var rows: [Album] = []
        while sqlite3_step(statement) == SQLITE_ROW {
            rows.append(
                Album(
                    id: sqlite3_column_int64(statement, 0),
                    name: string(at: 1, in: statement),
                    artist: string(at: 2, in: statement),
                    releaseDate: optionalString(at: 3, in: statement),
                    coverArtURL: optionalString(at: 4, in: statement),
                    coverArtMime: optionalString(at: 5, in: statement),
                    coverArtObjectKey: optionalString(at: 6, in: statement),
                    genre: nil,
                    totalDiscs: nil,
                    playCount: Int(sqlite3_column_int(statement, 7)),
                    createdAt: optionalString(at: 8, in: statement),
                    updatedAt: optionalString(at: 9, in: statement)
                )
            )
        }
        return rows
    }

    private func fetchTracks(sort: LibrarySort, clauses: (sql: String, binds: [String]), limit: Int, offset: Int) throws -> [Track] {
        let whereClause = clauses.sql.isEmpty ? "" : "WHERE \(clauses.sql)"
        let sql = """
        SELECT id, artist, album, track, play_count, track_number, disc_number, duration,
               is_apple_music_fav, is_last_fm_fav, created_at, updated_at
        FROM track_index
        \(whereClause)
        ORDER BY \(trackOrder(sort))
        LIMIT ? OFFSET ?;
        """
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        var bindIndex: Int32 = 1
        for value in clauses.binds {
            bindText(value, to: bindIndex, in: statement)
            bindIndex += 1
        }
        sqlite3_bind_int64(statement, bindIndex, Int64(limit))
        sqlite3_bind_int64(statement, bindIndex + 1, Int64(offset))

        var rows: [Track] = []
        while sqlite3_step(statement) == SQLITE_ROW {
            rows.append(
                Track(
                    id: sqlite3_column_int64(statement, 0),
                    artist: string(at: 1, in: statement),
                    album: string(at: 2, in: statement),
                    track: string(at: 3, in: statement),
                    playCount: Int(sqlite3_column_int(statement, 4)),
                    trackNumber: optionalInt(at: 5, in: statement),
                    discNumber: optionalInt(at: 6, in: statement),
                    duration: optionalInt64(at: 7, in: statement),
                    isAppleMusicFav: sqlite3_column_int(statement, 8) != 0,
                    isLastFmFav: sqlite3_column_int(statement, 9) != 0,
                    createdAt: optionalString(at: 10, in: statement),
                    updatedAt: optionalString(at: 11, in: statement)
                )
            )
        }
        return rows
    }

    private func scalarCount(sql: String, binds: [String]) throws -> Int {
        guard let statement = try prepare(sql) else {
            throw StoreError.prepareStatement(sql)
        }
        defer { sqlite3_finalize(statement) }

        var bindIndex: Int32 = 1
        for value in binds {
            bindText(value, to: bindIndex, in: statement)
            bindIndex += 1
        }
        return sqlite3_step(statement) == SQLITE_ROW ? Int(sqlite3_column_int(statement, 0)) : 0
    }

    private func ftsQuery(for keyword: String) -> String {
        "\"\(keyword.replacingOccurrences(of: "\"", with: "\"\""))\""
    }

    private func albumOrder(_ sort: LibrarySort) -> String {
        switch sort {
        case .recent:
            return "created_at DESC, id DESC"
        case .updated:
            return "updated_at DESC, id DESC"
        case .releaseDate:
            return "CASE WHEN release_date IS NULL OR release_date = '' THEN 1 ELSE 0 END ASC, release_date DESC, name COLLATE NOCASE ASC, id ASC"
        case .alpha:
            return "name COLLATE NOCASE ASC, artist COLLATE NOCASE ASC, id ASC"
        case .plays:
            return "play_count DESC, name COLLATE NOCASE ASC, id ASC"
        }
    }

    private func trackOrder(_ sort: LibrarySort) -> String {
        switch sort {
        case .recent:
            return "created_at DESC, id DESC"
        case .updated:
            return "updated_at DESC, id DESC"
        case .releaseDate:
            return "created_at DESC, id DESC"
        case .alpha:
            return "track COLLATE NOCASE ASC, artist COLLATE NOCASE ASC, id ASC"
        case .plays:
            return "play_count DESC, track COLLATE NOCASE ASC, id ASC"
        }
    }

    private func openIfNeeded() throws {
        if db != nil {
            return
        }

        let parent = databaseURL.deletingLastPathComponent()
        try FileManager.default.createDirectory(at: parent, withIntermediateDirectories: true, attributes: nil)

        var handle: OpaquePointer?
        if sqlite3_open(databaseURL.path, &handle) != SQLITE_OK {
            throw StoreError.openDatabase
        }
        db = handle
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
        sqlite3_bind_text(statement, index, nsValue.utf8String, -1, sqliteTransient)
    }

    private func bindOptionalText(_ value: String?, to index: Int32, in statement: OpaquePointer?) {
        guard let value else {
            sqlite3_bind_null(statement, index)
            return
        }
        bindText(value, to: index, in: statement)
    }

    private func bindOptionalInt(_ value: Int?, to index: Int32, in statement: OpaquePointer?) {
        guard let value else {
            sqlite3_bind_null(statement, index)
            return
        }
        sqlite3_bind_int64(statement, index, Int64(value))
    }

    private func bindOptionalInt64(_ value: Int64?, to index: Int32, in statement: OpaquePointer?) {
        guard let value else {
            sqlite3_bind_null(statement, index)
            return
        }
        sqlite3_bind_int64(statement, index, value)
    }

    private func string(at index: Int32, in statement: OpaquePointer?) -> String {
        optionalString(at: index, in: statement) ?? ""
    }

    private func optionalString(at index: Int32, in statement: OpaquePointer?) -> String? {
        guard let value = sqlite3_column_text(statement, index) else {
            return nil
        }
        return String(cString: value)
    }

    private func optionalInt(at index: Int32, in statement: OpaquePointer?) -> Int? {
        guard sqlite3_column_type(statement, index) != SQLITE_NULL else {
            return nil
        }
        return Int(sqlite3_column_int(statement, index))
    }

    private func optionalInt64(at index: Int32, in statement: OpaquePointer?) -> Int64? {
        guard sqlite3_column_type(statement, index) != SQLITE_NULL else {
            return nil
        }
        return sqlite3_column_int64(statement, index)
    }
}

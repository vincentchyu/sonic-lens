import Foundation

enum MediaLibraryAuthorizationState: String, Sendable {
    case notDetermined
    case authorized
    case denied
    case restricted
    case unsupported

    var displayTitle: String {
        switch self {
        case .notDetermined:
            return "未授权媒体库"
        case .authorized:
            return "已授权媒体库"
        case .denied:
            return "媒体库权限已拒绝"
        case .restricted:
            return "媒体库权限受限"
        case .unsupported:
            return "当前设备不支持"
        }
    }
}

struct MediaLibraryTrackSnapshot: Sendable, Equatable {
    let persistentID: String
    let title: String
    let artist: String
    let albumArtist: String
    let album: String
    let genre: String
    let composer: String
    let trackNumber: Int
    let discNumber: Int
    let duration: Int64
    let releaseDate: String
    let releaseYear: Int
    let playCount: Int
    let lastPlayedAt: Int64
}

struct PendingScrobbleItem: Codable, Sendable, Identifiable, Equatable {
    var id: String { idempotencyKey }

    var captureSourceTitle: String {
        idempotencyKey.hasPrefix("rt:") ? "实时捕获" : "媒体库差量"
    }

    let idempotencyKey: String
    let title: String
    let artist: String
    let albumArtist: String
    let album: String
    let genre: String
    let composer: String
    let trackNumber: Int
    let discNumber: Int
    let duration: Int64
    let releaseDate: String
    let releaseYear: Int
    let playedAt: String
    let playedAtTimestamp: Int64
    let uniqueID: String
    let bundleID: String
    let confidence: Int

    enum CodingKeys: String, CodingKey {
        case idempotencyKey = "idempotency_key"
        case title
        case artist
        case albumArtist = "album_artist"
        case album
        case genre
        case composer
        case trackNumber = "track_number"
        case discNumber = "disc_number"
        case duration
        case releaseDate = "release_date"
        case releaseYear = "release_year"
        case playedAt = "played_at"
        case playedAtTimestamp = "played_at_timestamp"
        case uniqueID = "unique_id"
        case bundleID = "bundle_id"
        case confidence
    }

    init(
        idempotencyKey: String,
        title: String,
        artist: String,
        albumArtist: String,
        album: String,
        genre: String = "",
        composer: String = "",
        trackNumber: Int = 0,
        discNumber: Int = 1,
        duration: Int64 = 0,
        releaseDate: String = "",
        releaseYear: Int = 0,
        playedAtTimestamp: Int64,
        uniqueID: String = "",
        bundleID: String = "com.apple.Music",
        confidence: Int = 2
    ) {
        self.idempotencyKey = idempotencyKey
        self.title = title
        self.artist = artist
        self.albumArtist = albumArtist
        self.album = album
        self.genre = genre
        self.composer = composer
        self.trackNumber = min(max(trackNumber, 0), 127)
        self.discNumber = min(max(discNumber, 0), 127)
        self.duration = max(duration, 0)
        self.releaseDate = releaseDate
        self.releaseYear = max(releaseYear, 0)
        self.playedAtTimestamp = playedAtTimestamp
        self.playedAt = Date(timeIntervalSince1970: TimeInterval(playedAtTimestamp)).formatted(.iso8601)
        self.uniqueID = uniqueID
        self.bundleID = bundleID
        self.confidence = confidence
    }

    func toUploadPayload() -> MobileScrobbleUploadRecord {
        MobileScrobbleUploadRecord(
            idempotencyKey: idempotencyKey,
            title: title,
            artist: artist,
            albumArtist: albumArtist,
            album: album,
            genre: genre,
            composer: composer,
            trackNumber: trackNumber,
            discNumber: discNumber,
            duration: duration,
            releaseDate: releaseDate,
            releaseYear: releaseYear,
            playedAt: playedAt,
            uniqueID: uniqueID,
            bundleID: bundleID,
            confidence: confidence
        )
    }
}

struct MobileScrobbleUploadRecord: Encodable, Sendable {
    let idempotencyKey: String
    let title: String
    let artist: String
    let albumArtist: String
    let album: String
    let genre: String
    let composer: String
    let trackNumber: Int
    let discNumber: Int
    let duration: Int64
    let releaseDate: String
    let releaseYear: Int
    let playedAt: String
    let uniqueID: String
    let bundleID: String
    let confidence: Int

    enum CodingKeys: String, CodingKey {
        case idempotencyKey = "idempotency_key"
        case title
        case artist
        case albumArtist = "album_artist"
        case album
        case genre
        case composer
        case trackNumber = "track_number"
        case discNumber = "disc_number"
        case duration
        case releaseDate = "release_date"
        case releaseYear = "release_year"
        case playedAt = "played_at"
        case uniqueID = "unique_id"
        case bundleID = "bundle_id"
        case confidence
    }
}

struct MobileScrobbleBatchRequest: Encodable, Sendable {
    let deviceID: String
    let deviceName: String
    let records: [MobileScrobbleUploadRecord]

    enum CodingKeys: String, CodingKey {
        case deviceID = "device_id"
        case deviceName = "device_name"
        case records
    }
}

struct MobileScrobbleItemResultPayload: Decodable, Sendable, Equatable {
    let idempotencyKey: String
    let status: String
    let scrobbled: Bool?
    let reason: String?

    enum CodingKeys: String, CodingKey {
        case idempotencyKey = "idempotency_key"
        case status
        case scrobbled
        case reason
    }
}

struct MobileScrobbleBatchResponse: Decodable, Sendable {
    let status: String
    let acceptedCount: Int
    let duplicateCount: Int
    let scrobbledCount: Int
    let failedCount: Int
    let acceptedKeys: [String]
    let failedKeys: [String]
    let results: [MobileScrobbleItemResultPayload]?

    enum CodingKeys: String, CodingKey {
        case status
        case acceptedCount = "accepted_count"
        case duplicateCount = "duplicate_count"
        case scrobbledCount = "scrobbled_count"
        case failedCount = "failed_count"
        case acceptedKeys = "accepted_keys"
        case failedKeys = "failed_keys"
        case results
    }
}

struct MobileScrobbleHistoryItem: Sendable, Identifiable, Equatable {
    var id: String { idempotencyKey }

    let idempotencyKey: String
    let title: String
    let artist: String
    let album: String
    let playedAtTimestamp: Int64
    let syncedAtTimestamp: Int64
    let status: String
    let scrobbled: Bool

    var captureSourceTitle: String {
        idempotencyKey.hasPrefix("rt:") ? "实时捕获" : "媒体库差量"
    }

    var statusDisplayTitle: String {
        if status == "duplicate" {
            return "已存在（去重）"
        }
        return scrobbled ? "已入库 · Last.fm 已上报" : "已入库 · Last.fm 待补推"
    }
}

struct RealtimeCaptureCheckpoint: Codable, Sendable, Equatable {
    let trackKey: String
    let startedAt: Int64
    let lastPosition: Double
    let capturedAt: Int64
    let thresholdReached: Bool
}
